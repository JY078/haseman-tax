package handler

import (
	"strings"
	"testing"

	"advertising-image/backend/internal/model"
)

func TestValidatePurchasePlanInput(t *testing.T) {
	valid := purchasePlanInput{
		ProductSKUID: 1, PlannedQuantity: 120, UnitsPerCarton: 12, CartonCount: 10,
		ExpectedArrivalDate: "2026-09-01", Remark: "  首批备货  ",
	}
	normalized, err := validatePurchasePlanInput(valid)
	if err != nil {
		t.Fatalf("valid input returned error: %v", err)
	}
	if normalized.Remark != "首批备货" {
		t.Fatalf("remark = %q", normalized.Remark)
	}

	tests := []struct {
		name    string
		mutate  func(*purchasePlanInput)
		wantErr string
	}{
		{"product required", func(input *purchasePlanInput) { input.ProductSKUID = 0 }, "请选择产品"},
		{"planned quantity positive", func(input *purchasePlanInput) { input.PlannedQuantity = 0 }, "计划采购量"},
		{"units per carton positive", func(input *purchasePlanInput) { input.UnitsPerCarton = -1 }, "单箱数量"},
		{"carton count positive", func(input *purchasePlanInput) { input.CartonCount = 0 }, "箱数"},
		{"date valid", func(input *purchasePlanInput) { input.ExpectedArrivalDate = "2026-02-30" }, "期望到货时间"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := valid
			tt.mutate(&input)
			_, err := validatePurchasePlanInput(input)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestPurchasePlanProductName(t *testing.T) {
	sku := model.ProductSKU{SKU: "SKU-01", Name: "SKU 名称", StyleName: "款式名称"}
	if got := purchasePlanProductName(sku, model.ProductSPU{Name: "SPU 产品名"}); got != "SPU 产品名" {
		t.Fatalf("got %q", got)
	}
	if got := purchasePlanProductName(sku, model.ProductSPU{}); got != "SKU 名称" {
		t.Fatalf("got %q", got)
	}
}
