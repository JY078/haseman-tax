package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	stdhtml "html"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const defaultFirecrawlAPIBase = "https://api.firecrawl.dev/v2"

var homeDepotInternetNumberPattern = regexp.MustCompile(`^\d{6,12}$`)

type homeDepotListingSpecification struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type homeDepotListingSource struct {
	InternetNumber  string                          `json:"internet_number"`
	CanonicalURL    string                          `json:"canonical_url"`
	Brand           string                          `json:"brand"`
	ModelNumber     string                          `json:"model_number"`
	ProductTitle    string                          `json:"product_title"`
	ProductCategory string                          `json:"product_category"`
	ProductOverview string                          `json:"product_overview"`
	BulletPoints    []string                        `json:"bullet_points"`
	Highlights      []string                        `json:"highlights"`
	Specifications  []homeDepotListingSpecification `json:"specifications"`
	Breadcrumbs     []string                        `json:"breadcrumbs"`
}

func normalizeHomeDepotInternetNumber(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !homeDepotInternetNumberPattern.MatchString(value) {
		return "", fmt.Errorf("home_depot_internet_number_invalid")
	}
	return value, nil
}

func homeDepotProductURL(internetNumber string) string {
	return "https://www.homedepot.com/p/" + internetNumber
}

func fetchHomeDepotListing(ctx context.Context, internetNumber string) (homeDepotListingSource, error) {
	apiKey := strings.TrimSpace(getSettingValue("firecrawl_api_key"))
	if apiKey == "" {
		return homeDepotListingSource{}, fmt.Errorf("firecrawl_api_key_not_configured")
	}
	client, err := newFirecrawlHTTPClient(strings.TrimSpace(getSettingValue("firecrawl_proxy")))
	if err != nil {
		return homeDepotListingSource{}, err
	}
	return fetchHomeDepotListingWithClient(ctx, client, defaultFirecrawlAPIBase, apiKey, internetNumber)
}

