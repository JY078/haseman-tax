package handler

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"advertising-image/backend/internal/model"
)

func TestImpossibleCrossModuleCoverageFindingDoesNotBlockValidAssignedFact(t *testing.T) {
	task, snapshot := creativeGateFixture()
	snapshot.ImagePlans[0].AssignedFactIDs = []string{"F12"}
	snapshot.ImagePlans[0].AssignedProductFacts = "Faucet height is 18.9 inches."
	issue := model.EcomSuiteCreativeIssue{ImageType: snapshot.ImagePlans[0].ImageType, ImageIndex: 1, Category: "narrative", Problem: "C2 also requires width and depth, omitted here.", Correction: "Add F28 to assigned_fact_ids and keep the existing height copy."}
	filtered := suppressImpossibleSuiteCoverageIssues(snapshot.ImagePlans, []model.EcomSuiteCreativeIssue{issue})
	if len(filtered) != 0 {
		t.Fatalf("impossible cross-module issue still blocks: %+v", filtered)
	}
	verdict := smartSuiteCreativeVerdict{Passed: func() *bool { v := false; return &v }(), Summary: "coverage", Checks: map[string]string{"narrative": "ok", "copy": "ok", "visual": "ok", "facts": "ok"}, Issues: []model.EcomSuiteCreativeIssue{issue}}
	raw, _ := json.Marshal(verdict)
	if err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, func(_ context.Context, mode, user string) (string, error) {
		if !strings.Contains(user, smartSuiteCreativeCoverageContract) {
			t.Fatal("coverage contract missing")
		}
		if mode != "evaluate" {
			t.Fatal("invalid issue should not trigger repair")
		}
		return string(raw), nil
	}); err != nil || !smartSuiteCreativeReviewReady(task, snapshot) {
		t.Fatalf("valid assigned fact was blocked: %v", err)
	}
}

func TestUnsupportedClaimStillRemainsCreativeIssue(t *testing.T) {
	_, snapshot := creativeGateFixture()
	issue := model.EcomSuiteCreativeIssue{ImageType: snapshot.ImagePlans[0].ImageType, ImageIndex: 1, Category: "facts", Problem: "Title claims guaranteed leak prevention without assigned evidence.", Correction: "Remove the unsupported claim from title and prompt."}
	if got := suppressImpossibleSuiteCoverageIssues(snapshot.ImagePlans, []model.EcomSuiteCreativeIssue{issue}); len(got) != 1 {
		t.Fatal("unsupported claim was incorrectly suppressed")
	}
}
