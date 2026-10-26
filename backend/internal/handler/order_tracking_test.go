package handler

import (
	"testing"
	"time"

	"advertising-image/backend/internal/model"
)

func TestNormalizeOrderTrackingItemsMergesDuplicateOrderSKU(t *testing.T) {
	items, err := normalizeOrderTrackingItems("TIKTOK", 7, []orderTrackingItemInput{
		{OrderID: " ORDER-1 ", PlatformSKU: " SKU-1 ", Quantity: 1, Recipient: "Alex", Address: "100 Main St"},
		{OrderID: "ORDER-1", PlatformSKU: "SKU-1", Quantity: 2, Recipient: "Alex", Address: "100 Main St"},
	})
	if err != nil {
		t.Fatalf("normalizeOrderTrackingItems returned error: %v", err)
	}
	if len(items) != 1 || items[0].Quantity != 3 || items[0].ImportedBy != 7 {
		t.Fatalf("unexpected normalized items: %#v", items)
	}
}

func TestNormalizeOrderTrackingItemsRejectsNullPackageKey(t *testing.T) {
	_, err := normalizeOrderTrackingItems("TEMU", 7, []orderTrackingItemInput{{
		OrderID: "ORDER-1", PlatformSKU: "SKU-1", PackageKey: "ORDER-1\x00SKU-1::part-1", Quantity: 2,
	}})
	if err == nil {
		t.Fatal("package key containing a null character should be rejected before database insertion")
	}
}

func TestNormalizeOrderTrackingItemsPreservesMergedOrderMetadata(t *testing.T) {
	items, err := normalizeOrderTrackingItems("TEMU", 7, []orderTrackingItemInput{{
		OrderID: "HB-1234-5678", PlatformSKU: "SKU-1", Quantity: 1,
		OriginalOrderIDs: []string{"ORDER-1234", "ORDER-5678"}, IsMergedOrder: true,
	}})
	if err != nil || len(items) != 1 {
		t.Fatalf("unexpected normalized items: %#v, %v", items, err)
	}
	if !items[0].IsMergedOrder || len(items[0].OriginalOrderIDs) != 2 || items[0].OriginalOrderIDs[1] != "ORDER-5678" {
		t.Fatalf("merged metadata was not preserved: %#v", items[0])
	}
}

func TestFindUnmappedOrderTrackingSKUs(t *testing.T) {
	missing := findUnmappedOrderTrackingSKUs(
		[]string{"MAPPED-A", "UNMAPPED", "MAPPED-B"},
		[]string{"MAPPED-B", "MAPPED-A"},
	)
	if len(missing) != 1 || missing[0] != "UNMAPPED" {
		t.Fatalf("unexpected missing SKUs: %#v", missing)
	}
}

func TestCompactOrderTrackingCheckValues(t *testing.T) {
	orderIDs, skus := compactOrderTrackingCheckValues([]orderTrackingItemInput{
		{OrderID: " ORDER-1 ", PlatformSKU: " SKU-1 "},
		{OrderID: "ORDER-1", PlatformSKU: "SKU-2"},
		{OrderID: "", PlatformSKU: ""},
	})
	if len(orderIDs) != 1 || orderIDs[0] != "ORDER-1" {
		t.Fatalf("unexpected order IDs: %#v", orderIDs)
	}
	if len(skus) != 2 || skus[0] != "SKU-1" || skus[1] != "SKU-2" {
		t.Fatalf("unexpected platform SKUs: %#v", skus)
	}
}

