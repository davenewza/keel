package testhelpers

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	cp "github.com/otiai10/copy"
	"github.com/teamkeel/keel/db"
	"github.com/teamkeel/keel/migrations"
	"go.opentelemetry.io/otel/trace"

	"github.com/teamkeel/keel/proto"
)

// WithTmpDir copies the contents of the src dir to a new temporary directory, returning the tmp dir path
func WithTmpDir(dir string) (string, error) {
	base := filepath.Base(dir)

	tmpDir, err := os.MkdirTemp("", base)

	if err != nil {
		return "", err
	}

	err = cp.Copy(dir, tmpDir)

	if err != nil {
		return "", err
	}

	return tmpDir, nil
}

func NpmInstall(dir string) (string, error) {
	npmInstall := exec.Command("npm", "install", "--progress=false")
	npmInstall.Dir = dir

	b, err := npmInstall.CombinedOutput()

	if err != nil {
		return "", err
	}

	return string(b), err
}

func SetupDatabaseForTestCase(ctx context.Context, dbConnInfo *db.ConnectionInfo, schema *proto.Schema, dbName string, resetDatabase bool) (db.Database, error) {
	mainDB, err := sql.Open("pgx/v5", dbConnInfo.String())
	if err != nil {
		return nil, err
	}
	defer mainDB.Close()

	_, err = mainDB.Exec("select pg_terminate_backend(pg_stat_activity.pid) from pg_stat_activity where datname = '" + dbName + "' and pg_stat_activity.pid <> pg_backend_pid();")
	if err != nil {
		return nil, err
	}

	if resetDatabase {
		// Drop the database if it already exists. The normal dropping of it at the end of the
		// test case is bypassed if you quit a debug run of the test in VS Code.
		_, err = mainDB.Exec("DROP DATABASE if exists " + dbName)
		if err != nil {
			return nil, err
		}

		// Create the database and drop at the end of the test
		_, err = mainDB.Exec("CREATE DATABASE " + dbName)
		if err != nil {
			return nil, err
		}
	}

	// Connect to the newly created test database and close connection
	// at the end of the test. We need to explicitly close the connection
	// so the mainDB connection can drop the database.
	testDBConnInfo := dbConnInfo.WithDatabase(dbName)
	fmt.Println(testDBConnInfo.String())
	database, err := db.New(ctx, testDBConnInfo.String())
	if err != nil {
		return nil, err
	}

	// Migrate the database to this test case's schema.
	m, err := migrations.New(ctx, schema, database)
	if err != nil {
		return nil, err
	}

	err = m.Apply(ctx)
	if err != nil {
		return nil, err
	}

	return database, nil
}

func DbNameForTestName(testName string) string {
	re := regexp.MustCompile(`[^\w]`)
	return strings.ToLower(re.ReplaceAllString(testName, ""))
}

//go:embed default.pem
var defaultPem []byte

func GetEmbeddedPrivateKey() (*rsa.PrivateKey, error) {

	privateKeyBlock, err := pem.Decode(defaultPem)
	if privateKeyBlock == nil {
		return nil, fmt.Errorf("private key PEM either invalid or empty: %s", err)
	}

	return x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
}

const (
	TraceId = "71f835dc7ac2750bed2135c7b30dc7fe"
	SpanId  = "b4c9e2a6a0d84702"
)

func WithTracing(ctx context.Context) (context.Context, error) {
	traceIdBytes, err := hex.DecodeString(TraceId)
	if err != nil {
		return nil, err
	}

	spanIdBytes, err := hex.DecodeString(SpanId)
	if err != nil {
		return nil, err
	}

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID(traceIdBytes),
		SpanID:     trace.SpanID(spanIdBytes),
		TraceFlags: trace.FlagsSampled,
	})

	if !spanContext.IsValid() {
		return nil, errors.New("spanContext is unexpectedly invalid")
	}

	return trace.ContextWithSpanContext(ctx, spanContext), nil
}
