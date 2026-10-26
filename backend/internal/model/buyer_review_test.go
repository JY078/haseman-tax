package model

import "testing"

func TestNormalizeReviewLink(t *testing.T) {
	base := "https://www.amazon.com/gp/customer-reviews/R1"
	same := []string{
		base,
		"https://www.amazon.com/gp/customer-reviews/R1",
		"https://WWW.AMAZON.COM/gp/customer-reviews/R1",
		"http://www.amazon.com/gp/customer-reviews/R1",
		base + "?filterByStar=five_star",
		base + "?filterByStar=one_star&pageNumber=2",
		base + "#review-comments",
		"https://www.amazon.com/gp/customer-reviews/R1/",
		"  " + base + "  ",
	}
	for _, link := range same {
		if got := NormalizeReviewLink(link); got != base {
			t.Fatalf("NormalizeReviewLink(%q) = %q; want %q", link, got, base)
		}
	}
	different := []string{
		"https://www.amazon.com/gp/customer-reviews/R2",
		"https://www.amazon.com/gp/customer-reviews/R1-extra",
		"https://smile.amazon.com/gp/customer-reviews/R1",
	}
	for _, link := range different {
		if got := NormalizeReviewLink(link); got == base {
			t.Fatalf("NormalizeReviewLink(%q) collapsed into %q", link, base)
		}
	}
	if got := NormalizeReviewLink("  "); got != "" {
		t.Fatalf("blank link = %q; want empty", got)
	}
	if got := NormalizeReviewLink("Not A URL"); got != "not a url" {
		t.Fatalf("fallback = %q; want trimmed lowercase", got)
	}
}

func TestBuyerReviewFingerprintLinkIdentity(t *testing.T) {
	link := "https://www.amazon.com/gp/customer-reviews/R1"
	variants := []string{
		link,
		"http://www.amazon.com/gp/customer-reviews/R1?filterByStar=five_star",
		"https://WWW.AMAZON.COM/gp/customer-reviews/R1/",
	}
	first := BuyerReviewFingerprint(variants[0], "B0A", "2026-01-01", "甲", "标题", "内容")
	for _, variant := range variants[1:] {
		if got := BuyerReviewFingerprint(variant, "B0B", "2026-02-02", "乙", "另一标题", "另一内容"); got != first {
			t.Fatalf("link variant %s produced different fingerprint", variant)
		}
	}
	if got := BuyerReviewFingerprint("https://www.amazon.com/gp/customer-reviews/R2", "B0A", "2026-01-01", "甲", "标题", "内容"); got == first {
		t.Fatal("different links must not share a fingerprint")
	}
}

func TestBuyerReviewFingerprintEmptyLinkFallback(t *testing.T) {
	a := BuyerReviewFingerprint("", "B0A", "2026-01-01", "甲", "标题", "内容")
	b := BuyerReviewFingerprint("   ", "B0A", "2026-01-01", "甲", "标题", "内容")
	c := BuyerReviewFingerprint("", "B0A", "2026-01-02", "甲", "标题", "内容")
	if a != b {
		t.Fatal("blank links must keep identical fallback fingerprints")
	}
	if a == c {
		t.Fatal("fallback identity must still distinguish rows")
	}
}
