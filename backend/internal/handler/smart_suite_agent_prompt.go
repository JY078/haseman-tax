package handler

import (
	"fmt"
	"log"
	"strings"

	"advertising-image/backend/internal/smartagent"
)

func smartSuiteAgentPrompt(name smartagent.Name, runtimeContract string) (string, error) {
	switch name {
	case smartagent.WriteSuiteCopy, smartagent.PlanSuiteImage, smartagent.EvaluateSuiteCreative:
		runtimeContract += "\n\n" + smartSuiteSharedExecutionContract()
	}
	if name == smartagent.EvaluateProductListing || name == smartagent.ReviewWayfairListing {
		writingRules, err := smartagent.Instructions(smartagent.WriteWayfairListing)
		if err != nil {
			return "", err
		}
		runtimeContract += "\n\n## Shared Wayfair Writing Standard\nApply the following only when the platform is Wayfair. Reuse its facts, style and priority rules. The evaluator/reviewer response schema takes precedence over the writer's JSON/evidence instructions. Preserve already effective copy when no specific issue requires rewriting.\n" + writingRules
	}
	return smartagent.SystemPrompt(name, runtimeContract)
}

// Send the actual backend constraints to every post-assignment producer and
// evaluator. Numerical bounds come from the validator, not another prompt copy.
func smartSuiteSharedExecutionContract() string {
	var rules strings.Builder
	rules.WriteString("## Shared backend execution contract\nThese constraints govern executable output if any other instruction conflicts. Quality preferences must not invent additional hard requirements.\n")
	rules.WriteString(smartSuiteCreativeFactContract + "\n" + smartSuiteCreativeCoverageContract + "\n" + smartSuiteCreativeEvidenceContract + "\n" + smartSuiteCreativeLayoutContract + "\n")
	for _, imageType := range []string{"smart_secondary", "smart_desktop_a_plus", "smart_mobile_a_plus"} {
		for _, language := range []struct{ name, sample string }{{"space-delimited", "English"}, {"Chinese", "中文"}} {
			for _, title := range []bool{true, false} {
				field := "subtitle"
				if title {
					field = "title"
				}
				min, max, unit := smartSuiteCopyLengthLimits(imageType, language.sample, title)
				fmt.Fprintf(&rules, "%s %s %s: %d..%d %s.\n", imageType, language.name, field, min, max, unit)
			}
		}
	}
	rules.WriteString("Count each visible field independently. Short meaningful titles are valid; do not pad them. Respect assigned facts when rewriting length failures. A structural pass is not proof of semantic quality; judge unsupported claims and unclear meaning separately. Return only the schema requested by your current role; these shared constraints do not authorize changing immutable assignments or returning extra fields.")
	return rules.String()
}

// Prompt builders historically return only two strings. Keep that API stable
// while still making the embedded skill the primary prompt source.
func smartSuiteAgentPromptText(name smartagent.Name, runtimeContract string) string {
	prompt, err := smartSuiteAgentPrompt(name, runtimeContract)
	if err != nil {
		log.Printf("[EcomSuite] smart agent skill unavailable: name=%s err=%v", name, err)
		return runtimeContract
	}
	return prompt
}
