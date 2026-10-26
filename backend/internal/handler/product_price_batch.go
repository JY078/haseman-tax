package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"advertising-image/backend/internal/database"
	"github.com/gin-gonic/gin"
)

const (
	// One worker per global scrape slot: the slot channel already caps how many
	// Firecrawl scrapes may run at once, so extra workers would only queue.
	amazonPriceBatchWorkers = amazonPriceScrapeParallel
	// A finished job is only kept for a short while; drop old ones so a
	// long-lived process does not retain every result forever.
	amazonBatchJobRetention = 30 * time.Minute
	amazonBatchJobLimit     = 20
)

type amazonPriceBatchFailure struct {
	ID     uint   `json:"id"`
	SKU    string `json:"sku"`
	Reason string `json:"reason"`
}

type amazonPriceBatchJob struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Total   int    `json:"total"`
	Done    int    `json:"done"`
	Success int    `json:"success"`
	// Several SKUs can share one ASIN, so the number of pages that actually
	// have to be scraped is often smaller than the number of products.
	Asins      int                       `json:"asins"`
	Scraped    int                       `json:"scraped"`
	Failures   []amazonPriceBatchFailure `json:"failures"`
	Error      string                    `json:"error,omitempty"`
	StartedAt  time.Time                 `json:"started_at"`
	FinishedAt *time.Time                `json:"finished_at,omitempty"`

	mu     sync.Mutex
	cancel context.CancelFunc
}

// A value copy would duplicate the mutex, so hand back a detached pointer.
func (j *amazonPriceBatchJob) snapshot() *amazonPriceBatchJob {
	j.mu.Lock()
	defer j.mu.Unlock()
	failures := make([]amazonPriceBatchFailure, len(j.Failures))
	copy(failures, j.Failures)
	return &amazonPriceBatchJob{
		ID: j.ID, Status: j.Status, Total: j.Total, Done: j.Done, Success: j.Success,
		Asins: j.Asins, Scraped: j.Scraped, Failures: failures, Error: j.Error,
		StartedAt: j.StartedAt, FinishedAt: j.FinishedAt,
	}
}

func (j *amazonPriceBatchJob) record(result func()) {
	j.mu.Lock()
	defer j.mu.Unlock()
	result()
}

func (j *amazonPriceBatchJob) fail(item productPriceRow, reason string) {
	j.record(func() {
		j.Done++
		j.Failures = append(j.Failures, amazonPriceBatchFailure{ID: item.ID, SKU: item.SKU, Reason: reason})
	})
}

func (j *amazonPriceBatchJob) succeed(item productPriceRow) {
	j.record(func() {
		j.Done++
		j.Success++
	})
}

var (
	amazonPriceBatchMu      sync.Mutex
	amazonPriceBatchJobs    = map[string]*amazonPriceBatchJob{}
	amazonPriceBatchRunning *amazonPriceBatchJob
)

func pruneAmazonPriceBatchJobsLocked() {
	for id, job := range amazonPriceBatchJobs {
		if job == amazonPriceBatchRunning {
			continue
		}
		if job.FinishedAt == nil || time.Since(*job.FinishedAt) < amazonBatchJobRetention {
			continue
		}
		delete(amazonPriceBatchJobs, id)
	}
	for len(amazonPriceBatchJobs) > amazonBatchJobLimit {
		var oldestID string
		var oldest time.Time
		for id, job := range amazonPriceBatchJobs {
			if job == amazonPriceBatchRunning {
				continue
			}
			if oldestID == "" || job.StartedAt.Before(oldest) {
				oldestID, oldest = id, job.StartedAt
			}
		}
		if oldestID == "" {
			return
		}
		delete(amazonPriceBatchJobs, oldestID)
	}
}

// One Amazon page to scrape plus every product that shares its ASIN.
type amazonPriceBatchTask struct {
	asin  string
	items []productPriceRow
}