func TestAccessibleOrderTrackingPlatform(t *testing.T) {
	admin := model.User{Roles: []model.Role{{Name: "admin"}}}
	if platform, ok := accessibleOrderTrackingPlatform(admin, ""); !ok || platform != "" {
		t.Fatalf("admin should access all platforms, got %q, %v", platform, ok)
	}
	temuUser := model.User{Platform: "TEMU"}
	if platform, ok := accessibleOrderTrackingPlatform(temuUser, ""); !ok || platform != "TEMU" {
		t.Fatalf("TEMU user should default to TEMU, got %q, %v", platform, ok)
	}
	if _, ok := accessibleOrderTrackingPlatform(temuUser, "TIKTOK"); ok {
		t.Fatal("TEMU user should not access TikTok tracking orders")
	}
	multiPlatformUser := model.User{Platform: "TEMU", OrderPlatforms: []string{"TEMU", "TIKTOK"}}
	if platform, ok := accessibleOrderTrackingPlatform(multiPlatformUser, ""); !ok || platform != "" {
		t.Fatalf("multi-platform user should default to all assigned platforms, got %q, %v", platform, ok)
	}
	if platform, ok := accessibleOrderTrackingPlatform(multiPlatformUser, "TIKTOK"); !ok || platform != "TIKTOK" {
		t.Fatalf("multi-platform user should access TikTok, got %q, %v", platform, ok)
	}
	if _, ok := accessibleOrderTrackingPlatform(multiPlatformUser, "LOWES"); ok {
		t.Fatal("multi-platform user should not access unassigned Lowe's orders")
	}
	if !canRequestOrderCancellation(multiPlatformUser, "TIKTOK") || canRequestOrderCancellation(multiPlatformUser, "LOWES") {
		t.Fatal("cancel request permission should follow assigned platforms")
	}
	dataUser := model.User{Platform: "data"}
	if platform, ok := accessibleOrderTrackingPlatform(dataUser, ""); !ok || platform != "" {
		t.Fatalf("data user should access all platforms, got %q, %v", platform, ok)
	}
	if platform, ok := accessibleOrderTrackingPlatform(dataUser, "TIKTOK"); !ok || platform != "TIKTOK" {
		t.Fatalf("data user should filter TikTok, got %q, %v", platform, ok)
	}
	if canEditOrderPlatform(dataUser, "TIKTOK") {
		t.Fatal("data user should not receive general TikTok platform maintenance permission")
	}
	if !canEditOrderTracking(dataUser, "TIKTOK") {
		t.Fatal("data user should edit TikTok tracking data")
	}
	lowesUser := model.User{Platform: "lowes"}
	if platform, ok := accessibleOrderTrackingPlatform(lowesUser, ""); !ok || platform != "LOWES" {
		t.Fatalf("Lowe's user should default to LOWES, got %q, %v", platform, ok)
	}
	if platform, ok := accessibleOrderTrackingPlatform(dataUser, "Lowe's"); !ok || platform != "LOWES" {
		t.Fatalf("data user should filter Lowe's, got %q, %v", platform, ok)
	}
}

func TestInferOrderCarrier(t *testing.T) {
	tests := map[string]string{
		"aftn123456789":          "Amazon MCF (TK)",
		"TBA123456789":           "Amazon Logistics",
		"GFUS123456789":          "GOFO",
		" gfus123456789 ":        "GOFO",
		"XGFUS123456789":         "",
		"1Z999AA10123456784":     "UPS",
		"9400111899223856928499": "USPS",
		"123456789012":           "FedEx",
		"1234567890":             "DHL Express",
		"unknown":                "",
	}
	for trackingNumber, expected := range tests {
		if actual := inferOrderCarrier(trackingNumber); actual != expected {
			t.Fatalf("inferOrderCarrier(%q) = %q, want %q", trackingNumber, actual, expected)
		}
	}
}

func TestNormalizeTrackingNumberKey(t *testing.T) {
	if actual := normalizeTrackingNumberKey(" 1z 999 aa "); actual != "1Z999AA" {
		t.Fatalf("unexpected tracking number key: %q", actual)
	}
}

func TestOrderTrackingStatusForNumber(t *testing.T) {
	if actual := orderTrackingStatusForNumber(" "); actual != orderStatusPendingShipment {
		t.Fatalf("empty tracking number status = %q", actual)
	}
	if actual := orderTrackingStatusForNumber("1Z999"); actual != orderStatusShipped {
		t.Fatalf("filled tracking number status = %q", actual)
	}
}

