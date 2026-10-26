package handler

import (
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Weekend transactions use the following Monday, without changing their period.
func taxRateDate(date time.Time) string {
	switch date.Weekday() {
	case time.Saturday:
		date = date.AddDate(0, 0, 2)
	case time.Sunday:
		date = date.AddDate(0, 0, 1)
	}
	return date.Format("2006-01-02")
}

func taxRateDateSQL(date string) string {
	return "to_char((" + date + ")::date + CASE EXTRACT(ISODOW FROM (" + date + ")::date)::int WHEN 6 THEN 2 WHEN 7 THEN 1 ELSE 0 END, 'YYYY-MM-DD')"
}

// Skip holidays using the next stored publication within the nearby date range.
func taxRateJoinSQL(alias, date, currency string) string {
	target := taxRateDateSQL(date)
	return "LEFT JOIN LATERAL (SELECT rate FROM tax_exchange_rates WHERE currency = " + currency +
		" AND rate > 0 AND date >= " + target +
		" AND date <= to_char((" + date + ")::date + interval '14 days', 'YYYY-MM-DD')" +
		" ORDER BY date LIMIT 1) " + alias + " ON TRUE"
}

func storedTaxRate(rates map[string]decimal.Decimal, currency string, posted time.Time) (decimal.Decimal, bool) {
	target := taxRateDate(posted)
	key := currency + ":" + target
	if rate, ok := rates[key]; ok && rate.IsPositive() {
		return rate, true
	}
	next := ""
	var rate decimal.Decimal
	prefix := currency + ":"
	for key, value := range rates {
		if !strings.HasPrefix(key, prefix) || !value.IsPositive() {
			continue
		}
		date := strings.TrimPrefix(key, prefix)
		if date >= target && (next == "" || date < next) {
			next, rate = date, value
		}
	}
	if next == "" {
		return decimal.Zero, false
	}
	// A long gap means history is missing, rather than a weekend or holiday.
	if next > posted.AddDate(0, 0, 14).Format("2006-01-02") {
		return decimal.Zero, false
	}
	rates[key] = rate
	return rate, true
}
