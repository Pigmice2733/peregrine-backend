package summary

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
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
			return Summary{}, fmt.Errorf("unable to summarize match: %w", err)
		}

		for statName, matchRecord := range matchRecords {
			// if there are multiple reports for one match we need to
			// average them so one match isn't weighted twice as much
			// as another if it has two reports

			var sum float64
			for _, reportGroup := range matchRecord {
				sum += sumJSONValues(reportGroup)
			}

			records[statName] = append(records[statName], sum/float64(len(matchRecord)))
		}
	}

	summary := make(Summary, 0)
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

// mapping of stat names to a list of report values: list of JSON values
// (float64, bool, string)
type rawRecords map[string][][]interface{}

func summarizeMatch(schema Schema, match Match) (rawRecords, error) {
	records := make(rawRecords)

	for _, statDescription := range schema {
		if statDescription.ReportReference != "" {
			if err := summarizeReportReference(statDescription, match, records); err != nil {
				return nil, fmt.Errorf("unable to summarize report reference: %w", err)
			}
		} else if statDescription.TBAReference != "" {
			if err := summarizeTBAReference(statDescription, match, records); err != nil {
				return nil, fmt.Errorf("unable to summarize TBA reference: %w", err)
			}
		} else if len(statDescription.Sum) != 0 {
			if err := summarizeSum(statDescription, match, records); err != nil {
				return nil, fmt.Errorf("unable to summarize sum stat: %w", err)
			}
		} else if len(statDescription.AnyOf) != 0 {
			if err := summarizeAnyOf(statDescription, match, records); err != nil {
				return nil, fmt.Errorf("unable to summarize any of stat: %w", err)
			}
		} else {
			return nil, errors.New("got invalid stat description: no ReportReference, TBAReference, Sum, or AnyOf")
		}
	}

	return records, nil
}

func summarizeReportReference(statDescription SchemaField, match Match, records rawRecords) error {
	for _, report := range match.Reports {
		var reportGroup []interface{}
		for _, reportField := range report {
			if reportField.Name == statDescription.ReportReference {
				reportGroup = append(reportGroup, reportField.Value)
			}
		}
		records[statDescription.Name] = append(records[statDescription.Name], reportGroup)
	}

	return nil
}

type templateData struct {
	RobotPosition int
}

func summarizeTBAReference(statDescription SchemaField, match Match, records rawRecords) error {
	tmpl, err := template.New("key").Parse(statDescription.TBAReference)
	if err != nil {
		return fmt.Errorf("unable to parse tba reference template: %w", err)
	}

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, templateData{RobotPosition: match.RobotPosition}); err != nil {
		return fmt.Errorf("unable to execute template: %w", err)
	}

	value, ok := match.ScoreBreakdown[buf.String()]
	if !ok {
		return nil
	}

	records[statDescription.Name] = append(records[statDescription.Name], []interface{}{value})

	return nil
}

func summarizeSum(statDescription SchemaField, match Match, records rawRecords) error {
	var sum float64

	for _, ref := range statDescription.Sum {
		refRecords := records[ref.Name]

		if len(refRecords) == 0 {
			// we can't resolve one of the records, return
			// this can happen if a match is missing a report field, or if
			// it's a match that only has TBA data and no reports (if you
			// are summing report data)
			return nil
		}

		var statsum float64
		for _, reportGroup := range refRecords {
			statsum += sumJSONValues(reportGroup)
		}

		sum += statsum / float64(len(refRecords))
	}

	records[statDescription.Name] = append(records[statDescription.Name], []interface{}{sum})

	return nil
}

func summarizeAnyOf(statDescription SchemaField, match Match, records rawRecords) error {
	for _, ref := range statDescription.AnyOf {
		refRecords, ok := records[ref.Name]
		if !ok {
			return nil
		}

		for _, reportGroup := range refRecords {
			for _, record := range reportGroup {
				if compareRecords(record, ref.Equals) {
					records[statDescription.Name] = append(records[statDescription.Name], []interface{}{1.0})
					return nil
				}
			}
		}
	}

	records[statDescription.Name] = append(records[statDescription.Name], []interface{}{0.0})
	return nil
}

func compareRecords(a, b interface{}) bool {
	aString, aOk := a.(string)
	bString, bOk := b.(string)
	if aOk && bOk {
		return aString == bString
	}

	aFloat, _ := a.(float64)
	bFloat, _ := b.(float64)

	if aBool, aOk := a.(bool); aOk && aBool {
		aFloat = 1.0
	}
	if bBool, bOk := b.(bool); bOk && bBool {
		bFloat = 1.0
	}

	return aFloat == bFloat
}