// Grouping by ASIN keeps one scrape per distinct product page instead of one
// per SKU. Products without a usable ASIN fail immediately, without spending
// a Firecrawl credit on them.
func groupAmazonPriceBatchItems(items []productPriceRow) ([]amazonPriceBatchTask, []amazonPriceBatchFailure) {
	order := []string{}
	grouped := map[string][]productPriceRow{}
	invalid := []amazonPriceBatchFailure{}
	for _, item := range items {
		asin := strings.ToUpper(strings.TrimSpace(item.ASIN))
		if !amazonASINPattern.MatchString(asin) {
			invalid = append(invalid, amazonPriceBatchFailure{ID: item.ID, SKU: item.SKU, Reason: "请先在产品库补全有效 ASIN"})
			continue
		}
		if _, ok := grouped[asin]; !ok {
			order = append(order, asin)
		}
		grouped[asin] = append(grouped[asin], item)
	}
	tasks := make([]amazonPriceBatchTask, 0, len(order))
	for _, asin := range order {
		tasks = append(tasks, amazonPriceBatchTask{asin: asin, items: grouped[asin]})
	}
	return tasks, invalid
}

func (j *amazonPriceBatchJob) scrapeTask(ctx context.Context, client *http.Client, apiBase, apiKey string, task amazonPriceBatchTask, userID uint) {
	prices, err := fetchAmazonPrice(ctx, client, apiBase, apiKey, task.asin)
	j.record(func() { j.Scraped++ })
	if err != nil {
		for _, item := range task.items {
			j.fail(item, err.Error())
		}
		return
	}
	now := time.Now()
	for _, item := range task.items {
		result := saveFetchedAmazonPrice(item, prices, userID, now)
		switch {
		case result.Error != nil:
			j.fail(item, "价格保存失败")
		case result.RowsAffected == 0 && !amazonPriceAlreadySaved(item.ID, prices):
			j.fail(item, "抓取期间商品或价格已变更，请刷新后重试")
		default:
			item.AmazonPrice, item.AmazonSalePrice, item.AmazonBestDeal = prices.Price, prices.SalePrice, prices.BestDeal
			if historyErr := recordProductPriceHistory(database.DB, item, now); historyErr != nil {
				j.fail(item, "价格历史保存失败")
				continue
			}
			j.succeed(item)
		}
	}
}

func (j *amazonPriceBatchJob) run(ctx context.Context, client *http.Client, apiBase, apiKey string, tasks []amazonPriceBatchTask, userID uint) {
	status := "done"
	defer func() {
		finished := time.Now()
		// Both locks are taken in this order everywhere so pruning, snapshots
		// and cancellation can never deadlock against the finishing job.
		amazonPriceBatchMu.Lock()
		j.mu.Lock()
		j.Status = status
		j.FinishedAt = &finished
		j.mu.Unlock()
		if amazonPriceBatchRunning == j {
			amazonPriceBatchRunning = nil
		}
		amazonPriceBatchMu.Unlock()
	}()
	slots := make(chan struct{}, amazonPriceBatchWorkers)
	var wg sync.WaitGroup
	for i := range tasks {
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			status = "cancelled"
			wg.Wait()
			return
		}
		task := tasks[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-slots }()
			if ctx.Err() != nil {
				return
			}
			j.scrapeTask(ctx, client, apiBase, apiKey, task, userID)
		}()
	}
	wg.Wait()
	if ctx.Err() != nil {
		status = "cancelled"
	}
}

