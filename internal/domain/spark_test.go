package domain

import (
	"strings"
	"testing"
	"time"
)

func TestSparkStatusValid(t *testing.T) {
	valid := []SparkStatus{
		SparkStatusNew,
		SparkStatusQuestioning,
		SparkStatusPromoted,
		SparkStatusArchived,
	}
	for _, s := range valid {
		if !s.Valid() {
			t.Errorf("expected %q to be valid", s)
		}
	}
	if SparkStatus("bogus").Valid() {
		t.Error("expected unknown status to be invalid")
	}
}

func TestNewSparkIDFormat(t *testing.T) {
	now := time.Date(2026, 4, 27, 14, 5, 30, 0, time.UTC)
	id := NewSparkID(now)
	if !strings.HasPrefix(id, "spark_20260427_140530_") {
		t.Errorf("unexpected prefix: %s", id)
	}
	// 6 hex chars after the second underscore-segment boundary.
	parts := strings.Split(id, "_")
	if len(parts) != 4 {
		t.Fatalf("expected 4 underscore-separated parts, got %d (%s)", len(parts), id)
	}
	if len(parts[3]) != 6 {
		t.Errorf("suffix should be 6 chars, got %q", parts[3])
	}
}

func TestNewSparkIDStable(t *testing.T) {
	now := time.Now()
	a := NewSparkID(now)
	b := NewSparkID(now)
	if a == b {
		t.Errorf("two IDs from the same instant should differ; both = %s", a)
	}
}
