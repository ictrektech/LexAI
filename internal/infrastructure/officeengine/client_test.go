package officeengine

import (
	"testing"

	enginev1 "github.com/Tencent/WeKnora/internal/infrastructure/officeengine/gen/office/engine/v1"
)

func TestDocumentResultPreservesOperationDiagnosticsOnWorkerError(t *testing.T) {
	result, err := documentResult(&enginev1.DocumentResponse{
		EngineName: "officecli", Status: "error", ErrorCode: "invalid_request", ErrorMessage: "ambiguous",
		OperationResults: []*enginev1.OperationResult{{
			OperationId: "op-1", Kind: "replace_text", Status: "failed", ActualMatches: 2,
			EngineName: "officecli", Message: "target matched twice",
		}},
	})
	if err == nil {
		t.Fatal("expected the worker error")
	}
	if result == nil || len(result.OperationResults) != 1 {
		t.Fatalf("operation diagnostics were lost: %#v", result)
	}
	if got := result.OperationResults[0].ActualMatches; got != 2 {
		t.Fatalf("actual matches = %d, want 2", got)
	}
}
