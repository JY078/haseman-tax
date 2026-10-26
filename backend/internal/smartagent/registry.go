package smartagent

import (
	"embed"
	"fmt"
	"sort"
	"strings"
)

type Name string

const (
	OrchestrateSuite            Name = "orchestrate-suite"
	UnderstandSuiteRequirements Name = "understand-suite-requirements"
	AnalyzeProductFacts         Name = "analyze-product-facts"
	MapBuyerConcerns            Name = "map-buyer-concerns"
	LockSuiteStyle              Name = "lock-suite-style"
	DirectSuiteCreative         Name = "direct-suite-creative"
	DirectSecondarySuite        Name = "direct-secondary-suite"
	WriteSuiteCopy              Name = "write-suite-copy"
	PlanSuiteImage              Name = "plan-suite-image"
	EvaluateSuiteCreative       Name = "evaluate-suite-creative"
	ReviewSuiteImage            Name = "review-suite-image"
	WriteProductListing         Name = "write-product-listing"
	WriteTikTokListing          Name = "write-tiktok-listing"
	WriteHomeDepotListing       Name = "write-home-depot-listing"
	WriteLowesListing           Name = "write-lowes-listing"
	WriteWayfairListing         Name = "write-wayfair-listing"
	WriteAmazonListing          Name = "write-amazon-listing"
	ResearchListingKeywords     Name = "research-listing-keywords"
	ReviewHomeDepotListing      Name = "review-home-depot-listing"
	ReviewLowesListing          Name = "review-lowes-listing"
	ReviewWayfairListing        Name = "review-wayfair-listing"
	EvaluateProductListing      Name = "evaluate-product-listing"
)

var skillNames = []Name{
	OrchestrateSuite,
	UnderstandSuiteRequirements,
	AnalyzeProductFacts,
	MapBuyerConcerns,
	LockSuiteStyle,
	DirectSuiteCreative,
	DirectSecondarySuite,
	WriteSuiteCopy,
	PlanSuiteImage,
	EvaluateSuiteCreative,
	ReviewSuiteImage,
	WriteProductListing,
	WriteTikTokListing,
	WriteHomeDepotListing,
	WriteLowesListing,
	WriteWayfairListing,
	WriteAmazonListing,
	ResearchListingKeywords,
	ReviewHomeDepotListing,
	ReviewLowesListing,
	ReviewWayfairListing,
	EvaluateProductListing,
}

//go:embed skills/*/SKILL.md
var skillFiles embed.FS

func Names() []Name {
	names := append([]Name(nil), skillNames...)
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	return names
}

func Instructions(name Name) (string, error) {
	if !known(name) {
		return "", fmt.Errorf("smart_agent_unknown_skill: %s", name)
	}
	content, err := skillFiles.ReadFile(fmt.Sprintf("skills/%s/SKILL.md", name))
	if err != nil {
		return "", fmt.Errorf("smart_agent_skill_unavailable: %s: %w", name, err)
	}
	instructions := stripFrontmatter(string(content))
	if instructions == "" {
		return "", fmt.Errorf("smart_agent_skill_empty: %s", name)
	}
	return instructions, nil
}

func SystemPrompt(name Name, runtimeContract string) (string, error) {
	instructions, err := Instructions(name)
	if err != nil {
		return "", err
	}
	runtimeContract = strings.TrimSpace(runtimeContract)
	if runtimeContract == "" {
		return instructions, nil
	}
	return instructions + "\n\n## Runtime Contract\n\n" + runtimeContract, nil
}

func known(name Name) bool {
	for _, candidate := range skillNames {
		if candidate == name {
			return true
		}
	}
	return false
}

func stripFrontmatter(content string) string {
	content = strings.TrimSpace(strings.ReplaceAll(content, "\r\n", "\n"))
	if !strings.HasPrefix(content, "---\n") {
		return content
	}
	if end := strings.Index(content[4:], "\n---\n"); end >= 0 {
		return strings.TrimSpace(content[end+9:])
	}
	return ""
}
