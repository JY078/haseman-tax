package handler

import (
	"reflect"
	"testing"
	"time"
)

func TestImageStatsMonthRange(t *testing.T) {
	start, end, err := imageStatsMonthRange("2026-07")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := start.Format("2006-01-02"); got != "2026-07-01" {
		t.Fatalf("unexpected start: %s", got)
	}
	if got := end.Format("2006-01-02"); got != "2026-08-01" {
		t.Fatalf("unexpected end: %s", got)
	}

	start, end, err = imageStatsMonthRange("")
	if err != nil || !start.Equal(time.Time{}) || !end.Equal(time.Time{}) {
		t.Fatalf("empty month should return an unbounded range")
	}

	if _, _, err = imageStatsMonthRange("2026/07"); err == nil {
		t.Fatal("invalid month should return an error")
	}
}

func TestImageStatsDepartmentCondition(t *testing.T) {
	condition, arg := imageStatsDepartmentCondition("美国1部")
	if condition != " AND users.department = ?" || arg != "美国1部" {
		t.Fatalf("primary department should use an exact match: %q %#v", condition, arg)
	}

	condition, arg = imageStatsDepartmentCondition("其他")
	if condition != " AND COALESCE(users.department, '') NOT IN ?" {
		t.Fatalf("other departments should use an exclusion condition: %q", condition)
	}
	if !reflect.DeepEqual(arg, imageStatsPrimaryDepartments) {
		t.Fatalf("unexpected primary department exclusions: %#v", arg)
	}
}
