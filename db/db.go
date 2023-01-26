package db

import (
	"context"
	"database/sql"
	"errors"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rdsdata"
	_ "github.com/lib/pq"
	"github.com/teamkeel/keel/cmd/database"
)

type ExecuteQueryResult struct {
	Rows []map[string]any
}

type ExecuteStatementResult struct {
	RowsAffected int64
}

type Db interface {
	// Executes SQL query statement and returns rows.
	ExecuteQuery(ctx context.Context, sql string, values ...any) (*ExecuteQueryResult, error)
	// Executes SQL statement and returns number of rows affected.
	ExecuteStatement(ctx context.Context, sql string, values ...any) (*ExecuteStatementResult, error)
	// Begins a new transaction.
	BeginTransaction(ctx context.Context) error
	// Commits the current transaction.
	CommitTransaction(ctx context.Context) error
	// Rolls back the current transaction.
	RollbackTransaction(ctx context.Context) error
}

// Local data operations for a local database connection.
func Local(ctx context.Context, dbConnInfo *database.ConnectionInfo) (Db, error) {
	sqlDb, err := sql.Open("postgres", dbConnInfo.String())
	if err != nil {
		return nil, err
	}
	return &localDb{conn: sqlDb}, nil
}

// LocalFromConnection using an existing sql.DB connection to provide data operations to a database.
// Typically used for testing where it makes sense to reuse an existing connection.
func LocalFromConnection(ctx context.Context, sqlDb *sql.DB) (Db, error) {
	return &localDb{conn: sqlDb}, nil
}

// DataAPI provides data operations for an Amazon Aurora database.
func DataAPI(ctx context.Context) (Db, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := rdsdata.NewFromConfig(cfg)
	dbResourceArn := os.Getenv("DB_RESOURCE_ARN")
	if dbResourceArn == "" {
		return nil, errors.New("the DB_RESOURCE_ARN env var is not set")
	}
	dbSecretArn := os.Getenv("DB_SECRET_ARN")
	if dbSecretArn == "" {
		return nil, errors.New("the DB_SECRET_ARN env var is not set")
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		return nil, errors.New("the DB_NAME env var is not set")
	}
	return &dataApi{
		client:               client,
		dbResourceArn:        dbResourceArn,
		dbSecretArn:          dbSecretArn,
		dbName:               dbName,
		ongoingTransactionId: nil,
	}, nil
}