func newFirecrawlHTTPClient(proxyAddress string) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// An empty setting explicitly means direct access, not environment proxies.
	transport.Proxy = nil
	transport.DialContext = (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	if proxyAddress != "" {
		if !strings.Contains(proxyAddress, "://") {
			proxyAddress = "http://" + proxyAddress
		}
		proxyURL, err := url.Parse(proxyAddress)
		if err != nil || proxyURL.Scheme == "" || proxyURL.Host == "" {
			return nil, fmt.Errorf("firecrawl_proxy_invalid")
		}
		if proxyURL.Scheme != "http" && proxyURL.Scheme != "https" && proxyURL.Scheme != "socks5" && proxyURL.Scheme != "socks5h" {
			return nil, fmt.Errorf("firecrawl_proxy_scheme_unsupported")
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return &http.Client{Timeout: 150 * time.Second, Transport: transport}, nil
}

func fetchHomeDepotListingWithClient(ctx context.Context, client *http.Client, apiBase, apiKey, internetNumber string) (homeDepotListingSource, error) {
	internetNumber, err := normalizeHomeDepotInternetNumber(internetNumber)
	if err != nil {
		return homeDepotListingSource{}, err
	}
	if client == nil {
		return homeDepotListingSource{}, fmt.Errorf("firecrawl_http_client_missing")
	}
	payload := map[string]interface{}{
		"url": homeDepotProductURL(internetNumber),
		"formats": []interface{}{
			"markdown",
			"rawHtml",
		},
		"onlyMainContent":    false,
		"waitFor":            7000,
		"timeout":            120000,
		"location":           map[string]interface{}{"country": "US", "languages": []string{"en-US"}},
		"removeBase64Images": true,
		"blockAds":           true,
		"proxy":              "auto",
		"storeInCache":       false,
	}
	var raw []byte
	var statusCode int
	for _, proxy := range []string{"auto", "enhanced"} {
		payload["proxy"] = proxy
		body, encodeErr := json.Marshal(payload)
		if encodeErr != nil {
			return homeDepotListingSource{}, fmt.Errorf("firecrawl_request_encode_failed: %w", encodeErr)
		}
		req, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(apiBase, "/")+"/scrape", bytes.NewReader(body))
		if requestErr != nil {
			return homeDepotListingSource{}, fmt.Errorf("firecrawl_request_create_failed: %w", requestErr)
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "hsm-home-depot-listing-evaluator/1.0")
		response, requestErr := client.Do(req)
		if requestErr != nil {
			return homeDepotListingSource{}, fmt.Errorf("firecrawl_request_failed: %w", requestErr)
		}
		statusCode = response.StatusCode
		raw, requestErr = io.ReadAll(io.LimitReader(response.Body, 32<<20))
		response.Body.Close()
		if requestErr != nil {
			return homeDepotListingSource{}, fmt.Errorf("firecrawl_response_read_failed: %w", requestErr)
		}
		if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
			break
		}
		if statusCode < http.StatusInternalServerError {
			return homeDepotListingSource{}, fmt.Errorf("firecrawl_http_%d: %s", statusCode, strings.TrimSpace(string(raw)))
		}
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return homeDepotListingSource{}, fmt.Errorf("firecrawl_http_%d: %s", statusCode, strings.TrimSpace(string(raw)))
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Markdown string `json:"markdown"`
			HTML     string `json:"html"`
			RawHTML  string `json:"rawHtml"`
			Metadata struct {
				Title       string `json:"title"`
				Description string `json:"description"`
				SourceURL   string `json:"sourceURL"`
			} `json:"metadata"`
		} `json:"data"`
		Error   string `json:"error"`
		Warning string `json:"warning"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return homeDepotListingSource{}, fmt.Errorf("firecrawl_response_invalid_json: %w", err)
	}
	if !envelope.Success {
		detail := strings.TrimSpace(envelope.Error)
		if detail == "" {
			detail = strings.TrimSpace(envelope.Warning)
		}
		return homeDepotListingSource{}, fmt.Errorf("firecrawl_listing_extract_failed: %s", detail)
	}
	documentListing := parseHomeDepotListingDocument(envelope.Data.Markdown, envelope.Data.RawHTML+"\n"+envelope.Data.HTML, envelope.Data.Metadata.Title, envelope.Data.Metadata.Description, internetNumber)
	listing := documentListing
	var detailErr error
	if listing.ProductOverview == "" || len(listing.BulletPoints) == 0 {
		var details homeDepotListingSource
		details, detailErr = fetchHomeDepotProductDetailsWithClient(ctx, client, apiBase, apiKey, internetNumber)
		if detailErr == nil {
			listing.ProductOverview = details.ProductOverview
			listing.BulletPoints = details.BulletPoints
		}
	}
	listing = normalizeHomeDepotListingSource(listing)
	// Specifications may be returned by a generic extractor, but are not part
	// of the HD listing generation/evaluation payload.
	listing.Specifications = nil
	// The requested Internet Number is the stable product key. Page content can
	// contain nearby product numbers, so it must not override the user's input.
	listing.InternetNumber = internetNumber
	listing.CanonicalURL = homeDepotProductURL(internetNumber)
	if listing.ProductTitle == "" {
		return homeDepotListingSource{}, fmt.Errorf("home_depot_listing_title_missing")
	}
	if listing.ProductOverview == "" && len(listing.BulletPoints) == 0 && len(listing.Highlights) == 0 {
		return homeDepotListingSource{}, fmt.Errorf("home_depot_listing_content_missing")
	}
	if listing.ProductOverview == "" || len(listing.BulletPoints) == 0 || len(listing.Highlights) == 0 {
		if detailErr != nil {
			return homeDepotListingSource{}, fmt.Errorf("home_depot_listing_product_details_missing: %w", detailErr)
		}
		return homeDepotListingSource{}, fmt.Errorf("home_depot_listing_product_details_missing")
	}
	return listing, nil
}

func fetchHomeDepotProductDetailsWithClient(ctx context.Context, client *http.Client, apiBase, apiKey, internetNumber string) (homeDepotListingSource, error) {
	payload := map[string]interface{}{
		"url":             homeDepotProductURL(internetNumber),
		"formats":         []string{"markdown", "rawHtml"},
		"onlyMainContent": false,
		"waitFor":         8000,
		"timeout":         120000,
		"location":        map[string]interface{}{"country": "US", "languages": []string{"en-US"}},
		"proxy":           "auto",
		"storeInCache":    false,
		"actions": []map[string]interface{}{
			{"type": "wait", "milliseconds": 4000},
			{"type": "executeJavascript", "script": homeDepotProductDetailsActionScript(internetNumber)},
		},
	}
	var raw []byte
	var statusCode int
	for _, proxy := range []string{"auto", "enhanced", "enhanced"} {
		payload["proxy"] = proxy
		body, encodeErr := json.Marshal(payload)
		if encodeErr != nil {
			return homeDepotListingSource{}, encodeErr
		}
		req, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(apiBase, "/")+"/scrape", bytes.NewReader(body))
		if requestErr != nil {
			return homeDepotListingSource{}, requestErr
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		response, requestErr := client.Do(req)
		if requestErr != nil {
			return homeDepotListingSource{}, requestErr
		}
		statusCode = response.StatusCode
		raw, requestErr = io.ReadAll(io.LimitReader(response.Body, 32<<20))
		response.Body.Close()
		if requestErr != nil {
			return homeDepotListingSource{}, requestErr
		}
		if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
			break
		}
		if statusCode < http.StatusInternalServerError {
			return homeDepotListingSource{}, fmt.Errorf("firecrawl_product_details_http_%d: %s", statusCode, strings.TrimSpace(string(raw)))
		}
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return homeDepotListingSource{}, fmt.Errorf("firecrawl_product_details_http_%d: %s", statusCode, strings.TrimSpace(string(raw)))
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Markdown string `json:"markdown"`
			RawHTML  string `json:"rawHtml"`
			Actions  struct {
				JavaScriptReturns []struct {
					Value string `json:"value"`
				} `json:"javascriptReturns"`
			} `json:"actions"`
		} `json:"data"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return homeDepotListingSource{}, err
	}
	if !envelope.Success {
		return homeDepotListingSource{}, fmt.Errorf("firecrawl_product_details_failed: %s", strings.TrimSpace(envelope.Error))
	}
	for _, result := range envelope.Data.Actions.JavaScriptReturns {
		if listing, parseErr := parseHomeDepotProductDetailsData(result.Value, internetNumber); parseErr == nil {
			return listing, nil
		}
	}
	listing := parseHomeDepotListingDocument(envelope.Data.Markdown, envelope.Data.RawHTML, "", "", internetNumber)
	if listing.ProductOverview != "" && len(listing.BulletPoints) > 0 {
		return listing, nil
	}
	return homeDepotListingSource{}, fmt.Errorf("home_depot_product_details_data_missing")
}

