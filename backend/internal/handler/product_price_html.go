package handler

import (
	"errors"
	"strings"

	htmlnode "golang.org/x/net/html"
)

const (
	amazonRowNew     = 1
	amazonRowUnknown = 0
	amazonRowUsed    = -1
	amazonRowNotUsed = 2
	amazonRowAny     = 3
)

func amazonPriceRegionID(id string) bool {
	switch id {
	case "corePriceDisplay_desktop_feature_div", "corePriceDisplay_mobile_feature_div", "corePrice_feature_div",
		"priceblock_ourprice", "priceblock_dealprice", "priceblock_saleprice", "price_inside_buybox",
		"newBuyBoxPrice", "unifiedPrice_feature_div", "desktop_unifiedPrice", "apex_desktop":
		return true
	}
	return false
}

// Amazon renders the new item and the used/other-seller offer as sibling
// accordion rows that each carry their own price block.
func amazonAccordionRowKind(n *htmlnode.Node, attr func(*htmlnode.Node, string) string) int {
	for p := n; p != nil; p = p.Parent {
		id := strings.ToLower(attr(p, "id"))
		if strings.Contains(id, "newaccordionrow") {
			return amazonRowNew
		}
		if strings.Contains(id, "usedaccordionrow") {
			return amazonRowUsed
		}
	}
	return amazonRowUnknown
}

// Container blocks such as apex_desktop and desktop_unifiedPrice wrap several
// offer rows, so only the innermost block describes a single offer.
func amazonInnermostPriceRegions(regions []*htmlnode.Node, attr func(*htmlnode.Node, string) string) []*htmlnode.Node {
	kept := make([]*htmlnode.Node, 0, len(regions))
	for _, region := range regions {
		nested := false
		var scan func(*htmlnode.Node)
		scan = func(n *htmlnode.Node) {
			if nested {
				return
			}
			if n != region && n.Type == htmlnode.ElementNode && amazonPriceRegionID(attr(n, "id")) {
				nested = true
				return
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				scan(c)
			}
		}
		scan(region)
		if !nested {
			kept = append(kept, region)
		}
	}
	return kept
}

type amazonHTMLScanner struct {
	attr     func(*htmlnode.Node, string) string
	hasClass func(*htmlnode.Node, string) bool
	content  func(*htmlnode.Node) string
	walk     func(*htmlnode.Node, func(*htmlnode.Node))
}

// Collect the sale and list amounts of one scope of price regions.
func (s *amazonHTMLScanner) regionPrices(regions []*htmlnode.Node, scope int) (map[string]bool, map[string]bool) {
	currents, lists := map[string]bool{}, map[string]bool{}
	for _, region := range regions {
		switch scope {
		case amazonRowNew:
			if amazonAccordionRowKind(region, s.attr) != amazonRowNew {
				continue
			}
		case amazonRowNotUsed:
			if amazonAccordionRowKind(region, s.attr) == amazonRowUsed {
				continue
			}
		}
		regionID := s.attr(region, "id")
		// Conditional and subscription offers must not become the public price.
		text := strings.ToLower(s.content(region))
		if strings.Contains(text, "prime exclusive") || strings.Contains(text, "subscribe") || strings.Contains(text, "per month") || strings.Contains(text, "/month") {
			continue
		}
		if regionID == "priceblock_ourprice" || regionID == "priceblock_dealprice" || regionID == "priceblock_saleprice" || regionID == "price_inside_buybox" || regionID == "newBuyBoxPrice" {
			if price, amount, e := normalizeAmazonAmountLoose(&text); e == nil && price != nil && amount > 0 {
				currents[*price] = true
				continue
			}
		}
		s.walk(region, func(n *htmlnode.Node) {
			if !s.hasClass(n, "a-offscreen") || amazonIsDerivedPriceNode(n, region, s.attr) {
				return
			}
			raw := strings.TrimSpace(s.content(n))
			if !strings.HasPrefix(raw, "$") && !strings.HasPrefix(raw, "US$") {
				return
			}
			price, amount, e := normalizeAmazonAmount(&raw)
			if e != nil || price == nil || amount <= 0 {
				return
			}
			list, priceNode := false, false
			for parent := n.Parent; parent != nil; parent = parent.Parent {
				if s.hasClass(parent, "a-text-price") {
					list = true
				}
				if s.hasClass(parent, "a-price") {
					priceNode = true
				}
				if parent == region {
					break
				}
			}
			if !priceNode {
				return
			}
			if list {
				lists[*price] = true
			} else {
				currents[*price] = true
			}
		})
	}
	return currents, lists
}