func TestNextOrderTrackingStatusPermissions(t *testing.T) {
	operator := model.User{Platform: "TEMU"}
	dataUser := model.User{Department: "数据部"}
	admin := model.User{Roles: []model.Role{{Name: "admin"}}}
	platformAdmin := model.User{Platform: "TEMU", Roles: []model.Role{{Name: "admin"}}}
	item := model.OrderTrackingItem{Platform: "TEMU", Status: orderStatusPendingShipment}

	if status, ok := nextOrderTrackingStatus(operator, item, "request_cancel"); !ok || status != orderStatusCancelRequested {
		t.Fatalf("platform operator should request cancellation, got %q, %v", status, ok)
	}
	if _, ok := nextOrderTrackingStatus(model.User{Platform: "TIKTOK"}, item, "request_cancel"); ok {
		t.Fatal("another platform operator should not request cancellation")
	}
	if status, ok := nextOrderTrackingStatus(platformAdmin, item, "request_cancel"); !ok || status != orderStatusCancelRequested {
		t.Fatalf("platform-bound admin should request cancellation, got %q, %v", status, ok)
	}
	item.Status = orderStatusShipped
	if _, ok := nextOrderTrackingStatus(operator, item, "request_cancel"); ok {
		t.Fatal("shipped order should not allow cancellation request")
	}
	item.Status = orderStatusCancelRequested
	if status, ok := nextOrderTrackingStatus(dataUser, item, "cancel"); !ok || status != orderStatusCancelled {
		t.Fatalf("data user should cancel a requested order, got %q, %v", status, ok)
	}
	if status, ok := nextOrderTrackingStatus(admin, item, "cancel"); !ok || status != orderStatusCancelled {
		t.Fatalf("admin should cancel a requested order, got %q, %v", status, ok)
	}
	if _, ok := nextOrderTrackingStatus(operator, item, "cancel"); ok {
		t.Fatal("platform operator should not confirm cancellation")
	}
	if status, ok := nextOrderTrackingStatus(operator, item, "revoke_cancel"); !ok || status != orderStatusPendingShipment {
		t.Fatalf("platform operator should revoke cancellation request, got %q, %v", status, ok)
	}
	if status, ok := nextOrderTrackingStatus(dataUser, item, "revoke_cancel"); !ok || status != orderStatusPendingShipment {
		t.Fatalf("data user should revoke cancellation request, got %q, %v", status, ok)
	}
	lowesItem := model.OrderTrackingItem{Platform: "LOWES", Status: orderStatusCancelRequested}
	if _, ok := nextOrderTrackingStatus(dataUser, lowesItem, "revoke_cancel"); ok {
		t.Fatal("Lowe's orders should not support revoking cancellation requests")
	}
}

func TestParseOrderTrackingDateRangeIncludesEndDate(t *testing.T) {
	from, to, err := parseOrderTrackingDateRange("2026-08-01", "2026-08-03")
	if err != nil {
		t.Fatalf("parseOrderTrackingDateRange returned error: %v", err)
	}
	if from == nil || from.Format("2006-01-02 15:04") != "2026-08-01 00:00" {
		t.Fatalf("unexpected start date: %v", from)
	}
	if to == nil || to.Format("2006-01-02 15:04") != "2026-08-04 00:00" {
		t.Fatalf("unexpected exclusive end date: %v", to)
	}
	if _, _, err := parseOrderTrackingDateRange("2026-08-04", "2026-08-03"); err == nil {
		t.Fatal("expected reversed date range to fail")
	}
}

func TestBuildOrderTrackingExportItemsDeduplicatesOrders(t *testing.T) {
	items := []model.OrderTrackingItem{
		{ID: 1, Platform: "TIKTOK", OrderID: "ORDER-1", TrackingNumber: "1Z999AA10123456784"},
		{ID: 2, Platform: "TIKTOK", OrderID: "ORDER-1", TrackingNumber: "1Z999AA10123456784"},
		{ID: 3, Platform: "TEMU", OrderID: "ORDER-1", TrackingNumber: "TBA123456789", ExportedAt: &time.Time{}},
	}
	exported := buildOrderTrackingExportItems(items)
	if len(exported) != 2 {
		t.Fatalf("expected 2 exported orders, got %#v", exported)
	}
	if exported[0].ID != 1 || exported[0].Carrier != "UPS" || exported[0].Exported || exported[1].ID != 3 || !exported[1].Exported {
		t.Fatalf("unexpected exported orders: %#v", exported)
	}
}

