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
	Attempts  MaxAvg `json:"attempts"`
	Successes MaxAvg `json:"successes"`
	Type      string `json:"type"`
}

// TeamStats holds the performance stats of one team
type TeamStats struct {
	Auto   map[string]*stat `json:"auto"`
	Teleop map[string]*stat `json:"teleop"`
	Team   string           `json:"team"`
}

// AnalyzeReports analyzes reports based on a schema
func AnalyzeReports(schema store.Schema, eventReports []store.Report, teamKeys []string) ([]TeamStats, error) {
	stats := make(map[string]TeamStats)

	concatedSchema := make(map[string]store.StatDescription)
	for _, v := range schema.Auto {
		concatedSchema[v.Name] = v
	}
	for _, v := range schema.Teleop {
		concatedSchema[v.Name] = v
	}

	for _, report := range eventReports {
		if _, ok := stats[report.TeamKey]; !ok {
			stats[report.TeamKey] = TeamStats{
				Auto:   make(map[string]*stat),
				Teleop: make(map[string]*stat),
			}
		}

		processStatFields(report.Data.Auto, concatedSchema, stats[report.TeamKey].Auto)
		processStatFields(report.Data.Teleop, concatedSchema, stats[report.TeamKey].Teleop)
	}

	for _, team := range teamKeys {
		if _, ok := stats[team]; !ok {
			stats[team] = TeamStats{Auto: make(map[string]*stat), Teleop: make(map[string]*stat)}
		}
	}

	statsList := make([]TeamStats, 0)
	for team, stats := range stats {
		stats.Team = team
		statsList = append(statsList, stats)
	}

	return statsList, nil
}

func processStatFields(data []store.Stat, schema map[string]store.StatDescription, stats map[string]*stat) {
	for _, datum := range data {
		if schemaField, ok := schema[datum.Name]; ok {
			if _, ok := stats[datum.Name]; !ok {
				stats[datum.Name] = &stat{
					Successes: MaxAvg{},
					Attempts:  MaxAvg{},
					Type:      schemaField.Type,
				}
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
