package handler

import (
	"testing"

	"advertising-image/backend/internal/model"
)

func TestTrimAndApplyProductCatalogImportRow(t *testing.T) {
	row := productCatalogImportRow{SPU: " SPU-1 ", SKU: " SKU-1 ", ASIN: " B012345678 ", SKUName: " 名称 ", Brand: " 品牌 "}
	trimProductCatalogImportRow(&row)
	if row.SPU != "SPU-1" || row.SKU != "SKU-1" || row.ASIN != "B012345678" {
		t.Fatalf("required identifiers were not trimmed: %#v", row)
	}

	spu := model.ProductSPU{ID: 7, SPU: row.SPU}
	sku := model.ProductSKU{SKU: row.SKU}
	applyImportedSPUFields(&spu, row)
	applyImportedSKUFields(&sku, spu, row, 9)
	if spu.Brand != "品牌" || sku.SPUID != 7 || sku.ASIN != "B012345678" || sku.Name != "名称" || sku.UpdatedBy != 9 {
		t.Fatalf("imported fields were not applied: spu=%#v sku=%#v", spu, sku)
	}
}

func TestProductCatalogImageImportPreservesAndReplaces(t *testing.T) {
	sku := model.ProductSKU{ImageURL: "/uploads/original.png"}
	applyImportedSKUFields(&sku, model.ProductSPU{}, productCatalogImportRow{}, 1)
	if sku.ImageURL != "/uploads/original.png" {
		t.Fatal("blank import erased photo")
	}
	row := productCatalogImportRow{ImageURL: " https://example.com/new.png "}
	trimProductCatalogImportRow(&row)
	applyImportedSKUFields(&sku, model.ProductSPU{}, row, 1)
	if sku.ImageURL != "https://example.com/new.png" {
		t.Fatal("photo not updated")
	}
}
