package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"advertising-image/backend/internal/adapter"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/smartagent"
)

func creativeGateFixture() (model.EcomSuiteTask, model.EcomSuitePlanSnapshot) {
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Control the Water Flow")
	plan.AssignedFactIDs = []string{"F1"}
	plan.AssignedProductFacts = "Single lever controls water flow."
	plan.Subtitle = "A single lever controls the water flow."
	plan.MustIncludeText = "No labels"
	task := model.EcomSuiteTask{WorkflowType: "smart_suite", Language: "English", SecondaryCount: 1, Images: []model.EcomSuiteImage{{ImageType: plan.ImageType, ImageIndex: 1}}}
	snapshot := model.EcomSuitePlanSnapshot{ImagePlans: []model.EcomSuiteImagePlan{plan}, Copywriting: []model.EcomSuiteCopywriting{{ImageType: plan.ImageType, ImageIndex: 1}}, CreativeDirections: []model.EcomSuiteCreativeDirection{{ImageType: plan.ImageType, ImageIndex: 1}}, ProductFactMap: model.EcomSuiteProductFactMap{ConfirmedFacts: []model.EcomSuiteProductFact{{ID: "F1", Statement: plan.AssignedProductFacts, EvidenceSource: "user_document", Evidence: plan.AssignedProductFacts}}}}
	return task, snapshot
}

func creativeVerdictJSON(passed bool) string {
	issues := []model.EcomSuiteCreativeIssue{}
	if !passed {
		issues = append(issues, model.EcomSuiteCreativeIssue{ImageType: "smart_secondary", ImageIndex: 1, Category: "visual", Problem: "Floating hand", Correction: "Show actual lever contact"})
	}
	b, _ := json.Marshal(smartSuiteCreativeVerdict{Passed: &passed, Summary: "Checked suite proof and copy", Checks: map[string]string{"narrative": "Distinct purchase tasks", "copy": "Native phrasing and complete units", "visual": "Hand contact checked", "facts": "Claims grounded"}, Issues: issues})
	return string(b)
}

func TestCreativeGateSuggestionsDoNotBlockAfterRepairBudget(t *testing.T) {
	for _, passed := range []bool{true, false} {
		task, snapshot := creativeGateFixture()
		snapshot.CreativeReview = &model.EcomSuiteCreativeReview{RepairAttempts: smartSuiteCreativeMaxRepairs}
		verdict, _ := parseSmartSuiteCreativeVerdict(creativeVerdictJSON(false), snapshot.ImagePlans)
		verdict.Passed = &passed
		verdict.Issues[0].Severity = "suggestion"
		verdict.Issues[0].Problem = "Placement beneath an overhead cabinet and cabinet-edge typography can reintroduce a clearance implication."
		verdict.Issues[0].Correction = "Prefer an open sink-deck scene."
		raw, _ := json.Marshal(verdict)
		calls := 0
		err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, func(_ context.Context, mode, user string) (string, error) {
			calls++
			if mode != "evaluate" || !strings.Contains(user, smartSuiteCreativeBlockingContract) {
				t.Fatal("suggestion triggered repair or missing threshold")
			}
			return string(raw), nil
		})
		if err != nil || calls != 1 || !smartSuiteCreativeReviewReady(task, snapshot) || len(snapshot.CreativeReview.Suggestions) != 1 {
			t.Fatalf("suggestion blocked generation: passed=%v calls=%d err=%v", passed, calls, err)
		}
	}
}

func TestCreativeGateKeepsBlockersAlongsideSuggestions(t *testing.T) {
	task, snapshot := creativeGateFixture()
	snapshot.CreativeReview = &model.EcomSuiteCreativeReview{RepairAttempts: smartSuiteCreativeMaxRepairs}
	verdict, _ := parseSmartSuiteCreativeVerdict(creativeVerdictJSON(false), snapshot.ImagePlans)
	verdict.Issues[0].Severity = "blocker"
	suggestion := verdict.Issues[0]
	suggestion.Severity = "suggestion"
	suggestion.Problem = "Prefer different text alignment."
	verdict.Issues = append(verdict.Issues, suggestion)
	raw, _ := json.Marshal(verdict)
	err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, func(_ context.Context, _, _ string) (string, error) { return string(raw), nil })
	if err == nil || smartSuiteCreativeReviewReady(task, snapshot) || len(snapshot.CreativeReview.Issues) != 1 || len(snapshot.CreativeReview.Suggestions) != 1 {
		t.Fatalf("real blocker was lost: %v", err)
	}
}

