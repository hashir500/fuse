package store

import (
	"context"
	"math"
	"path/filepath"
	"testing"
	"time"
)

func TestDBPersistsRequests(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	defer db.Close()

	req := RequestLog{
		Provider:         "anthropic",
		Model:            "claude-test",
		PromptTokens:     10,
		CompletionTokens: 20,
		EstimatedCost:    0.00033,
		WasBlocked:       false,
	}

	if err := db.LogRequest(context.Background(), req); err != nil {
		t.Fatal(err)
	}

	requests, err := db.Recent(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	if requests[0].EstimatedCost != 0.00033 {
		t.Fatalf("expected cost 0.00033, got %f", requests[0].EstimatedCost)
	}
}

func TestSpendAccumulatesCorrectly(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	defer db.Close()

	now := time.Now()
	for i := 0; i < 3; i++ {
		if err := db.LogRequest(context.Background(), RequestLog{
			Provider:      "anthropic",
			Model:         "claude-test",
			EstimatedCost: 0.01,
			WasBlocked:    false,
			Timestamp:     now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.LogRequest(context.Background(), RequestLog{
		Provider:      "anthropic",
		Model:         "claude-test",
		EstimatedCost: 0.05,
		WasBlocked:    true,
		Timestamp:     now,
	}); err != nil {
		t.Fatal(err)
	}

	spend, err := db.PeriodSpend(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	expected := 0.03
	if math.Abs(spend.Daily-expected) > 0.0001 {
		t.Fatalf("expected daily spend %.2f, got %.2f", expected, spend.Daily)
	}
}

func setupTestDB(t *testing.T) *Store {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "spend.db"))
	if err != nil {
		t.Fatal(err)
	}
	return db
}
