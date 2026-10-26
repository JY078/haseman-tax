package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeHomeDepotInternetNumber(t *testing.T) {
	if value, err := normalizeHomeDepotInternetNumber(" 322067088 "); err != nil || value != "322067088" {
		t.Fatalf("unexpected normalized value: %q, %v", value, err)
	}
	for _, value := range []string{"", "322-067-088", "abc", "12345", "1234567890123"} {
		if _, err := normalizeHomeDepotInternetNumber(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

func TestNewFirecrawlHTTPClientSupportsConfiguredProxy(t *testing.T) {
	client, err := newFirecrawlHTTPClient("10.10.65.24:10809")
	if err != nil || client == nil || client.Transport == nil {
		t.Fatalf("expected HTTP client with configured proxy, got client=%v err=%v", client, err)
	}
	if _, err := newFirecrawlHTTPClient("ftp://10.10.65.24:10809"); err == nil {
		t.Fatal("expected unsupported proxy scheme to be rejected")
	}
}

func TestFirecrawlEmptyProxyMeansDirect(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://unreachable-proxy.invalid:10809")
	client, err := newFirecrawlHTTPClient("")
	if err != nil {
		t.Fatal(err)
	}
	transport := client.Transport.(*http.Transport)
	if transport.Proxy != nil {
		t.Fatal("empty setting must not inherit environment proxy")
	}
	if transport.DialContext == nil {
		t.Fatal("connection timeout must be bounded")
	}
}

func TestHomeDepotTitleForEvaluationRemovesPlatformBrandPrefix(t *testing.T) {
	listing := homeDepotListingSource{Brand: "Forious pro", ProductTitle: "FORIOUS PRO Single Handle Kitchen Faucet in Matte Black"}
	if got := homeDepotTitleForEvaluation(listing); got != "Single Handle Kitchen Faucet in Matte Black" {
		t.Fatalf("unexpected evaluation title: %q", got)
	}
}

func TestFetchHomeDepotListingFetchesRawPageContent(t *testing.T) {
	requestProblems := make(chan string, 1)
	recordProblem := func(format string, args ...interface{}) {
		select {
		case requestProblems <- fmt.Sprintf(format, args...):
		default:
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scrape" {
			recordProblem("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			recordProblem("unexpected authorization header: %q", got)
		}
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			recordProblem("request decode failed: %v", err)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if payload["url"] != "https://www.homedepot.com/p/322067088" {
			recordProblem("unexpected product URL: %#v", payload["url"])
		}
		formats, ok := payload["formats"].([]interface{})
		if !ok || len(formats) != 2 || formats[0] != "markdown" || formats[1] != "rawHtml" {
			recordProblem("expected markdown and raw HTML formats, got %#v", payload["formats"])
		}
		if _, exists := payload["actions"]; exists {
			recordProblem("region-restricted Firecrawl actions must not be sent")
		}
		if payload["onlyMainContent"] != false {
			recordProblem("full page content must be retained, got onlyMainContent=%#v", payload["onlyMainContent"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"markdown":"# Kitchen Faucet\n- 3 spray settings\n- 360 degree swivel\nInternet # 322067088\nModel # HH0024BCH\n## Product Details\n### About This Product\nA kitchen faucet for everyday washing.\n### Highlights\n- Pull-down spray wand provides flexible reach\n- Single handle supports easy flow control\n## Specifications\n- Material: Stainless Steel","metadata":{"title":"Kitchen Faucet","description":"SEO text must not be used."}}}`))
	}))
	defer server.Close()

	listing, err := fetchHomeDepotListingWithClient(context.Background(), server.Client(), server.URL, "test-key", "322067088")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case problem := <-requestProblems:
		t.Fatal(problem)
	default:
	}
	if listing.ProductTitle != "Kitchen Faucet" || len(listing.Highlights) != 2 || len(listing.BulletPoints) != 2 || listing.ProductOverview != "A kitchen faucet for everyday washing." || len(listing.Specifications) != 0 {
		t.Fatalf("unexpected normalized listing: %+v", listing)
	}
	if listing.CanonicalURL != "https://www.homedepot.com/p/322067088" {
		t.Fatalf("canonical URL must be derived from the requested Internet Number: %s", listing.CanonicalURL)
	}
}

func TestFetchHomeDepotListingMergesVisibleDocumentFormats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"markdown":"# Kitchen Faucet\n- Markdown salient bullet\n## Product Details\n### About This Product\nMarkdown overview\n### Highlights\n- Markdown detail bullet","rawHtml":"<h2>Product Details</h2><h3>About This Product</h3><p>HTML overview</p><h3>Highlights</h3><ul><li>HTML detail bullet</li></ul>"}}`))
	}))
	defer server.Close()

	listing, err := fetchHomeDepotListingWithClient(context.Background(), server.Client(), server.URL, "test-key", "322067088")
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Highlights) != 1 {
		t.Fatalf("expected salient bullets only from the visible title area, got %#v", listing.Highlights)
	}
	if len(listing.BulletPoints) != 2 {
		t.Fatalf("expected Product Details bullets from visible document formats, got %#v", listing.BulletPoints)
	}
	if len(listing.Specifications) != 0 {
		t.Fatalf("specifications must be excluded from fetched listing, got %#v", listing.Specifications)
	}
}