func homeDepotProductDetailsActionScript(internetNumber string) string {
	itemID, _ := json.Marshal(internetNumber)
	return fmt.Sprintf(`(async () => {
  const sleep = (milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds));
  const viewMore = document.querySelector('[data-testid="view-more"]');
  if (viewMore) {
    viewMore.click();
    await sleep(5000);
  }
  const root = document.querySelector('#product-section-overview');
  const trigger = root && root.querySelector('[role="button"]');
  if (!root || !trigger) return JSON.stringify({ error: 'product_details_trigger_missing' });
  root.scrollIntoView({ block: 'center' });
  if (trigger.getAttribute('aria-expanded') !== 'true') trigger.click();
  await sleep(15000);
  let section = root.querySelector('#product-section-product-overview');
  if (!section || !section.innerText.trim()) {
    if (trigger.getAttribute('aria-expanded') === 'true') trigger.click();
    await sleep(1500);
    trigger.click();
    await sleep(20000);
    section = root.querySelector('#product-section-product-overview');
  }
  const headings = section ? Array.from(section.querySelectorAll('h1,h2,h3,h4')) : [];
  const aboutHeading = headings.find((element) => element.textContent.trim() === 'About This Product');
  const description = aboutHeading && aboutHeading.nextElementSibling ? aboutHeading.nextElementSibling.textContent.trim() : '';
  const highlights = section ? Array.from(section.querySelectorAll('[data-testid="bullet-list-item"]'))
    .map((element) => element.textContent.trim())
    .filter((value) => value && value.toLowerCase() !== 'return policy') : [];
  return JSON.stringify({ data: { product: { identifiers: { itemId: %s }, details: { description, highlights } } } });
})()`, string(itemID))
}

