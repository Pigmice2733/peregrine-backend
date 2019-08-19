package summary

import (
	"fmt"

	"github.com/pkg/errors"
)

// Report defines a report for a single team in a single match at a single event, which is
// just a list of name/values.
type Report []ReportField

// ReportField defines a single field of a report, a mapping of a name to a value.
type ReportField struct {
	Name  string
	Value float64
}

// ScoreBreakdown defines a TBA score breakdown (changes year to year) which is a mapping of
// strings to JSON values (float64, bool, string).
type ScoreBreakdown map[string]interface{}

// Match defines information relevant to summarizing matches (match key, reports, score
// breakdowns, alliances). RobotPosition should be the one-indexed position of the robot
// on the field, and the score breakdown should be the relevant score breakdown to the
// alliance the robot was on.
type Match struct {
	Key            string
	Reports        []Report
	RobotPosition  int
	ScoreBreakdown ScoreBreakdown
}

// Schema defines a list of schema fields for a schema. The Schema will outline how to summarize
// data from reports, TBA, and computed properties.
type Schema []SchemaField

// FieldDescriptor defines properties of a schema field that aren't related to how it should be
// summarized, but just information about the field (name, period, type).
type FieldDescriptor struct {
	Name string
}

// SchemaField is a singular schema field. Only specify one of: ReportReference, TBAReference,
// Sum, or AnyOf.
type SchemaField struct {
	FieldDescriptor
	ReportReference string
	TBAReference    string
	Sum             []FieldDescriptor
	AnyOf           []EqualExpression
}

// EqualExpression defines a reference that should equal some JSON value (float64, number,
// string).
type EqualExpression struct {
	FieldDescriptor
	Equals interface{}
}

// Summary defines a summarized list of matches.
type Summary []SummaryStat

// SummaryStat defines a single stat in a match.
type SummaryStat struct {
	FieldDescriptor
	Max     float64
	Average float64
}

// SummarizeTeam summarizes a singular team's performance in a single match. The matches
// passed must be ONLY for the team being analyzed and have RobotPosition and ScoreBreakdown
// set properly.
func SummarizeTeam(schema Schema, matches []Match) (Summary, error) {
	records := make(map[string][]float64)

	for _, match := range matches {
		matchRecords, err := summarizeMatch(schema, match)
		if err != nil {
			return Summary{}, errors.Wrap(err, "unable to summarize match")
		}

		for statName, matchRecord := range matchRecords {
			avg := sumJSONValues(matchRecord) / float64(len(matchRecord))
			records[statName] = append(records[statName], avg)
		}
	}

	var summary Summary
	for statName, record := range records {
		stat := SummaryStat{
			FieldDescriptor: FieldDescriptor{Name: statName},
			Max:             max(record),
			Average:         sum(record) / float64(len(record)),
		}

		summary = append(summary, stat)
	}

	return summary, nil
}

func max(values []float64) float64 {
	var max float64
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func sum(values []float64) float64 {
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum
}

func sumJSONValues(values []interface{}) float64 {
	var sum float64
	for _, value := range values {
		switch value := value.(type) {
		case float64:
			sum += value
		case bool:
			if value {
				sum++
			}
		}
	}
	return sum
}

func summarizeMatch(schema Schema, match Match) (map[string][]interface{}, error) {
	records := make(map[string][]interface{})

	for _, statDescription := range schema {
		if statDescription.ReportReference != "" {
			if err := summarizeReportReference(statDescription, match, records); err != nil {
				return nil, errors.Wrap(err, "unable to summarize report reference")
			}
		} else if statDescription.TBAReference != "" {
			// summarizeTBAReference(statDescription, match)
		} else if len(statDescription.Sum) != 0 {
			// summarizeSum(statDescription, match)
		} else if len(statDescription.AnyOf) != 0 {
			// summarizeAnyOf(statDescription, match)
		} else {
			return nil, fmt.Errorf("got invalid stat description: no ReportReference, TBAReference, Sum, or AnyOf")
		}
	}

	return records, nil
}

func summarizeReportReference(statDescription SchemaField, match Match, records map[string][]interface{}) error {
	for _, report := range match.Reports {
		for _, reportField := range report {
			if reportField.Name == statDescription.ReportReference {
				records[statDescription.Name] = append(records[statDescription.Name], reportField.Value)
			}
		}
	}

	return nil
}
