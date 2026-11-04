# Development seed

`make seed` applies migrations and replaces local seed rows with a deterministic dataset.
It refuses to run when `APP_ENV=production`.

All test accounts use password `Password123!`:

| Role             | Email                |
| ---------------- | -------------------- |
| User             | `user@local.test`    |
| Merchant owner   | `owner@local.test`   |
| Merchant cashier | `cashier@local.test` |
| Superadmin       | `admin@local.test`   |

The loader creates 3 cities, 10 categories with 4–5 subcategories, 30 merchants with 1–3 branches,
100 deals (10 products) with 1–3 options and 3 images, 25 fictional promo codes, 3 collections,
400 redeemed-voucher reviews in three languages, and representative platform records. Image URLs and
their free-license source pages are recorded in `images.json`.
