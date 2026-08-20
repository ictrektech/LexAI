package service

import (
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestUniqueWarnings(t *testing.T) {
	got := uniqueWarnings([]string{" first ", "first", "", "second", "second"})
	want := []string{"first", "second"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestLockedPlanRejectsOfficeCLICommentBeforeComparisonCreation(t *testing.T) {
	plan := &types.EditPlan{
		SchemaVersion: "1.0", Format: types.DocumentEditFormatDOCX, BaseSHA256: "source-sha", ApplyMode: "atomic",
		Operations: []types.EditOperation{{
			OperationID: "op-1", Kind: "add_comment",
			Target:  types.EditTarget{Quote: "payment term", ExpectedMatches: 1},
			Payload: types.EditPayload{Comment: "review this"},
		}},
	}
	if err := validateEditPlan(plan, "source-sha", types.DocumentEditModeAdeu); err != nil {
		t.Fatalf("Adeu should accept the locked comment plan: %v", err)
	}
	if err := validateEditPlan(plan, "source-sha", types.DocumentEditModeOfficeCLI); err == nil {
		t.Fatal("OfficeCLI comparison preflight should reject add_comment")
	}
}
