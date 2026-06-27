//go:build integration

package mongo_test

import (
	"admin-stats/config"
	"admin-stats/db"
	mongorepo "admin-stats/repository/mongo"
	"context"
	"log"
	"os"
	"testing"

	mongod "go.mongodb.org/mongo-driver/v2/mongo"
)

// testRepo and testDB are initialised once in TestMain and shared across all
// test files in this package. Use testDB for setup/teardown; testRepo is the
// subject under test.
var (
	testRepo *mongorepo.TransactionRepository
	testDB   *mongod.Database
)

// TestMain connects to MongoDB using the same config and db.NewMongoDB path
// as the production server so the test environment is never misconfigured
// independently. Reads conf/test.toml via config.LoadFrom("test").
//
// These tests only compile and run with: go test -tags integration ./repository/mongo/
// A missing config or unreachable MongoDB is a hard failure — the developer
// explicitly opted in so a silent skip would hide a real setup problem.
func TestMain(m *testing.M) {
	cfg, err := config.LoadFrom("test")
	if err != nil {
		log.Fatalf("integration test setup: cannot load test config: %v", err)
	}

	mongoDB := db.NewMongoDB(cfg.MongoConfig)
	ctx := context.Background()
	if err := mongoDB.Connect(ctx); err != nil {
		log.Fatalf("integration test setup: cannot connect to MongoDB (%s): %v", cfg.MongoConfig.URI, err)
	}

	testDB = mongoDB.DB
	testRepo = mongorepo.NewTransactionRepository(testDB)

	code := m.Run()

	_ = testDB.Drop(ctx)
	_ = mongoDB.Disconnect(ctx)
	os.Exit(code)
}
