package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/utils"
)

const (
	feishuOrderNotifyURLValue    = "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id"
	feishuAppID                  = "cli_aaf9b07532b9dcb5"
	feishuAppSecret              = "tnTGrTfvvqljJtUFQhS2LbMMW4NyqW61"
	feishuTenantTokenURL         = "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"
	feishuLowesOrderReceiveID    = "oc_53645b8ef64e091144f1836d8d42eb11"
	feishuLowesMentionOpenID     = "ou_6da069358fe167c03ce61b525255750f"
	feishuOtherOrderReceiveID    = "oc_16023167cf215c6d8d6f9dd17460c710"
	feishuTemuOrderReceiveID     = "oc_9f3cff1cc3175010c30e1e5f74d61cc6"
	feishuMCFExcludedMentionID   = "ou_eafd844841851d1f8b3f7f4edffd308f"
	feishuOrderNotifyQuietPeriod = 2 * time.Minute
)

func feishuCredentials() (string, string) {
	return feishuAppID, feishuAppSecret
}

var feishuTenantTokenCache struct {
	sync.Mutex
	token     string
	expiresAt time.Time
}

type orderNotifyBatch struct {
	mu    sync.Mutex
	keys  map[string]orderNotificationEvent
	timer *time.Timer
}

type orderNotificationEvent struct {
	Platform       string
	OrderID        string
	AmazonOrderID  string
	TrackingNumber string
	Status         string
	Reason         string
}

var pendingOrderNotifications = orderNotifyBatch{keys: make(map[string]orderNotificationEvent)}

// QueueOrderTrackingNotification starts or resets the quiet-period timer.
func QueueOrderTrackingNotification(platform, orderID string) {
	queueOrderNotification(orderNotificationEvent{Platform: platform, OrderID: orderID, Status: "tracking"})
}

// QueueOrderCancelFailedTrackingNotification reports that a Lowe's cancellation
// request could not proceed because tracking information was filled in.
func QueueOrderCancelFailedTrackingNotification(platform, orderID string) {
	queueOrderNotification(orderNotificationEvent{Platform: platform, OrderID: orderID, Status: "cancel_failed_tracking"})
}

// QueueOrderCancelledNotification adds a confirmed cancellation to the same quiet-period batch.
func QueueOrderCancelledNotification(platform, orderID string) {
	queueOrderNotification(orderNotificationEvent{Platform: platform, OrderID: orderID, Status: "cancelled"})
}

// QueueOrderCancelRequestedNotification adds a cancellation request to the same quiet-period batch.
func QueueOrderCancelRequestedNotification(platform, orderID string) {
	queueOrderNotification(orderNotificationEvent{Platform: platform, OrderID: orderID, Status: "cancel_requested"})
	go sendAmazonCancelRequestEmail(platform, orderID)
}

func sendAmazonCancelRequestEmail(platform, orderID string) {
	body := fmt.Sprintf("<p>平台：%s</p><p>订单号：%s</p><p>状态：申请取消</p>", html.EscapeString(platform), html.EscapeString(orderID))
	if err := utils.SendEmailWithAttachment("2818637484@qq.com", "亚马逊取消订单", body, "取消订单号.txt", "text/plain; charset=UTF-8", []byte(orderID+"\r\n")); err != nil {
		log.Printf("cancel request email send error: %v", err)
	}
}

// SendLowesMCFEmail sends the generated Lowe's MCF template through the same
// SMTP path and recipient used by cancellation request emails.
func SendLowesMCFEmail(fileName string, attachment []byte) error {
	body := fmt.Sprintf("<p>Lowes-MCF上传模板</p><p>平台：LOWES</p><p>附件：%s</p>", html.EscapeString(fileName))
	// MCF bytes remain UTF-16LE, but the MIME part must not declare a charset:
	// QQ Mail may apply that charset to the filename parameter and display it
	// as garbled text. The recipient can still open the .txt attachment normally.
	return utils.SendEmailWithAttachment("2818637484@qq.com", "Lowes-MCF上传模板", body, fileName, "application/octet-stream", attachment)
}

// QueueOrderPendingShipmentNotification adds newly archived MCF orders to the quiet-period batch.
func QueueOrderPendingShipmentNotification(platform, orderID string) {
	queueOrderNotification(orderNotificationEvent{Platform: platform, OrderID: orderID, Status: "pending_shipment"})
}