func parseHomeDepotProductDetailsData(content, internetNumber string) (homeDepotListingSource, error) {
	content = stdhtml.UnescapeString(stripHomeDepotMarkup(content))
	for index, char := range content {
		if char != '{' {
			continue
		}
		var response struct {
			Data struct {
				Product struct {
					Identifiers struct {
						ItemID string `json:"itemId"`
					} `json:"identifiers"`
					Details struct {
						Description string   `json:"description"`
						Highlights  []string `json:"highlights"`
					} `json:"details"`
				} `json:"product"`
			} `json:"data"`
		}
		if err := json.NewDecoder(strings.NewReader(content[index:])).Decode(&response); err != nil {
			continue
		}
		if response.Data.Product.Identifiers.ItemID != "" && response.Data.Product.Identifiers.ItemID != internetNumber {
			continue
		}
		listing := normalizeHomeDepotListingSource(homeDepotListingSource{
			InternetNumber:  internetNumber,
			ProductOverview: response.Data.Product.Details.Description,
			BulletPoints:    response.Data.Product.Details.Highlights,
		})
		if listing.ProductOverview != "" && len(listing.BulletPoints) > 0 {
			return listing, nil
		}
	}
	return homeDepotListingSource{}, fmt.Errorf("home_depot_product_details_data_missing")
}

func parseHomeDepotListingDocument(markdown, html, metadataTitle, _ string, internetNumber string) homeDepotListingSource {
	markdownListing := parseHomeDepotListingText(markdown, metadataTitle, internetNumber)
	htmlListing := parseHomeDepotListingText(homeDepotHTMLToMarkdown(html), metadataTitle, internetNumber)
	return mergeHomeDepotListingSources(markdownListing, htmlListing)
}

func parseHomeDepotListingText(content, metadataTitle, internetNumber string) homeDepotListingSource {
	lines := make([]string, 0)
	for _, rawLine := range strings.Split(content, "\n") {
		line := cleanHomeDepotDocumentLine(rawLine)
		if line != "" {
			lines = append(lines, line)
		}
	}
	listing := homeDepotListingSource{InternetNumber: internetNumber, ProductTitle: cleanHomeDepotProductTitle(metadataTitle)}
	for _, line := range lines {
		if listing.ProductTitle == "" && strings.HasPrefix(line, "# ") {
			listing.ProductTitle = cleanHomeDepotProductTitle(line[2:])
		}
		if match := regexp.MustCompile(`(?i)Internet\s*#\s*(\d+)`).FindStringSubmatch(line); len(match) == 2 {
			listing.InternetNumber = match[1]
		}
		if match := regexp.MustCompile(`(?i)Model\s*#\s*[:：]?\s*(.+)$`).FindStringSubmatch(line); len(match) == 2 {
			listing.ModelNumber = strings.TrimSpace(match[1])
		}
	}
	// Home Depot uses two distinct bullet groups: the short list beside the
	// title is Salient Bullets; Product Details > Highlights is the full list.
	listing.Highlights = parseHomeDepotTopBulletSection(lines)
	listing.BulletPoints = parseHomeDepotBulletSection(lines, "highlights")
	listing.Specifications = parseHomeDepotSpecificationSections(lines)
	if overview := parseHomeDepotTextSection(lines, "about this product"); overview != "" {
		listing.ProductOverview = overview
	} else if overview := parseHomeDepotTextSection(lines, "product overview"); overview != "" {
		listing.ProductOverview = overview
	}
	return normalizeHomeDepotListingSource(listing)
}

