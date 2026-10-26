package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Several SKUs can point at the same Amazon page, and products without a usable
// ASIN must not cost a scrape at all.
func TestAmazonBatchGroupsItemsByASIN(t *testing.T) {
	items := []productPriceRow{
		{ID: 1, SKU: "A", ASIN: "B012345678"},
		{ID: 2, SKU: "B", ASIN: " b012345678 "},
		{ID: 3, SKU: "C", ASIN: "B098765432"},
		{ID: 4, SKU: "D", ASIN: ""},
		{ID: 5, SKU: "E", ASIN: "nope"},
	}
	tasks, invalid := groupAmazonPriceBatchItems(items)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 distinct pages, got %d", len(tasks))
	}
	if tasks[0].asin != "B012345678" || len(tasks[0].items) != 2 {
		t.Fatalf("shared ASINs must be grouped, got %s with %d items", tasks[0].asin, len(tasks[0].items))
	}
	if len(invalid) != 2 || invalid[0].SKU != "D" {
		t.Fatalf("products without an ASIN must fail up front, got %+v", invalid)
	}
}

// A page is scraped once per ASIN, not once per SKU, and every product that
// shares it still gets its own result reported.
func TestAmazonBatchScrapesEachASINOnce(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		var payload struct {
			URL string `json:"url"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		asin := payload.URL[strings.LastIndex(payload.URL, "/")+1:]
		// A page without a price is a terminal answer, so it is not retried.
		w.Write([]byte(`{"success":true,"data":{"html":"<input id=\"ASIN\" value=\"` + asin + `\"><div>no price</div>"}}`))
	}))
	defer server.Close()

	items := []productPriceRow{
		{ID: 1, SKU: "A", ASIN: "B012345678"},
		{ID: 2, SKU: "B", ASIN: "B012345678"},
		{ID: 3, SKU: "C", ASIN: "B098765432"},
	}
	tasks, _ := groupAmazonPriceBatchItems(items)
	job := &amazonPriceBatchJob{ID: "job-1", Status: "running", Total: len(items), Asins: len(tasks), Failures: []amazonPriceBatchFailure{}, StartedAt: time.Now()}
	job.run(context.Background(), server.Client(), server.URL, "key", tasks, 1)

	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("3 SKUs over 2 ASINs must scrape twice, got %d", got)
	}
	snapshot := job.snapshot()
	if snapshot.Done != 3 || snapshot.Scraped != 2 {
		t.Fatalf("every product must be reported, got done=%d scraped=%d", snapshot.Done, snapshot.Scraped)
	}
	if len(snapshot.Failures) != 3 {
		t.Fatalf("expected 3 results, got %d", len(snapshot.Failures))
	}
	if snapshot.FinishedAt == nil || snapshot.Status != "done" {
		t.Fatalf("job must finish, status=%s", snapshot.Status)
	}
}

// Progress has to move product by product: a long run that only reports at the
// end looks stalled to whoever is waiting on it.
func TestAmazonBatchReportsProgressWhileRunning(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		var payload struct {
			URL string `json:"url"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		asin := payload.URL[strings.LastIndex(payload.URL, "/")+1:]
		w.Write([]byte(`{"success":true,"data":{"html":"<input id=\"ASIN\" value=\"` + asin + `\"><div>no price</div>"}}`))
	}))
	defer server.Close()

	items := []productPriceRow{{ID: 1, SKU: "A", ASIN: "B012345678"}, {ID: 2, SKU: "B", ASIN: "B098765432"}}
	tasks, _ := groupAmazonPriceBatchItems(items)
	job := &amazonPriceBatchJob{ID: "job-1", Status: "running", Total: len(items), Asins: len(tasks), Failures: []amazonPriceBatchFailure{}, StartedAt: time.Now()}
	done := make(chan struct{})
	go func() {
		job.run(context.Background(), server.Client(), server.URL, "key", tasks, 1)
		close(done)
	}()
	// Nothing has been scraped yet, so nothing may be reported as processed.
	if job.snapshot().Done != 0 {
		t.Fatal("progress must start at zero")
	}
	close(release)
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("job must finish")
	}
	if got := job.snapshot().Done; got != 2 {
		t.Fatalf("expected both products reported, got %d", got)
	}
}

