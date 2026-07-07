package service

import "testing"

func TestGraphLLMConcurrencyCapsAtHalfMainQA(t *testing.T) {
	t.Setenv("WEKNORA_MAIN_QA_MODEL_CONCURRENCY", "6")
	t.Setenv("WEKNORA_GRAPH_LLM_CONCURRENCY", "4")

	if got := graphLLMConcurrency(4); got != 3 {
		t.Fatalf("graphLLMConcurrency() = %d, want 3", got)
	}
}

func TestBackgroundLLMCapacityReservesChatSlots(t *testing.T) {
	if got := backgroundLLMCapacity(6, 2); got != 4 {
		t.Fatalf("backgroundLLMCapacity() = %d, want 4", got)
	}
	if got := backgroundLLMCapacity(2, 2); got != 1 {
		t.Fatalf("backgroundLLMCapacity() = %d, want 1", got)
	}
	if got := backgroundLLMCapacity(0, 2); got != 0 {
		t.Fatalf("backgroundLLMCapacity() = %d, want 0", got)
	}
}
