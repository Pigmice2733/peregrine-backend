package analysis

import (
	"github.com/Pigmice2733/peregrine-backend/internal/store"
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MaxAvg holds a max and average for a specific stat
type MaxAvg struct {
	Max     int     `json:"max"`
	Avg     float64 `json:"avg"`
	Total   int     `json:"-"`
	Matches int     `json:"-"`
}

func (ma *MaxAvg) update(n *int) {
	if n != nil {
		ma.Total += *n
		ma.Max = max(*n, ma.Max)
	}
	ma.Avg = float64(ma.Total) / float64(ma.Matches)
}

type stat struct {
	Name      string `json:"name"`
	Attempts  MaxAvg `json:"attempts"`
	Successes MaxAvg `json:"successes"`
}

// TeamStats holds the performance stats of one team
type TeamStats struct {
	Team   string
	Auto   map[string]*stat
	Teleop map[string]*stat
}

// AnalyzeReports analyzes reports based on a schema
func AnalyzeReports(schema store.Schema, eventReports []store.Report) (map[string]*TeamStats, error) {
	stats := make(map[string]*TeamStats)

	fields, err := getSchemaFields(schema)
	if err != nil {
		return nil, err
	}

	for _, report := range eventReports {
		if _, ok := stats[report.TeamKey]; !ok {
			rts := TeamStats{
				Team:   report.TeamKey,
				Auto:   make(map[string]*stat),
				Teleop: make(map[string]*stat),
			}
			stats[report.TeamKey] = &rts
		}

		processStatFields(report.Data.Auto, fields, stats[report.TeamKey].Auto)
		processStatFields(report.Data.Teleop, fields, stats[report.TeamKey].Teleop)
	}

	return stats, nil
}

// processStatFields processes report statistics based on field types and stores them into numericStat and booleanStat types.
func processStatFields(data []store.Stat, fields map[string]bool, stats map[string]*stat) {
	for _, datum := range data {
		if _, ok := fields[datum.Name]; ok {
			if _, ok := stats[datum.Name]; !ok {
				s := stat{
					Name:      datum.Name,
					Successes: MaxAvg{},
					Attempts:  MaxAvg{},
				}
				stats[datum.Name] = &s
			}

			stats[datum.Name].Attempts.Matches++
			stats[datum.Name].Successes.Matches++

			if datum.Attempts != nil && datum.Successes != nil {
				if *datum.Successes > *datum.Attempts {
					*datum.Successes = *datum.Attempts
				}
			}

			stats[datum.Name].Attempts.update(datum.Attempts)
			stats[datum.Name].Successes.update(datum.Successes)
		}
	}
}

// getSchemaFields creates a SchemaFields struct from a store.Schema
func getSchemaFields(s store.Schema) (map[string]bool, error) {
	fields := make(map[string]bool)

	for _, field := range s.Auto {
		fields[field.Name] = true
	}

	for _, field := range s.Teleop {
		fields[field.Name] = true
	}

	return fields, nil
}
