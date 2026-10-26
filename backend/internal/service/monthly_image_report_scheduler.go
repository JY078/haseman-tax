package service

import (
	"context"
	"log"
	"time"

	"advertising-image/backend/internal/cache"
)

const monthlyImageReportSendHour = 9
const monthlyImageReportTimeZone = "Asia/Shanghai"
const monthlyImageReportRetryCount = 3
const monthlyImageReportRetryInterval = 5 * time.Minute
const monthlyImageReportScheduleRetryInterval = 10 * time.Minute
const monthlyImageReportDispatchTTL = 45 * 24 * time.Hour

func nextMonthlyImageReportRun(now time.Time, location *time.Location) time.Time {
	localNow := now.In(location)
	next := time.Date(localNow.Year(), localNow.Month(), 1, monthlyImageReportSendHour, 0, 0, 0, location)
	if !next.After(localNow) {
		next = next.AddDate(0, 1, 0)
	}
	return next
}

func sendMonthlyImageReportWithRetry(month, chatID string) error {
	dispatchKey := "image-report:monthly:sent:" + month
	claimedDispatch := false
	if cache.RDB != nil {
		claimed, err := cache.RDB.SetNX(context.Background(), dispatchKey, "sending", 30*time.Minute).Result()
		if err != nil {
			log.Printf("[image-report] claim %s failed, continuing without deduplication: %v", month, err)
		} else if !claimed {
			log.Printf("[image-report] %s already sent or being sent, skipping", month)
			return nil
		} else {
			claimedDispatch = true
		}
	}

	var lastErr error
	for attempt := 1; attempt <= monthlyImageReportRetryCount; attempt++ {
		if err := GenerateAndSendMonthlyImageReport(month, chatID); err == nil {
			if cache.RDB != nil {
				if setErr := cache.RDB.Set(context.Background(), dispatchKey, "sent", monthlyImageReportDispatchTTL).Err(); setErr != nil {
					log.Printf("[image-report] mark %s as sent failed: %v", month, setErr)
				}
			}
			return nil
		} else {
			lastErr = err
			log.Printf("[image-report] send %s attempt %d/%d failed: %v", month, attempt, monthlyImageReportRetryCount, err)
		}
		if attempt < monthlyImageReportRetryCount {
			time.Sleep(monthlyImageReportRetryInterval)
		}
	}
	if cache.RDB != nil && claimedDispatch {
		if err := cache.RDB.Del(context.Background(), dispatchKey).Err(); err != nil {
			log.Printf("[image-report] release failed dispatch %s failed: %v", month, err)
		}
	}
	return lastErr
}

// StartMonthlyImageReportScheduler sends the previous month's report on the
// first day of each month when explicitly enabled and a chat ID is configured.
func StartMonthlyImageReportScheduler() {
	location, err := time.LoadLocation(monthlyImageReportTimeZone)
	if err != nil {
		log.Printf("[image-report] scheduler disabled: failed to load %s: %v", monthlyImageReportTimeZone, err)
		return
	}
	go func() {
		startupNow := time.Now().In(location)
		var nextRun time.Time
		if startupNow.Day() <= 7 && startupNow.Hour() >= monthlyImageReportSendHour {
			month := startupNow.AddDate(0, -1, 0).Format("2006-01")
			if err := sendMonthlyImageReportWithRetry(month, monthlyImageReportChatID); err != nil {
				log.Printf("[image-report] startup catch-up for %s failed: %v", month, err)
				nextRun = time.Now().Add(monthlyImageReportScheduleRetryInterval)
			} else {
				log.Printf("[image-report] startup catch-up for %s sent to Feishu chat", month)
			}
		}
		if nextRun.IsZero() {
			nextRun = nextMonthlyImageReportRun(time.Now(), location)
		}
		for {
			timer := time.NewTimer(time.Until(nextRun))
			<-timer.C
			month := time.Now().In(location).AddDate(0, -1, 0).Format("2006-01")
			if err := sendMonthlyImageReportWithRetry(month, monthlyImageReportChatID); err != nil {
				log.Printf("[image-report] send %s failed: %v", month, err)
				// Keep retrying after a transient Feishu/database failure. The
				// scheduler must recover without requiring a backend restart.
				nextRun = time.Now().Add(monthlyImageReportScheduleRetryInterval)
			} else {
				log.Printf("[image-report] sent %s to Feishu chat", month)
				nextRun = nextMonthlyImageReportRun(time.Now(), location)
			}
		}
	}()
	log.Printf("[image-report] monthly Feishu report scheduler started at day 1 %02d:00 (%s)", monthlyImageReportSendHour, monthlyImageReportTimeZone)
}
