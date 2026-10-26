package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var amazonASINPattern = regexp.MustCompile(`^[A-Z0-9]{10}$`)
var amazonEvidenceAmountPattern = regexp.MustCompile(`\b[0-9]+(?:\.[0-9]{1,2})?\b`)
var amazonSplitPricePattern = regexp.MustCompile(`\$\s*([0-9][0-9,]*)\s*[.\s]\s*([0-9]{2})\b`)
var amazonLooseAmountPattern = regexp.MustCompile(`(?i)(?:USD|US\$|\$)\s*([0-9][0-9,]*(?:\.[0-9]{1,2})?)|\b([0-9][0-9,]*\.[0-9]{1,2})\s*(?:USD|dollars?)\b`)

func normalizeAmazonPriceText(value string) string {
	value = html.UnescapeString(value)
	value = strings.NewReplacer("\\$", "$", "**", "", "__", "", "~~", "", "\u200b", "").Replace(value)
	value = amazonSplitPricePattern.ReplaceAllString(value, "$$${1}.${2}")
	return strings.Join(strings.Fields(value), " ")
}

func normalizeAmazonAmount(value *string) (*string, int64, error) {
	if value == nil {
		return nil, 0, nil
	}
	raw := strings.ToUpper(normalizeAmazonPriceText(*value))
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "USD"))
	for _, prefix := range []string{"USD", "US$", "$"} {
		raw = strings.TrimSpace(strings.TrimPrefix(raw, prefix))
	}
	// Only accept correctly grouped thousands, never a decimal comma.
	if strings.Contains(raw, ",") {
		if !regexp.MustCompile(`^[0-9]{1,3}(,[0-9]{3})+(\.[0-9]{1,2})?$`).MatchString(raw) {
			return nil, 0, errors.New("invalid amount")
		}
		raw = strings.ReplaceAll(raw, ",", "")
	}
	return normalizeProductPrice(&raw)
}

func normalizeAmazonAmountLoose(value *string) (*string, int64, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, 0, nil
	}
	match := amazonLooseAmountPattern.FindStringSubmatch(normalizeAmazonPriceText(*value))
	if match == nil {
		return normalizeAmazonAmount(value)
	}
	amount := match[1]
	if amount == "" {
		amount = match[2]
	}
	return normalizeAmazonAmount(&amount)
}

type amazonPriceConnectionError struct{ message string }

func (e *amazonPriceConnectionError) Error() string { return e.message }

func amazonPriceNetworkError(ctx context.Context, client *http.Client, req *http.Request, err error) error {
	message := "无法连接 Firecrawl，请检查服务器网络"
	viaProxy := false
	if transport, ok := client.Transport.(*http.Transport); ok && transport.Proxy != nil {
		proxy, proxyErr := transport.Proxy(req)
		viaProxy = proxyErr == nil && proxy != nil
	}
	var dnsErr *net.DNSError
	var netErr net.Error
	var opErr *net.OpError
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		message = "获取请求已取消，价格未更新"
	case errors.As(err, &dnsErr):
		message = "Firecrawl 域名解析失败，请检查服务器 DNS"
		if viaProxy {
			message = "代理或 Firecrawl 域名解析失败，请检查代理地址及服务器 DNS"
		}
	case errors.As(err, &opErr) && (opErr.Op == "dial" || opErr.Op == "proxyconnect"):
		if viaProxy {
			message = "Firecrawl 代理连接失败，请检查后台代理地址是否可从服务器访问；无需代理时清空该设置"
		}
	case errors.As(err, &netErr) && netErr.Timeout():
		message = "等待 Firecrawl 响应超时，价格未更新，请稍后重试"
		if viaProxy {
			message = "通过代理等待 Firecrawl 响应超时，请检查后台代理连接"
		}
	default:
		if viaProxy {
			message = "通过代理连接 Firecrawl 失败，请检查后台代理地址及服务器网络"
		}
	}
	return &amazonPriceConnectionError{message: message}
}

