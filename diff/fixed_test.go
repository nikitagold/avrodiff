package diff

import (
	"strings"
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

func fixedSchema(name string, size int) interface{} {
	return map[string]interface{}{
		"type": "fixed",
		"name": name,
		"size": float64(size), // JSON числа unmarshal в float64
	}
}

func TestDiffFixed(t *testing.T) {
	// size изменился → BREAKING с понятным сообщением
	base := schema(field("checksum", fixedSchema("MD5", 16)))
	head := schema(field("checksum", fixedSchema("MD5", 32)))

	changes := diffFields(base, head, "", newCtx(base, head, model.ModeFull))
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %v", len(changes), changes)
	}
	got := changes[0]
	if got.Severity != model.Breaking {
		t.Errorf("severity: got %q, want BREAKING", got.Severity)
	}
	// Проверяем что это именно "size changed", а не generic "type changed"
	if strings.Contains(got.Description, "type changed") {
		t.Errorf("description should not be generic 'type changed', got: %q", got.Description)
	}
	if !strings.Contains(got.Description, "16") || !strings.Contains(got.Description, "32") {
		t.Errorf("description should mention old size (16) and new size (32), got: %q", got.Description)
	}
}