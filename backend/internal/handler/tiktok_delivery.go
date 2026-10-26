package handler

import (
	"strings"

	"gorm.io/gorm"
)

// Choose exactly one shipping field for each matched settlement. The order join
// selects one SKU row, so importing multiple SKUs cannot multiply the fee.
func tikTokDeliverySummary(db *gorm.DB, currency string, stores ...string) *gorm.DB {
	store := ""
	if len(stores) > 0 {
		store = stores[0]
	}
	// Keep the raw site value: an empty site means "all countries".
	site := ""
	if len(stores) > 1 {
		site = strings.TrimSpace(stores[1])
	}
	// Fixed source field names; preserve signs and round only after quarterly aggregation.
	sumFields := func(fields ...string) string {
		terms := make([]string, 0, len(fields))
		for _, field := range fields {
			source := `s.data->>'` + field + `'`
			switch field {
			case "Signature confirmation service fee":
				source = `COALESCE(NULLIF(BTRIM(s.data->>'Signature confirmation service fee'), ''), s.data->>'Signature confirmation servicer fee')`
			case "Co-funded promotion (seller-funded)":
				source = `COALESCE(NULLIF(BTRIM(s.data->>'Co-funded promotion (seller-funded)'), ''), s.data->>'Co-funded promotion（seller-funded）')`
			case "Shipping protection service fee":
				source = `COALESCE(NULLIF(BTRIM(s.data->>'Shipping protection service fee'), ''), s.data->>'Protection service fee')`
			case "Sales proceed (negative balance)":
				source = `COALESCE(NULLIF(BTRIM(s.data->>'Sales proceed (negative balance)'), ''), s.data->>'Sales proceed（negative balance）')`
			}
			if field == "Sales tax payment" || field == "Retail delivery fee payment" {
				// Older sheets repeat payment headers; use the later tax-section value.
				source = `COALESCE(NULLIF(BTRIM(s.data->>'` + field + ` (2)'), ''), ` + source + `)`
			}
			terms = append(terms, `COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(`+source+`, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0)`)
		}
		return strings.Join(terms, " + ")
	}
	totalSettlementForType := func(typeName string) string {
		return "CASE WHEN LOWER(BTRIM(COALESCE(s.data->>'Type', ''))) = '" + strings.ToLower(typeName) + "' THEN (" + sumFields("Total settlement amount") + ") ELSE 0 END"
	}
	additionalCampaignPackage := totalSettlementForType("Additional Campaign Package")
	campaignPackage := totalSettlementForType("Campaign Package")
	campaignAmount := sumFields("Campaign service fee") + " + " + additionalCampaignPackage + " + " + campaignPackage
	commissionCampaignAmount := sumFields("Campaign service fee")
	salesAmount := sumFields("Subtotal after seller discounts")
	refundAmount := sumFields("Refund subtotal after seller discounts")
	smartPromotionAmount := sumFields("Smart Promotion fee", "Smart Promotion campaign period fee")
	promotionAmount := "0"
	otherFeesAmount := sumFields("Signature confirmation service fee", "Co-funded promotion (seller-funded)", "Shipping protection service fee", "Bill deduction for negative balance", "Sales proceed (negative balance)")
	shippingAmount := sumFields("Customer shipping fee offset", "Platform shipping fee discount", "Customer shipping fee", "Refunded customer shipping fee")
	taxAmount := sumFields("Sales tax", "Sales tax payment", "Sales tax refund", "Retail delivery fee", "Retail delivery fee payment", "Retail delivery fee refund")
	otherFeeSignature := sumFields("Signature confirmation service fee")
	otherFeeCofunded := sumFields("Co-funded promotion (seller-funded)")
	otherFeeProtection := sumFields("Shipping protection service fee")
	otherFeeNegativeDeduction := sumFields("Bill deduction for negative balance")
	otherFeeNegativeSales := sumFields("Sales proceed (negative balance)")
	campaignServiceItem := sumFields("Campaign service fee")
	smartPromotionItem := sumFields("Smart Promotion fee")
	smartPromotionPeriod := sumFields("Smart Promotion campaign period fee")
	taxSales := sumFields("Sales tax")
	taxSalesPayment := sumFields("Sales tax payment")
	taxSalesRefund := sumFields("Sales tax refund")
	taxRetailDelivery := sumFields("Retail delivery fee")
	taxRetailPayment := sumFields("Retail delivery fee payment")
	taxRetailRefund := sumFields("Retail delivery fee refund")
	factor := "CASE WHEN s.currency = 'CNY' THEN 1 ELSE NULLIF(fx.rate, 0) END"
	if currency == "USD" {
		factor = "CASE WHEN s.currency = 'USD' THEN 1 ELSE (" + factor + ") / NULLIF(usd.rate, 0) END"
	}
	// Read the latest order once per order ID instead of once per settlement.
	// Materialize parsed amounts below so each aggregate reuses the numeric values.
	matched := db.Table("tik_tok_settlements s").
		Joins(`LEFT JOIN (SELECT DISTINCT ON (order_id) order_id, data FROM tik_tok_orders WHERE order_id <> '' ORDER BY order_id, id DESC) o ON BTRIM(s.data->>'Type') = 'Order' AND o.order_id = BTRIM(s.data->>'Order/adjustment ID')`).
		Joins(taxRateJoinSQL("fx", "s.statement_date", "s.currency")).
		Joins(taxRateJoinSQL("usd", "s.statement_date", "'USD'")).
		Select("s.quarter, s.data, " + salesAmount + " AS sales_amount, " + refundAmount + " AS refund_amount, " + campaignAmount + " AS campaign_amount, " + commissionCampaignAmount + " AS commission_campaign_amount, " + smartPromotionAmount + " AS smart_promotion_amount, " + promotionAmount + " AS promotion_amount, " + otherFeesAmount + " AS other_fees_amount, " + shippingAmount + " AS shipping_amount, " + taxAmount + " AS tax_amount, " + otherFeeSignature + " AS other_fee_signature, " + otherFeeCofunded + " AS other_fee_cofunded, " + otherFeeProtection + " AS other_fee_protection, " + otherFeeNegativeDeduction + " AS other_fee_negative_deduction, " + otherFeeNegativeSales + " AS other_fee_negative_sales, " + campaignServiceItem + " AS campaign_service_item, " + additionalCampaignPackage + " AS additional_campaign_package, " + campaignPackage + " AS campaign_package, " + smartPromotionItem + " AS smart_promotion_item, " + smartPromotionPeriod + " AS smart_promotion_period, " + taxSales + " AS tax_sales, " + taxSalesPayment + " AS tax_sales_payment, " + taxSalesRefund + " AS tax_sales_refund, " + taxRetailDelivery + " AS tax_retail_delivery, " + taxRetailPayment + " AS tax_retail_payment, " + taxRetailRefund + " AS tax_retail_refund, " + tikTokSettlementLogisticsSQL + " AS logistics_method, " + factor + " AS factor")
	if strings.TrimSpace(store) != "" {
		matched = matched.Where("(s.store_id = ? OR s.store_id = '')", store)
	}
	matched = filterTaxSite(matched, site)
	return db.Raw(`WITH matched AS (?), selected AS (
	 SELECT quarter, logistics_method, factor, sales_amount, refund_amount, campaign_amount, commission_campaign_amount, smart_promotion_amount, promotion_amount, other_fees_amount, shipping_amount, tax_amount, other_fee_signature, other_fee_cofunded, other_fee_protection, other_fee_negative_deduction, other_fee_negative_sales, campaign_service_item, additional_campaign_package, campaign_package, smart_promotion_item, smart_promotion_period, tax_sales, tax_sales_payment, tax_sales_refund, tax_retail_delivery, tax_retail_payment, tax_retail_refund,
   data->>'Referral fee' AS raw_commission,
   data->>'Refund administration fee' AS raw_commission_refund,
   data->>'Actual return shipping fee' AS raw_return_shipping,
   data->>'Customer shipping fee offset' AS raw_customer_shipping_offset,
   data->>'FBT fulfillment fee reimbursement' AS raw_fbt_reimbursement,
   data->>'Customer shipping fee offset' AS raw_shipping_customer_offset,
   data->>'Platform shipping fee discount' AS raw_shipping_platform_discount,
   data->>'Customer shipping fee' AS raw_shipping_customer,
   data->>'Refunded customer shipping fee' AS raw_shipping_refunded_customer,
   CASE logistics_method
     WHEN 'MCF' THEN COALESCE(NULLIF(BTRIM(data->>'Seller shipping fee'), ''), data->>'Shipping fee')
     WHEN 'FBT' THEN data->>'FBT fulfillment fee'
     WHEN '海外仓' THEN data->>'TikTok Shop shipping fee'
     ELSE ''
   END AS raw_commission_shipping,
   data->>'Affiliate Commission' AS raw_affiliate_direct,
   data->>'Affiliate partner commission' AS raw_affiliate_partner,
   data->>'Affiliate Shop Ads commission' AS raw_affiliate_shop_ads,
   data->>'Affiliate commission deposit' AS raw_affiliate_deposit,
   data->>'Affiliate commission refund' AS raw_affiliate_refund,
   data->>'Affiliate Partner shop ads commission' AS raw_affiliate_partner_shop_ads,
   data->>'FBT fulfillment fee reimbursement' AS raw_other_fbt_reimbursement,
   data->>'Return shipping fee reimbursement' AS raw_other_return_shipping,
   COALESCE(data->>'Shipping protection reimbursement', data->>'Protection service reimbursement') AS raw_other_shipping_protection,
   data->>'FBT overall merchant subsidy' AS raw_other_fbt_subsidy,
   data->>'Logistics reimbursement' AS raw_other_logistics,
   data->>'Platform reimbursement' AS raw_other_platform,
   CASE logistics_method
     WHEN 'MCF' THEN COALESCE(NULLIF(BTRIM(data->>'Seller shipping fee'), ''), data->>'Shipping fee')
     WHEN 'FBT' THEN data->>'FBT fulfillment fee'
     WHEN '海外仓' THEN data->>'TikTok Shop shipping fee'
   END AS raw_amount
 FROM matched
), amounts AS MATERIALIZED (
	 SELECT quarter, logistics_method, factor, sales_amount, refund_amount, campaign_amount, commission_campaign_amount, smart_promotion_amount, promotion_amount, other_fees_amount, shipping_amount, tax_amount, other_fee_signature, other_fee_cofunded, other_fee_protection, other_fee_negative_deduction, other_fee_negative_sales, campaign_service_item, additional_campaign_package, campaign_package, smart_promotion_item, smart_promotion_period, tax_sales, tax_sales_payment, tax_sales_refund, tax_retail_delivery, tax_retail_payment, tax_retail_refund, raw_commission, raw_commission_refund,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_amount, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_commission, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0)
   + COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_commission_shipping, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0)
   + commission_campaign_amount
   + COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_return_shipping, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0)
   + CASE WHEN logistics_method = 'FBT' AND COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_fbt_reimbursement, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) > 0 THEN
       COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_customer_shipping_offset, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0)
       + COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_fbt_reimbursement, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0)
     ELSE 0 END AS commission_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_commission, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS commission_referral_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_commission_shipping, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS commission_shipping_amount,
   commission_campaign_amount AS commission_campaign_component_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_return_shipping, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS commission_return_shipping_amount,
   CASE WHEN logistics_method = 'FBT' AND COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_fbt_reimbursement, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) > 0 THEN COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_customer_shipping_offset, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) ELSE 0 END AS commission_fbt_shipping_offset_amount,
   CASE WHEN logistics_method = 'FBT' AND COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_fbt_reimbursement, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) > 0 THEN COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_fbt_reimbursement, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) ELSE 0 END AS commission_fbt_reimbursement_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_commission_refund, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS commission_refund_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_shipping_customer_offset, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS shipping_customer_offset_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_shipping_platform_discount, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS shipping_platform_discount_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_shipping_customer, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS shipping_customer_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_shipping_refunded_customer, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS shipping_refunded_customer_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_affiliate_direct, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS affiliate_direct_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_affiliate_partner, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS affiliate_partner_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_affiliate_shop_ads, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS affiliate_shop_ads_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_affiliate_deposit, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS affiliate_deposit_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_affiliate_refund, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS affiliate_refund_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_affiliate_partner_shop_ads, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS affiliate_partner_shop_ads_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_other_fbt_reimbursement, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS other_fbt_reimbursement_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_other_return_shipping, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS other_return_shipping_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_other_shipping_protection, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS other_shipping_protection_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_other_fbt_subsidy, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS other_fbt_subsidy_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_other_logistics, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS other_logistics_amount,
   COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(raw_other_platform, ''), '[$,\s]', '', 'g'), '^\((.*)\)$', '-\1'), '')::numeric, 0) AS other_platform_amount
 FROM selected
), converted AS (
	 SELECT quarter, logistics_method,
	   CASE WHEN sales_amount = 0 THEN 0 ELSE sales_amount * factor END AS sales_value,
	   CASE WHEN sales_amount <> 0 AND factor IS NULL THEN 1 ELSE 0 END AS sales_missing,
	   CASE WHEN refund_amount = 0 THEN 0 ELSE refund_amount * factor END AS refund_value,
	   CASE WHEN refund_amount <> 0 AND factor IS NULL THEN 1 ELSE 0 END AS refund_missing,
	   CASE WHEN other_fees_amount = 0 THEN 0 ELSE other_fees_amount * factor END AS other_fees_value,
	   CASE WHEN other_fees_amount <> 0 AND factor IS NULL THEN 1 ELSE 0 END AS other_fees_missing,
	   CASE WHEN shipping_amount = 0 THEN 0 ELSE shipping_amount * factor END AS shipping_value,
	   CASE WHEN shipping_amount <> 0 AND factor IS NULL THEN 1 ELSE 0 END AS shipping_missing,
   shipping_customer_offset_amount * factor AS shipping_customer_offset_value,
   shipping_platform_discount_amount * factor AS shipping_platform_discount_value,
   shipping_customer_amount * factor AS shipping_customer_value,
   shipping_refunded_customer_amount * factor AS shipping_refunded_customer_value,
	   CASE WHEN campaign_amount = 0 THEN 0 ELSE campaign_amount * factor END AS campaign_value,
	   CASE WHEN campaign_amount <> 0 AND factor IS NULL THEN 1 ELSE 0 END AS campaign_missing,
	   CASE WHEN smart_promotion_amount = 0 THEN 0 ELSE smart_promotion_amount * factor END AS smart_promotion_value,
	   CASE WHEN smart_promotion_amount <> 0 AND factor IS NULL THEN 1 ELSE 0 END AS smart_promotion_missing,
	   CASE WHEN promotion_amount = 0 THEN 0 ELSE promotion_amount * factor END AS promotion_value,
	   CASE WHEN promotion_amount <> 0 AND factor IS NULL THEN 1 ELSE 0 END AS promotion_missing,
   CASE WHEN tax_amount = 0 THEN 0 ELSE tax_amount * factor END AS tax_value,
   CASE WHEN tax_amount <> 0 AND factor IS NULL THEN 1 ELSE 0 END AS tax_missing,
   other_fee_signature * factor AS other_fee_signature_value,
   other_fee_cofunded * factor AS other_fee_cofunded_value,
   other_fee_protection * factor AS other_fee_protection_value,
   other_fee_negative_deduction * factor AS other_fee_negative_deduction_value,
   other_fee_negative_sales * factor AS other_fee_negative_sales_value,
   campaign_service_item * factor AS campaign_service_item_value,
   additional_campaign_package * factor AS additional_campaign_package_value,
   campaign_package * factor AS campaign_package_value,
   smart_promotion_item * factor AS smart_promotion_item_value,
   smart_promotion_period * factor AS smart_promotion_period_value,
   tax_sales * factor AS tax_sales_value,
   tax_sales_payment * factor AS tax_sales_payment_value,
   tax_sales_refund * factor AS tax_sales_refund_value,
   tax_retail_delivery * factor AS tax_retail_delivery_value,
   tax_retail_payment * factor AS tax_retail_payment_value,
   tax_retail_refund * factor AS tax_retail_refund_value,
   CASE WHEN amount = 0 THEN 0 ELSE amount * factor END AS value,
   CASE WHEN amount <> 0 AND factor IS NULL THEN 1 ELSE 0 END AS missing,
   CASE WHEN commission_amount = 0 THEN 0 ELSE commission_amount * factor END AS commission_value,
   CASE WHEN commission_refund_amount = 0 THEN 0 ELSE commission_refund_amount * factor END AS commission_refund_value,
   CASE WHEN commission_referral_amount = 0 THEN 0 ELSE commission_referral_amount * factor END AS commission_referral_value,
   CASE WHEN commission_shipping_amount = 0 THEN 0 ELSE commission_shipping_amount * factor END AS commission_shipping_value,
   CASE WHEN commission_campaign_component_amount = 0 THEN 0 ELSE commission_campaign_component_amount * factor END AS commission_campaign_component_value,
   CASE WHEN commission_return_shipping_amount = 0 THEN 0 ELSE commission_return_shipping_amount * factor END AS commission_return_shipping_value,
   CASE WHEN commission_fbt_shipping_offset_amount = 0 THEN 0 ELSE commission_fbt_shipping_offset_amount * factor END AS commission_fbt_shipping_offset_value,
   CASE WHEN commission_fbt_reimbursement_amount = 0 THEN 0 ELSE commission_fbt_reimbursement_amount * factor END AS commission_fbt_reimbursement_value,
   CASE WHEN affiliate_direct_amount = 0 THEN 0 ELSE affiliate_direct_amount * factor END AS affiliate_direct_value,
   CASE WHEN affiliate_partner_amount = 0 THEN 0 ELSE affiliate_partner_amount * factor END AS affiliate_partner_value,
   CASE WHEN affiliate_shop_ads_amount = 0 THEN 0 ELSE affiliate_shop_ads_amount * factor END AS affiliate_shop_ads_value,
   CASE WHEN affiliate_deposit_amount = 0 THEN 0 ELSE affiliate_deposit_amount * factor END AS affiliate_deposit_value,
   CASE WHEN affiliate_refund_amount = 0 THEN 0 ELSE affiliate_refund_amount * factor END AS affiliate_refund_value,
   CASE WHEN affiliate_partner_shop_ads_amount = 0 THEN 0 ELSE affiliate_partner_shop_ads_amount * factor END AS affiliate_partner_shop_ads_value,
   CASE WHEN (affiliate_direct_amount <> 0 OR affiliate_partner_amount <> 0 OR affiliate_shop_ads_amount <> 0 OR affiliate_deposit_amount <> 0 OR affiliate_refund_amount <> 0 OR affiliate_partner_shop_ads_amount <> 0) AND factor IS NULL THEN 1 ELSE 0 END AS affiliate_missing,
   CASE WHEN (commission_amount <> 0 OR commission_refund_amount <> 0) AND factor IS NULL THEN 1 ELSE 0 END AS commission_missing,
   CASE WHEN (other_fbt_reimbursement_amount <> 0 OR other_return_shipping_amount <> 0 OR other_shipping_protection_amount <> 0 OR other_fbt_subsidy_amount <> 0 OR other_logistics_amount <> 0 OR other_platform_amount <> 0) AND factor IS NULL THEN 1 ELSE 0 END AS other_income_missing,
   CASE WHEN other_fbt_reimbursement_amount = 0 THEN 0 ELSE other_fbt_reimbursement_amount * factor END AS other_fbt_reimbursement_value,
   CASE WHEN other_return_shipping_amount = 0 THEN 0 ELSE other_return_shipping_amount * factor END AS other_return_shipping_value,
   CASE WHEN other_shipping_protection_amount = 0 THEN 0 ELSE other_shipping_protection_amount * factor END AS other_shipping_protection_value,
   CASE WHEN other_fbt_subsidy_amount = 0 THEN 0 ELSE other_fbt_subsidy_amount * factor END AS other_fbt_subsidy_value,
   CASE WHEN (other_logistics_amount + other_platform_amount) = 0 THEN 0 ELSE (other_logistics_amount + other_platform_amount) * factor END AS other_settlement_value
 FROM amounts
)
	SELECT quarter,
	 CASE WHEN SUM(sales_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(sales_value), 0), 2) END AS sales_amount,
	 CASE WHEN SUM(sales_missing) > 0 THEN NULL ELSE COALESCE(SUM(sales_value), 0) END AS sales_unrounded,
	 CASE WHEN SUM(refund_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(refund_value), 0), 2) END AS settlement_refund_subtotal,
	 CASE WHEN SUM(other_fees_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(other_fees_value), 0), 2) END AS other_fees,
	 ROUND(COALESCE(SUM(other_fee_signature_value), 0), 2) AS other_fee_signature,
	 ROUND(COALESCE(SUM(other_fee_cofunded_value), 0), 2) AS other_fee_cofunded,
	 ROUND(COALESCE(SUM(other_fee_protection_value), 0), 2) AS other_fee_protection,
	 ROUND(COALESCE(SUM(other_fee_negative_deduction_value), 0), 2) AS other_fee_negative_deduction,
	 ROUND(COALESCE(SUM(other_fee_negative_sales_value), 0), 2) AS other_fee_negative_sales,
	 CASE WHEN SUM(shipping_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(shipping_value), 0), 2) END AS settlement_shipping_fee,
	 CASE WHEN SUM(shipping_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(shipping_value), 0) * -1, 2) END AS shipping_fee,
	 ROUND(COALESCE(SUM(shipping_customer_offset_value), 0), 2) AS shipping_customer_offset,
	 ROUND(COALESCE(SUM(shipping_platform_discount_value), 0), 2) AS shipping_platform_discount,
	 ROUND(COALESCE(SUM(shipping_customer_value), 0), 2) AS shipping_customer,
	 ROUND(COALESCE(SUM(shipping_refunded_customer_value), 0), 2) AS shipping_refunded_customer,
	 CASE WHEN SUM(campaign_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(campaign_value), 0), 2) END AS campaign_service_fee,
	 ROUND(COALESCE(SUM(campaign_service_item_value), 0), 2) AS campaign_service_item,
	 ROUND(COALESCE(SUM(additional_campaign_package_value), 0), 2) AS additional_campaign_package,
	 ROUND(COALESCE(SUM(campaign_package_value), 0), 2) AS campaign_package,
	 CASE WHEN SUM(smart_promotion_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(smart_promotion_value), 0), 2) END AS smart_promotion_fee,
	 ROUND(COALESCE(SUM(smart_promotion_item_value), 0), 2) AS smart_promotion_item,
	 ROUND(COALESCE(SUM(smart_promotion_period_value), 0), 2) AS smart_promotion_period,
	 CASE WHEN SUM(promotion_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(promotion_value), 0), 2) END AS promotion_fee,
 CASE WHEN SUM(tax_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(tax_value), 0), 2) END AS tax,
	 ROUND(COALESCE(SUM(tax_sales_value), 0), 2) AS tax_sales,
	 ROUND(COALESCE(SUM(tax_sales_payment_value), 0), 2) AS tax_sales_payment,
	 ROUND(COALESCE(SUM(tax_sales_refund_value), 0), 2) AS tax_sales_refund,
	 ROUND(COALESCE(SUM(tax_retail_delivery_value), 0), 2) AS tax_retail_delivery,
	 ROUND(COALESCE(SUM(tax_retail_payment_value), 0), 2) AS tax_retail_payment,
	 ROUND(COALESCE(SUM(tax_retail_refund_value), 0), 2) AS tax_retail_refund,
 CASE WHEN SUM(missing) FILTER (WHERE logistics_method = 'MCF') > 0 THEN NULL ELSE ROUND(COALESCE(SUM(value) FILTER (WHERE logistics_method = 'MCF'), 0), 2) END AS delivery_mcf,
 CASE WHEN SUM(missing) FILTER (WHERE logistics_method = 'FBT') > 0 THEN NULL ELSE ROUND(COALESCE(SUM(value) FILTER (WHERE logistics_method = 'FBT'), 0), 2) END AS delivery_fbt,
 CASE WHEN SUM(missing) FILTER (WHERE logistics_method = '海外仓') > 0 THEN NULL ELSE ROUND(COALESCE(SUM(value) FILTER (WHERE logistics_method = '海外仓'), 0), 2) END AS delivery_overseas,
 COUNT(*) FILTER (WHERE logistics_method = '') AS delivery_unmatched_rows,
 CASE WHEN SUM(commission_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(commission_value), 0), 2) END AS settlement_commission,
 CASE WHEN SUM(commission_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(commission_refund_value), 0), 2) END AS settlement_commission_refund,
 CASE WHEN SUM(commission_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(commission_value + commission_refund_value), 0), 2) END AS settlement_commission_total,
 CASE WHEN SUM(commission_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(commission_referral_value), 0), 2) END AS settlement_commission_referral,
 CASE WHEN SUM(commission_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(commission_shipping_value), 0), 2) END AS settlement_commission_shipping,
 CASE WHEN SUM(commission_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(commission_campaign_component_value), 0), 2) END AS settlement_commission_campaign,
 CASE WHEN SUM(commission_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(commission_return_shipping_value), 0), 2) END AS settlement_commission_return_shipping,
 CASE WHEN SUM(commission_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(commission_fbt_shipping_offset_value), 0), 2) END AS settlement_commission_fbt_shipping_offset,
 CASE WHEN SUM(commission_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(commission_fbt_reimbursement_value), 0), 2) END AS settlement_commission_fbt_reimbursement,
 CASE WHEN SUM(affiliate_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(affiliate_direct_value + affiliate_partner_value + affiliate_shop_ads_value + affiliate_deposit_value + affiliate_refund_value + affiliate_partner_shop_ads_value), 0), 2) END AS affiliate_commission,
 CASE WHEN SUM(affiliate_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(affiliate_direct_value), 0), 2) END AS affiliate_direct,
 CASE WHEN SUM(affiliate_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(affiliate_partner_value), 0), 2) END AS affiliate_partner,
 CASE WHEN SUM(affiliate_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(affiliate_shop_ads_value), 0), 2) END AS affiliate_shop_ads,
 CASE WHEN SUM(affiliate_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(affiliate_deposit_value), 0), 2) END AS affiliate_deposit,
 CASE WHEN SUM(affiliate_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(affiliate_refund_value), 0), 2) END AS affiliate_refund,
 CASE WHEN SUM(affiliate_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(affiliate_partner_shop_ads_value), 0), 2) END AS affiliate_partner_shop_ads,
 CASE WHEN SUM(other_income_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(other_fbt_reimbursement_value + other_return_shipping_value + other_shipping_protection_value + other_fbt_subsidy_value + other_settlement_value), 0), 2) END AS settlement_other_income,
 CASE WHEN SUM(other_income_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(other_fbt_reimbursement_value), 0), 2) END AS other_income_fbt_reimbursement,
 CASE WHEN SUM(other_income_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(other_return_shipping_value), 0), 2) END AS other_income_return_shipping,
 CASE WHEN SUM(other_income_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(other_shipping_protection_value), 0), 2) END AS other_income_shipping_protection,
 CASE WHEN SUM(other_income_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(other_fbt_subsidy_value), 0), 2) END AS other_income_fbt_subsidy,
 CASE WHEN SUM(other_income_missing) > 0 THEN NULL ELSE ROUND(COALESCE(SUM(other_settlement_value), 0), 2) END AS other_income_settlement,
 SUM(missing) AS missing_rates
FROM converted GROUP BY quarter`, matched)
}
