package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var taxExchangeLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

// The endpoint publishes one date for all currency pairs, including on holidays.
func fetchChinaMoneyRates(ctx context.Context) (map[string]model.TaxRate, error) {
	now := time.Now().UTC()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://www.chinamoney.com.cn/r/cms/www/chinamoney/data/fx/ccpr.json?t="+strconv.FormatInt(now.UnixMilli(), 10), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Origin", "https://www.chinamoney.com.cn")
	req.Header.Set("Referer", "https://www.chinamoney.com.cn/chinese/bkccpr/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	client := http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("中国货币网请求失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("中国货币网返回 HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Head struct {
			Code      string `json:"rep_code"`
			Timestamp int64  `json:"ts"`
		} `json:"head"`
		Data struct {
			LastDate string `json:"lastDate"`
		} `json:"data"`
		Records []struct {
			Pair  string `json:"vrtEName"`
			Price string `json:"price"`
		} `json:"records"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("中国货币网响应格式无效")
	}
	if payload.Head.Code != "200" || len(payload.Data.LastDate) < 10 {
		return nil, fmt.Errorf("中国货币网未返回有效报价")
	}
	date := payload.Data.LastDate[:10]
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, fmt.Errorf("中国货币网发布日期无效")
	}
	rates := make(map[string]model.TaxRate, len(exchangeCurrencies))
	for _, currency := range exchangeCurrencies {
		pair := currency + "/CNY"
		if currency == "MXN" || currency == "PLN" {
			pair = "CNY/" + currency
		}
		for _, record := range payload.Records {
			if record.Pair != pair {
				continue
			}
			rate, err := decimal.NewFromString(record.Price)
			if err != nil || !rate.IsPositive() || rate.GreaterThan(decimal.NewFromInt(1000)) {
				return nil, fmt.Errorf("中国货币网 %s/CNY 汇率无效", currency)
			}
			if currency == "MXN" || currency == "PLN" {
				// Invert yuan/foreign-currency quotes to store yuan per unit.
				rate = decimal.NewFromInt(1).DivRound(rate, 12)
			}
			rates[currency] = model.TaxRate{Rate: rate, Date: date, Source: "ChinaMoney", FetchedAt: now, ProviderTimestamp: payload.Head.Timestamp}
			break
		}
		if _, ok := rates[currency]; !ok {
			return nil, fmt.Errorf("中国货币网缺少 %s/CNY 汇率", currency)
		}
	}
	return rates, nil
}

func saveLatestTaxExchangeRates(ctx context.Context, db *gorm.DB) (string, error) {
	rates, err := fetchChinaMoneyRates(ctx)
	if err != nil {
		return "", err
	}
	rows := make([]model.TaxExchangeRate, 0, len(exchangeCurrencies))
	for _, currency := range exchangeCurrencies {
		rate := rates[currency]
		rows = append(rows, model.TaxExchangeRate{Date: rate.Date, Currency: currency, Rate: rate.Rate, Source: rate.Source, FetchedAt: rate.FetchedAt})
	}
	// Only the returned publication date is updated; historical imports stay intact.
	err = db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "date"}, {Name: "currency"}},
		DoUpdates: clause.AssignmentColumns([]string{"rate", "source", "fetched_at"}),
	}).Create(&rows).Error
	return rates["USD"].Date, err
}

func nextTaxExchangeFetch(now time.Time) time.Time {
	now = now.In(taxExchangeLocation)
	next := time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, taxExchangeLocation)
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func StartTaxExchangeRateScheduler() {
	log.Printf("[汇率定时获取] scheduler started，计划时间：每天 09:30（Asia/Shanghai）")
	go func() {
		// Fetch once when the service starts so a restart after 09:30 does not
		// leave the exchange-rate page stale until the following day.
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		date, err := saveLatestTaxExchangeRates(ctx, database.DB)
		cancel()
		if err != nil {
			log.Printf("[汇率定时获取] 启动获取失败：%v", err)
		} else {
			log.Printf("[汇率定时获取] 启动时已保存 %s 的 %d 种币种汇率", date, len(exchangeCurrencies))
		}

		for {
			timer := time.NewTimer(time.Until(nextTaxExchangeFetch(time.Now())))
			<-timer.C
			ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
			date, err = saveLatestTaxExchangeRates(ctx, database.DB)
			cancel()
			if err != nil {
				log.Printf("[汇率定时获取] 失败：%v", err)
			} else {
				log.Printf("[汇率定时获取] 已保存 %s 的 %d 种币种汇率", date, len(exchangeCurrencies))
			}
		}
	}()
}
