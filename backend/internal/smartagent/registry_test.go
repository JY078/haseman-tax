package smartagent

import (
	"strings"
	"testing"
)

func TestEveryRegisteredAgentLoadsItsSkill(t *testing.T) {
	if len(Names()) != 22 {
		t.Fatalf("expected 22 registered agents, got %d", len(Names()))
	}
	for _, name := range Names() {
		instructions, err := Instructions(name)
		if err != nil {
			t.Fatalf("load %s: %v", name, err)
		}
		if strings.Contains(instructions, "TODO") || !strings.Contains(instructions, "# ") {
			t.Fatalf("skill %s is incomplete", name)
		}
	}
}

func TestSystemPromptAppendsRuntimeContract(t *testing.T) {
	prompt, err := SystemPrompt(WriteSuiteCopy, "Return one requested item.")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "# Write Suite Copy") || !strings.HasSuffix(prompt, "Return one requested item.") {
		t.Fatalf("unexpected prompt: %s", prompt)
	}
}

func TestUnknownSkillFailsClosed(t *testing.T) {
	if _, err := Instructions(Name("missing-agent")); err == nil {
		t.Fatal("unknown skills must not fall back to a generic prompt")
	}
}