func TestOrderTrackingExportIncludesMetadataAndAllPackageProducts(t *testing.T) {
	created := time.Date(2026, 9, 15, 8, 30, 0, 0, time.UTC)
	first := model.OrderTrackingItem{ID: 1, Platform: "TEMU", OrderID: "ORDER-1", PackageKey: "part-1",
		AmazonOrderID: "AMZ-ORDER", Status: orderStatusShipped, AmazonSKU: "SKU-A", Quantity: 2,
		Recipient: "Alex", Phone: "00123456789", Address: "100 Main St", TrackingNumber: "TBA123",
		Remark: "包裹备注", ImportedBy: 7, ImportedByName: "张三", CreatedAt: created}
	second := first
	second.ID, second.AmazonSKU, second.Quantity = 2, "SKU-B", 3
	third := first
	third.ID, third.PackageKey, third.TrackingNumber, third.Quantity = 3, "part-2", "TBA456", 1
	rows := buildOrderTrackingExportRows([]model.OrderTrackingItem{first, second, third})
	if len(rows) != 2 || len(rows[0].Details) != 2 || rows[0].Details[1].AmazonSKU != "SKU-B" || rows[0].Details[1].Quantity != 3 || rows[1].TrackingNumber != "TBA456" {
		t.Fatalf("lost package products or tracking: %#v", rows)
	}
	r := rows[0]
	if r.AmazonOrderID != first.AmazonOrderID || r.Status != first.Status || r.Recipient != first.Recipient || r.Phone != first.Phone || r.Address != first.Address || r.Remark != first.Remark || r.ImportedByName != first.ImportedByName || !r.CreatedAt.Equal(created) {
		t.Fatalf("lost export metadata: %#v", r)
	}
}

func TestMergedExportDetailsBelongToOriginalOrder(t *testing.T) {
	items := []model.OrderTrackingItem{{ID: 1, Platform: "TEMU", OrderID: "HB-1-2", IsMergedOrder: true,
		OriginalOrderIDs: []string{"ORDER-1", "ORDER-2"},
		OriginalItems: []model.OrderTrackingSource{
			{OriginalOrderID: "ORDER-1", AmazonSKU: "SKU-A", Quantity: 2},
			{OriginalOrderID: "ORDER-2", AmazonSKU: "SKU-B", Quantity: 3},
		}}}
	rows := buildOrderTrackingExportRows(items)
	if len(rows) != 2 || len(rows[0].Details) != 1 || len(rows[1].Details) != 1 || rows[0].Details[0].AmazonSKU != "SKU-A" || rows[1].Details[0].Quantity != 3 {
		t.Fatalf("original order details were mixed: %#v", rows)
	}
}

func TestBuildOrderTrackingExportItemsKeepsSplitPackages(t *testing.T) {
	items := []model.OrderTrackingItem{
		{ID: 1, Platform: "TEMU", OrderID: "ORDER-SPLIT", PackageKey: "part-1", TrackingNumber: "T1"},
		{ID: 2, Platform: "TEMU", OrderID: "ORDER-SPLIT", PackageKey: "part-2", TrackingNumber: "T2"},
	}
	exported := buildOrderTrackingExportItems(items)
	if len(exported) != 2 || exported[0].ID != 1 || exported[1].ID != 2 {
		t.Fatalf("split packages should export separately: %#v", exported)
	}
}

func TestBuildOrderTrackingExportRowsExpandsMergedOriginalOrders(t *testing.T) {
	items := []model.OrderTrackingItem{
		{ID: 1, Platform: "TEMU", OrderID: "HB-1234-5678", PlatformSKU: "SKU-A", TrackingNumber: "TBA123", IsMergedOrder: true, OriginalOrderIDs: []string{"ORDER-1234", "ORDER-5678"}},
		{ID: 2, Platform: "TEMU", OrderID: "HB-1234-5678", PlatformSKU: "SKU-B", TrackingNumber: "TBA123", IsMergedOrder: true, OriginalOrderIDs: []string{"ORDER-1234", "ORDER-5678"}},
	}
	rows := buildOrderTrackingExportRows(items)
	if len(rows) != 2 || rows[0].OrderID != "ORDER-1234" || rows[1].OrderID != "ORDER-5678" {
		t.Fatalf("unexpected merged export rows: %#v", rows)
	}
}

