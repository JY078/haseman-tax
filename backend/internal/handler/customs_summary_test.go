package handler

import (
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCustomsSummaryFilters(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=127.0.0.1 user=test dbname=test"}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	oldDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = oldDB })
	for _, path := range []string{"/?date_from=2026-02-30", "/?date_from=2026-09-30&date_to=2026-09-01"} {
		response := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(response)
		c.Request = httptest.NewRequest("GET", path, nil)
		if _, ok := customsFilteredQuery(c); ok || response.Code != 400 {
			t.Fatalf("invalid date accepted: %s", path)
		}
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?keyword=FUS%25_&date_from=2026-09-01&date_to=2026-09-30", nil)
	query, ok := customsFilteredQuery(c)
	if !ok {
		t.Fatal("valid query rejected")
	}
	statement := query.Find(&[]model.CustomsDeclaration{}).Statement
	sql := statement.SQL.String()
	if !strings.Contains(sql, "(declaration_number ILIKE") || !strings.Contains(sql, "bill_number") || !strings.Contains(sql, "declaration_date >=") || !strings.Contains(sql, "declaration_date <=") {
		t.Fatalf("filters not grouped/applied: %s", sql)
	}
	if statement.Vars[0] != `%FUS\%\_%` || statement.Vars[5] != "2026-09-01" || statement.Vars[6] != "2026-09-30" {
		t.Fatalf("unexpected parameters: %+v", statement.Vars)
	}
	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?destination_port=Canada&date_from=2026-09-01", nil)
	query, ok = customsFilteredQuery(c)
	if !ok {
		t.Fatal("port filter rejected")
	}
	statement = query.Find(&[]model.CustomsDeclaration{}).Statement
	if !strings.Contains(statement.SQL.String(), "BTRIM(fields->>'destination_port') =") || len(statement.Vars) != 2 || statement.Vars[1] != "Canada" {
		t.Fatalf("port must be an exact parameterized filter: %s %+v", statement.SQL.String(), statement.Vars)
	}
}

func TestCustomsSummaryCurrenciesAndDeclarationTotals(t *testing.T) {
	records := []model.CustomsDeclaration{{
		DeclarationDate: "2026-09-18",
		Fields:          map[string]string{"packages": "205", "gross_weight": "3193.9", "net_weight": "2988.9", "freight": "USD/3908.95/总价", "insurance": "USD/101.89/总价", "destination_port": "加拿大"},
		Items:           []map[string]string{{"currency": "USD", "total_price": "37629.41"}, {"currency": "USD", "total_price": "2006.42"}, {"currency": "EUR", "total_price": "0.10"}, {"currency": "EUR", "total_price": "0.20"}},
	}}
	got := summarizeCustoms(records)
	if got.Total != 1 || got.ItemCount != 4 || got.ReviewCount != 0 || got.Packages.Value.String() != "205" || got.GrossWeight.Value.String() != "3193.9" || got.NetWeight.Value.String() != "2988.9" {
		t.Fatalf("declaration totals counted incorrectly: %+v", got)
	}
	if len(got.Amounts) != 2 || got.Amounts[0].Currency != "EUR" || got.Amounts[0].Goods.Value.String() != "0.3" || got.Amounts[0].Freight.Count != 0 {
		t.Fatalf("currency separation/precision: %+v", got.Amounts)
	}
	usd := got.Amounts[1]
	if usd.Currency != "USD" || usd.Goods.Value.String() != "39635.83" || usd.Freight.Value.String() != "3908.95" || usd.Insurance.Value.String() != "101.89" || usd.Freight.Count != 1 {
		t.Fatalf("amounts counted incorrectly: %+v", usd)
	}
	if len(got.Months) != 1 || got.Months[0].Name != "2026-09" || got.Destinations[0].Name != "加拿大" {
		t.Fatalf("dimensions: %+v", got)
	}
}

func TestCustomsSummaryRejectsMissingAmountsAndRates(t *testing.T) {
	got := summarizeCustoms([]model.CustomsDeclaration{{
		Fields: map[string]string{"packages": "0", "gross_weight": "1,234.50", "net_weight": "12x", "freight": "USD/10/单价", "insurance": "USD/0.3/费率"},
		Items:  []map[string]string{{"currency": "", "total_price": "20"}, {"currency": "USD", "total_price": "bad"}, {"currency": "USD", "total_price": "0"}},
	}})
	if got.ReviewCount != 1 || got.UnaggregatedAmounts != 4 || got.Packages.Count != 1 || got.Packages.Value.String() != "0" || got.GrossWeight.Value.String() != "1234.5" || got.NetWeight.Missing != 1 {
		t.Fatalf("missing/invalid values: %+v", got)
	}
	if len(got.Amounts) != 1 || got.Amounts[0].Goods.Count != 1 || got.Amounts[0].Goods.Missing != 1 || got.Amounts[0].Freight.Count != 0 {
		t.Fatalf("invalid amounts treated as totals: %+v", got.Amounts)
	}
}

func TestCustomsSummaryIncludesAllRecords(t *testing.T) {
	records := make([]model.CustomsDeclaration, 121)
	for i := range records {
		records[i].Fields = map[string]string{"packages": "1"}
		records[i].DeclarationDate = "2026-09-18"
	}
	got := summarizeCustoms(records)
	if got.Total != 121 || got.Packages.Value.String() != "121" || got.Months[0].Count != 121 {
		t.Fatalf("summary limited by pagination: %+v", got)
	}
	empty := summarizeCustoms(nil)
	if empty.Total != 0 || empty.Amounts == nil || empty.Months == nil || empty.Destinations == nil {
		t.Fatalf("empty response: %+v", empty)
	}
}
