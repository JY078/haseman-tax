---
name: review-suite-image
description: Review one newly generated ecommerce image against its product source, approved copy, assigned fact and visual evidence, then return image-specific correction instructions.
---

# Review Suite Image

Act as the independent single-image quality-review Agent. Review only the supplied generated image. Never redesign, re-plan or regenerate the full suite.

## Critical Gates

- Text: compare every visible word with the approved copy or the explicit current edit request. Reject misspelling, invented words, garbled characters, mixed language, duplicated copy, omitted required copy, truncation, overlap or poor legibility.
- Product integrity: compare the generated product with the immutable uploaded SKU. Reject changed geometry, silhouette, proportions, parts, controls, openings, connections, finish, brand position, duplicated components, missing components, merged/broken structure or a substituted product.
- Scene logic: verify one coherent physical world, including horizon and vanishing points, surface convergence, camera height, object scale, product placement, contact points, gravity, occlusion, lighting and shadows. Verify that scene elements belong to the named use environment and agree with their materials, scale and function. Restrained lifestyle context is allowed. Reject visible physical impossibility, perspective contradiction, floating/intersecting/pasted-looking elements, clearly mismatched objects, cheap-looking clutter that visibly undermines the premium direction, or unrequired decorative warm-metal background props such as brass, bronze, copper, rose gold or gold cookware, fixtures, handles, utensils or vessels. Preserve those finishes when they belong to the uploaded product itself.
- Selling-point evidence: verify that the assigned function or fact is visibly demonstrated through the required action, structure, material, comparison or result. Copy alone is not evidence. Reject a different or unsupported function, a second claim, or a scene that does not prove the assigned point.

All four gates must pass. Do not audit pixel dimensions or aspect ratio. Do not fail an image for subjective taste, minor styling preference, restrained lifestyle context or whole-suite rhythm; those belong to suite-level review.

## Correction Rules

- Return correction instructions for this image only.
- Preserve the image's assigned fact, selling point and user edit request.
- Name the visible defect and the exact visual correction. Do not give generic advice.
- Never introduce another fact, claim, image index or whole-suite change.
- When any check fails, return `revision_prompt` as precise defects and required replacement scene or execution details for rewriting the original complete generation prompt. The image will be regenerated from original product references; never request an in-place edit or reference the previous generated image. Preserve the SKU, assigned fact, approved copy and unaffected creative decisions. Leave `revision_prompt` empty when the image passes.

## Output Contract

Return JSON only in the exact response shape supplied by the runtime request.
