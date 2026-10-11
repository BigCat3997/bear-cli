package ado

import (
	"strings"
	"testing"
)

func TestMergeVariablesRequiresOverrideForExistingKey(t *testing.T) {
	oldValue := "old"
	newValue := "new"
	existing := map[string]VariableValue{"TOKEN": {Value: &oldValue, IsSecret: true}}
	incoming := map[string]VariableValue{"TOKEN": {Value: &newValue, IsSecret: true}}

	_, err := mergeVariables(existing, incoming, false)
	if err == nil || !strings.Contains(err.Error(), "--override") {
		t.Fatalf("error = %v, want override guidance", err)
	}
}

func TestMergeVariablesOverridesExistingKey(t *testing.T) {
	oldValue := "old"
	newValue := "new"
	existing := map[string]VariableValue{"TOKEN": {Value: &oldValue, IsSecret: true}}
	incoming := map[string]VariableValue{"TOKEN": {Value: &newValue, IsSecret: true}}

	merged, err := mergeVariables(existing, incoming, true)
	if err != nil {
		t.Fatalf("mergeVariables: %v", err)
	}
	if got := *merged["TOKEN"].Value; got != "new" {
		t.Fatalf("TOKEN = %q, want new", got)
	}
	if got := *existing["TOKEN"].Value; got != "old" {
		t.Fatalf("input map was mutated: TOKEN = %q", got)
	}
}
