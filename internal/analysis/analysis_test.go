package analysis

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
					{Name: "foobar-teleop", Attempts: newInt(5), Successes: newInt(3)},
				},
				Auto: []store.Stat{
					{Name: "foobar", Attempts: newInt(1), Successes: newInt(0)},
				},
			},
		},
		{
			ID:      1,
			TeamKey: "frc2733",
			Data: store.ReportData{
				Teleop: []store.Stat{
					{Name: "foobar-teleop", Attempts: newInt(8), Successes: newInt(6)},
				},
				Auto: []store.Stat{
					{Name: "foobar", Attempts: newInt(1), Successes: newInt(1)},
				},
			},
		},
		{
			ID:      2,
			TeamKey: "frc2733",
			Data: store.ReportData{
				Teleop: []store.Stat{
					{Name: "foobar-teleop", Attempts: newInt(1), Successes: newInt(0)},
				},
				Auto: []store.Stat{
					{Name: "foobar", Attempts: newInt(5), Successes: newInt(3)},
				},
			},
		},
		{
			ID:      3,
			TeamKey: "frc2471",
			Data: store.ReportData{
				Teleop: []store.Stat{
					{Name: "foobar-teleop", Attempts: newInt(2), Successes: newInt(2)},
				},
				Auto: []store.Stat{
					{Name: "foobar", Attempts: newInt(0), Successes: newInt(1)},
				},
			},
		},
	}

	schema := store.Schema{
		Auto:   []store.StatDescription{{Name: "foobar", Type: "boolean"}},
		Teleop: []store.StatDescription{{Name: "foobar-teleop", Type: "number"}},
	}

	analyzedStats, err := AnalyzeReports(schema, reports, []string{"frc4488"})
	if err != nil {
		t.Errorf("analysis failed with error: %v", err)
	}

	expected := []TeamStats{
		TeamStats{
			Auto: map[string]*stat{"foobar": &stat{
				Attempts:  MaxAvg{Max: 1, Avg: 0.5, Total: 1, Matches: 2},
				Successes: MaxAvg{Max: 0, Avg: 0, Total: 0, Matches: 2},
				Type:      "boolean",
			}},
			Teleop: map[string]*stat{"foobar-teleop": &stat{
				Attempts:  MaxAvg{Max: 5, Avg: 3.5, Total: 7, Matches: 2},
				Successes: MaxAvg{Max: 3, Avg: 2.5, Total: 5, Matches: 2},
				Type:      "number",
			}},
			Team: "frc2471",
		},
		TeamStats{
			Auto: map[string]*stat{"foobar": &stat{
				Attempts:  MaxAvg{Max: 5, Avg: 3, Total: 6, Matches: 2},
				Successes: MaxAvg{Max: 3, Avg: 2, Total: 4, Matches: 2},
				Type:      "boolean",
			}},
			Teleop: map[string]*stat{"foobar-teleop": &stat{
				Attempts:  MaxAvg{Max: 8, Avg: 4.5, Total: 9, Matches: 2},
				Successes: MaxAvg{Max: 6, Avg: 3, Total: 6, Matches: 2},
				Type:      "number",
			}},
			Team: "frc2733",
		},
		TeamStats{
			Auto:   make(map[string]*stat),
			Teleop: make(map[string]*stat),
			Team:   "frc4488",
		},
	}

	assert.ElementsMatch(t, expected, analyzedStats)
}