func TestCreativeReviewIsAdvisoryAndDeterministicPlanStillGatesRender(t *testing.T) {
	task, snapshot := creativeGateFixture()
	if !containsString(smartSuiteOrchestratorAvailableAgents(task, snapshot), smartSuiteOrchestratorRender) {
		t.Fatal("valid deterministic plan should render without editorial review")
	}
	calls := 0
	err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, func(_ context.Context, mode, user string) (string, error) {
		calls++
		if mode != "evaluate" || !strings.Contains(user, snapshot.ImagePlans[0].Title) {
			t.Fatal("evaluator did not receive actual content")
		}
		return creativeVerdictJSON(true), nil
	})
	if err != nil || calls != 1 || !smartSuiteCreativeReviewReady(task, snapshot) {
		t.Fatalf("pass not persisted: %v", err)
	}
	snapshot.ImagePlans[0].Subtitle = "Changed after approval"
	if smartSuiteCreativeReviewReady(task, snapshot) {
		t.Fatal("stale advisory review reused")
	}
}

func TestCreativeGateRepairsOnlyRequestedImageAndRequiresNewPlanEvaluation(t *testing.T) {
	task, snapshot := creativeGateFixture()
	second := snapshot.ImagePlans[0]
	second.ImageIndex = 2
	snapshot.ImagePlans = append(snapshot.ImagePlans, second)
	original := snapshot.ImagePlans[0]
	change := original
	change.VisualStrategy = "Hand visibly contacts the lever"
	change.AssignedFactIDs = []string{"F999"}
	change.Placement = "Invented mounting"
	encoded, _ := json.Marshal(map[string]interface{}{"plans": []model.EcomSuiteImagePlan{change}})
	calls := 0
	err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, func(_ context.Context, mode, user string) (string, error) {
		calls++
		if mode == "evaluate" {
			return creativeVerdictJSON(false), nil
		}
		return string(encoded), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || snapshot.CreativeReview.RepairAttempts != 1 || smartSuiteCreativeReviewReady(task, snapshot) {
		t.Fatal("repair incorrectly marked as approved")
	}
	if !reflect.DeepEqual(snapshot.ImagePlans[1], second) {
		t.Fatal("valid sibling image changed")
	}
	if !reflect.DeepEqual(snapshot.ImagePlans[0].AssignedFactIDs, original.AssignedFactIDs) || snapshot.ImagePlans[0].Placement != original.Placement {
		t.Fatal("repair replaced immutable facts or placement")
	}
	if snapshot.ImagePlans[0].VisualStrategy != change.VisualStrategy {
		t.Fatal("correction not applied")
	}
	if err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, func(_ context.Context, mode, user string) (string, error) {
		if mode != "evaluate" {
			t.Fatal("unexpected repair")
		}
		return creativeVerdictJSON(true), nil
	}); err != nil {
		t.Fatal(err)
	}
	if !smartSuiteCreativeReviewReady(task, snapshot) {
		t.Fatal("corrected plan did not pass")
	}
}

func TestCreativeGateBudgetAndMalformedVerdictsFailClosed(t *testing.T) {
	task, snapshot := creativeGateFixture()
	snapshot.CreativeReview = &model.EcomSuiteCreativeReview{RepairAttempts: smartSuiteCreativeMaxRepairs}
	calls := 0
	err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, func(_ context.Context, mode, user string) (string, error) {
		calls++
		return creativeVerdictJSON(false), nil
	})
	if err == nil || !strings.Contains(err.Error(), "suite_creative_quality_failed") || calls != 1 {
		t.Fatalf("budget not enforced: %v", err)
	}
	for _, raw := range []string{`{}`, `{"passed":true,"summary":"ok","issues":[]}`, strings.Replace(creativeVerdictJSON(false), `"passed":false`, `"passed":true`, 1), strings.Replace(creativeVerdictJSON(false), `"image_index":1`, `"image_index":99`, 1)} {
		if _, err := parseSmartSuiteCreativeVerdict(raw, snapshot.ImagePlans); err == nil {
			t.Fatalf("invalid verdict accepted: %s", raw)
		}
	}
}

