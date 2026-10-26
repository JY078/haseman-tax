package handler

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestJSONTotalsExcludeIdentifiersRatesAndWeight(t *testing.T) {
	data := map[string]string{
		"Customer Order #": "10001", "Customer Order line #": "1", "Purchase Order #": "20001", "Purchase Order line #": "2",
		"Commission Rate": "0.15", "Base Commission Rate": "0.15", "Partner GTIN": "123456789", "Product Tax Code": "100", "Ship to Zipcode": "90210",
		"Order ID": "10001", "SKU ID": "20001", "Product ID": "30001", "Zipcode": "90210", "Weight (kg)": "1.5", "Package ID": "40001",
		"亚马逊订单商品编号": "166052824406201", "配送邮政编码": "90210", "追踪编码": "123456", "Settlement ID": "26841843221", "Order Postal": "90210",
		"\uFEFFsettlement\u00a0id ": "1234", "重量（千克）": "1.5", "ＳＫＵ ＩＤ": "1234",
		"quantity": "3", "已发货数量": "4", "Amount": "10.50", "fba fees": "-6.25", "total": "4.25",
	}
	totals := map[string]decimal.Decimal{}
	addJSONTotals(totals, data)
	addJSONTotals(totals, data)
	want := map[string]string{"quantity": "6", "已发货数量": "8", "Amount": "21", "fba fees": "-12.5", "total": "8.5"}
	if len(totals) != len(want) {
		t.Fatalf("non-additive columns leaked into totals: %+v", totals)
	}
	for column, value := range want {
		if totals[column].String() != value {
			t.Fatalf("%s total = %s, want %s", column, totals[column], value)
		}
	}
}