// Use the same marketplace scope as the price ledger. Amazon price fetching
// does not require a current TikTok mapping; the mapping columns are optional.
func ListAmazonPriceFetchTargets(c *gin.Context) {
	if _, ok := getOrderMappingUser(c); !ok {
		return
	}
	if strings.TrimSpace(getSettingValue("firecrawl_api_key")) == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "请先在管理后台配置 Firecrawl API Key"})
		return
	}
	items := []productPriceRow{}
	if err := productPriceQuery().Select("DISTINCT p.id, p.sku, p.asin").Order("p.id").Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取待获取商品"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

const (
	// Firecrawl counts the time a job waits for a free browser against the
	// scrape timeout and rejects bursts with 429, so each attempt needs more
	// room than a plain page load and transient failures need a retry.
	// Amazon pages occasionally spend 60–80 seconds in Firecrawl's browser
	// queue. Keep the HTTP deadline above Firecrawl's scrape timeout so a slow
	// response body is not cut off by the client first.
	amazonPriceAttemptTimeout  = 90 * time.Second
	amazonPriceFetchBudget     = 300 * time.Second
	amazonPriceFetchAttempts   = 3
	amazonPriceRetryMax        = 20 * time.Second
	amazonPriceScrapeTimeoutMS = 60000
	// Two browser jobs improve batch throughput while staying below the rate
	// limit that caused 429 responses when four jobs were launched together.
	amazonPriceScrapeParallel = 2
)

// Overridden by tests to keep the retry suite fast.
var amazonPriceRetryBase = 3 * time.Second

// Terminal parse outcomes: the page was fetched, so another attempt would only
// spend another credit and reach the same answer.
var errAmazonPriceNotFound = errors.New("Firecrawl 已返回页面，但未找到当前 ASIN 标题下方的商品价格")
var errAmazonPriceUnavailable = errors.New("亚马逊页面显示该商品当前无货，未获取到价格")
var errAmazonFirecrawlRateLimited = errors.New("Firecrawl 请求过于频繁，请稍后重试")

func amazonPriceErrorIsTerminal(err error) bool {
	return errors.Is(err, errAmazonPriceNotFound) || errors.Is(err, errAmazonPriceUnavailable)
}

// Firecrawl rate limits are per team, so every caller of the same key shares
// one budget. Queue price scrapes instead of letting them collide.
var amazonPriceScrapeSlots = make(chan struct{}, amazonPriceScrapeParallel)

type amazonRetryableError struct {
	error
	wait time.Duration
}

func (e *amazonRetryableError) Unwrap() error { return e.error }

func retryableAmazonPriceError(err error, wait time.Duration) error {
	return &amazonRetryableError{error: err, wait: wait}
}

func amazonRetryAfter(header string) time.Duration {
	value := strings.TrimSpace(header)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil {
		if wait := time.Until(when); wait > 0 {
			return wait
		}
	}
	return 0
}

func amazonPriceRetryWait(err error, attempt int) (time.Duration, bool) {
	var retryable *amazonRetryableError
	if !errors.As(err, &retryable) {
		return 0, false
	}
	wait := retryable.wait
	if wait <= 0 {
		wait = amazonPriceRetryBase << (attempt - 2)
	}
	if wait > amazonPriceRetryMax {
		wait = amazonPriceRetryMax
	}
	return wait, true
}

type amazonPriceExtraction struct {
	ASIN         string  `json:"asin"`
	Currency     string  `json:"currency"`
	CurrentPrice *string `json:"current_price"`
	ListPrice    *string `json:"list_price"`
	Evidence     string  `json:"evidence"`
}

