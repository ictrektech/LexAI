package service

import (
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestBuildReviewClausesKeepsDocumentOrderAndOffsets(t *testing.T) {
	content := "# 付款条款\n合同价款应在交付后 30 日内支付。\n\n# Liability\nLiability is unlimited."
	clauses := buildReviewClauses("review-1", content)
	if len(clauses) == 0 {
		t.Fatal("expected at least one clause")
	}
	for index, clause := range clauses {
		if clause.Sequence != index {
			t.Fatalf("sequence=%d, want %d", clause.Sequence, index)
		}
		if clause.SourceStart < 0 || clause.SourceEnd <= clause.SourceStart || clause.SourceEnd > len([]rune(content)) {
			t.Fatalf("invalid source range: %+v", clause)
		}
	}
}

func TestParseReviewModelJSONAcceptsFencedJSONObject(t *testing.T) {
	var output reviewBatchOutput
	err := parseModelJSON("```json\n{\"issues\":[{\"risk_level\":\"high\",\"title\":\"Payment\",\"explanation\":\"Early payment\",\"original_quote\":\"pay now\",\"suggestion\":\"pay after delivery\"}]}\n```", &output)
	if err != nil || len(output.Issues) != 1 {
		t.Fatalf("parse output: %v %#v", err, output)
	}
}

func TestContractReviewRiskNormalization(t *testing.T) {
	if validRisk("HIGH") != types.ContractReviewRiskHigh {
		t.Fatal("HIGH should normalize to high")
	}
	if validRisk("unknown") != types.ContractReviewRiskMedium {
		t.Fatal("unknown risk should safely normalize to medium")
	}
}

func TestIssueFingerprintIsIdempotent(t *testing.T) {
	a := issueFingerprint("review", "clause", " Payment risk ", "pay  now")
	b := issueFingerprint("review", "clause", "payment risk", "pay now")
	if a != b {
		t.Fatalf("fingerprint changed across harmless whitespace: %s != %s", a, b)
	}
}

func TestContractReviewPromptsRequireChineseAnalysis(t *testing.T) {
	for name, prompt := range map[string]string{"clause": contractReviewClauseSystemPrompt, "overview": contractReviewOverviewSystemPrompt} {
		if !strings.Contains(prompt, "Simplified Chinese") {
			t.Fatalf("%s prompt must require Simplified Chinese output", name)
		}
	}
	if !strings.Contains(contractReviewClauseSystemPrompt, "original language and wording") {
		t.Fatal("clause prompt must preserve the original quotation for document positioning")
	}
}

func TestContractReviewCanBeRetriedAfterCompletion(t *testing.T) {
	if !canRetryContractReview(types.ContractReviewStatusCompleted) {
		t.Fatal("completed reviews must support an explicit rerun")
	}
	if !canRetryContractReview(types.ContractReviewStatusFailed) {
		t.Fatal("failed reviews must remain retryable")
	}
	if canRetryContractReview(types.ContractReviewStatusAnalyzing) {
		t.Fatal("running reviews must not support a second concurrent run")
	}
}

func TestCompletedReviewCanSaveNextConfigurationWithoutStarting(t *testing.T) {
	if !canUpdateContractReviewConfig(types.ContractReviewStatusCompleted) {
		t.Fatal("completed reviews must allow configuration changes before a rerun")
	}
	if canUpdateContractReviewConfig(types.ContractReviewStatusAnalyzing) {
		t.Fatal("running reviews must keep their configuration locked")
	}
}
