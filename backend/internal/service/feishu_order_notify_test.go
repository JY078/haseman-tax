package service

import (
	"strings"
	"testing"

	"advertising-image/backend/internal/model"
)

func TestFormatOrderTrackingNotification(t *testing.T) {
	actual := formatOrderTrackingNotification([]model.OrderTrackingItem{
		{Platform: "TEMU", OrderID: "ORDER-1", TrackingNumber: "TBA-1"},
		{Platform: "TIKTOK", OrderID: "ORDER-2", TrackingNumber: "1Z-2"},
	})
	expected := "已发货订单2条：\n平台:TEMU，订单号：ORDER-1，追踪号：TBA-1，亚马逊订单编号：未填写\n平台:TIKTOK，订单号：ORDER-2，追踪号：1Z-2，亚马逊订单编号：未填写"
	if actual != expected {
		t.Fatalf("unexpected notification text: %q", actual)
	}
}

func TestFormatOrderTrackingNotificationIncludesAmazonOrderID(t *testing.T) {
	actual := formatOrderTrackingNotification([]model.OrderTrackingItem{{Platform: "TEMU", OrderID: "ORDER-1", AmazonOrderID: "AMZ-ORDER-1", TrackingNumber: "TBA-1"}})
	expected := "已发货订单1条：\n平台:TEMU，订单号：ORDER-1，追踪号：TBA-1，亚马逊订单编号：AMZ-ORDER-1"
	if actual != expected {
		t.Fatalf("unexpected notification text: %q", actual)
	}
}

func TestBuildTrackingNotificationEventsExpandsMergedOrder(t *testing.T) {
	events := buildTrackingNotificationEvents([]model.OrderTrackingItem{
		{
			Platform: "TEMU", OrderID: "HB-1234-5678", TrackingNumber: "TBA-1",
			IsMergedOrder: true, OriginalOrderIDs: []string{"ORDER-1234", "ORDER-5678"},
		},
	})
	if len(events) != 2 || events[0].OrderID != "ORDER-1234" || events[1].OrderID != "ORDER-5678" {
		t.Fatalf("merged order should expand into original orders: %#v", events)
	}
	if events[0].TrackingNumber != "TBA-1" || events[1].TrackingNumber != "TBA-1" {
		t.Fatalf("expanded orders should share the package tracking number: %#v", events)
	}
}

func TestBuildTrackingNotificationEventsListsMultiplePackages(t *testing.T) {
	events := buildTrackingNotificationEvents([]model.OrderTrackingItem{
		{Platform: "TEMU", OrderID: "HB-1234-5678", TrackingNumber: "TBA-1", OriginalOrderIDs: []string{"ORDER-1234", "ORDER-5678"}},
		{Platform: "TEMU", OrderID: "HB-1234-5678", TrackingNumber: "1Z-2", OriginalOrderIDs: []string{"ORDER-1234", "ORDER-5678"}},
	})
	if len(events) != 2 || events[0].Status != "tracking" || events[1].Status != "tracking" {
		t.Fatalf("multiple package tracking numbers should expand to original orders: %#v", events)
	}
	if events[0].TrackingNumber != "TBA-1、1Z-2" || events[1].TrackingNumber != "TBA-1、1Z-2" {
		t.Fatalf("unexpected combined tracking numbers: %#v", events)
	}
}

func TestFormatOrderTrackingNotificationExpandsMergedOrder(t *testing.T) {
	actual := formatOrderTrackingNotification([]model.OrderTrackingItem{
		{
			Platform: "TEMU", OrderID: "HB-1234-5678", TrackingNumber: "TBA-1",
			IsMergedOrder: true, OriginalOrderIDs: []string{"ORDER-1234", "ORDER-5678"},
		},
	})
	expected := "已发货订单2条：\n平台:TEMU，订单号：ORDER-1234，追踪号：TBA-1，亚马逊订单编号：未填写\n平台:TEMU，订单号：ORDER-5678，追踪号：TBA-1，亚马逊订单编号：未填写"
	if actual != expected {
		t.Fatalf("unexpected merged order notification: %q", actual)
	}
}

