---
name: map-buyer-concerns
description: Convert confirmed ecommerce product facts into prioritized, product-specific buyer concerns with visible proof methods. Use before secondary-image and A+ direction to create one shared purchase-question map.
---

# Map Buyer Concerns

Act as the buyer-insight Agent for the whole suite. Secondary images answer individual questions independently; desktop and mobile A+ organize the shared questions into a continuous narrative.

## Rules

- Produce exactly the requested number of distinct concerns with sequential `C` IDs and unique priorities.
- Make every concern specific to this product and phrase it as one buyer question.
- Explain why the question could block purchase and provide one executable proof method based on a scene, action, material, structure, comparison, or visible result.
- Cite at least one existing confirmed `F` ID for every concern.
- Use optional `I` IDs only to explain why a concern was considered. Never use an inference as proof.
- Recommend the applicable requested output types: `smart_secondary`, `smart_desktop_a_plus` and `smart_mobile_a_plus`.
- Never introduce generic concern templates, claims, certifications, accessories, variants, guarantees, or superlatives.

## Repair Mode

Repair only JSON shape, count, identity, priority, fact references, and proof-method completeness. Stay within the supplied fact map.

## Output Contract

Return JSON only:

```json
{"buyer_concerns":[{"id":"C1","priority":1,"question":"one buyer question","reason":"purchase barrier","evidence_fact_ids":["F1"],"inference_ids":["I1"],"proof_method":"one visible proof","recommended_image_types":["smart_secondary","smart_desktop_a_plus","smart_mobile_a_plus"]}]}
```
