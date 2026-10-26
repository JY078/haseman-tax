package handler

import (
	"reflect"
	"testing"

	"advertising-image/backend/internal/model"
)

func TestNormalizeOrderPlatform(t *testing.T) {
	tests := map[string]string{
		"TEMU":        "TEMU",
		" temu ":      "TEMU",
		"TikTok":      "TIKTOK",
		"TikTok Shop": "TIKTOK",
		"TK":          "TIKTOK",
		"Lowe's":      "LOWES",
		"lowes":       "LOWES",
		"Wayfair":     "",
	}
	for input, expected := range tests {
		if actual := normalizeOrderPlatform(input); actual != expected {
			t.Fatalf("normalizeOrderPlatform(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestCompactOrderSKUs(t *testing.T) {
	actual := compactOrderSKUs([]string{" A ", "", "B", "A", " B "})
	expected := []string{"A", "B"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("compactOrderSKUs() = %#v, want %#v", actual, expected)
	}
}

func TestCanEditOrderPlatform(t *testing.T) {
	admin := model.User{Roles: []model.Role{{Name: "admin"}}}
	if !canEditOrderPlatform(admin, "TEMU") {
		t.Fatal("admin should be able to edit all platforms")
	}
	temuUser := model.User{Platform: "TEMU"}
	if !canEditOrderPlatform(temuUser, "TEMU") {
		t.Fatal("TEMU user should be able to edit TEMU mappings")
	}
	if canEditOrderPlatform(temuUser, "TIKTOK") {
		t.Fatal("TEMU user should not be able to edit TikTok mappings")
	}
	multiPlatformUser := model.User{Platform: "TEMU", OrderPlatforms: []string{"TEMU", "TIKTOK"}}
	if !canEditOrderPlatform(multiPlatformUser, "TIKTOK") || canEditOrderPlatform(multiPlatformUser, "LOWES") {
		t.Fatal("multi-platform permissions should only include assigned platforms")
	}
	lowesUser := model.User{Platform: "Lowe's"}
	if !canEditOrderPlatform(lowesUser, "LOWES") {
		t.Fatal("Lowe's user should be able to edit Lowe's mappings")
	}
	if canEditOrderPlatform(lowesUser, "TEMU") {
		t.Fatal("Lowe's user should not be able to edit TEMU mappings")
	}
}

func TestDefaultLowesFieldMappingsUseStoreSKUAndShippingAddress(t *testing.T) {
	items := defaultOrderFieldMappings["LOWES"]
	fields := make(map[string]orderFieldDefinition, len(items))
	for _, item := range items {
		fields[item.FieldKey] = item
	}
	if !reflect.DeepEqual(fields["sku"].SourceHeaders, []string{"Store SKU", "Offer SKU"}) {
		t.Fatalf("unexpected Lowe's SKU mapping: %#v", fields["sku"].SourceHeaders)
	}
	if !reflect.DeepEqual(fields["phone"].SourceHeaders, []string{"Shipping address phone"}) {
		t.Fatalf("unexpected Lowe's phone mapping: %#v", fields["phone"].SourceHeaders)
	}
	if !fields["recipientFirstName"].Required {
		t.Fatal("Lowe's recipient first name should be required")
	}
}

func TestNormalizeImportedOrderSKUs(t *testing.T) {
	items, err := normalizeImportedOrderSKUs([]orderSKUMappingInput{
		{PlatformSKU: " A ", AmazonSKU: " AMZ-A ", NonAmazonSKU: " NON-A "},
		{PlatformSKU: "A", AmazonSKU: "AMZ-A", NonAmazonSKU: "NON-A"},
		{PlatformSKU: "B", AmazonSKU: "AMZ-B", NonAmazonSKU: "NON-B"},
	})
	if err != nil {
		t.Fatalf("normalizeImportedOrderSKUs returned error: %v", err)
	}
	if len(items) != 2 || items[0].PlatformSKU != "A" || items[1].AmazonSKU != "AMZ-B" || items[1].NonAmazonSKU != "NON-B" {
		t.Fatalf("unexpected normalized items: %#v", items)
	}
}

func TestNormalizeImportedOrderSKUsTrimsRemark(t *testing.T) {
	items, err := normalizeImportedOrderSKUs([]orderSKUMappingInput{{
		PlatformSKU: "A", AmazonSKU: "AMZ-A", NonAmazonSKU: "NON-A", Remark: "  主推款  ",
	}})
	if err != nil || len(items) != 1 || items[0].Remark != "主推款" {
		t.Fatalf("unexpected normalized remark: %#v, %v", items, err)
	}
}

func TestNormalizeImportedOrderSKUsAllowsMultipleAmazonSKUs(t *testing.T) {
	items, err := normalizeImportedOrderSKUs([]orderSKUMappingInput{
		{PlatformSKU: "A", AmazonSKU: "AMZ-A", NonAmazonSKU: "NON-A"},
		{PlatformSKU: "A", AmazonSKU: "AMZ-B", NonAmazonSKU: "NON-A"},
	})
	if err != nil || len(items) != 2 {
		t.Fatalf("expected multiple Amazon SKUs for one platform SKU: %#v, %v", items, err)
	}
}

func TestNormalizeImportedOrderSKUsRejectsMissingNonAmazonSKU(t *testing.T) {
	_, err := normalizeImportedOrderSKUs([]orderSKUMappingInput{
		{PlatformSKU: "A", AmazonSKU: "AMZ-A"},
	})
	if err == nil {
		t.Fatal("expected missing non-Amazon SKU to fail")
	}
}

func TestPartitionImportedOrderSKUsByAmazonLibraryKeepsValidRows(t *testing.T) {
	items := []orderSKUMappingInput{
		{PlatformSKU: "PLATFORM-A", AmazonSKU: "AMZ-A"},
		{PlatformSKU: "PLATFORM-MISSING-1", AmazonSKU: "AMZ-MISSING"},
		{PlatformSKU: "PLATFORM-B", AmazonSKU: "AMZ-B"},
		{PlatformSKU: "PLATFORM-MISSING-2", AmazonSKU: "AMZ-MISSING"},
	}
	valid, missing := partitionImportedOrderSKUsByAmazonLibrary(items, []string{"AMZ-B", "AMZ-A"})
	if len(valid) != 2 || valid[0].PlatformSKU != "PLATFORM-A" || valid[1].PlatformSKU != "PLATFORM-B" {
		t.Fatalf("unexpected valid items: %#v", valid)
	}
	if !reflect.DeepEqual(missing, []string{"AMZ-MISSING"}) {
		t.Fatalf("unexpected missing SKUs: %#v", missing)
	}
}

func TestNormalizeOrderSKUMappingImportMode(t *testing.T) {
	if actual := normalizeOrderSKUMappingImportMode(" create "); actual != "create" {
		t.Fatalf("unexpected create mode: %q", actual)
	}
	if actual := normalizeOrderSKUMappingImportMode("UPDATE"); actual != "update" {
		t.Fatalf("unexpected update mode: %q", actual)
	}
	if actual := normalizeOrderSKUMappingImportMode("upsert"); actual != "" {
		t.Fatalf("unsupported mode should be rejected: %q", actual)
	}
}

func TestPartitionImportedOrderSKUsByMode(t *testing.T) {
	items := []orderSKUMappingInput{
		{PlatformSKU: "EXISTING"},
		{PlatformSKU: "NEW"},
	}
	existing := map[string]struct{}{"EXISTING": {}}

	created, createSkipped := partitionImportedOrderSKUsByMode(items, existing, "create")
	if len(created) != 1 || created[0].PlatformSKU != "NEW" || !reflect.DeepEqual(createSkipped, []string{"EXISTING"}) {
		t.Fatalf("unexpected create partition: %#v, %#v", created, createSkipped)
	}

	updated, updateSkipped := partitionImportedOrderSKUsByMode(items, existing, "update")
	if len(updated) != 1 || updated[0].PlatformSKU != "EXISTING" || !reflect.DeepEqual(updateSkipped, []string{"NEW"}) {
		t.Fatalf("unexpected update partition: %#v, %#v", updated, updateSkipped)
	}
}

func TestDefaultTikTokPhoneMappingDoesNotUseShippingInformation(t *testing.T) {
	items := defaultOrderFieldMappings["TIKTOK"]
	for _, item := range items {
		if item.FieldKey != "phone" {
			continue
		}
		for _, header := range item.SourceHeaders {
			if header == "Shipping Information" {
				t.Fatal("TikTok phone mapping must not use Shipping Information")
			}
		}
		if len(item.SourceHeaders) == 0 || item.SourceHeaders[0] != "Phone #" {
			t.Fatalf("unexpected TikTok phone mapping: %#v", item.SourceHeaders)
		}
		return
	}
	t.Fatal("TikTok phone field definition not found")
}

func TestDefaultTikTokUsesSubStatusAndDeliveryOptionType(t *testing.T) {
	fields := map[string][]string{}
	for _, item := range defaultOrderFieldMappings["TIKTOK"] {
		fields[item.FieldKey] = item.SourceHeaders
	}
	if !reflect.DeepEqual(fields["status"], []string{"Order SubStatus"}) {
		t.Fatalf("unexpected TikTok status mapping: %#v", fields["status"])
	}
	if !reflect.DeepEqual(fields["fulfillmentType"], []string{"Delivery Option Type"}) {
		t.Fatalf("unexpected TikTok delivery mapping: %#v", fields["fulfillmentType"])
	}
}

func TestLegacyTikTokDefaultMappingsUseNewDefaults(t *testing.T) {
	if !isLegacyTikTokDefaultMapping("TIKTOK", "status", []string{"Order Status"}) {
		t.Fatal("legacy TikTok status default should be recognized")
	}
	if !isLegacyTikTokDefaultMapping("TIKTOK", "fulfillmentType", []string{"Fulfillment Type"}) {
		t.Fatal("legacy TikTok fulfillment default should be recognized")
	}
	if isLegacyTikTokDefaultMapping("TIKTOK", "status", []string{"Custom Status"}) {
		t.Fatal("custom TikTok status mapping must be preserved")
	}
}

func TestValidateOrderFieldMappingsRequiresCompleteRuleSet(t *testing.T) {
	defaults := defaultOrderFieldMappings["TEMU"]
	items := make([]orderFieldDefinition, len(defaults)-1)
	copy(items, defaults[:len(defaults)-1])
	if _, err := validateOrderFieldMappings("TEMU", items); err == nil {
		t.Fatal("expected incomplete field mappings to fail")
	}
}

func TestValidateOrderFieldMappingsCompactsAliases(t *testing.T) {
	defaults := defaultOrderFieldMappings["TIKTOK"]
	items := make([]orderFieldDefinition, len(defaults))
	copy(items, defaults)
	for index := range items {
		if items[index].FieldKey == "phone" {
			items[index].SourceHeaders = []string{" Phone # ", "PHONE", "PHONE"}
		}
	}
	validated, err := validateOrderFieldMappings("TIKTOK", items)
	if err != nil {
		t.Fatalf("validateOrderFieldMappings returned error: %v", err)
	}
	for _, item := range validated {
		if item.FieldKey == "phone" && len(item.SourceHeaders) != 2 {
			t.Fatalf("phone aliases were not compacted: %#v", item.SourceHeaders)
		}
	}
}