func (v *amazonPriceExtraction) UnmarshalJSON(data []byte) error {
	var raw struct {
		ASIN     string          `json:"asin"`
		Currency string          `json:"currency"`
		Current  json.RawMessage `json:"current_price"`
		List     json.RawMessage `json:"list_price"`
		Evidence string          `json:"evidence"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	parse := func(value json.RawMessage) (*string, error) {
		if len(value) == 0 || string(value) == "null" {
			return nil, nil
		}
		var text string
		if json.Unmarshal(value, &text) == nil {
			return &text, nil
		}
		var number json.Number
		if json.Unmarshal(value, &number) == nil {
			s := number.String()
			return &s, nil
		}
		return nil, errors.New("invalid price type")
	}
	current, err := parse(raw.Current)
	if err != nil {
		return err
	}
	list, err := parse(raw.List)
	if err != nil {
		return err
	}
	*v = amazonPriceExtraction{ASIN: raw.ASIN, Currency: raw.Currency, CurrentPrice: current, ListPrice: list, Evidence: raw.Evidence}
	return nil
}

func validateAmazonExtraction(value amazonPriceExtraction, asin, markdown string) (productPriceInput, error) {
	failure := func(reason string) (productPriceInput, error) {
		return productPriceInput{}, errors.New(reason + "，已保留原值")
	}
	if strings.TrimSpace(value.ASIN) != "" && strings.ToUpper(strings.TrimSpace(value.ASIN)) != asin {
		return failure("抓取结果缺少目标 ASIN 或返回了其他变体")
	}
	if value.Currency != "" && strings.ToUpper(strings.TrimSpace(value.Currency)) != "USD" {
		return failure("抓取结果未确认美元币种")
	}
	current, amount, err := normalizeAmazonAmountLoose(value.CurrentPrice)
	if err != nil || current == nil || amount <= 0 {
		return failure("抓取页面未提供有效的主商品售价")
	}
	list, original, err := normalizeAmazonAmountLoose(value.ListPrice)
	if err != nil {
		return failure("抓取的价格格式无效")
	}
	// Firecrawl may label the two visible amounts in reverse order. Treat the
	// higher amount as list price and the lower amount as sale price instead of
	// discarding an otherwise usable result.
	if list != nil && original < amount {
		list, original = current, amount
		current, amount, _ = normalizeAmazonAmount(value.ListPrice)
	}
	result := productPriceInput{Price: current}
	if list != nil && original > amount {
		result.Price = list
		result.SalePrice = current
	}
	return result, nil
}

func fetchAmazonPrice(ctx context.Context, client *http.Client, apiBase, apiKey, asin string) (productPriceInput, error) {
	if !amazonASINPattern.MatchString(asin) {
		return productPriceInput{}, errors.New("ASIN 缺失或格式不正确")
	}
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > amazonPriceFetchBudget {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, amazonPriceFetchBudget)
		defer cancel()
	}
	select {
	case amazonPriceScrapeSlots <- struct{}{}:
	case <-ctx.Done():
		return productPriceInput{}, amazonPriceQueueError(ctx)
	}
	defer func() { <-amazonPriceScrapeSlots }()
	var lastErr error
	for attempt := 1; attempt <= amazonPriceFetchAttempts; attempt++ {
		if attempt > 1 {
			wait, retryable := amazonPriceRetryWait(lastErr, attempt)
			if !retryable {
				return productPriceInput{}, lastErr
			}
			timer := time.NewTimer(wait)
			select {
			case <-timer.C:
				timer.Stop()
			case <-ctx.Done():
				timer.Stop()
				return productPriceInput{}, lastErr
			}
		}
		prices, err := fetchAmazonPriceOnce(ctx, client, apiBase, apiKey, asin)
		if err == nil {
			return prices, nil
		}
		lastErr = err
	}
	return productPriceInput{}, lastErr
}

func amazonPriceQueueError(ctx context.Context) error {
	if errors.Is(ctx.Err(), context.Canceled) {
		return errors.New("获取请求已取消，价格未更新")
	}
	return errors.New("等待亚马逊价格抓取队列超时，请稍后重试")
}

func fetchAmazonPriceOnce(ctx context.Context, client *http.Client, apiBase, apiKey, asin string) (productPriceInput, error) {
	payload := map[string]interface{}{
		"url": "https://www.amazon.com/dp/" + asin,
		// Keep HTML so the parser can scope prices to the selected ASIN's
		// main product region. Markdown contains recommendations and other
		// seller prices without reliable DOM boundaries and must not be used
		// as a price fallback.
		"formats":  []interface{}{"html"},
		"location": map[string]interface{}{"country": "US", "languages": []string{"en-US"}},
		// Amazon frequently serves its price block outside Firecrawl's
		// main-content extraction. Keep the full HTML and let Firecrawl choose
		// the proxy mode; basic proxy requests are the common timeout source.
		// A short cache window prevents repeated clicks from creating another
		// Amazon browser job while still refreshing prices several times a day.
		"maxAge": 300000, "storeInCache": true, "onlyMainContent": false, "timeout": amazonPriceScrapeTimeoutMS,
		"proxy": "auto", "blockAds": true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return productPriceInput{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+"/scrape", bytes.NewReader(body))
	if err != nil {
		return productPriceInput{}, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Do(req)
	if err != nil {
		return productPriceInput{}, retryableAmazonPriceError(amazonPriceNetworkError(ctx, client, req, err), 0)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		prices, err := parseAmazonFirecrawlResponse(response.Body, asin)
		if err != nil && !amazonPriceErrorIsTerminal(err) {
			// Empty, truncated or unreadable bodies clear on a later attempt.
			return productPriceInput{}, retryableAmazonPriceError(err, 0)
		}
		return prices, err
	}
	wait := amazonRetryAfter(response.Header.Get("Retry-After"))
	switch response.StatusCode {
	case 401, 403:
		return productPriceInput{}, errors.New("Firecrawl API Key 无效或无权限")
	case 402:
		return productPriceInput{}, errors.New("Firecrawl 额度不足")
	case 429:
		return productPriceInput{}, retryableAmazonPriceError(errAmazonFirecrawlRateLimited, wait)
	default:
		message := fmt.Errorf("Firecrawl 抓取失败（HTTP %d）", response.StatusCode)
		if response.StatusCode >= http.StatusInternalServerError {
			return productPriceInput{}, retryableAmazonPriceError(message, wait)
		}
		return productPriceInput{}, message
	}
}

func parseAmazonFirecrawlResponse(body io.Reader, asin string) (productPriceInput, error) {
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			RawHTML  string `json:"rawHtml"`
			HTML     string `json:"html"`
			Markdown string `json:"markdown"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(body, 128<<20)).Decode(&envelope); err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return productPriceInput{}, errors.New("Firecrawl 响应读取超时，系统将自动重试")
		}
		if errors.Is(err, io.EOF) {
			return productPriceInput{}, errors.New("Firecrawl 返回了空响应，请重试")
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return productPriceInput{}, errors.New("Firecrawl 响应传输不完整，请重试")
		}
		var syntax *json.SyntaxError
		var field *json.UnmarshalTypeError
		if errors.As(err, &syntax) || errors.As(err, &field) {
			return productPriceInput{}, errors.New("Firecrawl 响应格式异常，无法读取页面数据")
		}
		return productPriceInput{}, errors.New("读取 Firecrawl 响应失败或超时，请重试")
	}
	if !envelope.Success {
		return productPriceInput{}, errors.New("Firecrawl 报告抓取失败，请稍后重试")
	}
	document := envelope.Data.RawHTML
	if document == "" {
		document = envelope.Data.HTML
	}
	prices, err := amazonHTMLPrice(document, asin)
	if err == nil {
		return prices, nil
	}
	// Keep the reason the page gave: an out-of-stock listing must not be
	// reported as a generic parse failure.
	if amazonPriceErrorIsTerminal(err) {
		return productPriceInput{}, err
	}
	if document == "" {
		return productPriceInput{}, errors.New("Firecrawl 任务完成但返回内容为空，请检查请求格式或稍后重试")
	}
	return productPriceInput{}, errAmazonPriceNotFound
}

