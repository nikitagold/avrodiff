package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

func TestPrintText(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		result model.DiffResult
		check  func(t *testing.T, out string)
	}{
		{
			name:   "major with breaking and safe changes",
			schema: "user-created.avsc",
			result: model.DiffResult{
				Level: model.LevelMajor,
				Changes: []model.Change{
					{
						Path:        "fields.email",
						Description: `field "email" removed`,
						Reason:      "consumers reading old messages will fail to deserialize",
						Severity:    model.Breaking,
					},
					{
						Path:        "fields.phone",
						Description: `field "phone" added (default: <nil>)`,
						Reason:      "backward and forward compatible",
						Severity:    model.Safe,
					},
				},
			},
			check: func(t *testing.T, out string) {
				for _, want := range []string{"BREAKING", "SAFE", "Result: MAJOR", "user-created.avsc"} {
					if !strings.Contains(out, want) {
						t.Errorf("expected %q in output", want)
					}
				}
			},
		},
		{
			name:   "no changes",
			schema: "user.avsc",
			result: model.DiffResult{Level: model.LevelNone},
			check: func(t *testing.T, out string) {
				if !strings.Contains(out, "no changes") {
					t.Error("expected 'no changes' in output")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			PrintText(&buf, tt.schema, tt.result)
			tt.check(t, buf.String())
		})
	}
}

func TestPrintJSON(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		result model.DiffResult
		check  func(t *testing.T, out map[string]interface{})
	}{
		{
			name:   "minor change",
			schema: "user.avsc",
			result: model.DiffResult{
				Level: model.LevelMinor,
				Changes: []model.Change{
					{
						Path:        "fields.phone",
						Description: "field added",
						Reason:      "safe",
						Severity:    model.Safe,
					},
				},
			},
			check: func(t *testing.T, out map[string]interface{}) {
				if out["schema"] != "user.avsc" {
					t.Errorf("schema: got %v", out["schema"])
				}
				if out["level"] != "MINOR" {
					t.Errorf("level: got %v", out["level"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := PrintJSON(&buf, tt.schema, tt.result); err != nil {
				t.Fatal(err)
			}
			var out map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
				t.Fatalf("invalid json: %v\n%s", err, buf.String())
			}
			tt.check(t, out)
		})
	}
}
