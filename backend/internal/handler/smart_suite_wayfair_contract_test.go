package handler

import (
	"encoding/json"
	"strings"
	"testing"

	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/smartagent"
)

func TestWayfairAllowsColorWithoutTitle(t *testing.T) {
	listing := model.EcomSuiteListing{
		Platform:         "wayfair",
		AboutThisProduct: "A brushed gold faucet adds a warm accent to the sink.",
		Features:         []string{"Brass body: A solid metal construction.", "Black finish: Coordinates with dark cabinet hardware."},
	}
	if err := validateSmartSuiteListing(listing); err != nil {
		t.Fatal(err)
	}
	listing.Title = "Legacy title"
	listing = constrainSmartSuiteListingToPlatform(listing, "wayfair")
	raw, err := json.Marshal(listing)
	if err != nil || strings.Contains(string(raw), `"title":`) {
		t.Fatalf("unexpected title in output: %s (%v)", raw, err)
	}
	value := normalizeListingEvaluation(listingEvaluation{
		Summary: "文案表达清晰", OptimizedListing: listing,
		Dimensions: []listingEvaluationDimension{
			{Key: "about_this_product", Score: 40, Comment: "表达清楚"},
			{Key: "features", Score: 60, Comment: "卖点明确"},
		},
	})
	if err := validateListingEvaluationResult(&value, "wayfair"); err != nil {
		t.Fatal(err)
	}
}

func TestWayfairReviewAndEvaluationShareWriterRules(t *testing.T) {
	rules, err := smartagent.Instructions(smartagent.WriteWayfairListing)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []smartagent.Name{smartagent.ReviewWayfairListing, smartagent.EvaluateProductListing} {
		prompt, err := smartSuiteAgentPrompt(name, "Return the requested schema.")
		if err != nil || !strings.Contains(prompt, rules) {
			t.Fatalf("%s does not load current writer rules: %v", name, err)
		}
	}
}
