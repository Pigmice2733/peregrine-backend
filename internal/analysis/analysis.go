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
	Total   int     `json:"-"`
	Matches int     `json:"-"`
}

type numericStat struct {
	Name      string `json:"statName"`
	Attempts  MaxAvg `json:"attempts"`
	Successes MaxAvg `json:"successes"`
}

type booleanStat struct {
	Name      string `json:"statName"`
	Attempts  int    `json:"attempts"`
	Successes int    `json:"successes"`
}

// TeamStats holds the performance stats of one team
type TeamStats struct {
	Team          string
	AutoNumeric   map[string]*numericStat
	AutoBoolean   map[string]*booleanStat
	TeleopNumeric map[string]*numericStat
	TeleopBoolean map[string]*booleanStat
}

// AnalyzeReports analyzes reports based on a schema
func AnalyzeReports(schema store.Schema, eventReports []store.Report) (map[string]*TeamStats, error) {
	stats := make(map[string]*TeamStats)

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
			rts := TeamStats{
				Team:          report.TeamKey,
				AutoNumeric:   make(map[string]*numericStat),
				AutoBoolean:   make(map[string]*booleanStat),
				TeleopNumeric: make(map[string]*numericStat),
				TeleopBoolean: make(map[string]*booleanStat),
			}
			stats[report.TeamKey] = &rts
		}

		processStatFields(data.Auto, schemaFields.Auto, stats[report.TeamKey].AutoNumeric, stats[report.TeamKey].AutoBoolean)
		processStatFields(data.Teleop, schemaFields.Teleop, stats[report.TeamKey].TeleopNumeric, stats[report.TeamKey].TeleopBoolean)
	}

	return stats, nil
}

// processStatFields processes report statistics based on field types and stores them into numericStat and booleanStat types.
func processStatFields(stats []store.Stat, fields map[string]string, numeric map[string]*numericStat, boolean map[string]*booleanStat) {
	for _, stat := range stats {
		if statType, ok := fields[stat.Name]; ok {
			if statType == "boolean" {
				if _, ok := boolean[stat.Name]; !ok {
					boolStat := booleanStat{
						Name:      stat.Name,
						Attempts:  0,
						Successes: 0,
					}
					boolean[stat.Name] = &boolStat
				}
			} else if statType == "number" {
				if _, ok := numeric[stat.Name]; !ok {
					succ := MaxAvg{}
					atmpt := MaxAvg{}
					numStat := numericStat{
						Name:      stat.Name,
						Successes: succ,
						Attempts:  atmpt,
					}
					numeric[stat.Name] = &numStat
				}
			}

			if statType == "boolean" {
				if stat.Attempted != nil && stat.Succeeded != nil {
					if *stat.Attempted {
						boolean[stat.Name].Attempts++
						if *stat.Succeeded {
							boolean[stat.Name].Successes++
						}
					}
				}
			} else if statType == "number" {
				numeric[stat.Name].Attempts.Matches++
				numeric[stat.Name].Successes.Matches++

				if stat.Attempts != nil && stat.Successes != nil {
					if *stat.Successes > *stat.Attempts {
						*stat.Successes = *stat.Attempts
					}

					numeric[stat.Name].Attempts.Total += *stat.Attempts
					currMax := numeric[stat.Name].Attempts.Max
					numeric[stat.Name].Attempts.Max = max(*stat.Attempts, currMax)

					numeric[stat.Name].Successes.Total += *stat.Successes
					currMax = numeric[stat.Name].Successes.Max
					numeric[stat.Name].Successes.Max = max(*stat.Successes, currMax)
				}

				totalAttempts := numeric[stat.Name].Attempts.Total
				nAttempts := numeric[stat.Name].Attempts.Matches
				numeric[stat.Name].Attempts.Avg = float32(totalAttempts) / float32(nAttempts)
				totalSuccesses := numeric[stat.Name].Successes.Total
				nSuccesses := numeric[stat.Name].Successes.Matches
				numeric[stat.Name].Successes.Avg = float32(totalSuccesses) / float32(nSuccesses)
			}
		}
	}
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
	if err := json.Unmarshal(s.Teleop, &teleopFields); err != nil {
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
