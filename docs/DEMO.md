# Demo accounts

Run `go run ./cmd/bootstrap` against the demo database before the walkthrough. In
`DEMO_MODE=true`, the six-digit OTP is shown on the login page. Staff accounts also
require the authenticator code documented in `docs/ENVIRONMENT.md`; do not use that
development factor for a hosted demo.

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
