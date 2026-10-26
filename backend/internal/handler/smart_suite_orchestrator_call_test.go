package handler

import (
	"strings"
	"testing"
)

func TestOrchestratorUsesBackendDependencyOrderWithoutRouterModel(t *testing.T) {
	decision, err := smartSuiteSelectNextAgent([]string{"analyze-product-facts", "lock-suite-style"})
	if err != nil || decision.NextAgent != "analyze-product-facts" {
		t.Fatalf("unexpected deterministic selection: %+v %v", decision, err)
	}
	if !strings.Contains(decision.Reason, "backend dependency order") {
		t.Fatal("selection reason does not explain no router call")
	}
	if _, err := smartSuiteSelectNextAgent(nil); err == nil {
		t.Fatal("empty available set did not fail")
	}
}
