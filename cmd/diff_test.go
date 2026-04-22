package cmd

import (
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

func TestParseCompatMode(t *testing.T) {
	tests := []struct {
		input   string
		want    model.CompatMode
		wantErr bool
	}{
		{"backward", model.ModeBackward, false},
		{"forward", model.ModeForward, false},
		{"full", model.ModeFull, false},
		{"none", model.ModeNone, false},
		{"bakward", "", true},  // опечатка → ошибка
		{"FULL", "", true},     // регистр не поддерживается
		{"", "", true},         // пустая строка → ошибка
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseCompatMode(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseCompatMode(%q): expected error, got %v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("parseCompatMode(%q): unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseCompatMode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}