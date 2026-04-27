package domain

import "testing"

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