// QueueOrderMCFExcludedNotification adds an excluded MCF order to the notification batch.
func QueueOrderMCFExcludedNotification(platform, orderID, reason string) {
	queueOrderNotification(orderNotificationEvent{Platform: platform, OrderID: orderID, Status: "mcf_excluded", Reason: reason})
}

func queueOrderNotification(event orderNotificationEvent) {
	event.Platform = strings.TrimSpace(event.Platform)
	event.OrderID = strings.TrimSpace(event.OrderID)
	event.Reason = strings.TrimSpace(event.Reason)
	if event.Platform == "" || event.OrderID == "" {
		return
	}
	if event.Status == "mcf_excluded" && event.Reason == "" {
		return
	}

	pendingOrderNotifications.mu.Lock()
	pendingOrderNotifications.keys[orderNotificationKey(event)] = event
	delay := feishuOrderNotifyQuietPeriod
	if pendingOrderNotifications.timer != nil {
		pendingOrderNotifications.timer.Stop()
	}
	pendingOrderNotifications.timer = time.AfterFunc(delay, flushOrderTrackingNotifications)
	pendingOrderNotifications.mu.Unlock()
}

func orderNotificationKey(event orderNotificationEvent) string {
	key := event.Status + "\x00" + event.Platform + "\x00" + event.OrderID
	if event.Status == "mcf_excluded" {
		key += "\x00" + event.Reason
	}
	return key
}

func flushOrderTrackingNotifications() {
	pendingOrderNotifications.mu.Lock()
	events := make([]orderNotificationEvent, 0, len(pendingOrderNotifications.keys))
	for _, event := range pendingOrderNotifications.keys {
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].Platform != events[j].Platform {
			return events[i].Platform < events[j].Platform
		}
		if events[i].OrderID != events[j].OrderID {
			return events[i].OrderID < events[j].OrderID
		}
		return events[i].Status < events[j].Status
	})
	pendingOrderNotifications.keys = make(map[string]orderNotificationEvent)
	pendingOrderNotifications.timer = nil
	pendingOrderNotifications.mu.Unlock()
	if len(events) == 0 {
		return
	}

	go sendOrderTrackingNotification(events)
}

func sendOrderTrackingNotification(events []orderNotificationEvent) {
	orders := make([]orderNotificationEvent, 0, len(events))
	excludedOrders := make([]orderNotificationEvent, 0)
	for _, event := range events {
		if event.Status == "mcf_excluded" {
			excludedOrders = append(excludedOrders, event)
			continue
		}
		if event.Status == "cancelled" || event.Status == "cancel_requested" || event.Status == "pending_shipment" {
			var items []model.OrderTrackingItem
			if err := database.DB.Where("platform = ? AND order_id = ?", event.Platform, event.OrderID).Order("id asc").Find(&items).Error; err != nil || len(items) == 0 {
				orders = append(orders, event)
				continue
			}
			orders = append(orders, buildOrderStatusNotificationEvents(items, event.Status)...)
			continue
		}
		var items []model.OrderTrackingItem
		if err := database.DB.Where("platform = ? AND order_id = ? AND tracking_number <> ''", event.Platform, event.OrderID).Order("id asc").Find(&items).Error; err != nil {
			continue
		}
		trackingEvents := buildTrackingNotificationEvents(items)
		if event.Status == "cancel_failed_tracking" {
			for index := range trackingEvents {
				trackingEvents[index].Reason = "取消订单失败，已存在追踪号"
			}
		}
		orders = append(orders, trackingEvents...)
	}
	if len(orders) == 0 && len(excludedOrders) == 0 {
		return
	}

	for receiveID, groupedOrders := range groupOrderNotificationsByReceiveID(orders) {
		sendOrderNotificationToReceiveID(receiveID, groupedOrders)
	}
	for receiveID, groupedOrders := range groupOrderNotificationsByReceiveID(excludedOrders) {
		sendMCFExcludedNotificationToReceiveID(receiveID, groupedOrders)
	}
}

