package oauth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dchest/uniuri"
	"github.com/teamkeel/keel/db"
	"github.com/teamkeel/keel/runtime/runtimectx"
	"golang.org/x/crypto/sha3"
)

const (
	// Character length of crypo-generated refresh token
	refreshTokenLength = 64
)

// NewRefreshToken generates a new refresh token for the identity using the
// configured or default expiry time.
func NewRefreshToken(ctx context.Context, identityId string) (string, error) {
	ctx, span := tracer.Start(ctx, "New Refresh Token")
	defer span.End()

	if identityId == "" {
		return "", errors.New("identity ID cannot be empty when generating new refresh token")
	}

	token := uniuri.NewLen(refreshTokenLength)
	hash, err := hashToken(token)
	if err != nil {
		return "", err
	}

	database, err := db.GetDatabase(ctx)
	if err != nil {
		return "", err
	}

	config, err := runtimectx.GetOAuthConfig(ctx)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	expiresAt := now.Add(config.RefreshTokenExpiry())

	sql := `
		INSERT INTO 
			keel_refresh_token (token, identity_id, expires_at, created_at) 
		VALUES 
			(?, ?, ?, ?)`

	db := database.GetDB().Exec(sql, hash, identityId, expiresAt, now)
	if db.Error != nil {
		return "", db.Error
	}

	if db.RowsAffected != 1 {
		return "", errors.New("failed to insert refresh token into database")
	}

	return token, nil
}

// RotateRefreshToken validates that the provided refresh token has not expired,
// and then rotates it for a new refresh token with the exact same expiry time and
// identity. The original refresh token is then revoked from future use.
func RotateRefreshToken(ctx context.Context, refreshTokenRaw string) (isValid bool, refreshToken string, identityId string, err error) {
	ctx, span := tracer.Start(ctx, "Rotate Refresh Token")
	defer span.End()

	tokenHash, err := hashToken(refreshTokenRaw)
	if err != nil {
		return false, "", "", err
	}

	newRefreshToken := uniuri.NewLen(refreshTokenLength)
	newTokenHash, err := hashToken(newRefreshToken)
	if err != nil {
		return false, "", "", err
	}

	database, err := db.GetDatabase(ctx)
	if err != nil {
		return false, "", "", err
	}

	// This query has the following (important) characteristics:
	//  - find and delete the refresh token
	//  - create a new refresh token with the identity_id and expire_at of the original token
	//  - only creates the new token if the original token had not expired
	sql := `
		WITH revoked_token AS (
			DELETE FROM 
				keel_refresh_token
			WHERE 
				token = ?
			RETURNING *)
		INSERT INTO 
			keel_refresh_token (token, identity_id, expires_at, created_at) 
		SELECT
			?, identity_id, expires_at, now()
		FROM 
			revoked_token
		WHERE
			expires_at >= now()
		RETURNING *`

	rows := []map[string]any{}
	err = database.GetDB().Raw(sql, tokenHash, newTokenHash).Scan(&rows).Error
	if err != nil {
		return false, "", "", err
	}

	// There was no refresh token found, and thus nothing to rotate.
	if len(rows) != 1 {
		return false, "", "", nil
	}

	identityId, ok := rows[0]["identity_id"].(string)
	if !ok {
		return false, "", "", errors.New("could not parse identity_id from database result")
	}

	return true, newRefreshToken, identityId, nil
}

// ValidateRefreshToken validates that the provided refresh token has no expired,
// and also returns the identity it is associated with. The refresh token is not revoked.
func ValidateRefreshToken(ctx context.Context, refreshTokenRaw string) (isValid bool, identityId string, err error) {
	ctx, span := tracer.Start(ctx, "Validate Refresh Token")
	defer span.End()

	tokenHash, err := hashToken(refreshTokenRaw)
	if err != nil {
		return false, "", err
	}

	database, err := db.GetDatabase(ctx)
	if err != nil {
		return false, "", err
	}

	sql := `
		SELECT
			token, identity_id, expires_at, now()
		FROM 
			keel_refresh_token
		WHERE 
			token = ? AND
			expires_at >= now()`

	rows := []map[string]any{}
	err = database.GetDB().Raw(sql, tokenHash).Scan(&rows).Error
	if err != nil {
		return false, "", err
	}

	// There was no refresh token found, and thus it is not valid
	if len(rows) != 1 {
		return false, "", nil
	}

	identityId, ok := rows[0]["identity_id"].(string)
	if !ok {
		return false, "", errors.New("could not parse identity_id from database result")
	}

	return true, identityId, nil
}

// RevokeRefreshToken will delete (revoke) the provided refresh token,
// which will prevent it from being used again.
func RevokeRefreshToken(ctx context.Context, refreshTokenRaw string) error {
	ctx, span := tracer.Start(ctx, "Revoke Refresh Token")
	defer span.End()

	tokenHash, err := hashToken(refreshTokenRaw)
	if err != nil {
		return err
	}

	database, err := db.GetDatabase(ctx)
	if err != nil {
		return err
	}

	sql := `
		DELETE FROM 
			keel_refresh_token
		WHERE 
			token = ?`

	err = database.GetDB().Exec(sql, tokenHash).Error
	if err != nil {
		return err
	}

	return nil
}

// hashToken will produce a 256-bit SHA3 hash without salt
func hashToken(input string) (string, error) {
	hash := sha3.New256()
	_, err := hash.Write([]byte(input))
	if err != nil {
		return "", err
	}

	sha3 := hash.Sum(nil)

	return fmt.Sprintf("%x", sha3), nil
}
