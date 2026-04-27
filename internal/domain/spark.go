package domain

import "time"

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