func FetchAmazonProductPrice(c *gin.Context) {
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(400, gin.H{"error": "无效的商品"})
		return
	}
	var item productPriceRow
	result := productPriceQuery().Where("p.id = ?", id).Select("p.id,p.sku,p.asin,p.amazon_price,p.amazon_sale_price,p.amazon_best_deal,p.price_updated_at").Limit(1).Scan(&item)
	if result.Error != nil {
		c.JSON(500, gin.H{"error": "读取商品失败"})
		return
	}
	if item.ID == 0 {
		c.JSON(404, gin.H{"error": "商品不存在或不在美国站范围内"})
		return
	}
	asin := strings.ToUpper(strings.TrimSpace(item.ASIN))
	if !amazonASINPattern.MatchString(asin) {
		c.JSON(422, gin.H{"error": "请先在产品库补全有效 ASIN"})
		return
	}
	key := strings.TrimSpace(getSettingValue("firecrawl_api_key"))
	if key == "" {
		c.JSON(503, gin.H{"error": "请先在管理后台配置 Firecrawl API Key"})
		return
	}
	client, err := newFirecrawlHTTPClient(strings.TrimSpace(getSettingValue("firecrawl_proxy")))
	if err != nil {
		c.JSON(503, gin.H{"error": "Firecrawl 代理配置无效"})
		return
	}
	// A scrape that queues behind another job still burns its own timeout, so
	// one attempt needs more room than a plain page load.
	client.Timeout = amazonPriceAttemptTimeout
	prices, err := fetchAmazonPrice(c.Request.Context(), client, defaultFirecrawlAPIBase, key, asin)
	if err != nil {
		var connectionErr *amazonPriceConnectionError
		switch {
		case errors.As(err, &connectionErr):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		case errors.Is(err, errAmazonFirecrawlRateLimited):
			// Let the client apply its own backoff on top of the server retries.
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		default:
			c.JSON(502, gin.H{"error": err.Error()})
		}
		return
	}
	now := time.Now()
	result = saveFetchedAmazonPrice(item, prices, user.ID, now)
	if result.Error != nil {
		c.JSON(500, gin.H{"error": "价格保存失败"})
		return
	}
	if result.RowsAffected == 0 && !amazonPriceAlreadySaved(item.ID, prices) {
		c.JSON(409, gin.H{"error": "抓取期间商品或价格已变更，请刷新后重试"})
		return
	}
	item.AmazonPrice, item.AmazonSalePrice, item.AmazonBestDeal = prices.Price, prices.SalePrice, prices.BestDeal
	if err := recordProductPriceHistory(database.DB, item, now); err != nil {
		c.JSON(500, gin.H{"error": "价格历史保存失败"})
		return
	}
	c.JSON(200, gin.H{"amazon_price": prices.Price, "amazon_sale_price": prices.SalePrice, "updated_at": now})
}

