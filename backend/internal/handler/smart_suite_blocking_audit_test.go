package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/smartagent"
)

func TestCopyRetryReportsRealFailureBeforeContext(t *testing.T) {
	task, snapshot := creativeGateFixture()
	plan := snapshot.ImagePlans[0]
	plan.ImageType = "smart_mobile_a_plus"
	bad := model.EcomSuiteCopywriting{ImageType: plan.ImageType, ImageIndex: plan.ImageIndex, Title: "Adjust Water Flow", Subtitle: "A single lever lets you adjust the water flow easily."}
	accepted := make([]model.EcomSuiteCopywriting, 20)
	for i := range accepted {
		accepted[i] = bad
	}
	feedback := smartSuiteCopywriterRetryFeedback(task, plan, []model.EcomSuiteCopywriting{bad}, accepted)
	summary := smartSuitePlanningFailureMessage("smart_suite_agent_failed", fmt.Errorf("%s", feedback))
	if !strings.Contains(summary, "subtitle_length") || !strings.Contains(summary, "actual=10") || strings.Contains(feedback, "copywriter_duplicate_title") || strings.Contains(feedback, "accepted_titles_do_not_repeat") {
		t.Fatalf("real failure masked: %s", summary)
	}
}

func TestCreativeSuggestionNeedsNoRepairInstruction(t *testing.T) {
	task, snapshot := creativeGateFixture()
	v, _ := parseSmartSuiteCreativeVerdict(creativeVerdictJSON(false), snapshot.ImagePlans)
	v.Issues[0].Severity = "suggestion"
	v.Issues[0].Correction = ""
	raw, _ := json.Marshal(v)
	calls := 0
	err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, func(_ context.Context, mode, _ string) (string, error) {
		calls++
		if mode != "evaluate" {
			t.Fatal("suggestion caused repair")
		}
		return string(raw), nil
	})
	if err != nil || calls != 1 || !smartSuiteCreativeReviewReady(task, snapshot) {
		t.Fatalf("advisory formatting blocked generation: %v", err)
	}
}

func TestPlannerRetriesOnlyMissingWithActualFeedback(t *testing.T) {
	first := completeSmartSuiteTestPlan("smart_secondary", 1, "Water Flow Control")
	second := completeSmartSuiteTestPlan("smart_secondary", 2, "Faucet Profile")
	calls := 0
	plans, matched, err := refineSmartSuitePlansWithCall([]model.EcomSuiteImagePlan{first, second}, func(requested []model.EcomSuiteImagePlan, feedback string) ([]model.EcomSuiteImagePlan, error) {
		calls++
		if calls == 1 {
			return []model.EcomSuiteImagePlan{first}, fmt.Errorf("incomplete_layout_spec: smart_secondary#2")
		}
		if len(requested) != 1 || requested[0].ImageIndex != 2 || !strings.Contains(feedback, "incomplete_layout_spec") {
			t.Fatalf("lost valid sibling or feedback: requested=%v feedback=%s", requested, feedback)
		}
		return []model.EcomSuiteImagePlan{second}, nil
	})
	if err != nil || matched != 2 || len(plans) != 2 || calls != 2 {
		t.Fatalf("planner recovery failed: matched=%d calls=%d err=%v", matched, calls, err)
	}
}

func TestPlannerDoesNotHideProviderFailureWithFallback(t *testing.T) {
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Water Flow Control")
	calls := 0
	_, _, err := refineSmartSuitePlansWithCall([]model.EcomSuiteImagePlan{plan}, func(_ []model.EcomSuiteImagePlan, _ string) ([]model.EcomSuiteImagePlan, error) {
		calls++
		return nil, fmt.Errorf("%w: unavailable", errSmartSuitePlanningCall)
	})
	if err == nil || !strings.Contains(err.Error(), "planning_agent_call_failed") || calls != 2 {
		t.Fatalf("provider failure hidden or unbounded retry: calls=%d err=%v", calls, err)
	}
}

func TestModeEvidenceRulesAgreeFromDirectorToRender(t *testing.T) {
	contracts := []string{smartSuiteFinalArtworkRules, smartSuiteGenerationArtworkConstraint}
	for _, agent := range []smartagent.Name{smartagent.DirectSuiteCreative, smartagent.PlanSuiteImage, smartagent.EvaluateSuiteCreative} {
		prompt, err := smartSuiteAgentPrompt(agent, "")
		if err != nil {
			t.Fatal(err)
		}
		contracts = append(contracts, prompt)
	}
	for _, contract := range contracts {
		for _, forbidden := range []string{"each panel is clearly labeled", "show each state as independent visible evidence", "make each state independently legible"} {
			if strings.Contains(contract, forbidden) {
				t.Fatalf("contradictory mode rule: %s", forbidden)
			}
		}
	}
}