func loadAmazonPriceBatchItems(c *gin.Context, ids []uint) ([]productPriceRow, error) {
	query := productPriceQuery().
		// Batch Amazon fetching is only available for products already mapped to
		// an active TikTok SKU.
		Where("m.id IS NOT NULL").
		Select("p.id,p.sku,p.asin,p.amazon_price,p.amazon_sale_price,p.amazon_best_deal,p.price_updated_at")
	if len(ids) > 0 {
		query = query.Where("p.id IN ?", ids)
	}
	items := []productPriceRow{}
	// Fetch in SKU order so batch processing follows the product price list's
	// SKU sequence, regardless of whether a product has a mapping.
	if err := query.Order("p.sku ASC, p.id ASC").Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// StartAmazonPriceBatchFetch scrapes every US product with a valid ASIN through a job id,
// so no single HTTP request has to stay open for the whole run and the page can
// show progress while it works.
func StartAmazonPriceBatchFetch(c *gin.Context) {
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	key := strings.TrimSpace(getSettingValue("firecrawl_api_key"))
	if key == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "请先在管理后台配置 Firecrawl API Key"})
		return
	}
	var body struct {
		IDs []uint `json:"ids"`
	}
	if c.Request.Body != nil {
		_ = json.NewDecoder(io.LimitReader(c.Request.Body, 1<<20)).Decode(&body)
	}
	items, err := loadAmazonPriceBatchItems(c, body.IDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取待获取商品"})
		return
	}
	if len(items) == 0 {
		c.JSON(http.StatusOK, gin.H{"started": false, "total": 0})
		return
	}
	tasks, invalid := groupAmazonPriceBatchItems(items)
	job := &amazonPriceBatchJob{
		ID:        strconv.FormatInt(time.Now().UnixNano(), 36),
		Status:    "running",
		Total:     len(items),
		Asins:     len(tasks),
		Failures:  append([]amazonPriceBatchFailure{}, invalid...),
		StartedAt: time.Now(),
	}
	job.Done = len(invalid)
	amazonPriceBatchMu.Lock()
	if amazonPriceBatchRunning != nil {
		running := amazonPriceBatchRunning.snapshot()
		amazonPriceBatchMu.Unlock()
		c.JSON(http.StatusConflict, gin.H{"error": "已有批量获取任务在进行中", "job": running})
		return
	}
	amazonPriceBatchRunning = job
	amazonPriceBatchJobs[job.ID] = job
	pruneAmazonPriceBatchJobsLocked()
	amazonPriceBatchMu.Unlock()

	client, err := newFirecrawlHTTPClient(strings.TrimSpace(getSettingValue("firecrawl_proxy")))
	if err != nil {
		amazonPriceBatchMu.Lock()
		amazonPriceBatchRunning = nil
		delete(amazonPriceBatchJobs, job.ID)
		amazonPriceBatchMu.Unlock()
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Firecrawl 代理配置无效"})
		return
	}
	// One attempt gets the same room as a single-product fetch; a slower
	// attempt is retried instead of blocking a worker for minutes.
	client.Timeout = amazonPriceAttemptTimeout
	// The request context dies with the handler, so the job needs its own.
	ctx, cancel := context.WithCancel(context.Background())
	job.mu.Lock()
	job.cancel = cancel
	job.mu.Unlock()
	go func() {
		defer cancel()
		job.run(ctx, client, defaultFirecrawlAPIBase, key, tasks, user.ID)
	}()
	c.JSON(http.StatusAccepted, gin.H{"started": true, "job": job.snapshot()})
}

func GetAmazonPriceBatchFetch(c *gin.Context) {
	if _, ok := getOrderMappingUser(c); !ok {
		return
	}
	amazonPriceBatchMu.Lock()
	job := amazonPriceBatchJobs[c.Param("jobId")]
	amazonPriceBatchMu.Unlock()
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "批量获取任务不存在或已清理"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"job": job.snapshot()})
}

func CancelAmazonPriceBatchFetch(c *gin.Context) {
	if _, ok := getOrderMappingUser(c); !ok {
		return
	}
	amazonPriceBatchMu.Lock()
	job := amazonPriceBatchJobs[c.Param("jobId")]
	amazonPriceBatchMu.Unlock()
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "批量获取任务不存在或已清理"})
		return
	}
	job.mu.Lock()
	if job.cancel != nil && job.Status == "running" {
		job.cancel()
		job.Status = "cancelling"
	}
	job.mu.Unlock()
	c.JSON(http.StatusOK, gin.H{"job": job.snapshot()})
}
