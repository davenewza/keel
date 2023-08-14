package auth

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/patrickmn/go-cache"
	"github.com/pquerna/cachecontrol"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type AuthConfig struct {
	// If enabled, skips signing validation of tokens
	AllowUnsigned bool `json:"AllowUnsigned"`
	// If enabled, will verify tokens using any OIDC compatible issuer
	AllowAnyIssuers bool             `json:"AllowAllIssuers"`
	Issuers         []ExternalIssuer `json:"issuers"`
	Keel            *KeelAuthConfig  `json:"keel"`
}

type KeelAuthConfig struct {
	// Allow new identities to be created through the authenticate endpoint
	AllowCreate bool `json:"allowCreate"`
	// In seconds
	TokenDuration int `json:"tokenDuration"`
}

type ExternalIssuer struct {
	Iss      string  `json:"iss"`
	Audience *string `json:"audience"`
}

type OpenidConfig struct {
	Issuer   string `json:"issuer"`
	AuthURL  string `json:"authorization_endpoint"`
	TokenURL string `json:"token_endpoint"`

	JWKSURL     string   `json:"jwks_uri"`
	UserInfoURL string   `json:"userinfo_endpoint"`
	Algorithms  []string `json:"id_token_signing_alg_values_supported"`
}

type UserInfo struct {
	Subject       string `json:"sub"`
	Profile       string `json:"profile"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`

	// OIDC Standard claims (non-exhaustive)
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Name       string `json:"name"`
	Picture    string `json:"picture"`
	Gender     string `json:"gender"`
	Zoneinfo   string `json:"zoneinfo"`
	Locale     string `json:"locale"`
	UpdatedAt  string `json:"updated_at"`

	Claims []byte
}

var tracer = otel.Tracer("github.com/teamkeel/keel/auth")

var (
	HttpClient   HTTPClient
	RequestCache *cache.Cache
	JwkCache     *jwk.Cache
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
	Get(string) (*http.Response, error)
}

func init() {
	HttpClient = &http.Client{}
	RequestCache = cache.New(5*time.Minute, 10*time.Minute)
	JwkCache = jwk.NewCache(context.Background())
}

// Loads OIDC config and JWKS into cache for each issuer and drops any incompatible provider
func CheckIssuers(ctx context.Context, issuers []ExternalIssuer) []ExternalIssuer {

	if len(issuers) == 0 {
		return issuers
	}

	ctx, span := tracer.Start(ctx, "ODIC")
	defer span.End()

	validIssuers := []ExternalIssuer{}

	for _, issuer := range issuers {

		ctx, issSpan := tracer.Start(ctx, issuer.Iss)

		oidc, err := GetOpenIDConnectConfig(ctx, issuer.Iss)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"issuer": issuer,
				"error":  err,
			}).Error("Failed to load OIDC config")

			issSpan.RecordError(err, trace.WithStackTrace(true))
			issSpan.SetStatus(codes.Error, err.Error())
			continue
		}

		issSpan.SetAttributes(attribute.String("JWKS url", oidc.JWKSURL))

		err = JwkCache.Register(oidc.JWKSURL, jwk.WithHTTPClient(HttpClient))
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"url":    oidc.JWKSURL,
				"issuer": issuer,
				"error":  err,
			}).Error("Couldn't register JWKS")

			issSpan.RecordError(err, trace.WithStackTrace(true))
			issSpan.SetStatus(codes.Error, err.Error())
			continue
		}

		// Check the url is valid
		if _, err = JwkCache.Refresh(ctx, oidc.JWKSURL); err != nil {
			logrus.WithFields(logrus.Fields{
				"url":    oidc.JWKSURL,
				"issuer": issuer,
				"error":  err,
			}).Error("Couldn't validate JWKS from cache")

			issSpan.RecordError(err, trace.WithStackTrace(true))
			issSpan.SetStatus(codes.Error, err.Error())
			continue
		}

		validIssuers = append(validIssuers, issuer)

		issSpan.End()
	}

	return validIssuers
}

