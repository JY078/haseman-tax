package handler

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"advertising-image/backend/internal/model"
)

func TestCopywriterRetryCarriesActualLengthAndPreservesValidSibling(t *testing.T) {
	task, snapshot := creativeGateFixture()
	first := snapshot.ImagePlans[0]
	mobile := first
	mobile.ImageType, mobile.ImageIndex = "smart_mobile_a_plus", 4
	plans := []model.EcomSuiteImagePlan{first, mobile}
	valid := model.EcomSuiteCopywriting{ImageType: first.ImageType, ImageIndex: 1, Title: first.Title, Subtitle: first.Subtitle}
	invalid := model.EcomSuiteCopywriting{ImageType: mobile.ImageType, ImageIndex: 4, Title: "Adjust Water Flow", Subtitle: "A single lever lets you adjust the water flow easily."}
	calls := 0
	copies, err := runSmartSuiteCopywriterWithCall(context.Background(), task, plans, func(_ context.Context, requested []model.EcomSuiteImagePlan, feedback string) ([]model.EcomSuiteCopywriting, error) {
		calls++
		if calls == 1 {
			return []model.EcomSuiteCopywriting{valid, invalid}, nil
		}
		if len(requested) != 1 || requested[0].ImageIndex != 4 {
			t.Fatal("valid sibling regenerated")
		}
		for _, detail := range []string{"actual=10", "allowed=4..9", invalid.Subtitle, valid.Title} {
			if !strings.Contains(feedback, detail) {
				t.Fatalf("feedback missing %q: %s", detail, feedback)
			}
		}
		invalid.Subtitle = "A single lever controls water flow."
		return []model.EcomSuiteCopywriting{invalid}, nil
	})
	if err != nil || calls != 2 || len(copies) != 2 {
		t.Fatalf("recovery failed: calls=%d err=%v", calls, err)
	}
	if copies[0].Title != valid.Title || copies[1].Subtitle != invalid.Subtitle {
		t.Fatal("copy lost")
	}
}

func TestCopyLengthBoundariesAndIndependentFieldCounting(t *testing.T) {
	for _, tc := range []struct{ imageType, title, subtitle, want string }{
		{"smart_mobile_a_plus", "Adjust Water Flow", "One lever controls flow.", ""},
		{"smart_mobile_a_plus", "Adjust Water Flow", "One lever controls the water flow at the sink.", ""},
		{"smart_mobile_a_plus", "Adjust Water Flow", "One lever controls the water flow easily at the sink.", "subtitle_length"},
		{"smart_mobile_a_plus", "Adjust Water Flow", "One lever controls.", "subtitle_length"},
		{"smart_mobile_a_plus", "冲洗更加顺手", "抽拉喷头调整水流", ""},
		{"smart_mobile_a_plus", "Adjust Water Flow", "抽拉喷头调整水流", ""},
	} {
		copy := model.EcomSuiteCopywriting{ImageType: tc.imageType, ImageIndex: 4, Title: tc.title, Subtitle: tc.subtitle}
		if got := smartSuiteCopywritingCandidateIssue(copy, tc.imageType); got != tc.want {
			t.Fatalf("%q: got %s want %s", tc.subtitle, got, tc.want)
		}
	}
}

func TestCopywriterDuplicateTitlesAreNonBlockingQualityWarnings(t *testing.T) {
	task, snapshot := creativeGateFixture()
	first := snapshot.ImagePlans[0]
	second := first
	second.ImageIndex = first.ImageIndex + 1
	copies := []model.EcomSuiteCopywriting{
		{ImageType: first.ImageType, ImageIndex: first.ImageIndex, Title: "Adjust Water Flow", Subtitle: "A single lever controls water flow."},
		{ImageType: second.ImageType, ImageIndex: second.ImageIndex, Title: "Adjust Water Flow", Subtitle: "A single lever controls water flow."},
	}
	if _, err := validateSmartSuiteCopywriting(task, []model.EcomSuiteImagePlan{first, second}, copies); err != nil {
		t.Fatalf("duplicate title should not block generation: %v", err)
	}
}

func TestCopywriterFailureReportsLatestCandidateAndStops(t *testing.T) {
	task, snapshot := creativeGateFixture()
	plan := snapshot.ImagePlans[0]
	plan.ImageType = "smart_mobile_a_plus"
	plan.ImageIndex = 4
	calls := 0
	_, err := runSmartSuiteCopywriterWithCall(context.Background(), task, []model.EcomSuiteImagePlan{plan}, func(_ context.Context, _ []model.EcomSuiteImagePlan, feedback string) ([]model.EcomSuiteCopywriting, error) {
		calls++
		subtitle := "A single lever lets you adjust the water flow easily."
		if calls > 1 {
			subtitle = "Single lever."
		}
		copy := model.EcomSuiteCopywriting{ImageType: plan.ImageType, ImageIndex: 4, Title: "Adjust Water Flow", Subtitle: subtitle}
		_, err := validateSmartSuiteCopywriting(task, []model.EcomSuiteImagePlan{plan}, []model.EcomSuiteCopywriting{copy})
		return []model.EcomSuiteCopywriting{copy}, err
	})
	if calls != 2 || err == nil || !strings.Contains(err.Error(), "actual=2") || strings.Contains(err.Error(), "actual=10") {
		t.Fatalf("stale error or unbounded retry: calls=%d err=%v", calls, err)
	}
}

func TestShortMeaningfulTitlesPassWriterAndCreativeRepair(t *testing.T) {
	for _, title := range []string{"Water-Use Specifications", "Brushed Nickel", "Installation", "拉丝镍"} {
		for _, kind := range []string{"smart_secondary", "smart_desktop_a_plus", "smart_mobile_a_plus"} {
			t.Run(kind+"/"+title, func(t *testing.T) {
				task, snapshot := creativeGateFixture()
				plan := snapshot.ImagePlans[0]
				plan.ImageType = kind
				plan.ImageIndex = 5
				copy := model.EcomSuiteCopywriting{ImageType: kind, ImageIndex: 5, Title: title, Subtitle: plan.Subtitle}
				if issue := smartSuiteCopywritingCandidateIssue(copy, kind); issue != "" {
					t.Fatalf("short title rejected: %s", issue)
				}
				plan.Title = title
				if !smartSuiteRefinedCopyUsable(plan) {
					t.Fatal("planning uses different title limits")
				}
				// Length validity is not a factual or semantic endorsement of the heading.
				if title != "Water-Use Specifications" {
					return
				}
				issues := []model.EcomSuiteCreativeIssue{{ImageType: kind, ImageIndex: 5}}
				raw, _ := json.Marshal(map[string]interface{}{"plans": []model.EcomSuiteCopywriting{copy}})
				if _, err := mergeSmartSuiteCreativeRepairs(task, []model.EcomSuiteImagePlan{plan}, issues, string(raw)); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestTitleUpperBoundsAndContentChecksRemain(t *testing.T) {
	for _, tc := range []struct{ title, want string }{
		{"", "missing_identity_or_copy"},
		{"Adjust Your Faucet Water Flow With One Simple Lever", "title_length"},
		{"Built with Purpose", "generic_copy"},
		{"Pictured Faucet", "caption_language"},
	} {
		copy := model.EcomSuiteCopywriting{ImageType: "smart_secondary", ImageIndex: 5, Title: tc.title, Subtitle: "A single lever controls water flow."}
		if got := smartSuiteCopywritingCandidateIssue(copy, copy.ImageType); got != tc.want {
			t.Fatalf("%q got=%s want=%s", tc.title, got, tc.want)
		}
	}
}