func homeDepotHTMLToMarkdown(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	value = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>|<style[^>]*>.*?</style>`).ReplaceAllString(value, "")
	for level := 1; level <= 6; level++ {
		opening := regexp.MustCompile(fmt.Sprintf(`(?is)<h%d[^>]*>`, level))
		value = opening.ReplaceAllString(value, "\n"+strings.Repeat("#", level)+" ")
		value = regexp.MustCompile(fmt.Sprintf(`(?is)</h%d\s*>`, level)).ReplaceAllString(value, "\n")
	}
	value = regexp.MustCompile(`(?is)<dt[^>]*>`).ReplaceAllString(value, "\n### ")
	value = regexp.MustCompile(`(?is)</dt\s*>|<dd[^>]*>|</dd\s*>`).ReplaceAllString(value, "\n")
	value = regexp.MustCompile(`(?is)<li[^>]*>`).ReplaceAllString(value, "\n- ")
	value = regexp.MustCompile(`(?is)</li\s*>`).ReplaceAllString(value, "\n")
	value = regexp.MustCompile(`(?is)<br\s*/?>|</p\s*>|</div\s*>|</section\s*>|</tr\s*>|</th\s*>|</td\s*>`).ReplaceAllString(value, "\n")
	value = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(value, "")
	return stdhtml.UnescapeString(value)
}

func mergeHomeDepotListingSources(sources ...homeDepotListingSource) homeDepotListingSource {
	merged := homeDepotListingSource{}
	for _, source := range sources {
		if merged.InternetNumber == "" {
			merged.InternetNumber = source.InternetNumber
		}
		if merged.CanonicalURL == "" {
			merged.CanonicalURL = source.CanonicalURL
		}
		if merged.Brand == "" {
			merged.Brand = source.Brand
		}
		if merged.ModelNumber == "" {
			merged.ModelNumber = source.ModelNumber
		}
		if merged.ProductTitle == "" {
			merged.ProductTitle = source.ProductTitle
		}
		if merged.ProductCategory == "" {
			merged.ProductCategory = source.ProductCategory
		}
		if merged.ProductOverview == "" || isHomeDepotMarkupArtifact(merged.ProductOverview) {
			merged.ProductOverview = source.ProductOverview
		}
		merged.BulletPoints = append(merged.BulletPoints, source.BulletPoints...)
		merged.Highlights = append(merged.Highlights, source.Highlights...)
		merged.Specifications = append(merged.Specifications, source.Specifications...)
		merged.Breadcrumbs = append(merged.Breadcrumbs, source.Breadcrumbs...)
	}
	return normalizeHomeDepotListingSource(merged)
}

func cleanHomeDepotDocumentLine(value string) string {
	value = strings.TrimSpace(value)
	value = regexp.MustCompile(`^[-*•]\s*`).ReplaceAllString(value, "- ")
	value = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`).ReplaceAllString(value, "$1")
	value = strings.ReplaceAll(value, "**", "")
	return strings.TrimSpace(value)
}

func cleanHomeDepotProductTitle(value string) string {
	value = strings.TrimSpace(value)
	value = regexp.MustCompile(`(?i)\s*[-|]\s*(?:The Home Depot|Home Depot)\s*$`).ReplaceAllString(value, "")
	value = regexp.MustCompile(`(?i)\s+HH\d+[A-Z0-9-]*\s*$`).ReplaceAllString(value, "")
	return strings.TrimSpace(value)
}

func homeDepotTitleForEvaluation(listing homeDepotListingSource) string {
	title := strings.TrimSpace(listing.ProductTitle)
	brand := strings.TrimSpace(listing.Brand)
	if brand != "" {
		pattern := regexp.MustCompile(`(?i)^` + regexp.QuoteMeta(brand) + `(?:\s*[-|:,.]\s*|\s+)`)
		title = strings.TrimSpace(pattern.ReplaceAllString(title, ""))
	}
	return title
}

func isHomeDepotMarkupArtifact(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return lower == "" || strings.Contains(lower, "aria-expanded") || strings.Contains(lower, "tabindex=") || strings.Contains(lower, ":first-child") || strings.Contains(lower, "sui-") || strings.Contains(lower, "<div") || strings.Contains(lower, "div>")
}

func stripHomeDepotMarkup(value string) string {
	value = regexp.MustCompile(`(?s)<script[^>]*>.*?</script>|<style[^>]*>.*?</style>`).ReplaceAllString(value, "")
	value = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(value, "\n")
	return value
}

func homeDepotHeading(line string) (string, bool) {
	match := regexp.MustCompile(`^#{1,6}\s+(.+)$`).FindStringSubmatch(strings.TrimSpace(line))
	if len(match) != 2 {
		return "", false
	}
	return strings.ToLower(strings.TrimSpace(match[1])), true
}

func parseHomeDepotBulletSection(lines []string, section string) []string {
	inside := false
	result := make([]string, 0)
	for _, line := range lines {
		if heading, ok := homeDepotHeading(line); ok {
			inside = strings.Contains(heading, section)
			if inside {
				continue
			}
			if len(result) > 0 {
				break
			}
		}
		if inside && strings.HasPrefix(line, "- ") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			if value != "" && !strings.EqualFold(value, "return policy") {
				result = append(result, value)
			}
		}
	}
	return uniqueTrimmedStrings(result)
}

