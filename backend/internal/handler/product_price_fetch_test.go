package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func init() {
	// Production waits seconds between attempts; tests only assert the count.
	amazonPriceRetryBase = 10 * time.Millisecond
}

func TestAmazonPriceNetworkFailureReasons(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "https://api.firecrawl.dev/v2/scrape", nil)
	for _, tc := range []struct {
		name, proxy, want string
		cause             error
	}{
		{"proxy dial", "http://user:secret@127.0.0.1:10809", "代理连接失败", &net.OpError{Op: "dial", Err: errors.New("connection refused secret")}},
		{"direct DNS", "", "域名解析失败", &net.DNSError{Err: "no such host", Name: "api.firecrawl.dev"}},
		{"direct timeout", "", "等待 Firecrawl 响应超时", context.DeadlineExceeded},
		{"proxy timeout", "http://127.0.0.1:10809", "通过代理等待", context.DeadlineExceeded},
		{"direct connection", "", "检查服务器网络", &net.OpError{Op: "dial", Err: errors.New("connection refused")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, err := newFirecrawlHTTPClient(tc.proxy)
			if err != nil {
				t.Fatal(err)
			}
			got := amazonPriceNetworkError(context.Background(), client, req, &url.Error{Op: "Post", URL: req.URL.String(), Err: tc.cause})
			var connectionErr *amazonPriceConnectionError
			if !errors.As(got, &connectionErr) || !strings.Contains(got.Error(), tc.want) {
				t.Fatalf("unexpected diagnostic: %v", got)
			}
			if strings.Contains(got.Error(), "secret") {
				t.Fatal("credentials leaked")
			}
		})
	}
}

func TestAmazonPriceUnreachableProxy(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	client, err := newFirecrawlHTTPClient("http://" + address)
	if err != nil {
		t.Fatal(err)
	}
	// Retries keep the proxy diagnostic, so keep the wait short in tests.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err = fetchAmazonPrice(ctx, client, "https://api.firecrawl.dev/v2", "test-key", "B012345678")
	var connectionErr *amazonPriceConnectionError
	if !errors.As(err, &connectionErr) || !strings.Contains(err.Error(), "代理连接失败") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAmazonPriceExtraction(t *testing.T) {
	ptr := func(s string) *string { return &s }
	for _, tc := range []struct {
		name        string
		value       amazonPriceExtraction
		valid       bool
		price, sale string
	}{
		{"single", amazonPriceExtraction{"B012345678", "USD", ptr("39.99"), nil, "Price $39.99"}, true, "39.99", ""},
		{"discount", amazonPriceExtraction{"B012345678", "USD", ptr("29.99"), ptr("39.99"), "List $39.99 Now $29.99"}, true, "39.99", "29.99"},
		{"unavailable", amazonPriceExtraction{"B012345678", "USD", nil, nil, "Unavailable"}, false, "", ""},
		{"wrong variant", amazonPriceExtraction{"B098765432", "USD", ptr("39.99"), nil, "Price $39.99"}, false, "", ""},
		{"wrong currency", amazonPriceExtraction{"B012345678", "CAD", ptr("39.99"), nil, "Price $39.99"}, false, "", ""},
		{"reversed labels", amazonPriceExtraction{"B012345678", "USD", ptr("49.99"), ptr("39.99"), "List $39.99 Now $49.99"}, true, "49.99", "39.99"},
		{"zero", amazonPriceExtraction{"B012345678", "USD", ptr("0"), nil, "Price $0.00"}, false, "", ""},
		{"price without matching evidence text", amazonPriceExtraction{"B012345678", "USD", ptr("19.99"), nil, "Price unavailable"}, true, "19.99", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateAmazonExtraction(tc.value, "B012345678", tc.value.Evidence)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
			if tc.valid {
				if *got.Price != tc.price {
					t.Fatal(got)
				}
				if tc.sale != "" && (got.SalePrice == nil || *got.SalePrice != tc.sale) {
					t.Fatal(got)
				}
				if tc.sale == "" && got.SalePrice != nil {
					t.Fatal("stale discount must clear")
				}
			}
		})
	}
	if _, err := validateAmazonExtraction(amazonPriceExtraction{"B098765432", "USD", ptr("39.99"), nil, "Price $39.99"}, "B012345678", "unrelated page"); err == nil {
		t.Fatal("other ASIN accepted")
	}
}

