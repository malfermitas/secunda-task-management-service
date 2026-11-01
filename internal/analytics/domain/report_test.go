package domain

import (
	"encoding/json"
	"testing"
)

func TestTeamStatsAvgCloseSecondsJSON(t *testing.T) {
	var stats TeamStats
	if err := json.Unmarshal([]byte(`{"avg_close_seconds":null}`), &stats); err != nil {
		t.Fatalf("Unmarshal(null) error = %v", err)
	}
	if stats.AvgCloseSeconds.IsSome() {
		t.Fatalf("AvgCloseSeconds = %v, want None", stats.AvgCloseSeconds)
	}

	if err := json.Unmarshal([]byte(`{"avg_close_seconds":12.5}`), &stats); err != nil {
		t.Fatalf("Unmarshal(number) error = %v", err)
	}
	value, ok := stats.AvgCloseSeconds.TryUnwrap()
	if !ok || value != 12.5 {
		t.Fatalf("AvgCloseSeconds = %v, want Some(12.5)", stats.AvgCloseSeconds)
	}
}
