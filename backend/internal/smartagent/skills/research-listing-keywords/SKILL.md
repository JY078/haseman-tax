---
name: research-listing-keywords
description: Research marketplace search phrases for Listing generation.
---
# Listing Keyword Research

Research high-intent category search phrases from the AMZ123 US top-keywords search page using the official web search tool. The caller supplies an exact URL in the form `https://www.amz123.com/usatopkeywords/1?k=<URL-encoded product category>`; open/search that URL, then extract the first 20 ranked keywords returned for that category query, preserving order. This agent performs category research only: use product category for the query, not the SKU's product facts, images, specifications or product name. Return broad category phrases, even when some may later be filtered out for this SKU. The Listing Writer performs product-fact filtering later. Combine operator-provided keywords after the AMZ123 results as a separately labeled source. Return JSON only:

`{"keywords":[{"term":"","source":"operator|marketplace|autocomplete|product_title","tags":[],"reason":"","platform_frequency":0,"search_volume":0,"competition":0,"evidence_urls":[]}]}`

Do not invent search volume or rankings. Copy the first 20 AMZ123 terms exactly as displayed; when 20 rows are available, returning fewer than 20 is an error. Remove only duplicate rows that are literally repeated. Return the 20 AMZ123 phrases plus operator terms.
