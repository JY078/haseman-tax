package handler

import (
	"reflect"
	"strings"
	"unicode"

	"github.com/shopspring/decimal"
	"golang.org/x/text/unicode/norm"
)

func wantsTaxTotals(value string) bool {
	return value == "1" || strings.EqualFold(value, "true")
}

func parseTotalAmount(value string) (decimal.Decimal, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return decimal.Zero, false
	}
	value = strings.ReplaceAll(value, "$", "")
	value = strings.ReplaceAll(value, "¥", "")
	value = strings.ReplaceAll(value, ",", "")
	value = strings.ReplaceAll(value, " ", "")
	if strings.HasPrefix(value, "(") && strings.HasSuffix(value, ")") {
		value = "-" + strings.TrimSuffix(strings.TrimPrefix(value, "("), ")")
	}
	number, err := decimal.NewFromString(value)
	return number, err == nil
}

func addTotal(target map[string]decimal.Decimal, key, value string) {
	if number, ok := parseTotalAmount(value); ok {
		target[key] = target[key].Add(number)
	}
}

func addJSONTotals(target map[string]decimal.Decimal, data map[string]string) {
	for key, value := range data {
		if excludedTaxTotalColumns[normalizeTaxTotalColumn(key)] {
			continue
		}
		addTotal(target, key, value)
	}
}

func normalizeTaxTotalColumn(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || r == '\uFEFF' {
			return -1
		}
		return unicode.ToLower(r)
	}, norm.NFKC.String(value))
}

var excludedTaxTotalColumns = func() map[string]bool {
	columns := []string{
		"Customer Order #", "Customer Order line #", "Purchase Order #", "Purchase Order line #",
		"Commission Rate", "Base Commission Rate", "Partner GTIN", "Product Tax Code", "Ship to Zipcode",
		"Order ID", "SKU ID", "Product ID", "Zipcode", "Weight(kg)", "Package ID",
		"亚马逊订单商品编号", "配送邮政编码", "追踪编码", "settlement id", "order postal",
		"客户单号", "客户订单号", "客户订单行号", "采购订单号", "采购订单行号", "佣金比例",
		"基础佣金比例", "商品条码", "商品税码", "收货邮编", "订单ID", "产品ID", "邮编", "重量(千克)", "包裹ID",
	}
	excluded := make(map[string]bool, len(columns))
	for _, column := range columns {
		excluded[normalizeTaxTotalColumn(column)] = true
	}
	return excluded
}()

func decimalStructTotals(rows interface{}) map[string]decimal.Decimal {
	totals := map[string]decimal.Decimal{}
	value := reflect.ValueOf(rows)
	if value.Kind() != reflect.Slice {
		return totals
	}
	decimalType := reflect.TypeOf(decimal.Decimal{})
	for i := 0; i < value.Len(); i++ {
		row := value.Index(i)
		if row.Kind() == reflect.Pointer {
			row = row.Elem()
		}
		if row.Kind() != reflect.Struct {
			continue
		}
		rowType := row.Type()
		for field := 0; field < row.NumField(); field++ {
			if rowType.Field(field).Type != decimalType {
				continue
			}
			name := strings.Split(rowType.Field(field).Tag.Get("json"), ",")[0]
			if name == "" || name == "-" {
				name = rowType.Field(field).Name
			}
			totals[name] = totals[name].Add(row.Field(field).Interface().(decimal.Decimal))
		}
	}
	return totals
}
