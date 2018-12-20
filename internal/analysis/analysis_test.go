package analysis_test

import (
	"testing"

	"github.com/Pigmice2733/peregrine-backend/internal/analysis"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
)

func TestAnalsyzeReports(t *testing.T) {
	reports := []store.Report{
		store.Report{
			ID:      0,
			TeamKey: "frc2471",
			Data: []byte(`{
				"teleop": [{"statName": "Cubes", "attempts": 5, "successes": 3}],
				"auto": [{"statName": "Line", "attempted": true, "succeeded": false}]
			}`),
		},
		store.Report{
			ID:      1,
			TeamKey: "frc2733",
			Data: []byte(`{
				"teleop": [{"statName": "Cubes", "attempts": 8, "successes": 6}],
				"auto": [{"statName": "Line", "attempted": true, "succeeded": true}]
			}`),
		},
		store.Report{
			ID:      2,
			TeamKey: "frc2733",
			Data: []byte(`{
				"teleop": [{"statName": "Line", "attempted": true, "succeeded": false}],
				"auto": [{"statName": "Cubes", "attempts": 5, "successes": 3}]
			}`),
		},
		store.Report{
			ID:      3,
			TeamKey: "frc2471",
			Data: []byte(`{
				"teleop": [{"statName": "Cubes", "attempts": 2, "successes": 2}],
				"auto": [{"statName": "Line", "attempted": true, "succeeded": true}]
			}`),
		},
	}

	schema := store.Schema{
		Auto:   []byte(`[{"name": "Line", "type": "boolean"}]`),
		Teleop: []byte(`[{"name: "Cubes", "type": "number"}]`),
	}

	analyzedStats, err := analysis.AnalyzeReports(schema, reports)
	if err != nil {
		t.Errorf("analysis failed with error: %v", err)
	}

	if _, ok := analyzedStats["frc2733"]; !ok {
		t.Errorf("analysis doesn't contain team frc2733")
	}
	if _, ok := analyzedStats["frc2471"]; !ok {
		t.Errorf("analysis doesn't contain team frc2471")
	}

	if analyzedStats["frc2733"].Team != "frc2733" {
		t.Errorf("analysis for team frc2733 has wrong team key")
	}
	if analyzedStats["frc2471"].Team != "frc2471" {
		t.Errorf("analysis for team frc2471 has wrong team key")
	}

	for name, stat := range analyzedStats["frc2471"].Auto {
		if stat.Name != name {
			t.Errorf("stat %s has wrong stat name: %s", name, stat.Name)
		}

		if stat.Name == "Line" {
			if stat.BooleanAttempts == nil || stat.BooleanSuccesses == nil || *stat.BooleanAttempts != 2 || *stat.BooleanSuccesses != 1 {
				t.Errorf("analysis for frc2471 'Line' is wrong")
			}
			if stat.NumericAttempts != nil || stat.NumericSuccesses != nil {
				t.Errorf("analysis for frc2471 boolean stat has non-nil numeric stats")
			}
		} else if stat.Name == "Cubes" {
			attempts := analysis.MaxAvg{
				Max:     5,
				Avg:     3.5,
				Total:   7,
				Matches: 2,
			}

			successes := analysis.MaxAvg{
				Max:     3,
				Avg:     2.5,
				Total:   5,
				Matches: 2,
			}
			if stat.NumericAttempts == nil || stat.NumericSuccesses == nil || *stat.NumericAttempts != attempts || *stat.NumericSuccesses != successes {
				t.Errorf("analysis for frc2471 'Cubes' is wrong")
			}
			if stat.BooleanAttempts != nil || stat.BooleanSuccesses != nil {
				t.Errorf("analysis for frc2471 numeric stat has non-nil boolean stats")
			}
		}
	}

	for name, stat := range analyzedStats["frc2733"].Auto {
		if stat.Name != name {
			t.Errorf("stat %s has wrong stat name: %s", name, stat.Name)
		}

		if stat.Name == "Line" {
			if stat.BooleanAttempts == nil || stat.BooleanSuccesses == nil || *stat.BooleanAttempts != 1 || *stat.BooleanSuccesses != 1 {
				t.Errorf("analysis for frc2733 'Line' is wrong")
			}
			if stat.NumericAttempts != nil || stat.NumericSuccesses != nil {
				t.Errorf("analysis for frc2733 boolean stat has non-nil numeric stats")
			}
		} else if stat.Name == "Cubes" {
			attempts := analysis.MaxAvg{
				Max:     8,
				Avg:     8,
				Total:   8,
				Matches: 1,
			}

			successes := analysis.MaxAvg{
				Max:     6,
				Avg:     6,
				Total:   6,
				Matches: 1,
			}
			if stat.NumericAttempts == nil || stat.NumericSuccesses == nil || *stat.NumericAttempts != attempts || *stat.NumericSuccesses != successes {
				t.Errorf("analysis for frc2733 'Cubes' is wrong")
			}
			if stat.BooleanAttempts != nil || stat.BooleanSuccesses != nil {
				t.Errorf("analysis for frc2733 numeric stat has non-nil boolean stats")
			}
		}
	}
}
