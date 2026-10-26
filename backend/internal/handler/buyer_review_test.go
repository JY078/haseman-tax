package handler

import (
	"advertising-image/backend/internal/model"
	"testing"
)

func TestBuyerReviewProductImages(t *testing.T) {
	products := []model.ProductSKU{
		{SKU: "DO1-B-93", ASIN: "B0PRODUCT1", ImageURL: "/uploads/product-1.png"},
		{SKU: "B0PRODUCT1", ASIN: "B0PRODUCT2", ImageURL: "/uploads/exact-sku.png"},
		{SKU: "DO1-R-75", ASIN: "B0PRODUCT3", ImageURL: "https://example.com/product-3.png"},
		{SKU: "NO-IMAGE", ASIN: "B0EMPTY"},
		{SKU: "OTHER", ASIN: "NO-IMAGE", ImageURL: "/uploads/wrong.png"},
		{SKU: "DUPLICATE-1", ASIN: "B0SHARED", ImageURL: "/uploads/first.png"},
		{SKU: "DUPLICATE-2", ASIN: "B0SHARED", ImageURL: "/uploads/second.png"},
		{SKU: "DUPLICATE-3", ASIN: "B0SHARED", ImageURL: "/uploads/first.png"},
		{SKU: "SAME-1", ASIN: "B0SAME", ImageURL: "/uploads/shared.png"},
		{SKU: "SAME-2", ASIN: "B0SAME", ImageURL: "/uploads/shared.png"},
	}
	tests := []struct{ key, want string }{
		{" do1-b-93 ", ""},
		{"B0PRODUCT1", "/uploads/product-1.png"},
		{" b0product3 ", "https://example.com/product-3.png"},
		{"NO-IMAGE", "/uploads/wrong.png"}, {"B0EMPTY", ""}, {"B0SHARED", ""},
		{"B0SAME", "/uploads/shared.png"},
		{"MISSING", ""}, {"DO1-B", ""}, {"", ""},
	}
	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			reviews := []model.BuyerReview{{ChildASIN: test.key, Image: "imported-image.png", ReviewImages: "review-image.png", ProductImageURL: "stale.png"}}
			matchBuyerReviewProductImages(reviews, products)
			if reviews[0].ProductImageURL != test.want {
				t.Fatalf("image = %q; want %q", reviews[0].ProductImageURL, test.want)
			}
			if reviews[0].Image != "imported-image.png" || reviews[0].ReviewImages != "review-image.png" {
				t.Fatal("imported images were modified")
			}
		})
	}
}
