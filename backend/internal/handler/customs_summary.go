package handler

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func customsFilteredQuery(c *gin.Context) (*gorm.DB, bool) {
	query := database.DB.Model(&model.CustomsDeclaration{})
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(keyword) + "%"
		query = query.Where("(declaration_number ILIKE ? OR file_name ILIKE ? OR fields->>'consignor' ILIKE ? OR fields->>'contract_number' ILIKE ? OR fields->>'bill_number' ILIKE ?)", like, like, like, like, like)
	}
	from, to := c.Query("date_from"), c.Query("date_to")
	for _, value := range []string{from, to} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				c.JSON(400, gin.H{"error": "日期格式应为 YYYY-MM-DD"})
				return nil, false
			}
		}
	}
	if from != "" && to != "" && from > to {
		c.JSON(400, gin.H{"error": "开始日期不能晚于结束日期"})
		return nil, false
	}
	if from != "" {
		query = query.Where("declaration_date >= ?", from)
	}
	if to != "" {
		query = query.Where("declaration_date <= ?", to)
	}
	if port := strings.TrimSpace(c.Query("destination_port")); port != "" {
		query = query.Where("BTRIM(fields->>'destination_port') = ?", port)
	}
	return query, true
}

type customsMeasure struct {
	Value   decimal.Decimal `json:"value"`
	Missing int             `json:"missing"`
	Count   int             `json:"count"`
}

var customsNumber = regexp.MustCompile(`^(?:[0-9]+|[0-9]{1,3}(?:,[0-9]{3})+)(?:\.[0-9]+)?$`)
var customsCurrency = regexp.MustCompile(`^[A-Z]{3}$`)

func (m *customsMeasure) add(raw string) bool {
	raw = strings.TrimSpace(raw)
	if !customsNumber.MatchString(raw) {
		m.Missing++
		return false
	}
	value, err := decimal.NewFromString(strings.ReplaceAll(raw, ",", ""))
	if err != nil {
		m.Missing++
		return false
	}
	m.Value = m.Value.Add(value)
	m.Count++
	return true
}

type customsAmounts struct {
	Currency  string         `json:"currency"`
	Goods     customsMeasure `json:"goods"`
	Freight   customsMeasure `json:"freight"`
	Insurance customsMeasure `json:"insurance"`
}
type customsCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
type customsSummary struct {
	Total               int               `json:"total"`
	ItemCount           int               `json:"item_count"`
	ReviewCount         int               `json:"review_count"`
	Packages            customsMeasure    `json:"packages"`
	GrossWeight         customsMeasure    `json:"gross_weight"`
	NetWeight           customsMeasure    `json:"net_weight"`
	UnaggregatedAmounts int               `json:"unaggregated_amounts"`
	Amounts             []*customsAmounts `json:"amounts"`
	Months              []customsCount    `json:"months"`
	Destinations        []customsCount    `json:"destinations"`
	DestinationOptions  []string          `json:"destination_options"`
}

func summarizeCustoms(records []model.CustomsDeclaration) customsSummary {
	result := customsSummary{Total: len(records), Amounts: []*customsAmounts{}, Months: []customsCount{}, Destinations: []customsCount{}}
	amounts := map[string]*customsAmounts{}
	months, destinations := map[string]int{}, map[string]int{}
	currencyGroup := func(code string) *customsAmounts {
		if amounts[code] == nil {
			amounts[code] = &customsAmounts{Currency: code}
		}
		return amounts[code]
	}
	for _, row := range records {
		review := len(row.Warnings) > 0
		result.ItemCount += len(row.Items)
		for _, pair := range []struct {
			key     string
			measure *customsMeasure
		}{{"packages", &result.Packages}, {"gross_weight", &result.GrossWeight}, {"net_weight", &result.NetWeight}} {
			if !pair.measure.add(row.Fields[pair.key]) {
				review = true
			}
		}
		month := "日期缺失"
		if date, err := time.Parse("2006-01-02", row.DeclarationDate); err == nil {
			month = date.Format("2006-01")
		}
		months[month]++
		destination := strings.TrimSpace(row.Fields["destination_port"])
		if destination == "" {
			destination = "未识别"
		}
		destinations[destination]++
		for _, item := range row.Items {
			currency := strings.ToUpper(strings.TrimSpace(item["currency"]))
			if !customsCurrency.MatchString(currency) {
				result.UnaggregatedAmounts++
				review = true
				continue
			}
			if !currencyGroup(currency).Goods.add(item["total_price"]) {
				result.UnaggregatedAmounts++
				review = true
			}
		}
		for _, key := range []string{"freight", "insurance"} {
			// Only explicit total-price charges may be added; rates/unit prices are not totals.
			parts := strings.Split(strings.ReplaceAll(row.Fields[key], "／", "/"), "/")
			if len(parts) != 3 || !customsCurrency.MatchString(strings.ToUpper(strings.TrimSpace(parts[0]))) || strings.TrimSpace(parts[2]) != "总价" {
				result.UnaggregatedAmounts++
				review = true
				continue
			}
			group := currencyGroup(strings.ToUpper(strings.TrimSpace(parts[0])))
			measure := &group.Freight
			if key == "insurance" {
				measure = &group.Insurance
			}
			if !measure.add(parts[1]) {
				result.UnaggregatedAmounts++
				review = true
			}
		}
		if len(row.Items) == 0 {
			review = true
		}
		if review {
			result.ReviewCount++
		}
	}
	for _, group := range amounts {
		result.Amounts = append(result.Amounts, group)
	}
	sort.Slice(result.Amounts, func(i, j int) bool { return result.Amounts[i].Currency < result.Amounts[j].Currency })
	for name, count := range months {
		result.Months = append(result.Months, customsCount{name, count})
	}
	sort.Slice(result.Months, func(i, j int) bool { return result.Months[i].Name < result.Months[j].Name })
	for name, count := range destinations {
		result.Destinations = append(result.Destinations, customsCount{name, count})
	}
	sort.Slice(result.Destinations, func(i, j int) bool {
		if result.Destinations[i].Count == result.Destinations[j].Count {
			return result.Destinations[i].Name < result.Destinations[j].Name
		}
		return result.Destinations[i].Count > result.Destinations[j].Count
	})
	return result
}

func SummarizeCustomsDeclarations(c *gin.Context) {
	query, ok := customsFilteredQuery(c)
	if !ok {
		return
	}
	var records []model.CustomsDeclaration
	if err := query.Select("declaration_date", "fields", "items", "warnings").Find(&records).Error; err != nil {
		c.JSON(500, gin.H{"error": "报关单看板加载失败"})
		return
	}
	result := summarizeCustoms(records)
	result.DestinationOptions = []string{}
	if err := database.DB.Model(&model.CustomsDeclaration{}).
		Distinct("BTRIM(fields->>'destination_port')").
		Where("BTRIM(COALESCE(fields->>'destination_port', '')) <> ''").
		Order("BTRIM(fields->>'destination_port')").
		Pluck("BTRIM(fields->>'destination_port')", &result.DestinationOptions).Error; err != nil {
		c.JSON(500, gin.H{"error": "指运港选项加载失败"})
		return
	}
	c.JSON(200, result)
}