// Firecrawl's rate limit is a per-team budget, so the job must never run more
// scrapes at once than the shared slot channel allows.
func TestAmazonBatchRespectsScrapeConcurrency(t *testing.T) {
	var inFlight, peak int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&inFlight, 1)
		for {
			peakValue := atomic.LoadInt32(&peak)
			if current <= peakValue || atomic.CompareAndSwapInt32(&peak, peakValue, current) {
				break
			}
		}
		time.Sleep(150 * time.Millisecond)
		atomic.AddInt32(&inFlight, -1)
		var payload struct {
			URL string `json:"url"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		asin := payload.URL[strings.LastIndex(payload.URL, "/")+1:]
		w.Write([]byte(`{"success":true,"data":{"html":"<input id=\"ASIN\" value=\"` + asin + `\"><div>no price</div>"}}`))
	}))
	defer server.Close()

	items := make([]productPriceRow, 6)
	for i := range items {
		items[i] = productPriceRow{ID: uint(i + 1), SKU: "SKU" + strconv.Itoa(i), ASIN: "B01234567" + strconv.Itoa(i)}
	}
	tasks, _ := groupAmazonPriceBatchItems(items)
	job := &amazonPriceBatchJob{ID: "job-1", Status: "running", Total: len(items), Asins: len(tasks), Failures: []amazonPriceBatchFailure{}, StartedAt: time.Now()}
	job.run(context.Background(), server.Client(), server.URL, "key", tasks, 1)

	if got := atomic.LoadInt32(&peak); got > amazonPriceBatchWorkers {
		t.Fatalf("at most %d scrapes may run at once, peaked at %d", amazonPriceBatchWorkers, got)
	}
	if got := job.snapshot().Done; got != len(items) {
		t.Fatalf("expected %d results, got %d", len(items), got)
	}
}

// Cancelling must stop the pages that have not started yet and still report the
// ones that did, and polling a running job must never see a half-written state.
func TestAmazonBatchCancelStopsRemainingPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		var payload struct {
			URL string `json:"url"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		asin := payload.URL[strings.LastIndex(payload.URL, "/")+1:]
		w.Write([]byte(`{"success":true,"data":{"html":"<input id=\"ASIN\" value=\"` + asin + `\"><div>no price</div>"}}`))
	}))
	defer server.Close()

	items := make([]productPriceRow, 12)
	for i := range items {
		items[i] = productPriceRow{ID: uint(i + 1), SKU: "SKU" + strconv.Itoa(i), ASIN: "B01234567" + strconv.Itoa(i%10)}
	}
	tasks, _ := groupAmazonPriceBatchItems(items)
	job := &amazonPriceBatchJob{ID: "job-1", Status: "running", Total: len(items), Asins: len(tasks), Failures: []amazonPriceBatchFailure{}, StartedAt: time.Now()}
	ctx, cancel := context.WithCancel(context.Background())
	job.cancel = cancel
	finished := make(chan struct{})
	go func() {
		job.run(ctx, server.Client(), server.URL, "key", tasks, 1)
		close(finished)
	}()
	polling := make(chan struct{})
	go func() {
		for i := 0; i < 300; i++ {
			job.snapshot()
			time.Sleep(time.Millisecond)
		}
		close(polling)
	}()
	time.Sleep(500 * time.Millisecond)
	cancel()
	select {
	case <-finished:
	case <-time.After(15 * time.Second):
		t.Fatal("cancelling must stop the job")
	}
	<-polling
	snapshot := job.snapshot()
	if snapshot.Status != "cancelled" {
		t.Fatalf("expected cancelled, got %s", snapshot.Status)
	}
	if snapshot.Done == 0 || snapshot.Done >= len(items) {
		t.Fatalf("cancelling must leave part of the %d products unprocessed, got %d done", len(items), snapshot.Done)
	}
	if len(snapshot.Failures) != snapshot.Done {
		t.Fatalf("every processed product must be reported, %d done vs %d results", snapshot.Done, len(snapshot.Failures))
	}
	if snapshot.FinishedAt == nil {
		t.Fatal("a finished job must record its finish time")
	}
}
