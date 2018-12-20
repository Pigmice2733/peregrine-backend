package analysis

import (
	"encoding/json"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
)

func newInt(a int) *int {
	return &a
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// SchemaFields stores all the stat fields from a schema
type SchemaFields struct {
	Auto   map[string]string
	Teleop map[string]string
}

// MaxAvg holds a max and average for a specific stat
type MaxAvg struct {
	Max     int     `json:"max"`
	Avg     float32 `json:"avg"`
	Total   int
	Matches int
}

// AnalyzedStat represents a single analyzed statistic
type AnalyzedStat struct {
	Name             string  `json:"statName"`
	NumericAttempts  *MaxAvg `json:"attempts,omitempty"`
	NumericSuccesses *MaxAvg `json:"successes,omitempty"`
	BooleanAttempts  *int    `json:"attempts,omitempty"`
	BooleanSuccesses *int    `json:"successes,omitempty"`
}

// TeamStats holds the performance stats of one team
type TeamStats struct {
	Team   string `json:"team"`
	Auto   map[string]AnalyzedStat
	Teleop map[string]AnalyzedStat
}

// AnalyzeReports analyzes reports based on a schema
func AnalyzeReports(schema store.Schema, eventReports []store.Report) (map[string]TeamStats, error) {
	stats := make(map[string]TeamStats)

	schemaFields, err := getSchemaFields(schema)
	if err != nil {
		return nil, err
	}

	for _, report := range eventReports {
		var data store.ReportData

		if err := json.Unmarshal(report.Data, &data); err != nil {
			return nil, err
		}

		if _, ok := stats[report.TeamKey]; !ok {
			stats[report.TeamKey] = TeamStats{
				Team:   report.TeamKey,
				Auto:   make(map[string]AnalyzedStat),
				Teleop: make(map[string]AnalyzedStat),
			}
		}

		for _, stat := range data.Auto {
			if statType, ok := schemaFields.Auto[stat.Name]; ok {
				if _, ok := stats[report.TeamKey].Auto[stat.Name]; !ok {
					if statType == "boolean" {
						stats[report.TeamKey].Auto[stat.Name] = AnalyzedStat{
							Name:             stat.Name,
							BooleanAttempts:  newInt(0),
							BooleanSuccesses: newInt(0),
						}
					} else if statType == "numeric" {
						succ := MaxAvg{}
						atmpt := MaxAvg{}
						stats[report.TeamKey].Auto[stat.Name] = AnalyzedStat{
							Name:             stat.Name,
							NumericSuccesses: &succ,
							NumericAttempts:  &atmpt,
						}
					}
				}

				if statType == "boolean" {
					if stat.Attempted != nil && *stat.Attempted {
						*stats[report.TeamKey].Auto[stat.Name].BooleanAttempts++
					}
					if stat.Succeeded != nil && *stat.Succeeded {
						*stats[report.TeamKey].Auto[stat.Name].BooleanSuccesses++
					}
				} else if statType == "numeric" {
					stats[report.TeamKey].Auto[stat.Name].NumericAttempts.Matches++
					stats[report.TeamKey].Auto[stat.Name].NumericSuccesses.Matches++

					if stat.Attempts != nil {
						stats[report.TeamKey].Auto[stat.Name].NumericAttempts.Total += *stat.Attempts
						currMax := stats[report.TeamKey].Auto[stat.Name].NumericAttempts.Max
						stats[report.TeamKey].Auto[stat.Name].NumericAttempts.Max = max(*stat.Attempts, currMax)
					}
					if stat.Successes != nil {
						stats[report.TeamKey].Auto[stat.Name].NumericSuccesses.Total += *stat.Successes
						currMax := stats[report.TeamKey].Auto[stat.Name].NumericSuccesses.Max
						stats[report.TeamKey].Auto[stat.Name].NumericSuccesses.Max = max(*stat.Successes, currMax)
					}

					totalAttempts := stats[report.TeamKey].Auto[stat.Name].NumericAttempts.Total
					nAttempts := stats[report.TeamKey].Auto[stat.Name].NumericAttempts.Matches
					stats[report.TeamKey].Auto[stat.Name].NumericAttempts.Avg = float32(totalAttempts) / float32(nAttempts)
					totalSuccesses := stats[report.TeamKey].Auto[stat.Name].NumericSuccesses.Total
					nSuccesses := stats[report.TeamKey].Auto[stat.Name].NumericSuccesses.Matches
					stats[report.TeamKey].Auto[stat.Name].NumericSuccesses.Avg = float32(totalSuccesses) / float32(nSuccesses)
				}
			}
		}
	}

	return stats, nil
}

// getSchemaFields creates a SchemaFields struct from a store.Schema
func getSchemaFields(s store.Schema) (SchemaFields, error) {
	sf := SchemaFields{
		Auto:   make(map[string]string),
		Teleop: make(map[string]string),
	}

	autoFields := []store.StatDescription{}
	if err := json.Unmarshal(s.Auto, &autoFields); err != nil {
		return sf, err
	}

	teleopFields := []store.StatDescription{}
	if err := json.Unmarshal(s.Auto, &teleopFields); err != nil {
		return sf, err
	}

	for _, field := range autoFields {
		sf.Auto[field.Name] = field.Type
	}

	for _, field := range teleopFields {
		sf.Teleop[field.Name] = field.Type
	}

	return sf, nil
}