func saveFetchedAmazonPrice(item productPriceRow, prices productPriceInput, userID uint, now time.Time) *gorm.DB {
	// Re-check the product and avoid overwriting a manual edit made during scraping.
	return database.DB.Table("product_skus").Where("id = ? AND sku = ? AND asin = ? AND price_updated_at IS NOT DISTINCT FROM ? AND amazon_price IS NOT DISTINCT FROM ? AND amazon_sale_price IS NOT DISTINCT FROM ? AND amazon_best_deal IS NOT DISTINCT FROM ?", item.ID, item.SKU, item.ASIN, item.PriceUpdatedAt, item.AmazonPrice, item.AmazonSalePrice, item.AmazonBestDeal).
		Updates(map[string]interface{}{"amazon_price": prices.Price, "amazon_sale_price": prices.SalePrice, "amazon_best_deal": prices.BestDeal, "price_updated_at": now, "updated_by": userID})
}

// A duplicate fetch can lose the conditional update after another request has
// already written the exact same prices. Treat that idempotent result as
// success; only a different newer value remains a real concurrency conflict.
func amazonPriceAlreadySaved(id uint, prices productPriceInput) bool {
	var count int64
	database.DB.Table("product_skus").Where("id = ? AND amazon_price IS NOT DISTINCT FROM ? AND amazon_sale_price IS NOT DISTINCT FROM ? AND amazon_best_deal IS NOT DISTINCT FROM ?", id, prices.Price, prices.SalePrice, prices.BestDeal).Count(&count)
	return count > 0
}
