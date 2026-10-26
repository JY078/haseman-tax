package handler

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"advertising-image/backend/internal/model"
)

func TestReviewRewriteReplacesExecutionAndPreservesCompleteContract(t *testing.T) {
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Control the Water Flow")
	plan.AssignedFactIDs = []string{"F1"}
	plan.AssignedProductFacts = "Single lever controls water flow."
	plan.Placement = "Installed on the sink deck"
	plan.Prompt = "OLD_INSTRUCTION: the handle floats above the counter."
	original := buildSmartSuiteImageGenerationPromptForTask(model.EcomSuiteTask{WorkflowType: "smart_suite", Language: "English"}, plan, "")
	before := decodeSmartSuiteGenerationContract(t, original)
	replacement := smartSuiteReviewExecution{
		Scene:                 "A side-lit basin with the faucet mounted on its deck",
		VisualRequirements:    "A hand contacts the lever; the mounted base casts a contact shadow.",
		SceneEvidence:         "Hand and lever meet at the grip to demonstrate single-lever operation.",
		LayoutSpec:            before.ImagePlan.LayoutSpec,
		LayoutInstruction:     "Keep the hand and lever clear of the title area.",
		TypographyInstruction: "Render both approved copy items with clear separation from the lever.",
	}
	encoded, _ := json.Marshal(replacement)
	result, err := mergeSmartSuiteReviewExecution(original, string(encoded), "")
	if err != nil {
		t.Fatal(err)
	}
	after := decodeSmartSuiteGenerationContract(t, result)
	if strings.Contains(result, "OLD_INSTRUCTION") || strings.Contains(result, "REVIEWER CORRECTION") {
		t.Fatal("old conflicting instruction or appended feedback reached the drawing contract")
	}
	if after.ImagePlan.VisualRequirements != replacement.VisualRequirements {
		t.Fatal("replacement did not reach final prompt")
	}
	// Everything except the six explicitly editable execution fields must survive.
	after.ImagePlan.Scene = before.ImagePlan.Scene
	after.ImagePlan.VisualRequirements = before.ImagePlan.VisualRequirements
	after.ImagePlan.SceneEvidence = before.ImagePlan.SceneEvidence
	after.ImagePlan.LayoutSpec = before.ImagePlan.LayoutSpec
	after.ImagePlan.LayoutInstruction = before.ImagePlan.LayoutInstruction
	after.ImagePlan.TypographyInstruction = before.ImagePlan.TypographyInstruction
	if !reflect.DeepEqual(before, after) {
		t.Fatal("rewrite changed a fact, copy, reference role or other locked requirement")
	}
	for name, response := range map[string]string{
		"incomplete":     `{ "scene": "new scene" }`,
		"copy injection": strings.TrimSuffix(string(encoded), "}") + `,"copy_items":[{"text":"Invented claim"}]}`,
		"wrong schema":   `{"revision_prompt":"Edit the current image in place"}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := mergeSmartSuiteReviewExecution(original, response, ""); err == nil {
				t.Fatal("unsafe/incomplete rewrite accepted")
			}
		})
	}
	if _, err := mergeSmartSuiteReviewExecution("Short old edit request", string(encoded), ""); err == nil {
		t.Fatal("missing original complete contract accepted")
	}
}

func TestSecondaryConcernLinksReachCopywriterAndFinalScene(t *testing.T) {
	facts := completeSmartSuiteTestFactMap()
	concerns := completeSmartSuiteTestConcerns()
	task := model.EcomSuiteTask{WorkflowType: "smart_suite", Language: "English", PlanSnapshot: &model.EcomSuitePlanSnapshot{ProductFactMap: facts, BuyerConcerns: concerns}}
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Product detail")
	direction := model.EcomSuiteCreativeDirection{
		ImageType: plan.ImageType, ImageIndex: 1, BuyerConcernIDs: []string{concerns[0].ID}, BuyerQuestion: concerns[0].Question,
		PrimarySellingPoint: "Keep a second roll ready", AssignedFactIDs: []string{"F1"},
		VisualEvidence: "two rolls visible in the installed holder", ShotScale: "medium", SceneDirection: "A bathroom wall with soft side light and two installed rolls",
		Placement: "Mounted on the bathroom wall",
	}
	directions, err := reconcileSmartSuiteCreativeDirectionsWithMode(task, []model.EcomSuiteImagePlan{plan}, []model.EcomSuiteCreativeDirection{direction}, true)
	if err != nil {
		t.Fatal(err)
	}
	plans := applySmartSuiteCreativeDirections([]model.EcomSuiteImagePlan{plan}, directions)
	_, copyPrompt := buildSmartSuiteCopywriterPrompts(task, plans)
	if !strings.Contains(copyPrompt, concerns[0].Question) || !strings.Contains(copyPrompt, direction.PrimarySellingPoint) {
		t.Fatal("buyer question or value dropped before copywriter")
	}
	task.PlanSnapshot.ImagePlans = plans
	// Simulate image rows, whose schema does not contain placement/fact fields.
	rowPlan := model.EcomSuiteImagePlan{ImageType: plan.ImageType, ImageIndex: 1, Style: plans[0].Style, Prompt: plans[0].Prompt}
	contract := decodeSmartSuiteGenerationContract(t, buildSmartSuiteImageGenerationPromptForTask(task, rowPlan, ""))
	if contract.ImagePlan.Placement != direction.Placement || contract.ConfirmedFact.ID != "F1" {
		t.Fatal("placement or assigned fact dropped from final image request")
	}
	direction.BuyerConcernIDs = nil
	if _, err := reconcileSmartSuiteCreativeDirectionsWithMode(task, []model.EcomSuiteImagePlan{plan}, []model.EcomSuiteCreativeDirection{direction}, true); err == nil {
		t.Fatal("secondary direction without buyer-concern link accepted")
	}
}
