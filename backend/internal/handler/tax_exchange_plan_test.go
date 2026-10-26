package handler

import (
	"testing"
	"time"

	"advertising-image/backend/internal/model"
	"github.com/shopspring/decimal"
)

func TestTaxExchangeDateRange(t *testing.T) {
	parse := func(value string) time.Time {
		date, err := time.Parse(time.RFC3339, value)
		if err != nil {
			t.Fatal(err)
		}
		return date
	}
	dates := taxExchangeDateRange(parse("2026-01-31T23:00:00+08:00"), parse("2026-02-02T01:00:00+08:00"))
	if len(dates) != 3 || dates[0] != "2026-01-31" || dates[2] != "2026-02-02" {
		t.Fatalf("must include every date through today: %#v", dates)
	}
	if dates := taxExchangeDateRange(time.Time{}, time.Now()); len(dates) != 0 {
		t.Fatal("empty history has a range")
	}
	if dates := taxExchangeDateRange(parse("2027-01-01T00:00:00+08:00"), parse("2026-01-01T00:00:00+08:00")); len(dates) != 0 {
		t.Fatal("future transactions should not request future rates")
	}
}

func TestMissingTaxExchangeRequests(t *testing.T) {
	existing := []model.TaxExchangeRate{
		{Date: "2026-01-01", Currency: "USD", Rate: decimal.NewFromInt(7)},
		{Date: "2026-01-01", Currency: "EUR", Rate: decimal.Zero},
		{Date: "2026-01-01", Currency: "CAD", Rate: decimal.NewFromInt(-1)},
		{Date: "2025-01-01", Currency: "GBP", Rate: decimal.NewFromInt(9)},
	}
	missing := missingTaxExchangeRequests([]string{"2026-01-01", "2026-01-02"}, existing)
	if len(missing) != 13 {
		t.Fatalf("expected 13 missing pairs, got %#v", missing)
	}
	for _, item := range missing {
		if item.Date == "2026-01-01" && item.Currency == "USD" {
			t.Fatal("valid rate must be skipped")
		}
	}
	if len(missingTaxExchangeRequests(nil, existing)) != 0 {
		t.Fatal("no date range must mean no requests")
	}
}

func TestWeekendTaxRateDate(t *testing.T) {
	for input, want := range map[string]string{"2026-01-02": "2026-01-02", "2026-01-03": "2026-01-05", "2026-01-04": "2026-01-05", "2026-01-05": "2026-01-05", "2026-01-31": "2026-02-02", "2028-12-31": "2029-01-01"} {
		date, err := time.Parse("2006-01-02", input)
		if err != nil {
			t.Fatal(err)
		}
		if got := taxRateDate(date); got != want {
			t.Fatalf("%s: got %s want %s", input, got, want)
		}
	}
	missing := missingTaxExchangeRequests([]string{"2026-01-02", "2026-01-03", "2026-01-04", "2026-01-05"}, nil)
	if len(missing) != 14 {
		t.Fatalf("only Friday and Monday should be fetched: %+v", missing)
	}
	for _, row := range missing {
		if row.Date == "2026-01-03" || row.Date == "2026-01-04" {
			t.Fatal("requested weekend rate")
		}
	}
}

func TestStoredTaxRateHoliday(t *testing.T) {
	rates := map[string]decimal.Decimal{
		"USD:2025-10-14": decimal.RequireFromString("7"),
		"USD:2026-04-07": decimal.RequireFromString("7.1457"),
		"USD:2026-04-08": decimal.RequireFromString("7.2"),
	}
	date, _ := time.Parse("2006-01-02", "2025-10-13")
	if rate, ok := storedTaxRate(rates, "USD", date); !ok || rate.String() != "7" {
		t.Fatalf("holiday before first publication: got %s, %v", rate, ok)
	}
	for _, input := range []string{"2026-04-04", "2026-04-05", "2026-04-06", "2026-04-07"} {
		date, _ := time.Parse("2006-01-02", input)
		rate, ok := storedTaxRate(rates, "USD", date)
		if !ok || rate.String() != "7.1457" {
			t.Fatalf("%s: got %s, %v", input, rate, ok)
		}
	}
	for _, input := range []string{"2025-01-01", "2025-11-01", "2026-04-09"} {
		date, _ := time.Parse("2006-01-02", input)
		if _, ok := storedTaxRate(rates, "USD", date); ok {
			t.Fatalf("out-of-coverage date %s must stay pending", input)
		}
	}
}
