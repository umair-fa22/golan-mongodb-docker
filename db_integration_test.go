package main

import (
    "context"
    "database/sql"
    "os"
    "testing"
    "time"

    _ "github.com/lib/pq"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

// TestMongoIntegration connects to MongoDB using MONGODB_URI and performs a small insert/find.
func TestMongoIntegration(t *testing.T) {
    uri := os.Getenv("MONGODB_URI")
    if uri == "" {
        t.Skip("MONGODB_URI not set; skipping MongoDB integration test")
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
    if err != nil {
        t.Fatalf("mongo connect error: %v", err)
    }
    defer client.Disconnect(ctx)

    if err := client.Ping(ctx, nil); err != nil {
        t.Fatalf("mongo ping failed: %v", err)
    }

    coll := client.Database("test").Collection("ci_items")
    _, err = coll.InsertOne(ctx, bson.M{"name": "ci-test", "value": 1})
    if err != nil {
        t.Fatalf("mongo insert failed: %v", err)
    }

    var out bson.M
    if err := coll.FindOne(ctx, bson.M{"name": "ci-test"}).Decode(&out); err != nil {
        t.Fatalf("mongo find failed: %v", err)
    }

    // cleanup
    _, _ = coll.DeleteMany(ctx, bson.M{"name": "ci-test"})
}

// TestPostgresIntegration connects to Postgres using POSTGRES_DSN and performs simple DDL/DML.
func TestPostgresIntegration(t *testing.T) {
    dsn := os.Getenv("POSTGRES_DSN")
    if dsn == "" {
        t.Skip("POSTGRES_DSN not set; skipping Postgres integration test")
    }

    db, err := sql.Open("postgres", dsn)
    if err != nil {
        t.Fatalf("postgres open error: %v", err)
    }
    defer db.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        t.Fatalf("postgres ping failed: %v", err)
    }

    _, err = db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS ci_test (id SERIAL PRIMARY KEY, name TEXT)")
    if err != nil {
        t.Fatalf("create table failed: %v", err)
    }

    _, err = db.ExecContext(ctx, "INSERT INTO ci_test (name) VALUES ($1)", "ci-test")
    if err != nil {
        t.Fatalf("insert failed: %v", err)
    }

    var name string
    err = db.QueryRowContext(ctx, "SELECT name FROM ci_test WHERE name = $1 LIMIT 1", "ci-test").Scan(&name)
    if err != nil {
        t.Fatalf("select failed: %v", err)
    }

    if name != "ci-test" {
        t.Fatalf("unexpected name: %v", name)
    }

    // cleanup
    _, _ = db.ExecContext(ctx, "DELETE FROM ci_test WHERE name = $1", "ci-test")
}
