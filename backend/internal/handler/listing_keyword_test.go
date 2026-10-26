package handler

import "testing"

func TestNormalizeListingPlatformAliases(t *testing.T) {
	cases := map[string]string{
		"TikTok Shop": "tiktok",
		"Home Depot":  "home_depot",
		"Lowe’s":      "lowes",
		"Wayfair":     "wayfair",
		"Amazon":      "amazon",
	}
	for input, expected := range cases {
		if got := normalizeListingPlatform(input); got != expected {
			t.Fatalf("normalizeListingPlatform(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestSplitListingKeywordsDeduplicatesAndNormalizes(t *testing.T) {
	got := splitListingKeywords("Bathroom Faucet, deck mount\nBATHROOM FAUCET； 3 hole faucet ")
	if len(got) != 3 {
		t.Fatalf("expected 3 terms, got %#v", got)
	}
	if got[0] != "bathroom faucet" || got[1] != "deck mount" || got[2] != "3 hole faucet" {
		t.Fatalf("unexpected terms: %#v", got)
	}
}
