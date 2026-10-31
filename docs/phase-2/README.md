# Phase 2 verification

The public frontend uses OpenAPI-generated types and the shared API client. In Phase 2, MSW serves
deterministic catalog data expanded from `seed/data.json`; setting both `USE_MSW=false` and
`NEXT_PUBLIC_USE_MSW=false` switches the same client to the configured Go API.

## Required viewport evidence

Capture the first screen of `/uz/deal/beauty-and-spas-special-2` at `360x640`, `390x844`, and
`1440x900`. At each size the selected option, price, discount, savings, and red Buy button must be
visible without scrolling. Store the images in `docs/phase-2/screenshots/`.

Verified screenshots: [360×640](screenshots/deal-360x640.png),
[390×844](screenshots/deal-390x844.png), and [1440×900](screenshots/deal-1440x900.png).

## Performance budgets

`apps/web/lighthouserc.cjs` checks the homepage, a category page, and a deal page. Each must score at
least 90 performance and 95 accessibility, best practices, and SEO; LCP must be at most 2.5s, CLS
at most 0.1, and TBT at most 200ms.

The production audit ran three times per route. Observed ranges: performance 98–100,
accessibility 100, best practices 96, SEO 100, LCP 676–1,045ms, CLS 0, and TBT 0ms.