// Amazon prints derived amounts next to the real offer: a unit rate such as
// "$13.63 per count" and the struck-through "List Price: $69.99". Neither is
// the current offer, so neither may be collected as a price.
func amazonIsDerivedPriceNode(n *htmlnode.Node, stop *htmlnode.Node, attr func(*htmlnode.Node, string) string) bool {
	for p := n; p != nil; p = p.Parent {
		if p.Type == htmlnode.ElementNode {
			marker := strings.ToLower(attr(p, "class") + " " + attr(p, "id"))
			if strings.Contains(marker, "priceperunit") || strings.Contains(marker, "perunit") || strings.Contains(marker, "basisprice") {
				return true
			}
		}
		if p == stop {
			break
		}
	}
	return false
}

func isAmazonAvailabilityID(id string) bool {
	switch id {
	case "availability", "availability_feature_div", "outOfStock":
		return true
	}
	return false
}

func amazonAvailabilitySaysUnavailable(text string) bool {
	message := strings.ToLower(text)
	for _, phrase := range []string{"currently unavailable", "out of stock", "no longer available"} {
		if strings.Contains(message, phrase) {
			return true
		}
	}
	return false
}

func amazonPriceResult(currents, lists map[string]bool) productPriceInput {
	current := ""
	for p := range currents {
		current = p
	}
	result := productPriceInput{Price: &current}
	for original := range lists {
		_, base, _ := normalizeProductPrice(&original)
		_, sale, _ := normalizeProductPrice(&current)
		if base < sale {
			original, current = current, original
			base, sale = sale, base
		}
		if base > sale {
			result.Price = &original
			result.SalePrice = &current
		}
	}
	return result
}

type amazonPriceNode struct {
	node  *htmlnode.Node
	order int
	value string
}