func TestCreativeGateRetriesMalformedResponseWithoutChangingPlans(t *testing.T) {
	task, snapshot := creativeGateFixture()
	calls := 0
	err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, func(_ context.Context, mode, user string) (string, error) {
		calls++
		if calls == 1 {
			return `{}`, nil
		}
		if !strings.Contains(user, "Previous validation error") {
			t.Fatal("retry missing error")
		}
		return creativeVerdictJSON(true), nil
	})
	if err != nil || calls != 2 || !smartSuiteCreativeReviewReady(task, snapshot) {
		t.Fatalf("schema recovery failed: %v", err)
	}
}

func TestCreativeRepairPreservesOmittedLayoutFields(t *testing.T) {
	for _, tc := range []struct {
		name, patch string
	}{
		{"copy only", `"title":"Control Your Water Flow"`},
		{"partial layout", `"layout_spec":{"camera_angle":"Side view with natural perspective"}`},
		{"nested zero coordinate", `"layout_spec":{"text_safe_area":{"x_percent":0}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			task, snapshot := creativeGateFixture()
			originalJSON, _ := json.Marshal(snapshot.ImagePlans)
			verdict, _ := parseSmartSuiteCreativeVerdict(creativeVerdictJSON(false), snapshot.ImagePlans)
			raw := `{"plans":[{"image_type":"smart_secondary","image_index":1,` + tc.patch + `}]}`
			plans, err := mergeSmartSuiteCreativeRepairs(task, snapshot.ImagePlans, verdict.Issues, raw)
			if err != nil {
				t.Fatal(err)
			}
			want := snapshot.ImagePlans[0].LayoutSpec
			switch tc.name {
			case "partial layout":
				want.CameraAngle = "Side view with natural perspective"
			case "nested zero coordinate":
				want.TextSafeArea.XPercent = 0
			case "copy only":
				if plans[0].Title != "Control Your Water Flow" {
					t.Fatal("copy patch lost")
				}
			}
			if !reflect.DeepEqual(plans[0].LayoutSpec, want) {
				t.Fatalf("layout changed unexpectedly: %+v", plans[0].LayoutSpec)
			}
			afterJSON, _ := json.Marshal(snapshot.ImagePlans)
			if string(originalJSON) != string(afterJSON) {
				t.Fatal("input plans mutated")
			}
		})
	}
}

func TestCreativeRepairRejectsExplicitInvalidLayout(t *testing.T) {
	for _, layout := range []string{`null`, `{"product_occupancy_percent":0}`, `{"text_safe_area":{"width_percent":0}}`, `{"text_safe_area":null}`, `{"forbidden_layouts":[]}`} {
		task, snapshot := creativeGateFixture()
		before, _ := json.Marshal(snapshot.ImagePlans)
		verdict, _ := parseSmartSuiteCreativeVerdict(creativeVerdictJSON(false), snapshot.ImagePlans)
		_, err := mergeSmartSuiteCreativeRepairs(task, snapshot.ImagePlans, verdict.Issues,
			`{"plans":[{"image_type":"smart_secondary","image_index":1,"layout_spec":`+layout+`}]}`)
		if err == nil {
			t.Fatalf("invalid layout accepted: %s", layout)
		}
		after, _ := json.Marshal(snapshot.ImagePlans)
		if string(before) != string(after) {
			t.Fatal("failed repair mutated original")
		}
	}
}

func TestCreativeRepairRetryReceivesActionableLayoutError(t *testing.T) {
	task, snapshot := creativeGateFixture()
	repairs := 0
	err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, func(_ context.Context, mode, user string) (string, error) {
		if mode == "evaluate" {
			return creativeVerdictJSON(false), nil
		}
		repairs++
		if !strings.Contains(user, smartSuiteCreativeLayoutContract) {
			t.Fatal("repair missing layout contract")
		}
		if repairs == 1 {
			return `{"plans":[{"image_type":"smart_secondary","image_index":1,"layout_spec":{"panel_count":1}}]}`, nil
		}
		for _, detail := range []string{"smart_secondary#1", "panel_count=1", "expected 0"} {
			if !strings.Contains(user, detail) {
				t.Fatalf("retry missing %q", detail)
			}
		}
		return `{"plans":[{"image_type":"smart_secondary","image_index":1,"layout_spec":{"panel_count":0}}]}`, nil
	})
	if err != nil || repairs != 2 {
		t.Fatalf("repair did not recover: repairs=%d err=%v", repairs, err)
	}
}

func TestLayoutValidationExplainsEachConstraint(t *testing.T) {
	_, snapshot := creativeGateFixture()
	for _, tc := range []struct {
		name, detail string
		change       func(*model.EcomSuiteLayoutSpec)
	}{
		{"occupancy", "product_occupancy_percent=20", func(s *model.EcomSuiteLayoutSpec) { s.ProductOccupancyPercent = 20 }},
		{"anchor", `product_anchor="upper_left"`, func(s *model.EcomSuiteLayoutSpec) { s.ProductAnchor = "upper_left" }},
		{"comparison", `comparison_mode="single"`, func(s *model.EcomSuiteLayoutSpec) { s.ComparisonMode = "single" }},
		{"panel count", "panel_count=1", func(s *model.EcomSuiteLayoutSpec) { s.PanelCount = 1 }},
		{"panel mode", "requires comparison_mode=side_by_side", func(s *model.EcomSuiteLayoutSpec) { s.PanelCount = 3 }},
		{"text overflow", "x+width<=100", func(s *model.EcomSuiteLayoutSpec) { s.TextSafeArea.XPercent = 90 }},
		{"camera", "camera_angle", func(s *model.EcomSuiteLayoutSpec) { s.CameraAngle = "" }},
		{"forbidden layouts", `missing required entry "editorial_header"`, func(s *model.EcomSuiteLayoutSpec) { s.ForbiddenLayouts = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := snapshot.ImagePlans[0].LayoutSpec
			tc.change(&spec)
			err := smartSuiteLayoutSpecValidationError(spec)
			if err == nil || !strings.Contains(err.Error(), tc.detail) || smartSuiteLayoutSpecComplete(spec) {
				t.Fatalf("missing diagnostic %q: %v", tc.detail, err)
			}
		})
	}
}

func TestCreativeRepairTargetsOneImageAndRetriesOnlyMissingTarget(t *testing.T) {
	task, snapshot := creativeGateFixture()
	second := snapshot.ImagePlans[0]
	second.ImageIndex = 2
	snapshot.ImagePlans = append(snapshot.ImagePlans, second)
	verdict, _ := parseSmartSuiteCreativeVerdict(creativeVerdictJSON(false), snapshot.ImagePlans)
	duplicate := verdict.Issues[0]
	duplicate.Category = "copy"
	secondIssue := verdict.Issues[0]
	secondIssue.ImageIndex = 2
	verdict.Issues = append(verdict.Issues, duplicate, secondIssue)
	encoded, _ := json.Marshal(verdict)
	repairs := 0
	err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, func(_ context.Context, mode, user string) (string, error) {
		if mode == "evaluate" {
			return string(encoded), nil
		}
		repairs++
		switch repairs {
		case 1:
			if !strings.Contains(user, "This call repairs ONLY smart_secondary#1") {
				t.Fatal("missing first target")
			}
			return `{"plans":[{"image_type":"smart_secondary","image_index":1,"title":"Control Your Water Flow"}]}`, nil
		case 2:
			if !strings.Contains(user, "This call repairs ONLY smart_secondary#2") {
				t.Fatal("duplicate findings triggered a duplicate repair")
			}
			return `{"plans":[]}`, nil
		default:
			if repairs != 3 || !strings.Contains(user, "missing=smart_secondary#2") {
				t.Fatal("retry missing actionable target")
			}
			return `{"plans":[{"image_type":"smart_secondary","image_index":2,"title":"Adjust Your Water Flow"}]}`, nil
		}
	})
	if err != nil || repairs != 3 {
		t.Fatalf("targeted repair failed: %v calls=%d", err, repairs)
	}
	if snapshot.ImagePlans[0].Title != "Control Your Water Flow" || snapshot.ImagePlans[1].Title != "Adjust Your Water Flow" {
		t.Fatal("valid sibling repair lost")
	}
}

func TestCreativeRepairIdentifiesMissingExtraAndDuplicateImages(t *testing.T) {
	task, snapshot := creativeGateFixture()
	verdict, _ := parseSmartSuiteCreativeVerdict(creativeVerdictJSON(false), snapshot.ImagePlans)
	for _, tc := range []struct{ raw, expected string }{
		{`{"plans":[]}`, "missing=smart_secondary#1"},
		{`{"plans":[{"image_type":"smart_secondary","image_index":2}]}`, "unrequested_image:smart_secondary#2"},
		{`{"plans":[{"image_type":"smart_secondary","image_index":1},{"image_type":"smart_secondary","image_index":1}]}`, "duplicate_image:smart_secondary#1"},
	} {
		_, err := mergeSmartSuiteCreativeRepairs(task, snapshot.ImagePlans, verdict.Issues, tc.raw)
		if err == nil || !strings.Contains(err.Error(), tc.expected) {
			t.Fatalf("expected %s, got %v", tc.expected, err)
		}
	}
}

func TestCreativeRepairRetriesCaptionCopyWithSpecificFeedback(t *testing.T) {
	for _, inherited := range []bool{false, true} {
		task, snapshot := creativeGateFixture()
		if inherited {
			snapshot.ImagePlans[0].Subtitle = "Single lever pictured beside the spout."
		}
		repairs := 0
		err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, func(_ context.Context, mode, user string) (string, error) {
			if mode == "evaluate" {
				return creativeVerdictJSON(false), nil
			}
			repairs++
			if !strings.Contains(user, "Visible copy contract") {
				t.Fatal("repair missing copy rules")
			}
			if repairs == 1 {
				if inherited {
					return `{"plans":[{"image_type":"smart_secondary","image_index":1,"visual_strategy":"Hand visibly contacts lever"}]}`, nil
				}
				return `{"plans":[{"image_type":"smart_secondary","image_index":1,"subtitle":"Single lever pictured beside the spout."}]}`, nil
			}
			for _, detail := range []string{"field=subtitle", `matched="pictured"`, "buyer-facing product copy", "not a target-language error"} {
				if !strings.Contains(user, detail) {
					t.Fatalf("missing corrective feedback %q", detail)
				}
			}
			return `{"plans":[{"image_type":"smart_secondary","image_index":1,"subtitle":"A single lever controls the water flow."}]}`, nil
		})
		if err != nil || repairs != 2 {
			t.Fatalf("caption retry failed: inherited=%v calls=%d err=%v", inherited, repairs, err)
		}
		if snapshot.ImagePlans[0].Subtitle != "A single lever controls the water flow." {
			t.Fatal("corrected copy not saved")
		}
	}
}

func TestCreativeGateRoutesInvalidUpstreamPlansBackToPlanner(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*model.EcomSuiteImagePlan)
	}{
		{"missing layout", func(p *model.EcomSuiteImagePlan) { p.LayoutSpec = model.EcomSuiteLayoutSpec{} }},
		{"zero occupancy", func(p *model.EcomSuiteImagePlan) { p.LayoutSpec.ProductOccupancyPercent = 0 }},
		{"invalid anchor", func(p *model.EcomSuiteImagePlan) { p.LayoutSpec.ProductAnchor = "upper_left" }},
		{"single panel", func(p *model.EcomSuiteImagePlan) { p.LayoutSpec.PanelCount = 1 }},
		{"text overflow", func(p *model.EcomSuiteImagePlan) { p.LayoutSpec.TextSafeArea.WidthPercent = 100 }},
		{"missing camera", func(p *model.EcomSuiteImagePlan) { p.LayoutSpec.CameraAngle = "" }},
		{"missing exclusions", func(p *model.EcomSuiteImagePlan) { p.LayoutSpec.ForbiddenLayouts = nil }},
		{"missing scene", func(p *model.EcomSuiteImagePlan) { p.Style = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			task, snapshot := creativeGateFixture()
			tc.change(&snapshot.ImagePlans[0])
			snapshot.CreativeReview = &model.EcomSuiteCreativeReview{Passed: true, InputFingerprint: smartSuiteCreativeFingerprint(task, snapshot)}
			agents := smartSuiteOrchestratorAvailableAgents(task, snapshot)
			if !containsString(agents, string(smartagent.PlanSuiteImage)) || containsString(agents, string(smartagent.EvaluateSuiteCreative)) || containsString(agents, smartSuiteOrchestratorRender) {
				t.Fatalf("invalid plan routed incorrectly: %v", agents)
			}
			if smartSuiteCreativeReviewReady(task, snapshot) {
				t.Fatal("invalid plan reused approval")
			}
			err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, func(context.Context, string, string) (string, error) {
				t.Fatal("invalid upstream plan made a model call")
				return "", nil
			})
			if err == nil || !strings.Contains(err.Error(), "creative_review_input_invalid:smart_secondary#1") {
				t.Fatalf("wrong failure source: %v", err)
			}
		})
	}
}

func TestCreativeRepairMalformedInputsAreAtomic(t *testing.T) {
	for _, patch := range []string{
		`"layout_spec":{"product_occupancy_percent":0}`, `"layout_spec":{"product_occupancy_percent":"55%"}`,
		`"layout_spec":{"product_occupancy_percent":0.55}`, `"layout_spec":[]`, `"layout_spec":null`,
		`"layout_spec":{"text_safe_area":{"height_percent":0}}`, `"title":null`, `"title":""`,
	} {
		task, snapshot := creativeGateFixture()
		before, _ := json.Marshal(snapshot.ImagePlans)
		verdict, _ := parseSmartSuiteCreativeVerdict(creativeVerdictJSON(false), snapshot.ImagePlans)
		_, err := mergeSmartSuiteCreativeRepairs(task, snapshot.ImagePlans, verdict.Issues, `{"plans":[{"image_type":"smart_secondary","image_index":1,`+patch+`}]}`)
		if err == nil {
			t.Fatalf("invalid patch accepted: %s", patch)
		}
		after, _ := json.Marshal(snapshot.ImagePlans)
		if string(before) != string(after) {
			t.Fatal("invalid response corrupted original plan")
		}
	}
}

func TestCreativeRepairRecoversAcrossDifferentValidationErrors(t *testing.T) {
	task, snapshot := creativeGateFixture()
	repairs := 0
	err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, func(_ context.Context, mode, user string) (string, error) {
		if mode == "evaluate" {
			return creativeVerdictJSON(false), nil
		}
		repairs++
		switch repairs {
		case 1:
			return `{"plans":[]}`, nil
		case 2:
			return `{"plans":[{"image_type":"smart_secondary","image_index":1,"layout_spec":{"product_occupancy_percent":0}}]}`, nil
		case 3:
			return `{"plans":[{"image_type":"smart_secondary","image_index":1,"subtitle":"Single lever pictured beside the spout."}]}`, nil
		default:
			return `{"plans":[{"image_type":"smart_secondary","image_index":1,"subtitle":"A single lever controls the water flow."}]}`, nil
		}
	})
	if err == nil || repairs != 2 || !strings.Contains(err.Error(), "product_occupancy_percent=0") {
		t.Fatalf("stage stopped without a bounded actionable error: calls=%d err=%v", repairs, err)
	}
}

func TestCreativeRepairCancelledBeforeAnyModelCall(t *testing.T) {
	task, snapshot := creativeGateFixture()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := runSmartSuiteCreativeGateWithCall(ctx, task, &snapshot, func(context.Context, string, string) (string, error) {
		t.Error("cancelled task still invoked model")
		return creativeVerdictJSON(true), nil
	})
	if err == nil {
		t.Fatal("cancellation ignored")
	}
}

func TestCreativeRepairStopsWithoutProgressAndBoundsChangingErrors(t *testing.T) {
	for _, repeat := range []bool{true, false} {
		task, snapshot := creativeGateFixture()
		calls := 0
		err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, func(_ context.Context, mode, user string) (string, error) {
			if mode == "evaluate" {
				return creativeVerdictJSON(false), nil
			}
			calls++
			value := 0
			if !repeat {
				value = calls
			}
			return fmt.Sprintf(`{"plans":[{"image_type":"smart_secondary","image_index":1,"layout_spec":{"product_occupancy_percent":%d}}]}`, value), nil
		})
		want := smartSuiteCreativeMaxCorrectionCalls
		if repeat {
			want = 2
		}
		if err == nil || calls != want {
			t.Fatalf("repeat=%v calls=%d expected=%d err=%v", repeat, calls, want, err)
		}
		if snapshot.ImagePlans[0].LayoutSpec.ProductOccupancyPercent != 55 {
			t.Fatal("failed corrections changed original")
		}
	}
}

func TestCreativeGateThroughVisionHTTPToFinalPrompt(t *testing.T) {
	task, snapshot := creativeGateFixture()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		calls++
		var raw string
		switch calls {
		case 1:
			raw = creativeVerdictJSON(false)
		case 2:
			raw = `{"plans":[]}`
		case 3:
			body, _ := json.Marshal(request)
			if !strings.Contains(string(body), "missing=smart_secondary#1") {
				t.Error("HTTP retry omitted validation feedback")
			}
			raw = `{"plans":[{"image_type":"smart_secondary","image_index":1,"title":"Adjust Your Water Flow"}]}`
		default:
			raw = creativeVerdictJSON(true)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"output": []interface{}{map[string]interface{}{"content": []interface{}{map[string]interface{}{"type": "output_text", "text": raw}}}}})
	}))
	defer server.Close()
	call := func(ctx context.Context, mode, user string) (string, error) {
		raw, _, _, err := adapter.NewChatAdapter("test", server.URL).AnalyzeImagesWithVisionContext(ctx, "fixture-model", nil, "Mode: "+mode, user, 12000)
		return raw, err
	}
	for round := 0; round < 2; round++ {
		if err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, call); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 4 || !smartSuiteCreativeReviewReady(task, snapshot) {
		t.Fatalf("HTTP loop incomplete: calls=%d", calls)
	}
	prompt := buildSmartSuiteImageGenerationPromptForTask(task, snapshot.ImagePlans[0], "")
	if !strings.Contains(prompt, "Adjust Your Water Flow") || strings.Contains(prompt, "Control the Water Flow") {
		t.Fatal("final image prompt did not receive repaired copy")
	}
}

func TestCreativeEvidenceContractSharedAndPriorFindingsReachEvaluation(t *testing.T) {
	task, snapshot := creativeGateFixture()
	_, planner := buildSmartSuitePlanRefinementPrompts(task, snapshot.ImagePlans)
	if !strings.Contains(planner, smartSuiteCreativeEvidenceContract) || strings.Contains(planner, "set panel_count to N") {
		t.Fatal("planner retains contradictory mode contract")
	}
	snapshot.CreativeReview = &model.EcomSuiteCreativeReview{Summary: "Prior ambiguous mode evidence", RepairAttempts: 0}
	calls := 0
	err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, func(_ context.Context, mode, user string) (string, error) {
		calls++
		if !strings.Contains(user, smartSuiteCreativeEvidenceContract) {
			t.Fatal("missing renderer/evidence constraints")
		}
		if mode == "evaluate" {
			if !strings.Contains(user, "Prior ambiguous mode evidence") {
				t.Fatal("review lost prior findings")
			}
			return creativeVerdictJSON(false), nil
		}
		return `{"plans":[{"image_type":"smart_secondary","image_index":1,"visual_strategy":"Hand visibly contacts the real lever"}]}`, nil
	})
	if err != nil || calls != 2 {
		t.Fatalf("repair failed: %v calls=%d", err, calls)
	}
	if snapshot.ImagePlans[0].MustIncludeText != "No labels" {
		t.Fatal("repair introduced forbidden labels")
	}
	err = runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, func(_ context.Context, mode, user string) (string, error) { return creativeVerdictJSON(false), nil })
	if err == nil || !strings.Contains(err.Error(), "smart_secondary#1 [visual]") || !strings.Contains(err.Error(), "correction:") {
		t.Fatalf("terminal failure missing actionable findings: %v", err)
	}
}

func TestCreativeGateRepairsMissingUnitWithoutInventingEvidence(t *testing.T) {
	task, snapshot := creativeGateFixture()
	plan := &snapshot.ImagePlans[0]
	plan.AssignedFactIDs = []string{"F4"}
	plan.AssignedProductFacts = "Height: 18.9. Unit not supplied."
	plan.Title = "18.9-inch Height"
	plan.Subtitle = "An 18.9-inch profile fits beneath cabinets."
	plan.Role = "Will this fit beneath my cabinet?"
	snapshot.ProductFactMap.ConfirmedFacts = []model.EcomSuiteProductFact{{ID: "F4", Statement: plan.AssignedProductFacts, EvidenceSource: "user_document", Evidence: "Height: 18.9"}}
	verdict, _ := parseSmartSuiteCreativeVerdict(creativeVerdictJSON(false), snapshot.ImagePlans)
	verdict.Issues = []model.EcomSuiteCreativeIssue{{ImageType: plan.ImageType, ImageIndex: plan.ImageIndex, Category: "facts", Problem: "18.9-inch invents a unit and cabinet compatibility is unsupported.", Correction: "Remove the measurement and cabinet fit promise."}}
	failure, _ := json.Marshal(verdict)
	modes := []string{}
	call := func(_ context.Context, mode, user string) (string, error) {
		modes = append(modes, mode)
		if !strings.Contains(user, "omit the entire numeric measurement") {
			t.Fatal("missing shared incomplete-measurement rule")
		}
		if len(modes) == 1 {
			return string(failure), nil
		}
		if mode == "repair" {
			return `{"plans":[{"image_type":"smart_secondary","image_index":1,"title":"Faucet Profile","subtitle":"Explore the faucet's upright product profile.","role":"What is the faucet profile?","cta_direction":"Inspect the product profile","prompt":"Photograph the reference faucet upright with an unobstructed view of its profile. No clearance demonstration or measurement annotations.","visual_strategy":"An unnumbered product profile view","style":"Neutral product study"}]}`, nil
		}
		if strings.Contains(snapshot.ImagePlans[0].Title+snapshot.ImagePlans[0].Subtitle+snapshot.ImagePlans[0].Role, "18.9") {
			t.Fatal("unsupported measurement survived correction")
		}
		return creativeVerdictJSON(true), nil
	}
	if err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, call); err != nil {
		t.Fatal(err)
	}
	if smartSuiteCreativeReviewReady(task, snapshot) {
		t.Fatal("repair bypassed factual review")
	}
	if err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, call); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(modes, []string{"evaluate", "repair", "evaluate"}) || !smartSuiteCreativeReviewReady(task, snapshot) {
		t.Fatalf("gate did not advance after correction: %v", modes)
	}
	if snapshot.ImagePlans[0].AssignedFactIDs[0] != "F4" || snapshot.ImagePlans[0].AssignedProductFacts != "Height: 18.9. Unit not supplied." || snapshot.Copywriting[0].Title != snapshot.ImagePlans[0].Title {
		t.Fatal("source facts changed or corrected copy was not synchronized")
	}
}

func TestCreativeFactRepairRejectsReviewerRequestToExpandFacts(t *testing.T) {
	task, snapshot := creativeGateFixture()
	plan := &snapshot.ImagePlans[0]
	plan.AssignedFactIDs = []string{"F12"}
	plan.AssignedProductFacts = "Requires one installation hole."
	plan.Title = "One-Hole Deck Mount"
	plan.Subtitle = "One hole supports the deck mount."
	snapshot.ProductFactMap.ConfirmedFacts = []model.EcomSuiteProductFact{
		{ID: "F12", Statement: "Requires one installation hole.", EvidenceSource: "user_document", Evidence: "One hole"},
		{ID: "F28", Statement: "Deck mounted.", EvidenceSource: "user_document", Evidence: "Deck mounted"},
	}
	verdict, _ := parseSmartSuiteCreativeVerdict(creativeVerdictJSON(false), snapshot.ImagePlans)
	verdict.Issues = []model.EcomSuiteCreativeIssue{{ImageType: "smart_secondary", ImageIndex: 1, Category: "facts", Problem: "Deck mounting needs F28, while this image is assigned only F12.", Correction: "Add F28 to assigned_fact_ids and keep existing copy."}}
	raw, _ := json.Marshal(verdict)
	calls := 0
	err := runSmartSuiteCreativeGateWithCall(context.Background(), task, &snapshot, func(_ context.Context, mode, user string) (string, error) {
		calls++
		if !strings.Contains(user, smartSuiteCreativeFactContract) {
			t.Fatal("missing factual repair authority")
		}
		if mode == "evaluate" {
			return string(raw), nil
		}
		if strings.Contains(user, "Add F28 to assigned_fact_ids and keep existing copy.") {
			t.Fatal("impossible correction sent to repair")
		}
		if !strings.Contains(user, "Deck mounting needs F28") {
			t.Fatal("original factual problem lost")
		}
		return `{"plans":[{"image_type":"smart_secondary","image_index":1,"title":"Single-Hole Installation","subtitle":"Requires only one installation hole.","assigned_fact_ids":["F12","F28"],"assigned_product_facts":"Requires one hole and deck mounting."}]}`, nil
	})
	if err != nil || calls != 2 {
		t.Fatalf("factual wording was not repaired: %v calls=%d", err, calls)
	}
	if len(snapshot.ImagePlans[0].AssignedFactIDs) != 1 || snapshot.ImagePlans[0].AssignedFactIDs[0] != "F12" || snapshot.ImagePlans[0].AssignedProductFacts != "Requires one installation hole." {
		t.Fatal("expanded immutable evidence")
	}
	if snapshot.ImagePlans[0].Title != "Single-Hole Installation" || smartSuiteCreativeReviewReady(task, snapshot) {
		t.Fatal("copy was not narrowed or was approved without review")
	}
}
