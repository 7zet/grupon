# Demo accounts

Start the demo with `scripts\demo.ps1` (it loads the demo data on the first run; `-Reset` reloads it). The site is
at <http://localhost:3000/uz>, or at your ngrok address when shared (see `docs/SHARE.md`). In
`DEMO_MODE=true`, the six-digit OTP is shown on the login page. Staff accounts also
require the authenticator code documented in `docs/ENVIRONMENT.md`; do not use that
development factor outside your own PC.

| Account | Role / business | Phone | Demonstrates |
| --- | --- | --- | --- |
| Aziz — xaridor | Buyer | `+998900000001` | Browse, test payment, vouchers and refund/dispute entry points. |
| Zarina — xaridor | Buyer | `+998910000009` | A second independent buyer account. |
| Dilnoza — Lotus Spa egasi | Owner — Lotus Spa | `+998900000002` | Merchant dashboard, deals, staff and payout visibility. |
| Sherzod — Lotus Spa menejeri | Manager — Lotus Spa | `+998910000010` | Deal and business operations without owner-only access. |
| Bekzod — Lotus Spa kassiri (Chilonzor) | Cashier — Lotus Spa | `+998900000003` | Voucher lookup, redemption and own redemption history. |
| Nodira — Urban Grill kassiri | Cashier — Urban Grill | `+998900000009` | A cashier isolated to a second business. |
| Madina — moderator | Moderator | `+998910000011` | Deal moderation and review reports. |
| Umida — qo'llab-quvvatlash | Support | `+998910000012` | Customer support and dispute evidence/closure actions. |
| Farhod — moliya tayyorlovchi | Finance | `+998910000013` | Finance queue preparation; cannot self-approve. |
| Shahnoza — moliya tasdiqlovchi | Finance | `+998910000014` | Second finance identity for four-eyes approval. |
| Saida — superadmin | Superadmin | `+998900000004` | Administrative audit, moderation and demo-wide oversight. |

Every business also has its own staff: phone `+998912MMRPP`, where `MM` is the business number
(`01`-`30`, the `seed-merchant-MM` slug) and `R` is `1` owner, `2` manager, `3` cashier. For
example business 27 (Dolphin Swim Club): owner `+99891227100`, manager `+99891227200`, cashier
`+99891227300`. The login page does not list demo accounts; this table is the only list.

The extra seeded owner and administrator accounts exist for parallel automated tests;
they are not part of the recommended walkthrough.

## Recommended walkthrough

1. Sign in as Aziz, open a deal, use the demo payment and show the issued voucher.
2. Sign in as Bekzod, redeem that voucher and show redemption history.
3. Sign in as Dilnoza, show the Lotus Spa merchant dashboard and deal workflow.
4. Sign in as Madina, approve or reject a review/deal in moderation.
5. Sign in as Farhod and Shahnoza separately to demonstrate the two-person finance flow.
6. Finish as Saida to show the audit trail and role-specific navigation.

Notes: every demo business already has a placeholder price proof (a PDF marked DEMO), so
Dilnoza can submit a new deal without uploading anything. Madina and Saida also see the admin
overview totals; Saida can open the fraud-signal page and suspend or reinstate a business under
Admin → Businesses.


## Where each role lands and what to show

Every role has its own workspace. After sign-in (or when opening `/` while signed in) the account goes to its home:

| Role (demo account) | Lands on | Menu it sees | Show |
| --- | --- | --- | --- |
| Buyer, `+998900000001` | `/uz` marketplace | Marketplace only; no staff or business links | Buy a deal, pay, see the voucher QR. |
| Business owner, `+99891227100` (business 27) | `/uz/merchant/dashboard` ("Dealo Business") | Dashboard, deals, new deal, orders, redeem, payouts, customers, analytics, reviews, calculator, staff, business, settings | Dashboard totals, payouts, staff. On a phone the bottom bar has Dashboard, Deals, Redeem, Orders and More. |
| Manager, `+99891227200` | `/uz/merchant/dashboard` | The owner menu without payouts, staff, business and settings | Deals, orders and reviews; payouts and staff show "No access". |
| Cashier, `+99891227300` | `/uz/merchant/redeem` ("Dealo Kassa") | Only the scanner, the code field and today's own redemptions | Scan or type a code, confirm, cancel within 10 minutes. |
| Moderator, `+998910000011` | `/uz/admin/moderation` ("Dealo Admin") | Deal moderation, businesses, fraud signals, review reports | Approve or reject a deal in the queue. |
| Support, `+998910000012` | `/uz/admin/finance` | Finance desk, showing refunds and disputes only | Dispute and refund queue. Payouts are not shown. |
| Finance, `+998910000013` | `/uz/admin/finance` | Finance desk: payouts, refunds, disputes | Payout approval (two people: finance prepares, a second person approves). |
| Superadmin, `+998900000004` | `/uz/admin` (overview) | Every admin section, including audit | Totals, fraud signals, business suspend and reinstate, audit log. |

Each workspace has an account menu (name, role, Logout), a "View marketplace" link and, when the account has
several roles, "Switch workspace". Opening another role's page shows "No access" with a button back to the
account's own home. The server still enforces every rule; the page check is only what the user sees.

Any signed-in account can open its menu from the profile icon. On a phone that is the bottom bar; on a
computer it is the top right corner.