func TestCollapseOrderTrackingItemsKeepsAllDetails(t *testing.T) {
	items := []model.OrderTrackingItem{
		{ID: 1, Platform: "TEMU", OrderID: "HB-1234-5678", PlatformSKU: "SKU-A", AmazonSKU: "AMZ-A", Quantity: 1, OriginalItems: []model.OrderTrackingSource{{OriginalOrderID: "ORDER-1234", PlatformSKU: "SKU-A", AmazonSKU: "AMZ-A", Quantity: 1}}},
		{ID: 2, Platform: "TEMU", OrderID: "HB-1234-5678", PlatformSKU: "SKU-B", AmazonSKU: "AMZ-B", Quantity: 3, OriginalItems: []model.OrderTrackingSource{{OriginalOrderID: "ORDER-5678", PlatformSKU: "SKU-B", AmazonSKU: "AMZ-B", Quantity: 3}}},
	}
	orders := collapseOrderTrackingItems(items)
	if len(orders) != 1 || len(orders[0].Details) != 2 || orders[0].Quantity != 4 {
		t.Fatalf("unexpected collapsed order: %#v", orders)
	}
	if orders[0].Details[0].OriginalOrderID != "ORDER-1234" || orders[0].Details[1].OriginalOrderID != "ORDER-5678" || orders[0].Details[1].AmazonSKU != "AMZ-B" || orders[0].Details[1].Quantity != 3 {
		t.Fatalf("order details were not preserved: %#v", orders[0].Details)
	}
}

func TestCollapseOrderTrackingItemsKeepsSplitPackages(t *testing.T) {
	items := []model.OrderTrackingItem{
		{ID: 1, Platform: "TEMU", OrderID: "ORDER-SPLIT", PlatformSKU: "SKU-A", PackageKey: "part-1", Quantity: 2, TrackingNumber: "T1", Status: orderStatusShipped, CanEditTracking: true},
		{ID: 2, Platform: "TEMU", OrderID: "ORDER-SPLIT", PlatformSKU: "SKU-A", PackageKey: "part-2", Quantity: 1, TrackingNumber: "T2", Status: orderStatusShipped, CanEditTracking: true},
	}
	orders := collapseOrderTrackingItems(items)
	if len(orders) != 1 || len(orders[0].Packages) != 2 {
		t.Fatalf("split packages should remain visible under one order: %#v", orders)
	}
	if orders[0].TrackingNumber != "T1、T2" || orders[0].Status != orderStatusShipped {
		t.Fatalf("unexpected split package summary: %#v", orders[0])
	}
	if orders[0].Packages[0].ID != 1 || orders[0].Packages[0].Quantity != 2 || orders[0].Packages[1].TrackingNumber != "T2" || orders[0].Packages[1].Quantity != 1 {
		t.Fatalf("package identity or tracking number was lost: %#v", orders[0].Packages)
	}
}

func TestCollapseOrderTrackingItemsShowsCancellationRequest(t *testing.T) {
	items := []model.OrderTrackingItem{
		{ID: 1, Platform: "TEMU", OrderID: "ORDER-CANCEL", PlatformSKU: "SKU-A", Quantity: 1, Status: orderStatusCancelRequested},
		{ID: 2, Platform: "TEMU", OrderID: "ORDER-CANCEL", PlatformSKU: "SKU-B", Quantity: 2, Status: orderStatusCancelRequested},
	}
	orders := collapseOrderTrackingItems(items)
	if len(orders) != 1 || orders[0].Status != orderStatusCancelRequested {
		t.Fatalf("cancellation request should remain visible on collapsed order: %#v", orders)
	}
}