// parseHomeDepotTopBulletSection captures the customer-facing bullet list that
// appears beside the product title before the expandable detail sections.
func parseHomeDepotTopBulletSection(lines []string) []string {
	result := make([]string, 0, 5)
	started := false
	seenProductTitle := false
	for _, line := range lines {
		if heading, ok := homeDepotHeading(line); ok {
			if strings.HasPrefix(strings.TrimSpace(line), "# ") && !strings.Contains(heading, "internet #") && !strings.Contains(heading, "model #") {
				seenProductTitle = true
			}
			if started {
				break
			}
			if strings.Contains(heading, "specification") || heading == "details" || strings.Contains(heading, "warranty") || strings.Contains(heading, "highlight") {
				break
			}
			continue
		}
		if seenProductTitle && strings.HasPrefix(line, "- ") {
			started = true
			value := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			if value != "" && !strings.EqualFold(value, "return policy") {
				result = append(result, value)
			}
			continue
		}
		if started {
			break
		}
	}
	return uniqueTrimmedStrings(result)
}

func parseHomeDepotTextSection(lines []string, section string) string {
	inside := false
	paragraphs := make([]string, 0)
	for _, line := range lines {
		if heading, ok := homeDepotHeading(line); ok {
			if inside {
				break
			}
			inside = heading == section
			continue
		}
		if inside && !strings.HasPrefix(line, "- ") {
			paragraphs = append(paragraphs, strings.TrimSpace(line))
		}
	}
	return strings.TrimSpace(strings.Join(uniqueTrimmedStrings(paragraphs), " "))
}