func TestFetchHomeDepotListingLoadsCollapsedProductDetailsFromProductData(t *testing.T) {
	requestPaths := make([]string, 0, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPaths = append(requestPaths, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/scrape":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode scrape request: %v", err)
			}
			actions, _ := payload["actions"].([]interface{})
			if len(actions) == 0 {
				_, _ = w.Write([]byte(`{"success":true,"data":{"markdown":"# Kitchen Faucet\n- Salient one\n- Salient two\n- Salient three"}}`))
				return
			}
			encodedPayload, _ := json.Marshal(payload)
			if !strings.Contains(string(encodedPayload), `data-testid=\"view-more\"`) || !strings.Contains(string(encodedPayload), "product-section-overview") {
				t.Errorf("details action must click View More Details and expand Product Details: %s", encodedPayload)
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"actions":{"javascriptReturns":[{"type":"string","value":"{\"data\":{\"product\":{\"identifiers\":{\"itemId\":\"321844381\"},\"details\":{\"description\":\"Actual About This Product copy.\",\"highlights\":[\"Actual detail bullet one\",\"Actual detail bullet two\"]}}}}"}]}}}`))
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	listing, err := fetchHomeDepotListingWithClient(context.Background(), server.Client(), server.URL, "test-key", "321844381")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(requestPaths, ","); got != "POST /scrape,POST /scrape" {
		t.Fatalf("unexpected Firecrawl request sequence: %s", got)
	}
	if listing.ProductOverview != "Actual About This Product copy." || len(listing.BulletPoints) != 2 || len(listing.Highlights) != 3 {
		t.Fatalf("unexpected collapsed details mapping: %+v", listing)
	}
}

func TestParseHomeDepotListingDocumentKeepsAllSections(t *testing.T) {
	listing := parseHomeDepotListingDocument(`# Kitchen Faucet
