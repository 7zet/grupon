# Architecture (plain language)

Dealo is a marketplace for local deals in Uzbekistan. Buyers purchase vouchers, merchants publish deals, staff check vouchers at the counter, and platform admins approve deals and handle reports.

## The pieces

- **Web app (`apps/web`)**: what people see in the browser. Next.js renders the public site, the buyer account, the merchant panel and the moderation pages. It talks to the API through a small same-origin proxy, so the browser never holds the access token directly.
- **API (`services/api`)**: one Go program that answers all requests. It checks who is calling, applies the business rules and reads or writes the database.
- **Worker (`services/api/cmd/worker`)**: a second Go program that runs scheduled housekeeping: releasing unpaid stock reservations, expiring old vouchers, moving deals between statuses and recalculating ratings and sold counts. Every job is safe to run twice.
- **Database (PostgreSQL)**: the single source of truth for users, merchants, deals, orders, vouchers, reviews and the audit log.
- **Cache (Redis, optional)**: keeps public catalog pages fast and holds rate-limit counters. The system works without it, with rate limits off.
- **Media storage**: merchant photos are stored on a disk folder behind an adapter, so a cloud bucket can replace it later.
- **Mock payments**: a stand-in for a real payment provider. It calls the same webhook a real provider would.

## How a purchase works

1. A visitor browses deals. No login is needed.
2. At checkout the visitor logs in, and the selected deal option is preserved without a cart.
3. The API creates an order and reserves stock for 15 minutes. The reservation is one conditional database update, so two buyers can never take the last unit.
4. The payment succeeds through the webhook. The order becomes paid, stock is sold, and one voucher is issued per unit.
5. If payment fails, the order stays pending until the reservation expires, and the buyer can retry.

## How a deal goes live

1. A merchant creates a draft with options, prices, photos and locations.
2. Submitting it moves it to `in_review`.
3. A moderator approves it (it becomes active or scheduled) or requests changes with a reason.
4. Changing the price, terms or options of a live deal sends it back to review. Changing only the stock does not.

## How a voucher is used

1. Staff scan the voucher's QR code or type its short code.
2. The API checks that the voucher belongs to their business and shows its state: valid, already used, expired, refunded or cancelled.
3. Confirming marks it used with one conditional update, so it can be used once even if two counters try at the same moment. The action is audited.

The QR code carries a signed token, never the raw voucher id, so it cannot be forged.

## Who can do what

- **Buyer**: buy, view own vouchers, review after using a voucher, report a review.
- **Merchant owner and manager**: edit business, locations, deals and media, redeem vouchers.
- **Merchant cashier**: redeem vouchers only. Edit requests are refused by the API.
- **Moderator**: approve or reject deals, handle review reports, hide reviews.
- **Admin and superadmin**: the same as moderator, plus reading the audit log.

Each rule is enforced twice: the HTTP layer checks the caller, and the business layer checks the action again.

## Code layout

Each feature follows the same three layers:

1. **HTTP handlers** (`services/api/internal/http`): read the request, call the domain, write the response. No business rules here.
2. **Domain** (`services/api/internal/domain`): the rules, written in plain Go with no database code.
3. **Store** (`services/api/internal/store`): the database queries, generated from SQL files by sqlc.

The API contract lives in `packages/api-spec/openapi.yaml`. Go server types and TypeScript client types are generated from it, so the contract is the first thing to change.

## Money and time

- Amounts are whole numbers of the smallest unit (UZS). No floating-point money anywhere.
- UZS is the only supported currency.
- All times are stored in UTC and shown in the buyer's local time.

## Audit

Every admin action and every sensitive merchant action (deal changes, voucher redemption) writes a row to `audit_logs` in the same transaction as the change. If the audit row cannot be written, the change does not happen.

## Deployment

- Three container images: `api`, `worker` and `migrate` (Go), plus `web` (Node). See `services/api/Dockerfile` and `apps/web/Dockerfile`.
- Run `migrate` once per release, before starting the new API and worker.
- `seed` is for development only and refuses to run in production.
- Settings are environment variables. See `docs/ENVIRONMENT.md`.

## Known limits

- Media is stored on a local disk; a shared object store is needed before running more than one API instance.
- The mock payment provider is the only one; real providers are out of scope for now.
- Support tickets and system settings screens are not built; finance dispute handling exists, but the full PRD dispute transition chain is incomplete.
- The admin screens currently live inside the web app; they will move to their own app.