func sendOrderNotificationToReceiveID(receiveID string, orders []orderNotificationEvent) {
	text := formatOrderNotificationForReceiveID(receiveID, orders, formatOrderNotification(orders))
	content, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		log.Printf("feishu order notification content error: %v", err)
		return
	}
	body, err := json.Marshal(map[string]interface{}{
		"receive_id": receiveID,
		"msg_type":   "text",
		"content":    string(content),
	})
	if err != nil {
		log.Printf("feishu order notification request error: %v", err)
		return
	}
	if err := postFeishuOrderNotification(body, false); err != nil {
		log.Printf("feishu order notification send error: %v", err)
	}
}

func sendMCFExcludedNotificationToReceiveID(receiveID string, orders []orderNotificationEvent) {
	content, err := json.Marshal(map[string]string{"text": formatMCFExcludedNotification(orders)})
	if err != nil {
		log.Printf("feishu MCF excluded notification content error: %v", err)
		return
	}
	body, err := json.Marshal(map[string]interface{}{
		"receive_id": receiveID,
		"msg_type":   "text",
		"content":    string(content),
	})
	if err != nil {
		log.Printf("feishu MCF excluded notification request error: %v", err)
		return
	}
	if err := postFeishuOrderNotification(body, false); err != nil {
		log.Printf("feishu MCF excluded notification send error: %v", err)
	}
}

func formatOrderNotificationForReceiveID(receiveID string, orders []orderNotificationEvent, text string) string {
	if strings.TrimSpace(receiveID) != feishuLowesOrderReceiveID {
		return text
	}
	shouldMention := false
	for _, order := range orders {
		if order.Status == "tracking" || order.Status == "cancelled" {
			shouldMention = true
			break
		}
	}
	if !shouldMention {
		return text
	}
	return fmt.Sprintf("<at user_id=\"%s\">负责人</at>\n%s", feishuLowesMentionOpenID, text)
}

func groupOrderNotificationsByReceiveID(orders []orderNotificationEvent) map[string][]orderNotificationEvent {
	groups := make(map[string][]orderNotificationEvent, 3)
	for _, order := range orders {
		receiveID := feishuOtherOrderReceiveID
		switch strings.ToUpper(strings.TrimSpace(order.Platform)) {
		case "LOWES":
			receiveID = feishuLowesOrderReceiveID
		case "TEMU":
			receiveID = feishuTemuOrderReceiveID
		}
		groups[receiveID] = append(groups[receiveID], order)
	}
	return groups
}