// Amazon labels the buy-box rows it wants a shopper to compare: a "Regular
// Price" row, a "Prime Member Price" row, and — while a deal runs — a "Buy New"
// row under a "Limited time deal" badge. Read those labels instead of guessing
// from whichever struck-through amount happens to sit nearby.
func amazonAccordionOffers(root *htmlnode.Node, attr func(*htmlnode.Node, string) string, content func(*htmlnode.Node) string) (productPriceInput, bool) {
	order := map[*htmlnode.Node]int{}
	prices := []amazonPriceNode{}
	var collect func(*htmlnode.Node)
	collect = func(n *htmlnode.Node) {
		order[n] = len(order)
		if n.Type == htmlnode.ElementNode && strings.Contains(" "+attr(n, "class")+" ", " a-offscreen ") {
			if !amazonIsDerivedPriceNode(n, nil, attr) {
				raw := strings.TrimSpace(content(n))
				if price, amount, err := normalizeAmazonAmount(&raw); err == nil && price != nil && amount > 0 {
					prices = append(prices, amazonPriceNode{node: n, order: order[n], value: *price})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			collect(c)
		}
	}
	collect(root)

	// The price of a row is the first amount Amazon prints after its caption,
	// searched from the closest enclosing block outwards so a neighbouring
	// row's price is never picked up first.
	priceAfter := func(caption *htmlnode.Node) *string {
		for ancestor, depth := caption, 0; ancestor != nil && depth < 8; ancestor, depth = ancestor.Parent, depth+1 {
			best, bestOrder := "", 0
			for _, candidate := range prices {
				if candidate.order <= order[caption] {
					continue
				}
				inside := false
				for p := candidate.node; p != nil; p = p.Parent {
					if p == ancestor {
						inside = true
						break
					}
				}
				if !inside {
					continue
				}
				if best == "" || candidate.order < bestOrder {
					best, bestOrder = candidate.value, candidate.order
				}
			}
			if best != "" {
				return &best
			}
		}
		return nil
	}

	var regular, prime, dealPriceCaption, buyNew *string
	deal := false
	var scan func(*htmlnode.Node)
	scan = func(n *htmlnode.Node) {
		if n.Type == htmlnode.ElementNode {
			id := attr(n, "id")
			label := strings.ToLower(strings.TrimSpace(content(n)))
			switch id {
			case "newAccordionCaption_feature_div":
				switch {
				case strings.Contains(label, "regular price"):
					regular = priceAfter(n)
				case strings.Contains(label, "buy new"):
					buyNew = priceAfter(n)
				}
			case "primeSavingsUpsellCaption_feature_div":
				switch {
				case strings.Contains(label, "prime member price"):
					prime = priceAfter(n)
				case strings.Contains(label, "deal price"):
					dealPriceCaption = priceAfter(n)
				}
			case "dealBadge_feature_div":
				// The badge also exists on the used row, where it stays empty.
				for p := n; p != nil; p = p.Parent {
					if strings.Contains(strings.ToLower(attr(p, "id")), "newaccordionrow") {
						if strings.Contains(label, "limited time deal") {
							deal = true
						}
						break
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			scan(c)
		}
	}
	scan(root)

	switch {
	case regular != nil || prime != nil:
		var dealPrice *string
		if deal {
			dealPrice = buyNew
		}
		if dealPrice == nil {
			dealPrice = dealPriceCaption
		}
		return productPriceInput{Price: regular, SalePrice: prime, BestDeal: dealPrice}, true
	case dealPriceCaption != nil:
		return productPriceInput{BestDeal: dealPriceCaption}, true
	case deal && buyNew != nil:
		// A running deal replaces the standing price, so no regular price is
		// stored rather than falling back to the struck-through list price.
		return productPriceInput{BestDeal: buyNew}, true
	}
	return productPriceInput{}, false
}

// Read only Amazon's main price region, never recommendation or other-seller prices.
func amazonHTMLPrice(document, asin string) (productPriceInput, error) {
	invalid := errors.New("主商品 HTML 中没有可确认的美元售价")
	root, err := htmlnode.Parse(strings.NewReader(document))
	if err != nil {
		return productPriceInput{}, invalid
	}
	attr := func(n *htmlnode.Node, key string) string {
		for _, a := range n.Attr {
			if a.Key == key {
				return a.Val
			}
		}
		return ""
	}
	hasClass := func(n *htmlnode.Node, class string) bool {
		return strings.Contains(" "+attr(n, "class")+" ", " "+class+" ")
	}
	var content func(*htmlnode.Node) string
	content = func(n *htmlnode.Node) string {
		if n.Type == htmlnode.TextNode {
			return n.Data
		}
		var b strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			b.WriteString(content(c))
		}
		return b.String()
	}
	var walk func(*htmlnode.Node, func(*htmlnode.Node))
	walk = func(n *htmlnode.Node, visit func(*htmlnode.Node)) {
		visit(n)
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, visit)
		}
	}
	scanner := amazonHTMLScanner{attr: attr, hasClass: hasClass, content: content, walk: walk}
	matched, mismatch, priceUnavailable, unavailableOffer := false, false, false, false
	var found []*htmlnode.Node
	walk(root, func(n *htmlnode.Node) {
		if n.Type != htmlnode.ElementNode {
			return
		}
		id := attr(n, "id")
		if isAmazonAvailabilityID(id) && amazonAvailabilitySaysUnavailable(content(n)) {
			unavailableOffer = true
		}
		if id == "priceNotAvailable_feature_div" {
			message := strings.ToLower(content(n))
			priceUnavailable = strings.Contains(message, "add this item to your cart") || strings.Contains(message, "to see product details") || strings.Contains(message, "currently unavailable")
		}
		if n.Data == "input" && id == "ASIN" {
			selected := strings.ToUpper(strings.TrimSpace(attr(n, "value")))
			if selected == asin {
				matched = true
			} else if selected != "" {
				mismatch = true
			}
		}
		if strings.EqualFold(strings.TrimSpace(attr(n, "data-csa-c-asin")), asin) {
			matched = true
		}
		if amazonPriceRegionID(id) {
			found = append(found, n)
			matched = true
		}
	})
	if !matched || mismatch {
		return productPriceInput{}, invalid
	}
	// Amazon's labelled offer rows describe the price far more reliably than
	// whichever amounts appear near the buy box.
	if offers, ok := amazonAccordionOffers(root, attr, content); ok {
		return offers, nil
	}
	regions := amazonInnermostPriceRegions(found, attr)
	if len(regions) == 0 {
		regions = found
	}
	currents, lists := scanner.regionPrices(regions, amazonRowAny)
	if !priceUnavailable {
		// The used/other-seller row repeats its own price inside the same buy
		// box, so the new-item row must win before any wider scope is used.
		for _, scope := range []int{amazonRowNew, amazonRowNotUsed} {
			scoped, scopedLists := scanner.regionPrices(regions, scope)
			if len(scoped) == 1 && len(scopedLists) <= 1 {
				return amazonPriceResult(scoped, scopedLists), nil
			}
		}
		if len(currents) == 1 && len(lists) <= 1 {
			return amazonPriceResult(currents, lists), nil
		}
	}
	// An out-of-stock listing has no offer at all and keeps only recommendation
	// and A+ content prices, so report the missing offer instead of scanning
	// those widgets for something to fill in.
	if unavailableOffer {
		return productPriceInput{}, errAmazonPriceUnavailable
	}
	// When Amazon hides the new-item price behind "add to cart", the
	// current ASIN may still expose an Offers/Used-from price. Use that
	// explicit ASIN offer as the sole original price; never inspect cards
	// or recommendation carousels.
	if priceUnavailable || len(currents) == 0 {
		var offers string
		walk(root, func(n *htmlnode.Node) {
			if n.Type == htmlnode.ElementNode && attr(n, "id") == "aod-ingress-link" {
				offers = content(n)
			}
		})
		if price, _, err := normalizeAmazonAmountLoose(&offers); err == nil && price != nil {
			return productPriceInput{Price: price}, nil
		}
	}
	// Firecrawl's cleaned HTML can omit Amazon's feature IDs. In that
	// response shape, keep only prices inside a node explicitly tagged for
	// the requested ASIN and ignore recommendation/carousel containers.
	if matched {
		currents, lists := map[string]bool{}, map[string]bool{}
		walk(root, func(n *htmlnode.Node) {
			if n.Type != htmlnode.ElementNode || !hasClass(n, "a-offscreen") || amazonIsDerivedPriceNode(n, nil, attr) {
				return
			}
			raw := strings.TrimSpace(content(n))
			if !strings.HasPrefix(raw, "$") && !strings.HasPrefix(raw, "US$") {
				return
			}
			price, amount, e := normalizeAmazonAmountLoose(&raw)
			if e != nil || price == nil || amount <= 0 {
				return
			}
			insideASIN, excluded := false, false
			for parent := n.Parent; parent != nil; parent = parent.Parent {
				if strings.EqualFold(strings.TrimSpace(attr(parent, "data-csa-c-asin")), asin) {
					insideASIN = true
				}
				// Variant swatches list the price of other colors/sizes.
				id := strings.ToLower(attr(parent, "id"))
				if strings.Contains(id, "usedaccordionrow") {
					excluded = true
				}
				classes := strings.ToLower(attr(parent, "class") + " " + attr(parent, "id"))
				// cerberus/atcbtn mark Amazon's shared "consider these
				// available items" recommendations shown on out-of-stock pages.
				if strings.Contains(classes, "p13n") || strings.Contains(classes, "carousel") || strings.Contains(classes, "recommend") || strings.Contains(classes, "alternative") ||
					strings.Contains(classes, "apex_price") || strings.Contains(classes, "apex_on_twister_price") || strings.Contains(classes, "dimension-slot-info") ||
					strings.Contains(classes, "cerberus") || strings.Contains(classes, "atcbtn") {
					excluded = true
				}
			}
			if !insideASIN || excluded {
				return
			}
			isList := false
			for parent := n.Parent; parent != nil; parent = parent.Parent {
				if hasClass(parent, "a-text-price") {
					isList = true
				}
				if strings.EqualFold(strings.TrimSpace(attr(parent, "data-csa-c-asin")), asin) {
					break
				}
			}
			if isList {
				lists[*price] = true
			} else {
				currents[*price] = true
			}
		})
		if len(currents) == 1 && len(lists) <= 1 {
			return amazonPriceResult(currents, lists), nil
		}
	}
	return productPriceInput{}, invalid
}
