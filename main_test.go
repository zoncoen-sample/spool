package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/gcpug/handy-spanner/fake"
	"github.com/google/uuid"
	"google.golang.org/api/option"
	"google.golang.org/grpc"

	"github.com/zoncoen-sample/spool/models"
)

func TestNewSpannerClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	projectID, instanceID, databaseID := os.Getenv("SPANNER_PROJECT_ID"), os.Getenv("SPANNER_INSTANCE_ID"), os.Getenv("SPANNER_DATABASE_ID")
	if os.Getenv("USE_SPANNER_EMULATOR") != "" {
		projectID, instanceID, databaseID = "fake", "fake", "fake"
		srv, _, err := fake.Run()
		if err != nil {
			t.Fatalf("failed to run handy-spanner: %s", err)
		}
		defer srv.Stop()

		f, err := os.Open("./db/schema.sql")
		if err != nil {
			t.Fatalf("failed to open schema: %s", err)
		}

		dbname := fmt.Sprintf("projects/%s/instances/%s/databases/%s", projectID, instanceID, databaseID)
		srv.ParseAndApplyDDL(ctx, dbname, f)
		if err := os.Setenv("SPANNER_EMULATOR_HOST", srv.Addr()); err != nil {
			t.Fatalf("failed to set env var: %s", err)
		}
	}

	client, err := newSpannerClient(ctx, projectID, instanceID, databaseID)
	if err != nil {
		t.Fatalf("failed to create spanner client: %s", err)
	}

	id := uuid.New().String()
	t.Run("write", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
			txt := &models.Text{
				TextID:    id,
				Body:      "hello",
				CreatedAt: time.Now(),
			}
			return tx.BufferWrite([]*spanner.Mutation{txt.Insert(ctx)})
		}); err != nil {
			t.Fatalf("failed to write: %s", err)
		}
	})
	t.Run("read", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		t.Run("found", func(t *testing.T) {
			tx := client.ReadOnlyTransaction()
			defer tx.Close()
			txt, err := models.FindText(ctx, tx, id)
			if err != nil {
				t.Fatalf("failed to find text: %s", err)
			}
			if got, expect := txt.Body, "hello"; got != expect {
				t.Errorf("expect %s but got %s", expect, got)
			}
			if txt.CreatedAt.IsZero() {
				t.Error("CreatedAt is zero")
			}
		})
		t.Run("not found", func(t *testing.T) {
			tx := client.ReadOnlyTransaction()
			defer tx.Close()
			if _, err := models.FindText(ctx, tx, uuid.New().String()); err == nil {
				t.Fatal("should not be found")
			}
		})
	})
}

func newSpannerClient(ctx context.Context, projectID, instanceID, databaseID string) (*spanner.Client, error) {
	dbname := fmt.Sprintf("projects/%s/instances/%s/databases/%s", projectID, instanceID, databaseID)

	spannerClientOptions := []option.ClientOption{}
	if host := os.Getenv("SPANNER_EMULATOR_HOST"); host != "" {
		conn, err := grpc.DialContext(ctx, host, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			return nil, err
		}
		spannerClientOptions = append(spannerClientOptions, option.WithGRPCConn(conn))
	}

	return spanner.NewClientWithConfig(ctx, dbname,
		spanner.ClientConfig{
			SessionPoolConfig: spanner.SessionPoolConfig{
				MinOpened:     10,
				MaxIdle:       30,
				WriteSessions: 0,
			},
		},
		spannerClientOptions...,
	)
}