func TestAmazonPriceExtractionAcceptsNumericAndBlankListPrice(t *testing.T) {
	var value amazonPriceExtraction
	if err := json.Unmarshal([]byte(`{"asin":"B012345678","currency":"USD","current_price":97.49,"list_price":"","evidence":""}`), &value); err != nil {
		t.Fatal(err)
	}
	got, err := validateAmazonExtraction(value, "B012345678", "")
	if err != nil || got.Price == nil || *got.Price != "97.49" || got.SalePrice != nil {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestAmazonHTMLPriceUsesMainOffer(t *testing.T) {
	html := `<input id="ASIN" value="B012345678"><div id="corePriceDisplay_desktop_feature_div"><span class="a-price"><span class="a-offscreen">$29.99</span></span><span class="a-price a-text-price"><span class="a-offscreen">$39.99</span></span></div><div id="rh">$9.99</div>`
	got, err := amazonHTMLPrice(html, "B012345678")
	if err != nil || got.Price == nil || *got.Price != "39.99" || got.SalePrice == nil || *got.SalePrice != "29.99" {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestAmazonHTMLPriceRejectsOtherASIN(t *testing.T) {
	html := `<input id="ASIN" value="B098765432"><div id="corePriceDisplay_desktop_feature_div"><span class="a-price"><span class="a-offscreen">$29.99</span></span></div>`
	if _, err := amazonHTMLPrice(html, "B012345678"); err == nil {
		t.Fatal("other ASIN must be rejected")
	}
}

func TestAmazonHTMLPriceUsesCurrentASINOffersFallback(t *testing.T) {
	html := `<input id="ASIN" value="B012345678"><div id="corePriceDisplay_desktop_feature_div"></div><a id="aod-ingress-link"><span>Used (2) from</span><span class="a-price"><span class="a-offscreen">$97.49</span></span></a><div class="recommendation"><span class="a-offscreen">$199.99</span></div>`
	got, err := amazonHTMLPrice(html, "B012345678")
	if err != nil || got.Price == nil || *got.Price != "97.49" || got.SalePrice != nil {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestAmazonHTMLPriceUnavailablePrefersCurrentASINOfferOverBuyBoxNoise(t *testing.T) {
	html := `<input id="ASIN" value="B012345678"><div id="priceNotAvailable_feature_div">To see product details, add this item to your cart.</div><div id="corePriceDisplay_desktop_feature_div"><span class="a-price"><span class="a-offscreen">$139.99</span></span><span class="a-text-price"><span class="a-offscreen">$199.99</span></span></div><a id="aod-ingress-link"><span>Used (2) from</span><span class="a-price"><span class="a-offscreen">$97.49</span></span></a>`
	got, err := amazonHTMLPrice(html, "B012345678")
	if err != nil || got.Price == nil || *got.Price != "97.49" || got.SalePrice != nil {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestAmazonHTMLPriceCleanHTMLUsesASINScopedPrice(t *testing.T) {
	html := `<div class="alternative-items"><span class="a-offscreen">$139.99</span></div><section data-csa-c-asin="B012345678"><h1>Product title</h1><span class="a-price"><span class="a-offscreen">$97.49</span></span></section>`
	got, err := amazonHTMLPrice(html, "B012345678")
	if err != nil || got.Price == nil || *got.Price != "97.49" {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestAmazonHTMLPriceSupportsBuyBoxPriceblock(t *testing.T) {
	html := `<div id="alternative-items"><span class="a-offscreen">$139.99</span></div><div id="buyBox"><h1>Product title</h1><span id="priceblock_dealprice">$97.49</span></div>`
	got, err := amazonHTMLPrice(html, "B012345678")
	if err != nil || got.Price == nil || *got.Price != "97.49" {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestAmazonHTMLPricePrefersNewOfferRowOverUsedRow(t *testing.T) {
	html := `<input id="ASIN" value="B0DWK6K6L5">
	<div id="apex_desktop">
		<div id="apex_desktop_newAccordionRow"><div id="corePriceDisplay_desktop_feature_div"><span class="a-price"><span class="a-offscreen">$49.78</span></span><span class="a-price a-text-price"><span class="a-offscreen">$55.99</span></span></div></div>
		<div id="apex_desktop_usedAccordionRow"><div id="corePriceDisplay_desktop_feature_div"><span class="a-price"><span class="a-offscreen">$38.99</span></span><span class="a-price a-text-price"><span class="a-offscreen">$55.99</span></span></div></div>
	</div>
	<div id="accordionRows">
		<div id="newAccordionRow_0"><div id="corePrice_feature_div"><span class="a-price"><span class="a-offscreen">$49.78</span></span></div></div>
		<div id="usedAccordionRow"><div id="corePrice_feature_div"><span class="a-price"><span class="a-offscreen">$38.99</span></span></div></div>
	</div>`
	got, err := amazonHTMLPrice(html, "B0DWK6K6L5")
	if err != nil || got.Price == nil || *got.Price != "55.99" || got.SalePrice == nil || *got.SalePrice != "49.78" {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestAmazonHTMLPriceIgnoresUsedRowAndVariantSwatches(t *testing.T) {
	html := `<div data-csa-c-asin="B0G7H6Q7BZ">
	<div id="apex_desktop">
		<div id="apex_desktop_newAccordionRow"><div id="corePriceDisplay_desktop_feature_div"><span class="a-price"><span class="a-offscreen">$66.99</span></span></div></div>
		<div id="apex_desktop_usedAccordionRow"><div id="corePriceDisplay_desktop_feature_div"><span class="a-price"><span class="a-offscreen">$46.89</span></span><span class="a-price a-text-price"><span class="a-offscreen">$66.99</span></span></div></div>
	</div>
	<div id="apex_price" class="apex_on_twister_price"><span class="a-price a-text-price"><span class="a-offscreen">$139.99</span></span></div>
	</div>`
	got, err := amazonHTMLPrice(html, "B0G7H6Q7BZ")
	if err != nil || got.Price == nil || *got.Price != "66.99" || got.SalePrice != nil {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestAmazonHTMLPriceReportsOutOfStockInsteadOfRecommendationPrices(t *testing.T) {
	html := `<input id="ASIN" value="B0B4G46RC3">
	<div id="availability_feature_div"><span>Currently unavailable.</span> We don't know when or if this item will be back in stock.</div>
	<div id="CardInstanceX" class="_Y2VyY_mainCardContainer cerberusSharedCards-widgetTypeOos">
		<div class="_Y2VyY_afFixedGridContentWithAtcBtn_3p"><span class="a-price"><span class="a-offscreen">$57.79</span></span></div>
	</div>
	<div class="p13n-asin"><span class="a-price"><span class="a-offscreen">$39.99</span></span></div>`
	if _, err := amazonHTMLPrice(html, "B0B4G46RC3"); !errors.Is(err, errAmazonPriceUnavailable) {
		t.Fatalf("out of stock must be reported, got %v", err)
	}
}

func TestAmazonHTMLPriceIgnoresOutOfStockWidgetPrices(t *testing.T) {
	// No availability block at all: the shared-card recommendations shown on
	// out-of-stock pages must still never become the product price.
	html := `<div data-csa-c-asin="B0B4G46RC3">
	<div id="unifiedPrice_feature_div"></div>
	<div class="_Y2VyY_afFixedGridContentWithAtcBtn_3p"><span class="a-price"><span class="a-offscreen">$57.79</span></span></div>
	</div>`
	got, err := amazonHTMLPrice(html, "B0B4G46RC3")
	if err == nil || got.Price != nil {
		t.Fatalf("recommendation widget price must not be used, got=%v err=%v", got, err)
	}
}

func TestAmazonHTMLPriceKeepsPriceWhenAvailabilityWidgetIsStale(t *testing.T) {
	// A variation page can carry an unavailable sibling while the selected
	// ASIN still has an offer, so a real buy-box price must win.
	html := `<input id="ASIN" value="B012345678">
	<div id="availability_feature_div">Currently unavailable.</div>
	<div id="apex_desktop"><div id="apex_desktop_newAccordionRow"><div id="corePriceDisplay_desktop_feature_div"><span class="a-price"><span class="a-offscreen">$29.99</span></span></div></div></div>`
	got, err := amazonHTMLPrice(html, "B012345678")
	if err != nil || got.Price == nil || *got.Price != "29.99" {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestAmazonHTMLPriceIgnoresUnitPrice(t *testing.T) {
	// "$13.63 per count" is a derived rate printed beside the real price, not
	// a discount. Its span carries both a-price and a-text-price, so without
	// the check it lands in the list-price set and becomes the sale price.
	html := `<input id="ASIN" value="B08CSBQ9G2">
	<div id="corePrice_feature_div"><div class="a-section apex-core-price-identifier">
		<span class="a-price aok-align-center apex-pricetopay-value"><span class="a-offscreen">$26.99</span></span>
		<span class="a-size-mini a-color-base aok-align-center"><span class="a-price a-text-price apex-priceperunit-value"><span class="a-offscreen">$13.63</span></span></span>
		<span class="a-size-mini aok-offscreen apex-priceperunit-accessibility-label">$13.63 per count</span>
	</div></div>`
	got, err := amazonHTMLPrice(html, "B08CSBQ9G2")
	if err != nil || got.Price == nil || *got.Price != "26.99" || got.SalePrice != nil {
		t.Fatalf("unit price must not become a discount, got price=%v sale=%v err=%v", got.Price, got.SalePrice, err)
	}
}

func TestAmazonHTMLPriceKeepsRealDiscountBesideUnitPrice(t *testing.T) {
	// Skipping the unit price must not cost the real discount: the list price
	// and the pay price still describe a genuine deal.
	html := `<input id="ASIN" value="B012345678">
	<div id="corePrice_feature_div"><div class="a-section apex-core-price-identifier">
		<span class="a-price aok-align-center apex-pricetopay-value"><span class="a-offscreen">$19.99</span></span>
		<span class="a-price a-text-price"><span class="a-offscreen">$26.99</span></span>
		<span class="a-size-mini a-color-base aok-align-center"><span class="a-price a-text-price apex-priceperunit-value"><span class="a-offscreen">$9.99</span></span></span>
	</div></div>`
	got, err := amazonHTMLPrice(html, "B012345678")
	if err != nil || got.Price == nil || *got.Price != "26.99" || got.SalePrice == nil || *got.SalePrice != "19.99" {
		t.Fatalf("real discount must survive, got price=%v sale=%v err=%v", got.Price, got.SalePrice, err)
	}
}

// A listing with labelled accordion rows: the row captions name the prices, so
// they are read instead of whatever struck-through amount sits nearby.
func TestAmazonHTMLPriceReadsRegularAndPrimeRows(t *testing.T) {
	html := `<input id="ASIN" value="B0CJFG5W4B">
	<h5><div class="a-column a-span12 accordion-caption"><div id="newAccordionCaption_feature_div"><span class="a-text-bold">Regular Price</span></div></div></h5>
	<div class="a-accordion-inner"><div id="corePrice_feature_div"><span class="a-price apex-pricetopay-value"><span class="a-offscreen">$75.99</span></span></div></div>
	<h5><div id="primeSavingsUpsellCaption_feature_div"><span class="a-size-base a-text-bold">Prime Member Price</span></div></h5>
	<div class="a-accordion-inner"><div id="corePrice_feature_div"><span class="a-price apex-pricetopay-value"><span class="a-offscreen">$71.99</span></span></div></div>`
	got, err := amazonHTMLPrice(html, "B0CJFG5W4B")
	if err != nil || got.Price == nil || *got.Price != "75.99" || got.SalePrice == nil || *got.SalePrice != "71.99" {
		t.Fatalf("got price=%v prime=%v err=%v", got.Price, got.SalePrice, err)
	}
	if got.BestDeal != nil {
		t.Fatalf("best deal must stay empty without a deal badge, got %v", *got.BestDeal)
	}
}

func TestAmazonHTMLPriceReadsDealPriceRow(t *testing.T) {
	html := `<input id="ASIN" value="B0F43WMKPK">
	<h5><div id="primeSavingsUpsellCaption_feature_div"><span class="a-size-base a-text-bold">Deal price</span></div></h5>
	<div class="a-accordion-inner"><div id="corePrice_feature_div"><span class="a-price apex-pricetopay-value"><span class="a-offscreen">$55.99</span></span></div></div>
	<h5><div id="newAccordionCaption_feature_div"><span class="a-text-bold">Regular Price</span></div></h5>
	<div class="a-accordion-inner"><div id="corePrice_feature_div"><span class="a-price apex-pricetopay-value"><span class="a-offscreen">$69.99</span></span></div></div>`
	got, err := amazonHTMLPrice(html, "B0F43WMKPK")
	if err != nil || got.Price == nil || *got.Price != "69.99" || got.SalePrice != nil || got.BestDeal == nil || *got.BestDeal != "55.99" {
		t.Fatalf("got price=%v prime=%v deal=%v err=%v", got.Price, got.SalePrice, got.BestDeal, err)
	}
}

// While a deal runs, the Buy New row holds the deal price and no regular price
// is stored, because the only other amount is the struck-through list price.
func TestAmazonHTMLPriceDealBadgeUsesBuyNewPrice(t *testing.T) {
	html := `<input id="ASIN" value="B07X395JXY">
	<div id="apex_desktop_newAccordionRow">
		<div id="dealBadge_feature_div"><span class="dealBadge">Limited time deal</span></div>
		<h5><div class="a-column a-span12 accordion-caption"><div id="newAccordionCaption_feature_div"><span class="a-text-bold">Buy New</span></div></div></h5>
		<div class="a-accordion-inner"><div id="corePrice_feature_div"><span class="a-price apex-pricetopay-value"><span class="a-offscreen">$53.98</span></span>
		<span class="a-price a-text-price apex-basisprice-value"><span class="a-offscreen">$69.99</span></span></div></div>
	</div>`
	got, err := amazonHTMLPrice(html, "B07X395JXY")
	if err != nil || got.BestDeal == nil || *got.BestDeal != "53.98" {
		t.Fatalf("deal price must come from the Buy New row, got bestDeal=%v err=%v", got.BestDeal, err)
	}
	if got.Price != nil || got.SalePrice != nil {
		t.Fatalf("a running deal leaves the list price out, got price=%v prime=%v", got.Price, got.SalePrice)
	}
}

// The deal badge also appears on the used row, where it is empty; that must not
// turn a normal listing into a deal.
func TestAmazonHTMLPriceIgnoresEmptyDealBadge(t *testing.T) {
	html := `<input id="ASIN" value="B012345678">
	<div id="apex_desktop_usedAccordionRow"><div id="dealBadge_feature_div"></div></div>
	<div id="corePrice_feature_div"><span class="a-price"><span class="a-offscreen">$29.99</span></span></div>`
	got, err := amazonHTMLPrice(html, "B012345678")
	if err != nil || got.Price == nil || *got.Price != "29.99" || got.BestDeal != nil {
		t.Fatalf("an empty badge must not create a deal, got price=%v bestDeal=%v err=%v", got.Price, got.BestDeal, err)
	}
}

// "List Price: $69.99" is Amazon's struck-through reference, never the price a
// shopper pays, so it must not become the regular price.
func TestAmazonHTMLPriceIgnoresListPrice(t *testing.T) {
	html := `<input id="ASIN" value="B012345678">
	<div id="corePriceDisplay_desktop_feature_div">
		<span class="a-price apex-pricetopay-value"><span class="a-offscreen">$53.98</span></span>
		<span class="apex-basisprice-feature"><span class="a-price a-text-price apex-basisprice-value"><span class="a-offscreen">$69.99</span></span></span>
	</div>`
	got, err := amazonHTMLPrice(html, "B012345678")
	if err != nil || got.Price == nil || *got.Price != "53.98" {
		t.Fatalf("list price must be ignored, got price=%v sale=%v err=%v", got.Price, got.SalePrice, err)
	}
}

func TestAmazonPriceUnavailableIsNotRetried(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		body, _ := json.Marshal(map[string]interface{}{"success": true, "data": map[string]interface{}{
			"rawHtml": `<input id="ASIN" value="B012345678"><div id="availability">Currently unavailable.</div>`,
		}})
		w.Write(body)
	}))
	defer server.Close()
	if _, err := fetchAmazonPrice(context.Background(), server.Client(), server.URL, "key", "B012345678"); !errors.Is(err, errAmazonPriceUnavailable) {
		t.Fatalf("unexpected error: %v", err)
	}
	if requests != 1 {
		t.Fatalf("an out-of-stock page must not be refetched, got %d requests", requests)
	}
}

func TestAmazonMarkdownCannotSupplyUnscopedPrice(t *testing.T) {
	_, err := parseAmazonFirecrawlResponse(strings.NewReader(`{"success":true,"data":{"markdown":"Product title $10.00 Consider these alternative items $139.99"}}`), "B012345678")
	if err == nil {
		t.Fatal("unscoped markdown price must not be accepted")
	}
}

func TestFirecrawlAmazonRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" || r.URL.Path != "/scrape" {
			t.Error("incorrect Firecrawl request")
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["url"] != "https://www.amazon.com/dp/B012345678" || body["maxAge"] != float64(300000) {
			t.Error("must use fixed US URL and bounded cache")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true,"data":{"markdown":"Price $39.99","rawHtml":"<input id=\"ASIN\" value=\"B012345678\"><div id=\"corePriceDisplay_desktop_feature_div\"><span class=\"a-price\"><span class=\"a-offscreen\">$39.99</span></span></div>"}}}`))
	}))
	defer server.Close()
	got, err := fetchAmazonPrice(context.Background(), server.Client(), server.URL, "test-key", "B012345678")
	if err != nil || got.Price == nil || *got.Price != "39.99" {
		t.Fatalf("got=%v error=%v", got, err)
	}
	if _, err = fetchAmazonPrice(context.Background(), server.Client(), server.URL, "test-key", "invalid/url"); err == nil {
		t.Fatal("invalid ASIN accepted")
	}
}

func TestFirecrawlAmazonFailures(t *testing.T) {
	for _, tc := range []struct {
		status, requests int
	}{
		{401, 1}, {402, 1},
		{429, amazonPriceFetchAttempts}, {500, amazonPriceFetchAttempts},
		{200, amazonPriceFetchAttempts},
	} {
		var requests int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests++
			w.WriteHeader(tc.status)
			w.Write([]byte(`{"success":false,"error":"private upstream details"}`))
		}))
		_, err := fetchAmazonPrice(context.Background(), server.Client(), server.URL, "key", "B012345678")
		server.Close()
		if err == nil {
			t.Fatalf("status %d must fail", tc.status)
		}
		if requests != tc.requests {
			t.Fatalf("status %d requested %d times, want %d", tc.status, requests, tc.requests)
		}
	}
}

func TestAmazonPriceRetryStopsWhenThePageHasNoPrice(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Write([]byte(`{"success":true,"data":{"rawHtml":"<input id=\"ASIN\" value=\"B012345678\">"}}`))
	}))
	defer server.Close()
	if _, err := fetchAmazonPrice(context.Background(), server.Client(), server.URL, "key", "B012345678"); !errors.Is(err, errAmazonPriceNotFound) {
		t.Fatalf("unexpected error: %v", err)
	}
	if requests != 1 {
		t.Fatalf("a page without a price must not be refetched, got %d requests", requests)
	}
}

func TestAmazonPriceRetryRecoversAfterRateLimit(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"success":true,"data":{"rawHtml":"<input id=\"ASIN\" value=\"B012345678\"><div id=\"corePriceDisplay_desktop_feature_div\"><span class=\"a-price\"><span class=\"a-offscreen\">$39.99</span></span></div>"}}`))
	}))
	defer server.Close()
	got, err := fetchAmazonPrice(context.Background(), server.Client(), server.URL, "key", "B012345678")
	if err != nil || got.Price == nil || *got.Price != "39.99" {
		t.Fatalf("got=%v err=%v", got, err)
	}
	if requests != 2 {
		t.Fatalf("want one retry after the rate limit, got %d requests", requests)
	}
}

func TestAmazonPriceRateLimitOutlastsRetries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	_, err := fetchAmazonPrice(context.Background(), server.Client(), server.URL, "key", "B012345678")
	if !errors.Is(err, errAmazonFirecrawlRateLimited) {
		t.Fatalf("rate limit must stay identifiable after retries: %v", err)
	}
}

func TestAmazonRetryAfterHeader(t *testing.T) {
	for _, tc := range []struct {
		header string
		want   time.Duration
	}{
		{"", 0},
		{"5", 5 * time.Second},
		{" 12 ", 12 * time.Second},
		{"0", 0},
		{"soon", 0},
	} {
		if got := amazonRetryAfter(tc.header); got != tc.want {
			t.Fatalf("header %q => %v, want %v", tc.header, got, tc.want)
		}
	}
}

func TestAmazonPriceScrapeTimeoutsCoverQueueWait(t *testing.T) {
	if amazonPriceAttemptTimeout <= amazonPriceScrapeTimeoutMS*time.Millisecond {
		t.Fatal("one attempt must outlast the scrape timeout Firecrawl is given")
	}
	if amazonPriceFetchBudget <= amazonPriceAttemptTimeout {
		t.Fatal("the fetch budget must leave room for more than one attempt")
	}
}

func TestAmazonResponseHTMLSurvivesMalformedExtraction(t *testing.T) {
	page := `<input id="ASIN" value="B012345678"><div id="corePriceDisplay_desktop_feature_div"><span class="a-price"><span class="a-offscreen">$39.99</span></span></div>`
	for _, extraction := range []string{`{"current_price":{"amount":39.99}}`, `[]`, `"invalid"`, `null`} {
		body, err := json.Marshal(map[string]interface{}{"success": true, "data": map[string]interface{}{"rawHtml": page}})
		if err != nil {
			t.Fatal(err)
		}
		got, err := parseAmazonFirecrawlResponse(strings.NewReader(string(body)), "B012345678")
		if err != nil || got.Price == nil || *got.Price != "39.99" || got.SalePrice != nil {
			t.Fatalf("extraction=%s got=%v err=%v", extraction, got, err)
		}
	}
}

func TestAmazonResponseFailureDiagnostics(t *testing.T) {
	for _, tc := range []struct{ body, want string }{
		{"", "空响应"},
		{`{"success":true`, "传输不完整"},
		{`<html>gateway error</html>`, "响应格式异常"},
		{`{"success":false}`, "报告抓取失败"},
		{`{"success":true,"data":{"rawHtml":""}}`, "返回内容为空"},
	} {
		_, err := parseAmazonFirecrawlResponse(strings.NewReader(tc.body), "B012345678")
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("body=%s error=%v want=%s", tc.body, err, tc.want)
		}
	}
}
