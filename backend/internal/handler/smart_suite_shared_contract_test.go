package handler

import (
	"fmt"
	"strings"
	"testing"

	"advertising-image/backend/internal/smartagent"
)

func TestSuiteProducersAndEvaluatorShareExecutableConstraints(t *testing.T) {
	for _, agent := range []smartagent.Name{smartagent.WriteSuiteCopy, smartagent.PlanSuiteImage, smartagent.EvaluateSuiteCreative} {
		t.Run(string(agent), func(t *testing.T) {
			prompt, err := smartSuiteAgentPrompt(agent, "Return the role's JSON schema.")
			if err != nil {
				t.Fatal(err)
			}
			for _, constraint := range []string{smartSuiteCreativeFactContract, smartSuiteCreativeEvidenceContract, smartSuiteCreativeLayoutContract, "Short meaningful titles are valid"} {
				if !strings.Contains(prompt, constraint) {
					t.Fatal("producer/reviewer constraint mismatch")
				}
			}
			if !strings.Contains(prompt, smartSuiteCreativeCoverageContract) {
				t.Fatal("suite coverage contract missing")
			}
			for _, kind := range []string{"smart_secondary", "smart_desktop_a_plus", "smart_mobile_a_plus"} {
				for _, lang := range []struct{ name, sample string }{{"space-delimited", "English"}, {"Chinese", "中文"}} {
					for _, title := range []bool{true, false} {
						field := "subtitle"
						if title {
							field = "title"
						}
						min, max, unit := smartSuiteCopyLengthLimits(kind, lang.sample, title)
						line := fmt.Sprintf("%s %s %s: %d..%d %s.", kind, lang.name, field, min, max, unit)
						if !strings.Contains(prompt, line) {
							t.Fatalf("validator bounds missing: %s", line)
						}
					}
				}
			}
		})
	}
}

func TestSuiteExecutionRulesDoNotLeakIntoFactExtractionOrListing(t *testing.T) {
	for _, agent := range []smartagent.Name{smartagent.AnalyzeProductFacts, smartagent.WriteProductListing, smartagent.ReviewSuiteImage} {
		prompt, err := smartSuiteAgentPrompt(agent, "role contract")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(prompt, "## Shared backend execution contract") {
			t.Fatalf("unrelated agent received plan repair authority: %s", agent)
		}
	}
}
