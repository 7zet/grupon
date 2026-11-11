# Architecture (plain language)

Dealo is a marketplace for local deals in Tashkent. Buyers purchase vouchers, businesses publish deals, cashiers
check vouchers at the counter, and platform staff approve deals and businesses and handle money and disputes.

## The pieces

- **Web app (`apps/web`)**: everything people see in the browser (Next.js). It has four workspaces, each with its
  own layout (route groups under `apps/web/src/app/[locale]`):
  - `(shop)`: the buyer marketplace and the buyer account;
  - `(business)/merchant`: **Dealo Business** for owners and managers, or **Dealo Kassa** for a cashier;
  - `(admin)/admin`: **Dealo Admin** for moderator, support, finance and superadmin.
- **Same-origin routes in the web app**: the browser never calls the Go API directly (the API sends no CORS headers).
  `/api/auth/*` signs in and stores the session cookie, `/api/bff/*` forwards signed-in requests (an allow-list of
  operations), `/api/suggest` forwards public search suggestions, `/api/revalidate` lets the API refresh cached pages.
- **API (`services/api/cmd/api`)**: one Go program that answers all requests. It checks who is calling, applies the
  business rules and reads or writes the database. In the local demo it also runs the worker
  (`RUN_WORKER_IN_API=true`).
- **Worker (`services/api/cmd/worker`)**: every 30 seconds it releases unpaid stock reservations, expires old
  vouchers, moves deals between statuses, books unused-voucher breakage and runs a ledger reconciliation check.
  Every job is safe to run twice.
- **Database (PostgreSQL)**: the single source of truth for users, businesses, deals, orders, vouchers, the ledger,
  reviews and the audit log. Migrations are in `services/api/migrations`.
- **Redis (optional)**: a response cache for the catalog and the counters for sign-in rate limits. Without it the
  system works and rate limits are off; `APP_ENV=production` requires it.
- **Media storage**: photos and business documents are stored in a folder on the API machine behind an adapter, so
  an object store can replace it later.
- **Mock payments**: a stand-in for a real payment provider. It sends the same signed webhook a real provider would.

## How a purchase works

1. A visitor browses and searches deals (by name, by Uzbek or Russian spelling, or by street address). No login.
2. "Buy now" keeps the chosen deal, option and quantity; the visitor signs in with a phone code and returns to checkout.
3. The API creates an order and reserves stock for 15 minutes. The reservation is one conditional database update, so
   two buyers can never take the last unit.
4. The payment webhook marks the order paid, sells the stock, issues one voucher per unit and writes the ledger.
5. If payment fails, the buyer can retry; an unpaid reservation is released when it expires.

## How a deal goes live

1. A business owner or manager creates a draft with options, prices, photos and locations (a six-step wizard).
   Submitting needs a price proof document on the business.
2. Submitting moves the deal to `in_review`.
3. A moderator approves it (active or scheduled) or rejects it with a reason the business sees.
4. Changing the price, terms or options of a live deal sends it back to review. Changing only the stock does not.
   A live deal can be paused, resumed and archived.

A new business also goes through review: it registers, uploads documents, and a moderator approves or rejects it.
Superadmin can suspend, terminate and reinstate a business.

## How a voucher is used

1. The cashier scans the voucher's QR code or types its code in Dealo Kassa.
2. The API checks that the voucher belongs to their business and shows its state: valid, used, expired, refunded or
   on hold.
3. Confirming marks it used with one conditional update, so it is used once even if two counters try at once.
   The cashier can undo their own redemption within 10 minutes; after that only support can reverse it.

The QR code carries a signed token that changes every 30 seconds, never the raw voucher id, so it cannot be forged
and a screenshot goes stale.

## Who can do what

| Role | Workspace and home | Can do |
| --- | --- | --- |
| Buyer (`user`) | Marketplace, `/` | Buy, own vouchers and orders, refunds, reviews, report a review |
| Owner (`merchant_owner`) | Dealo Business, `/merchant/dashboard` | Everything for the business: deals, media, orders, customers, analytics, reviews, calculator, redeem, business data, staff, payouts, settings |
| Manager (`merchant_manager`) | Dealo Business, `/merchant/dashboard` | Same as owner without business data, staff, payouts and settings |
| Cashier (`merchant_cashier`) | Dealo Kassa, `/merchant/redeem` | Redeem vouchers and undo their own within 10 minutes |
| Moderator | Dealo Admin, `/admin/moderation` | Deal moderation, business review, fraud signals, review reports |
| Support | Dealo Admin, `/admin/finance` | Refunds and disputes on the finance desk (no payouts) |
| Finance | Dealo Admin, `/admin/finance` | Payouts (prepare or approve, never both for one payout), refunds, disputes |
| Superadmin (and `admin`) | Dealo Admin, `/admin` | Everything above, plus overview, business suspension and the audit log |

The section rules live in `apps/web/src/lib/roles.ts`; opening another role's page shows "No access". The API checks
every rule again, so the web check is only what the user sees. Staff roles sign in with a phone code plus an
authenticator code.

## Code layout

Each API feature follows the same three layers:

1. **HTTP handlers** (`services/api/internal/http`): read the request, call the domain, write the response.
2. **Domain** (`services/api/internal/domain`): the rules, in plain Go with no database code.
3. **Store** (`services/api/internal/store`): the database queries, generated from `services/api/queries/*.sql` by sqlc.

The API contract lives in `packages/api-spec/openapi.yaml`. Go server types and TypeScript client types are generated
from it (`go generate ./...` in `services/api`, `corepack pnpm --filter @dealo/api-client run gen`), so the contract
changes first. Translations are in `packages/i18n/messages/{uz,ru,en}.json`; design tokens in
`packages/ui/src/styles/tokens.css`.

## Money and time

- Amounts are whole numbers of tiyin (1/100 so'm). No floating-point money anywhere. UZS is the only currency.
- The ledger is double-entry and append-only; corrections are reversing entries.
- All times are stored in UTC and shown in the buyer's local time.

## Audit

Every admin action and every sensitive business action (deal changes, redemptions, reversals, payouts) writes a row
to `audit_logs` in the same transaction as the change. A database trigger makes `audit_logs` append-only.

## Running it

- The demo runs on one Windows PC: `scripts/demo.ps1` checks Postgres, migrates, seeds when empty, builds the web app
  and starts the API (with the worker) and the web app. See `README.md`.
- To show it to others, one ngrok HTTPS tunnel points at the web app (`docs/SHARE.md`).
- `cmd/bootstrap` migrates and seeds (demo only); `cmd/seed` refuses to run with `APP_ENV=production`.
- Settings are environment variables (`docs/ENVIRONMENT.md`). Dockerfiles exist but are not verified.

## Known limits

- Media is on the API machine's disk; an object store is needed before running more than one API instance.
- The mock payment provider is the only one; real providers are out of scope.
- Categories, cities, commission and users have no admin screens; they come from the seed.
- See `docs/HANDOVER.md` for what is not built.