func TestBuildOrderStatusNotificationEventsExpandsMergedOrder(t *testing.T) {
	events := buildOrderStatusNotificationEvents([]model.OrderTrackingItem{
		{
			Platform: "TEMU", OrderID: "HB-1234-5678",
			OriginalOrderIDs: []string{"ORDER-1234", "ORDER-5678"},
		},
	}, "pending_shipment")
	if len(events) != 2 || events[0].OrderID != "ORDER-1234" || events[1].OrderID != "ORDER-5678" {
		t.Fatalf("merged status notification should expand into original orders: %#v", events)
	}
	if events[0].Status != "pending_shipment" || events[1].Status != "pending_shipment" {
		t.Fatalf("unexpected expanded status: %#v", events)
	}
}

func TestFormatOrderNotificationIncludesCancellation(t *testing.T) {
	actual := formatOrderNotification([]orderNotificationEvent{
		{Platform: "TEMU", OrderID: "ORDER-1", TrackingNumber: "TBA-1", Status: "tracking"},
		{Platform: "TIKTOK", OrderID: "ORDER-2", Status: "cancelled"},
	})
	expected := "已发货订单1条，取消订单1条：\n平台:TEMU，订单号：ORDER-1，追踪号：TBA-1，亚马逊订单编号：未填写\n平台: TIKTOK，订单号：ORDER-2，状态：已取消"
	if actual != expected {
		t.Fatalf("unexpected notification text: %q", actual)
	}
}

func TestFormatOrderNotificationIncludesPendingShipment(t *testing.T) {
	actual := formatOrderNotification([]orderNotificationEvent{
		{Platform: "TEMU", OrderID: "ORDER-1", Status: "pending_shipment"},
	})
	expected := "新增订单1条：\n平台：TEMU，订单号：ORDER-1，状态：待发货"
	if actual != expected {
		t.Fatalf("unexpected notification text: %q", actual)
	}
}

func TestFormatOrderNotificationIncludesCancellationRequest(t *testing.T) {
	actual := formatOrderNotification([]orderNotificationEvent{
		{Platform: "TIKTOK", OrderID: "ORDER-REQUESTED", Status: "cancel_requested"},
	})
	expected := "申请取消订单1条：\n平台：TIKTOK，订单号：ORDER-REQUESTED，状态：申请取消"
	if actual != expected {
		t.Fatalf("unexpected cancellation request notification text: %q", actual)
	}
}

func TestFormatOrderNotificationIncludesCancelFailureRemark(t *testing.T) {
	actual := formatOrderNotification([]orderNotificationEvent{
		{Platform: "LOWES", OrderID: "ORDER-TRACKED", TrackingNumber: "1Z-1", Status: "tracking", Reason: "取消订单失败，已存在追踪号"},
	})
	expected := "已发货订单1条：\n平台:LOWES，订单号：ORDER-TRACKED，追踪号：1Z-1，亚马逊订单编号：未填写，备注：取消订单失败，已存在追踪号"
	if actual != expected {
		t.Fatalf("unexpected cancel failure notification text: %q", actual)
	}
}

func TestFormatMCFExcludedNotification(t *testing.T) {
	actual := formatMCFExcludedNotification([]orderNotificationEvent{
		{Platform: "TEMU", OrderID: "ORDER-1", Reason: "该订单号ORDER-1状态不是待发货"},
		{Platform: "TIKTOK", OrderID: "ORDER-2", Reason: "该订单号ORDER-2的配送类型不是商家物流"},
	})
	expected := "<at user_id=\"ou_eafd844841851d1f8b3f7f4edffd308f\">负责人</at>\n订单转MCF中异常订单：\n平台：TEMU，订单号：ORDER-1，问题：该订单号ORDER-1状态不是待发货\n平台：TIKTOK，订单号：ORDER-2，问题：该订单号ORDER-2的配送类型不是商家物流"
	if actual != expected {
		t.Fatalf("unexpected MCF excluded notification text: %q", actual)
	}
}

func TestFormatMCFExcludedNotificationDoesNotMentionForAlreadyGeneratedOrders(t *testing.T) {
	actual := formatMCFExcludedNotification([]orderNotificationEvent{
		{Platform: "TEMU", OrderID: "ORDER-1", Reason: "该订单商品已生成过 MCF，本次已排除"},
	})
	expected := "订单转MCF中异常订单：\n平台：TEMU，订单号：ORDER-1，问题：该订单商品已生成过 MCF，本次已排除"
	if actual != expected {
		t.Fatalf("already generated MCF exclusions should not mention anyone: %q", actual)
	}
}