func GetOpenIDConnectConfig(ctx context.Context, issuer string) (*OpenidConfig, error) {

	span := trace.SpanFromContext(ctx)

	trimmed := strings.TrimSuffix(issuer, "/")

	configUrl := trimmed + "/.well-known/openid-configuration"

	span.AddEvent("Fetching OIDC configuration",
		trace.WithAttributes(
			attribute.String("issuer", issuer),
			attribute.String("url", configUrl),
		))

	req, err := http.NewRequest("GET", configUrl, nil)
	if err != nil {
		return nil, err
	}
	body, _, err := cachedRequest(ctx, req.URL.String(), req)
	if err != nil {
		return nil, err
	}

	config := &OpenidConfig{}
	err = json.Unmarshal(body, config)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal: %s", err)
	}

	if issuer != config.Issuer {
		return nil, fmt.Errorf("oidc issuer did not match the issuer returned by provider, expected %q got %q", config.Issuer, issuer)
	}

	return config, nil

}

func GetUserInfo(ctx context.Context, issuer string, token string) (*UserInfo, error) {

	sub, err := extractSubFromToken(token)
	if err != nil {
		return nil, err
	}

	oidc, err := GetOpenIDConnectConfig(ctx, issuer)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", oidc.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	body, _, err := cachedRequest(ctx, fmt.Sprintf("%s-%s", req.URL.String(), sub), req)
	if err != nil {
		return nil, fmt.Errorf("Fetch failed: %s", err)
	}

	userInfo := &UserInfo{}
	err = json.Unmarshal(body, userInfo)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal: %s", err)
	}

	return userInfo, nil

}

func extractSubFromToken(token string) (string, error) {
	// Parse the JWT without verifying the signature
	t, _, err := new(jwt.Parser).ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return "", fmt.Errorf("Error parsing JWT: %s", err)
	}

	// Extract the "sub" claim
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("Claims not found")

	}

	subClaim, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("Sub claim not found or not a string")
	}

	return subClaim, nil
}

func GetJWKS(ctx context.Context, issuer string) (jwk.Set, error) {

	var emptySet jwk.Set
	odic, err := GetOpenIDConnectConfig(ctx, issuer)
	if err != nil {
		return emptySet, nil
	}

	return JwkCache.Get(ctx, odic.JWKSURL)
}

func cachedRequest(ctx context.Context, key string, req *http.Request) (body []byte, cacheHit bool, err error) {

	span := trace.SpanFromContext(ctx)

	if cached, found := RequestCache.Get(key); found {
		span.SetAttributes(attribute.String("cache", "hit"))
		cachedBody := cached.([]byte)
		return cachedBody, true, nil
	}

	span.SetAttributes(attribute.String("cache", "miss"))

	resp, err := HttpClient.Do(req)
	if err != nil {
		return []byte{}, cacheHit, err
	}
	defer resp.Body.Close()

	span.AddEvent("ODIC request", trace.WithAttributes(
		attribute.String("url", req.URL.String()),
		attribute.Int("statusCode", resp.StatusCode),
	))

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, cacheHit, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("failed to fetch url: %s  Status: %d  ", req.URL.String(), resp.StatusCode)
	}

	// Cache the response based on the cache control information
	reasons, expires, err := cachecontrol.CachableResponse(req, resp, cachecontrol.Options{})
	if err == nil {
		shouldCache := len(reasons) == 0

		if shouldCache {
			duration := time.Until(expires)
			RequestCache.Set(key, body, duration)
		}
	}

	return body, cacheHit, nil
}

func PublicKeyForIssuer(ctx context.Context, issuerUri string, tokenKid string) (*rsa.PublicKey, error) {
	jwks, err := GetJWKS(ctx, issuerUri)

	if err != nil {
		return nil, err
	}

	publicKey, err := ExtractJWKSPublicKey(ctx, jwks, tokenKid)

	if err != nil {
		return nil, err
	}

	return publicKey, nil
}

func ExtractJWKSPublicKey(ctx context.Context, jwks jwk.Set, tokenKid string) (*rsa.PublicKey, error) {
	allKeys := jwks.Keys(ctx)
	found := false
	var publicKey rsa.PublicKey

	span := trace.SpanFromContext(ctx)

	if jwks.Len() > 1 && tokenKid == "" {
		span.AddEvent("Multiple jwks but no kid in token, using first jwk")
	}

	for allKeys.Next(ctx) {
		curr := allKeys.Pair()

		switch v := curr.Value.(type) {
		case jwk.RSAPublicKey:
			kid := v.KeyID()

			if tokenKid == "" || tokenKid == kid {
				found = true
				err := v.Raw(&publicKey)
				if err != nil {
					found = false

				}

				if found {
					break
				}
			}
		}
	}

	if !found {
		return nil, errors.New("No RSA public key found")
	}

	return &publicKey, nil
}
