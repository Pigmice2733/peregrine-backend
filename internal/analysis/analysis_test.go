package analysis

import (
	"testing"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
)

func newInt(a int) *int {
	return &a
}

func newBool(a bool) *bool {
	return &a
}

func TestAnalsyzeReports(t *testing.T) {
	reports := []store.Report{
		{
			ID:      0,
			TeamKey: "frc2471",
			Data: store.ReportData{
				Teleop: []store.Stat{
					{Name: "Cubes", Attempts: newInt(5), Successes: newInt(3)},
				},
				Auto: []store.Stat{
					{Name: "Line", Attempts: newInt(1), Successes: newInt(0)},
				},
			},
		},
		{
			ID:      1,
			TeamKey: "frc2733",
			Data: store.ReportData{
				Teleop: []store.Stat{
					{Name: "Cubes", Attempts: newInt(8), Successes: newInt(6)},
				},
				Auto: []store.Stat{
					{Name: "Line", Attempts: newInt(1), Successes: newInt(1)},
				},
			},
		},
		{
			ID:      2,
			TeamKey: "frc2733",
			Data: store.ReportData{
				Teleop: []store.Stat{
					{Name: "Line", Attempts: newInt(1), Successes: newInt(0)},
				},
				Auto: []store.Stat{
					{Name: "Cubes", Attempts: newInt(5), Successes: newInt(3)},
				},
			},
		},
		{
			ID:      3,
			TeamKey: "frc2471",
			Data: store.ReportData{
				Teleop: []store.Stat{
					{Name: "Cubes", Attempts: newInt(2), Successes: newInt(2)},
				},
				Auto: []store.Stat{
					{Name: "Line", Attempts: newInt(0), Successes: newInt(1)},
				},
			},
		},
	}

	schema := store.Schema{
		Auto:   []store.StatDescription{{Name: "Line", Type: "boolean"}},
		Teleop: []store.StatDescription{{Name: "Cubes", Type: "number"}},
	}

	analyzedStats, err := AnalyzeReports(schema, reports)
	if err != nil {
		t.Errorf("analysis failed with error: %v", err)
	}

	if _, ok := analyzedStats["frc2733"]; !ok {
		t.Errorf("analysis doesn't contain team frc2733")
	}
	if _, ok := analyzedStats["frc2471"]; !ok {
		t.Errorf("analysis doesn't contain team frc2471")
	}

	for name, stat := range analyzedStats["frc2471"].Auto {
		if stat.Name != name {
			t.Errorf("stat %s has wrong stat name: %s", name, stat.Name)
		}

		if stat.Name != "Line" {
			attempts := MaxAvg{
				Max:     1,
				Avg:     0.5,
				Total:   1,
				Matches: 2,
			}

			successes := MaxAvg{
				Max:     0,
				Avg:     0,
				Total:   0,
				Matches: 2,
			}
			if stat.Attempts != attempts || stat.Successes != successes {
				t.Errorf("analysis for frc2471 'Line' is wrong")
			}
		}
	}

	for name, stat := range analyzedStats["frc2471"].Teleop {
		if stat.Name != name {
			t.Errorf("stat %s has wrong stat name: %s", name, stat.Name)
		}

		if stat.Name == "Cubes" {
			attempts := MaxAvg{
				Max:     5,
				Avg:     3.5,
				Total:   7,
				Matches: 2,
			}

			successes := MaxAvg{
				Max:     3,
				Avg:     2.5,
				Total:   5,
				Matches: 2,
			}
			if stat.Attempts != attempts || stat.Successes != successes {
				t.Errorf("analysis for frc2471 'Cubes' is wrong")
			}
		}
	}
}
