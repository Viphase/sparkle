package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

type SparkStatus string

const (
	SparkStatusNew         SparkStatus = "new"
	SparkStatusQuestioning SparkStatus = "questioning"
	SparkStatusPromoted    SparkStatus = "promoted"
	SparkStatusArchived    SparkStatus = "archived"
)

func (s SparkStatus) Valid() bool {
	switch s {
	case SparkStatusNew, SparkStatusQuestioning, SparkStatusPromoted, SparkStatusArchived:
		return true
	}
	return false
}

type Spark struct {
	ID                string
	Title             string
	Description       string
	Status            SparkStatus
	Tags              []string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	PromotedProjectID string
}

// NewSparkID returns a sortable, collision-resistant ID rooted at the given
// time. Format: spark_YYYYMMDD_HHMMSS_<6 hex chars>.
func NewSparkID(now time.Time) string {
	var b [3]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fall back to nanoseconds — not perfect but extremely unlikely to
		// collide for human-driven flows.
		return fmt.Sprintf("spark_%s_%06d",
			now.UTC().Format("20060102_150405"),
			now.UTC().Nanosecond()/1000)
	}
	return fmt.Sprintf("spark_%s_%s",
		now.UTC().Format("20060102_150405"),
		hex.EncodeToString(b[:]))
}