func parseHomeDepotSpecificationSections(lines []string) []homeDepotListingSpecification {
	inside := false
	pendingName := ""
	result := make([]homeDepotListingSpecification, 0)
	for _, line := range lines {
		if heading, ok := homeDepotHeading(line); ok {
			wasInside := inside
			isSection := strings.Contains(heading, "specification") || heading == "details" || strings.Contains(heading, "warranty")
			headingLevel := len(line) - len(strings.TrimLeft(line, "#"))
			if wasInside && !isSection && headingLevel >= 3 {
				pendingName = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#"))
			}
			if isSection {
				inside = true
			} else if headingLevel <= 2 {
				inside = false
			}
			continue
		}
		if !inside {
			continue
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "|") {
			cells := strings.Split(strings.Trim(line, "|"), "|")
			if len(cells) >= 2 && !strings.Contains(cells[0], "---") && !strings.EqualFold(strings.TrimSpace(cells[0]), "name") {
				result = append(result, homeDepotListingSpecification{Name: cleanHomeDepotDocumentLine(cells[0]), Value: cleanHomeDepotDocumentLine(strings.Join(cells[1:], " "))})
			}
			continue
		}
		if pendingName != "" {
			if !strings.EqualFold(line, "see similar items") && !strings.EqualFold(line, "view more details") {
				result = append(result, homeDepotListingSpecification{Name: cleanHomeDepotDocumentLine(pendingName), Value: cleanHomeDepotDocumentLine(line)})
				pendingName = ""
			}
			continue
		}
		if strings.HasPrefix(line, "- ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "- "))
		}
		parts := regexp.MustCompile(`\s*[:：]\s*`).Split(line, 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && strings.TrimSpace(parts[1]) != "" {
			result = append(result, homeDepotListingSpecification{Name: cleanHomeDepotDocumentLine(parts[0]), Value: cleanHomeDepotDocumentLine(parts[1])})
		}
	}
	return normalizeHomeDepotListingSource(homeDepotListingSource{Specifications: result}).Specifications
}

func normalizeHomeDepotListingSource(listing homeDepotListingSource) homeDepotListingSource {
	listing.InternetNumber = strings.TrimSpace(listing.InternetNumber)
	listing.CanonicalURL = strings.TrimSpace(listing.CanonicalURL)
	listing.Brand = strings.TrimSpace(listing.Brand)
	listing.ModelNumber = strings.TrimSpace(listing.ModelNumber)
	listing.ProductTitle = cleanHomeDepotProductTitle(listing.ProductTitle)
	listing.ProductCategory = strings.TrimSpace(listing.ProductCategory)
	listing.ProductOverview = strings.TrimSpace(listing.ProductOverview)
	if strings.Contains(listing.ProductOverview, "<") && strings.Contains(listing.ProductOverview, ">") {
		listing.ProductOverview = strings.TrimSpace(strings.Join(strings.Fields(stripHomeDepotMarkup(listing.ProductOverview)), " "))
	}
	if isHomeDepotMarkupArtifact(listing.ProductOverview) {
		listing.ProductOverview = ""
	}
	if strings.EqualFold(listing.ProductOverview, listing.ProductTitle) || containsHomeDepotString(listing.BulletPoints, listing.ProductOverview) || containsHomeDepotString(listing.Highlights, listing.ProductOverview) {
		listing.ProductOverview = ""
	}
	listing.BulletPoints = uniqueTrimmedStrings(listing.BulletPoints)
	listing.Highlights = uniqueTrimmedStrings(listing.Highlights)
	if sameHomeDepotStringSet(listing.BulletPoints, listing.Highlights) {
		listing.BulletPoints = nil
	}
	listing.Breadcrumbs = uniqueTrimmedStrings(listing.Breadcrumbs)
	specifications := make([]homeDepotListingSpecification, 0, len(listing.Specifications))
	seen := make(map[string]struct{}, len(listing.Specifications))
	for _, specification := range listing.Specifications {
		specification.Name = strings.TrimSpace(specification.Name)
		specification.Value = strings.TrimSpace(specification.Value)
		if specification.Name == "" || specification.Value == "" {
			continue
		}
		key := strings.ToLower(specification.Name + "\x00" + specification.Value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		specifications = append(specifications, specification)
	}
	listing.Specifications = specifications
	return listing
}

func containsHomeDepotString(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}

func sameHomeDepotStringSet(a, b []string) bool {
	if len(a) == 0 || len(a) != len(b) {
		return false
	}
	for _, value := range a {
		if !containsHomeDepotString(b, value) {
			return false
		}
	}
	return true
}

func uniqueTrimmedStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		key := strings.ToLower(value)
		if value == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func homeDepotListingSourceResponse(listing homeDepotListingSource) map[string]interface{} {
	return map[string]interface{}{
		"internet_number": listing.InternetNumber, "canonical_url": listing.CanonicalURL,
		"brand": listing.Brand, "model_number": listing.ModelNumber, "product_category": listing.ProductCategory,
		"title": listing.ProductTitle, "salient_bullets": listing.Highlights,
		"bullet_points": listing.BulletPoints, "about_this_product": listing.ProductOverview,
	}
}

func homeDepotListingForEvaluation(listing homeDepotListingSource) (productFeatures, productProof, existingListing string) {
	facts := make([]string, 0, 3)
	if listing.Brand != "" {
		facts = append(facts, "Brand: "+listing.Brand)
	}
	if listing.ModelNumber != "" {
		facts = append(facts, "Model Number: "+listing.ModelNumber)
	}
	if len(facts) == 0 {
		facts = append(facts, "Internet Number: "+listing.InternetNumber)
	}
	proof := []string{"Home Depot Internet Number: " + listing.InternetNumber, "Source URL: " + listing.CanonicalURL}
	parts := []string{"Title:\n" + homeDepotTitleForEvaluation(listing)}
	if listing.ProductOverview != "" {
		parts = append(parts, "About This Product:\n"+listing.ProductOverview)
	}
	if len(listing.BulletPoints) > 0 {
		lines := make([]string, 0, len(listing.BulletPoints))
		for index, bullet := range listing.BulletPoints {
			lines = append(lines, fmt.Sprintf("%d. %s", index+1, bullet))
		}
		parts = append(parts, "Bullet Points:\n"+strings.Join(lines, "\n"))
	}
	if len(listing.Highlights) > 0 {
		lines := make([]string, 0, len(listing.Highlights))
		for index, highlight := range listing.Highlights {
			lines = append(lines, fmt.Sprintf("%d. %s", index+1, highlight))
		}
		parts = append(parts, "Salient Bullets:\n"+strings.Join(lines, "\n"))
	}
	return strings.Join(facts, "\n"), strings.Join(proof, "\n"), strings.Join(parts, "\n\n")
}