- Salient one
- Salient two
- Salient three
## Product Details
### About This Product
A durable faucet designed for everyday use.
### Highlights
- Ceramic cartridges
- Stainless steel
- Braided supply lines
- Lever handle
- Pop-up drain kit included
- 1.5 GPM flow rate
- Easy to install
- Return Policy
## Specifications
| Color/Finish | Black |
| Faucet Hole Fit | 3 Hole |
| Installation Type | 8 in. Widespread |
## Details
Color Family: Black
Included Components: All Mounting Hardware, Drain
## Warranty / Certifications
Certifications and Listings: cUPC`, "", "Kitchen Faucet", "", "322067088")
	if len(listing.Highlights) != 3 || len(listing.BulletPoints) != 7 {
		t.Fatalf("expected three salient and seven detail bullets, got salient=%#v bullets=%#v", listing.Highlights, listing.BulletPoints)
	}
	if len(listing.Specifications) != 6 {
		t.Fatalf("expected specifications and details, got %d: %#v", len(listing.Specifications), listing.Specifications)
	}
	if listing.ProductOverview != "A durable faucet designed for everyday use." {
		t.Fatalf("expected visible Product Overview to replace metadata, got %q", listing.ProductOverview)
	}
}

func TestParseHomeDepotSpecificationHeadingValuePairs(t *testing.T) {
	listing := parseHomeDepotListingDocument("## Specifications\n### Number of Faucet Handles\n2 Handle\nSee Similar Items\n### Finish Family\nMatte\n### Flow rate (gallons per minute)\n1.5", "", "Faucet", "", "322067088")
	if len(listing.Specifications) != 3 || listing.Specifications[0].Value != "2 Handle" || listing.Specifications[2].Name != "Flow rate (gallons per minute)" {
		t.Fatalf("unexpected heading/value specifications: %#v", listing.Specifications)
	}
}

func TestFetchHomeDepotListingUsesRequestedInternetNumber(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"markdown":"# Kitchen Faucet\n- Salient feature\nInternet # 999999999\n## Product Details\n### About This Product\nA faucet.\n### Highlights\n- Detail feature"}}`))
	}))
	defer server.Close()

	listing, err := fetchHomeDepotListingWithClient(context.Background(), server.Client(), server.URL, "test-key", "322067088")
	if err != nil {
		t.Fatal(err)
	}
	if listing.InternetNumber != "322067088" || listing.CanonicalURL != "https://www.homedepot.com/p/322067088" {
		t.Fatalf("requested Internet Number must remain canonical: %+v", listing)
	}
}

func TestFetchHomeDepotListingAllowsMissingSpecifications(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"markdown":"# Kitchen Faucet\n- Salient feature\n## Product Details\n### About This Product\nA faucet.\n### Highlights\n- Detail feature"}}`))
	}))
	defer server.Close()

	_, err := fetchHomeDepotListingWithClient(context.Background(), server.Client(), server.URL, "test-key", "322067088")
	if err != nil {
		t.Fatalf("specifications are optional for optimization, got %v", err)
	}
}

func TestFetchHomeDepotListingRejectsMissingVisibleProductDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"markdown":"# Kitchen Faucet\n## Specifications\n- Material: Stainless Steel","metadata":{"description":"SEO description is not visible Product Details."}}}`))
	}))
	defer server.Close()

	_, err := fetchHomeDepotListingWithClient(context.Background(), server.Client(), server.URL, "test-key", "322067088")
	if err == nil || !strings.Contains(err.Error(), "home_depot_listing_content_missing") {
		t.Fatalf("expected missing listing content error, got %v", err)
	}
}

func TestHomeDepotListingForEvaluationKeepsCopyOutOfFacts(t *testing.T) {
	listing := homeDepotListingSource{
		InternetNumber:  "322067088",
		CanonicalURL:    "https://www.homedepot.com/p/322067088",
		Brand:           "Forious pro",
		ModelNumber:     "HH0024BCH",
		ProductTitle:    "Kitchen Faucet",
		ProductOverview: "Unverified marketing sentence",
		Highlights:      []string{"Unverified marketing highlight"},
		Specifications:  []homeDepotListingSpecification{{Name: "Material", Value: "Stainless Steel"}},
	}
	facts, _, existing := homeDepotListingForEvaluation(listing)
	if strings.Contains(facts, "Unverified marketing") {
		t.Fatalf("marketing copy must not be treated as product facts: %s", facts)
	}
	if !strings.Contains(facts, "Brand: Forious pro") || !strings.Contains(existing, "Unverified marketing sentence") {
		t.Fatalf("unexpected evaluation input\nfacts=%s\nlisting=%s", facts, existing)
	}
}
