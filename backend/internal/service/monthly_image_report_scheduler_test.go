package service

import (
	"testing"
	"time"
)

func TestNextMonthlyImageReportRunUsesConfiguredTimezone(t *testing.T) {
	location, err := time.LoadLocation(monthlyImageReportTimeZone)
	if err != nil {
		t.Fatalf("load timezone: %v", err)
	}

	utcNow := time.Date(2026, 9, 1, 0, 30, 0, 0, time.UTC)
	next := nextMonthlyImageReportRun(utcNow, location)
	if got := next.Format(time.RFC3339); got != "2026-09-01T09:00:00+08:00" {
		t.Fatalf("unexpected next run: %s", got)
	}

	afterRun := time.Date(2026, 9, 1, 1, 30, 0, 0, time.UTC)
	next = nextMonthlyImageReportRun(afterRun, location)
	if got := next.Format(time.RFC3339); got != "2026-10-01T09:00:00+08:00" {
		t.Fatalf("unexpected following run: %s", got)
	}
}