func postFeishuOrderNotification(body []byte, forceRefresh bool) error {
	token, err := feishuTenantAccessToken(forceRefresh)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequest(http.MethodPost, feishuOrderNotifyURL(), bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
		if err != nil {
			return err
		}
		responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			return nil
		}
		if attempt == 0 && feishuResponseNeedsTokenRefresh(responseBody) {
			token, err = feishuTenantAccessToken(true)
			if err != nil {
				return err
			}
			continue
		}
		return fmt.Errorf("feishu returned status %s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}
	return fmt.Errorf("feishu notification retry exhausted")
}

func feishuTenantAccessToken(forceRefresh bool) (string, error) {
	appID, appSecret := feishuCredentials()
	if appID == "" || appSecret == "" {
		return "", fmt.Errorf("feishu app credentials are not configured")
	}
	feishuTenantTokenCache.Lock()
	defer feishuTenantTokenCache.Unlock()
	if !forceRefresh && feishuTenantTokenCache.token != "" && time.Until(feishuTenantTokenCache.expiresAt) > time.Minute {
		return feishuTenantTokenCache.token, nil
	}
	payload, _ := json.Marshal(map[string]string{"app_id": appID, "app_secret": appSecret})
	req, err := http.NewRequest(http.MethodPost, feishuTenantTokenURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int64  `json:"expire"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&result); err != nil {
		return "", err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices || result.Code != 0 || result.TenantAccessToken == "" {
		return "", fmt.Errorf("feishu token request failed: status=%s code=%d msg=%s", resp.Status, result.Code, result.Msg)
	}
	feishuTenantTokenCache.token = result.TenantAccessToken
	feishuTenantTokenCache.expiresAt = time.Now().Add(time.Duration(result.Expire) * time.Second)
	return result.TenantAccessToken, nil
}

func feishuResponseNeedsTokenRefresh(body []byte) bool {
	var result struct {
		Code int `json:"code"`
	}
	if json.Unmarshal(body, &result) != nil {
		return false
	}
	return result.Code == 99991663 || result.Code == 99991664
}

func formatOrderNotification(orders []orderNotificationEvent) string {
	newOrderCount := 0
	shippedOrderCount := 0
	cancelRequestedOrderCount := 0
	cancelledOrderCount := 0
	for _, item := range orders {
		switch item.Status {
		case "pending_shipment":
			newOrderCount++
		case "cancel_requested":
			cancelRequestedOrderCount++
		case "cancelled":
			cancelledOrderCount++
		default:
			shippedOrderCount++
		}
	}
	summary := make([]string, 0, 3)
	if newOrderCount > 0 {
		summary = append(summary, fmt.Sprintf("新增订单%d条", newOrderCount))
	}
	if shippedOrderCount > 0 {
		summary = append(summary, fmt.Sprintf("已发货订单%d条", shippedOrderCount))
	}
	if cancelRequestedOrderCount > 0 {
		summary = append(summary, fmt.Sprintf("申请取消订单%d条", cancelRequestedOrderCount))
	}
	if cancelledOrderCount > 0 {
		summary = append(summary, fmt.Sprintf("取消订单%d条", cancelledOrderCount))
	}
	lines := []string{strings.Join(summary, "，") + "："}
	for _, item := range orders {
		if item.Status == "cancelled" {
			lines = append(lines, fmt.Sprintf("平台: %s，订单号：%s，状态：已取消", item.Platform, item.OrderID))
			continue
		} else if item.Status == "pending_shipment" {
			lines = append(lines, fmt.Sprintf("平台：%s，订单号：%s，状态：待发货", item.Platform, item.OrderID))
			continue
		} else if item.Status == "cancel_requested" {
			lines = append(lines, fmt.Sprintf("平台：%s，订单号：%s，状态：申请取消", item.Platform, item.OrderID))
			continue
		}
		amazonOrderID := strings.TrimSpace(item.AmazonOrderID)
		if amazonOrderID == "" {
			amazonOrderID = "未填写"
		}
		line := fmt.Sprintf("平台:%s，订单号：%s，追踪号：%s，亚马逊订单编号：%s", item.Platform, item.OrderID, item.TrackingNumber, amazonOrderID)
		if item.Reason != "" {
			line += fmt.Sprintf("，备注：%s", item.Reason)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func formatOrderTrackingNotification(orders []model.OrderTrackingItem) string {
	groups := make(map[string][]model.OrderTrackingItem)
	groupOrder := make([]string, 0, len(orders))
	for _, item := range orders {
		key := item.Platform + "\x00" + item.OrderID
		if _, exists := groups[key]; !exists {
			groupOrder = append(groupOrder, key)
		}
		groups[key] = append(groups[key], item)
	}
	events := make([]orderNotificationEvent, 0, len(orders))
	for _, key := range groupOrder {
		events = append(events, buildTrackingNotificationEvents(groups[key])...)
	}
	return formatOrderNotification(events)
}

func buildTrackingNotificationEvents(items []model.OrderTrackingItem) []orderNotificationEvent {
	if len(items) == 0 {
		return nil
	}
	trackingNumbers := make([]string, 0, len(items))
	trackingKeys := make(map[string]struct{}, len(items))
	for _, item := range items {
		trackingNumber := strings.TrimSpace(item.TrackingNumber)
		if trackingNumber == "" {
			continue
		}
		key := strings.ToUpper(strings.ReplaceAll(trackingNumber, " ", ""))
		if _, exists := trackingKeys[key]; exists {
			continue
		}
		trackingKeys[key] = struct{}{}
		trackingNumbers = append(trackingNumbers, trackingNumber)
	}
	if len(trackingNumbers) == 0 {
		return nil
	}
	base := items[0]
	amazonOrderID := ""
	for _, item := range items {
		if value := strings.TrimSpace(item.AmazonOrderID); value != "" {
			amazonOrderID = value
			break
		}
	}
	originalOrderIDs := notificationOriginalOrderIDs(items)
	if len(originalOrderIDs) == 0 {
		originalOrderIDs = []string{strings.TrimSpace(base.OrderID)}
	}

	result := make([]orderNotificationEvent, 0, len(originalOrderIDs))
	for _, orderID := range originalOrderIDs {
		result = append(result, orderNotificationEvent{
			Platform: base.Platform, OrderID: orderID, AmazonOrderID: amazonOrderID,
			TrackingNumber: strings.Join(trackingNumbers, "、"), Status: "tracking",
		})
	}
	return result
}

func buildOrderStatusNotificationEvents(items []model.OrderTrackingItem, status string) []orderNotificationEvent {
	if len(items) == 0 {
		return nil
	}
	base := items[0]
	originalOrderIDs := notificationOriginalOrderIDs(items)
	if len(originalOrderIDs) == 0 {
		originalOrderIDs = []string{strings.TrimSpace(base.OrderID)}
	}
	result := make([]orderNotificationEvent, 0, len(originalOrderIDs))
	for _, orderID := range originalOrderIDs {
		result = append(result, orderNotificationEvent{Platform: base.Platform, OrderID: orderID, Status: status})
	}
	return result
}

func notificationOriginalOrderIDs(items []model.OrderTrackingItem) []string {
	originalOrderIDs := make([]string, 0, len(items))
	seenOrderIDs := make(map[string]struct{}, len(items))
	addOrderID := func(orderID string) {
		orderID = strings.TrimSpace(orderID)
		if orderID == "" {
			return
		}
		if _, exists := seenOrderIDs[orderID]; exists {
			return
		}
		seenOrderIDs[orderID] = struct{}{}
		originalOrderIDs = append(originalOrderIDs, orderID)
	}
	for _, item := range items {
		for _, orderID := range item.OriginalOrderIDs {
			addOrderID(orderID)
		}
		for _, source := range item.OriginalItems {
			addOrderID(source.OriginalOrderID)
		}
	}
	return originalOrderIDs
}

func formatMCFExcludedNotification(orders []orderNotificationEvent) string {
	lines := []string{"订单转MCF中异常订单："}
	if shouldMentionMCFExcludedOrders(orders) {
		lines = append([]string{fmt.Sprintf("<at user_id=\"%s\">负责人</at>", feishuMCFExcludedMentionID)}, lines...)
	}
	for _, item := range orders {
		lines = append(lines, fmt.Sprintf("平台：%s，订单号：%s，问题：%s", item.Platform, item.OrderID, item.Reason))
	}
	return strings.Join(lines, "\n")
}

func shouldMentionMCFExcludedOrders(orders []orderNotificationEvent) bool {
	for _, item := range orders {
		if strings.TrimSpace(item.Reason) != "该订单商品已生成过 MCF，本次已排除" {
			return true
		}
	}
	return false
}

func feishuOrderNotifyURL() string {
	return feishuOrderNotifyURLValue
}

// SendPDFToFeishuChat uploads a PDF and sends it as a file message to a chat.
func SendPDFToFeishuChat(chatID, fileName string, pdf []byte) error {
	chatID = strings.TrimSpace(chatID)
	fileName = strings.TrimSpace(fileName)
	if chatID == "" {
		return fmt.Errorf("feishu chat id is not configured")
	}
	if len(pdf) == 0 {
		return fmt.Errorf("pdf is empty")
	}
	token, err := feishuTenantAccessToken(false)
	if err != nil {
		return err
	}
	fileKey, err := uploadFeishuFile(token, fileName, pdf)
	if err != nil {
		return err
	}
	content, _ := json.Marshal(map[string]string{"file_key": fileKey})
	body, _ := json.Marshal(map[string]interface{}{
		"receive_id": chatID,
		"msg_type":   "file",
		"content":    string(content),
	})
	return postFeishuMessage(body, token)
}

func uploadFeishuFile(token, fileName string, pdf []byte) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("file_type", "stream"); err != nil {
		return "", err
	}
	if err := writer.WriteField("file_name", fileName); err != nil {
		return "", err
	}
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(pdf); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, "https://open.feishu.cn/open-apis/im/v1/files", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			FileKey string `json:"file_key"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8192)).Decode(&result); err != nil {
		return "", err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices || result.Code != 0 || result.Data.FileKey == "" {
		return "", fmt.Errorf("feishu file upload failed: status=%s code=%d msg=%s", resp.Status, result.Code, result.Msg)
	}
	return result.Data.FileKey, nil
}

func postFeishuMessage(body []byte, token string) error {
	req, err := http.NewRequest(http.MethodPost, feishuOrderNotifyURL(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("feishu file message failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(responseBody)))
	}
	return nil
}
