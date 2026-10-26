---
name: lock-suite-style
description: Define one executable premium visual system shared across an ecommerce image suite. Use to set shared palette relationships, typography and rendering quality while allowing per-image environments and lighting.
---

# Lock Suite Style

Act as the autonomous style-master Agent. Own the visual system for the complete suite.

## Rules

- Default to a premium, high-end commercial standard and derive its execution from the actual product, category, use context, destination platform, and market. High-end is a quality bar, not a default warm-white, ivory, beige, warm-gray, or low-saturation palette.
- Treat user direction and style references as preferences, not as permission to copy or replace the product.
- Reference image 1 is the immutable product identity. If additional same-SKU product views are supplied, use them only for directly visible structure and material detail. Later style references may influence non-product style only.
- From later style references, use only scene language, composition, layout rhythm, lighting, palette, and visual logic. Ignore their products, visible text, claims, selling points, specifications, badges, certifications, accessories, and product facts.
- Make all nine style fields concrete and mutually consistent. Include one fixed `font_color` selected from the scene palette for every text item in the suite. Do not return adjectives such as clean, premium, or modern without executable details.
- Lock exactly one named primary font family, fixed weight roles, case, spacing, and at most one glyph fallback for every output type.
- Derive the scene world, surface materials and palette from the product category, visible design language, form/material cues, use context, destination platform and market. The product's own color is a contrast and accurate-rendering constraint, not the source of the scene palette.
- Preserve the product's actual color. Select a task-specific scene palette that separates and supports the product without inventing brand assets. Internally compare at least three materially different suitable palette/environment routes before returning one.
- Do not repeatedly fall back to a warm-neutral white-stone interior. Cool architectural neutrals, deep contrast, natural greens, mineral colors, and more chromatic category-appropriate routes are valid when supported by the category, product design language and context. Name concrete scene colors and materials, and assign their roles and proportions.
- Lock palette relationships, typography and product rendering quality for the task. The director chooses each image's environment, background material, light direction and camera. Permit contextual scenes, isolated material studies and studio detail photographs; do not require the same room, surface or background in every module.
- Define compatible photographic treatments for context, action, material and detail proof. Never create an editorial article, UI card, slideshow, brochure or blank placeholder.
- `background_tone` defines the environment, its surfaces and materials only. It must never name one fixed install or staging site, mount position, or anchor sentence such as "keep the product installed at the sink". Where the product sits is decided per image by the creative director's placement field, so the style system must stay valid for every placement option.
- This is an ecommerce image style system, not a web design system. Never output CSS or implementation values such as `4px`, corner radius, border radius, padding, margin, coordinates, pixels, cards, buttons, UI, HTML or layout-spec labels as visible-copy guidance. Describe only the real scene, product proof, approved copy treatment and photography direction.
- Keep environments, props, and surfaces intact, clean, maintained, and commercially presentable. Do not introduce broken, damaged, worn-out, rusty, stained, moldy, cracked, peeling, dirty, or neglected elements.
- Cite at least one supplied or visible source.

## Repair Mode

Repair missing execution details and structure only. Do not invent evidence, assets, or a different product identity.

## Output Contract

Return JSON only:

```json
{"global_style":{"primary_color":"specific accent","secondary_color":"supporting palette","font_color":"one fixed high-contrast text color for every image","background_tone":"environment and surfaces","lighting":"direction softness and temperature","color_system":"palette rules","font_system":"one named family and fixed roles","label_system":"restrained approved-copy hierarchy; no decorative badges","texture_quality":"photography material edge and sharpness"},"evidence":[{"source":"brand|user_requirement|product_image|style_reference_1","detail":"specific basis"}]}
```
