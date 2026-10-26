package handler

import "testing"

func TestUniqueProductSPUIDs(t *testing.T) {
	actual := uniqueProductSPUIDs([]uint{3, 0, 2, 3, 2, 8})
	expected := []uint{3, 2, 8}
	if len(actual) != len(expected) {
		t.Fatalf("uniqueProductSPUIDs() = %v; want %v", actual, expected)
	}
	for index := range expected {
		if actual[index] != expected[index] {
			t.Fatalf("uniqueProductSPUIDs() = %v; want %v", actual, expected)
		}
	}
}

func TestIsDuplicateProductSPUError(t *testing.T) {
	for _, message := range []string{
		`duplicate key value violates unique constraint "idx_product_spus_spu"`,
		"ERROR: duplicate key value violates unique constraint (SQLSTATE 23505)",
	} {
		if !isDuplicateProductSPUError(testError(message)) {
			t.Fatalf("duplicate error %q was not detected", message)
		}
	}
	if isDuplicateProductSPUError(testError("connection refused")) {
		t.Fatal("unrelated error was detected as duplicate")
	}
}

type testError string

func (e testError) Error() string { return string(e) }

func TestNormalizeProductCertificationType(t *testing.T) {
	tests := map[string]string{
		"CEC":     "CEC",
		" doe ":   "DOE",
		"CUPC":    "cUPC",
		"NSF 61":  "NSF 61",
		"nsf 372": "NSF 372",
		"WARS":    "WARS",
		"acs":     "ACS",
	}
	for input, expected := range tests {
		actual, ok := normalizeProductCertificationType(input)
		if !ok || actual != expected {
			t.Fatalf("normalizeProductCertificationType(%q) = %q, %v; want %q, true", input, actual, ok, expected)
		}
	}
	if _, ok := normalizeProductCertificationType("FCC"); ok {
		t.Fatal("FCC should not be accepted")
	}
}

func TestNormalizeProductCertificationDate(t *testing.T) {
	if actual, ok := normalizeProductCertificationDate("2026-07-27"); !ok || actual != "2026-07-27" {
		t.Fatalf("valid date = %q, %v", actual, ok)
	}
	for _, value := range []string{"", "2026/07/27", "2026-02-30"} {
		if _, ok := normalizeProductCertificationDate(value); ok {
			t.Fatalf("invalid date %q was accepted", value)
		}
	}
}