func TestFormatMCFExcludedNotificationMentionsForMixedReasons(t *testing.T) {
	actual := formatMCFExcludedNotification([]orderNotificationEvent{
		{Platform: "TEMU", OrderID: "ORDER-1", Reason: "该订单商品已生成过 MCF，本次已排除"},
		{Platform: "TEMU", OrderID: "ORDER-2", Reason: "该订单SKU未映射：SKU-2"},
	})
	if !strings.HasPrefix(actual, "<at user_id=\""+feishuMCFExcludedMentionID+"\">负责人</at>\n") {
		t.Fatalf("mixed excluded orders should mention the responsible user: %q", actual)
	}
}

func TestMCFExcludedNotificationKeyKeepsDifferentReasons(t *testing.T) {
	first := orderNotificationKey(orderNotificationEvent{Platform: "TEMU", OrderID: "ORDER-1", Status: "mcf_excluded", Reason: "状态异常"})
	second := orderNotificationKey(orderNotificationEvent{Platform: "TEMU", OrderID: "ORDER-1", Status: "mcf_excluded", Reason: "配送异常"})
	if first == second {
		t.Fatal("different exclusion reasons must not overwrite each other")
	}
}

func TestGroupOrderNotificationsByReceiveID(t *testing.T) {
	groups := groupOrderNotificationsByReceiveID([]orderNotificationEvent{
		{Platform: "LOWES", OrderID: "LOWES-1"},
		{Platform: "TEMU", OrderID: "TEMU-1"},
		{Platform: "TIKTOK", OrderID: "TIKTOK-1"},
	})

	lowesOrders := groups[feishuLowesOrderReceiveID]
	if len(lowesOrders) != 1 || lowesOrders[0].OrderID != "LOWES-1" {
		t.Fatalf("unexpected Lowe's notification group: %#v", lowesOrders)
	}
	temuOrders := groups[feishuTemuOrderReceiveID]
	if len(temuOrders) != 1 || temuOrders[0].OrderID != "TEMU-1" {
		t.Fatalf("unexpected TEMU notification group: %#v", temuOrders)
	}
	otherOrders := groups[feishuOtherOrderReceiveID]
	if len(otherOrders) != 1 || otherOrders[0].OrderID != "TIKTOK-1" {
		t.Fatalf("unexpected TikTok notification group: %#v", otherOrders)
	}
}

func TestFormatOrderNotificationForReceiveIDMentionsLowesOwner(t *testing.T) {
	text := "已发货订单1条：\n平台:LOWES，订单号：LOWES-1，追踪号：1Z-1"
	orders := []orderNotificationEvent{{Platform: "LOWES", OrderID: "LOWES-1", Status: "tracking"}}
	actual := formatOrderNotificationForReceiveID(feishuLowesOrderReceiveID, orders, text)
	expected := "<at user_id=\"ou_6da069358fe167c03ce61b525255750f\">负责人</at>\n" + text
	if actual != expected {
		t.Fatalf("unexpected Lowe's mention: %q", actual)
	}
	if other := formatOrderNotificationForReceiveID(feishuOtherOrderReceiveID, orders, text); other != text {
		t.Fatalf("other platform notification should not mention Lowe's owner: %q", other)
	}
}

func TestFormatOrderNotificationForReceiveIDSkipsNonActionableLowesStatuses(t *testing.T) {
	text := "新增订单1条，申请取消订单1条"
	orders := []orderNotificationEvent{
		{Platform: "LOWES", OrderID: "LOWES-1", Status: "pending_shipment"},
		{Platform: "LOWES", OrderID: "LOWES-2", Status: "cancel_requested"},
	}
	if actual := formatOrderNotificationForReceiveID(feishuLowesOrderReceiveID, orders, text); actual != text {
		t.Fatalf("non-actionable Lowe's statuses should not mention owner: %q", actual)
	}
}

func TestFeishuResponseNeedsTokenRefresh(t *testing.T) {
	if !feishuResponseNeedsTokenRefresh([]byte(`{"code":99991663}`)) {
		t.Fatal("expected invalid token response to refresh")
	}
	if feishuResponseNeedsTokenRefresh([]byte(`{"code":0}`)) {
		t.Fatal("successful response should not refresh")
	}
}
