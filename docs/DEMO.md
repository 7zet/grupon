# Dealo demo guide

Start the demo with `powershell -ExecutionPolicy Bypass -File scripts\demo.ps1 -Reset` (reloads the demo data) and
open <http://localhost:3000/uz>. To show it on a phone, see `docs/SHARE.md`.

**Signing in:** type the phone number, press "Kod yuborish", and the six-digit code appears on the page (demo mode
only). Admin roles (moderator, support, finance, superadmin) also ask for an authenticator code: add the key
`JBSWY3DPEHPK3PXP` to Google Authenticator or a similar app. This key is for the demo on your own PC only.

## Accounts and where each one lands

| Person | Role | Phone | Lands on | What to show |
| --- | --- | --- | --- | --- |
| Aziz | Buyer | `+998900000001` | Marketplace `/uz` | Search by street, buy, voucher QR |
| Zarina | Buyer | `+998910000009` | Marketplace `/uz` | A second buyer |
| Dilnoza | Owner, Lotus Spa | `+998900000002` | `/uz/merchant/dashboard` (Dealo Business) | Sales chart, deals, payouts, staff |
| Sherzod | Manager, Lotus Spa | `+998910000010` | `/uz/merchant/dashboard` | Same menu without payouts, staff, settings |
| Bekzod | Cashier, Lotus Spa | `+998900000003` | `/uz/merchant/redeem` (Dealo Kassa) | Scan, confirm, cancel within 10 minutes |
| Nodira | Cashier, Urban Grill | `+998900000009` | `/uz/merchant/redeem` | Cannot redeem Lotus Spa vouchers |
| Madina | Moderator | `+998910000011` | `/uz/admin/moderation` (Dealo Admin) | Two deals and one business wait for review |
| Umida | Support | `+998910000012` | `/uz/admin/finance` | One open dispute, refunds; no payouts |
| Farhod | Finance (prepares) | `+998910000013` | `/uz/admin/finance` | Payouts; cannot approve his own |
| Shahnoza | Finance (approves) | `+998910000014` | `/uz/admin/finance` | Approves the waiting Lotus Spa payout |
| Saida | Superadmin | `+998900000004` | `/uz/admin` | Everything, including the audit log |
| Laylo | Owner of a new business | `+998935550099` | `/uz/merchant/dashboard` | Yashil Bog Kafe, waiting for approval |

Every one of the 30 demo businesses also has its own staff: phone `+998912MMRPP`, where `MM` is the business number
(`01`–`30`) and `R` is `1` owner, `2` manager, `3` cashier. Example, business 27 (Dolphin Swim Club): owner
`+99891227100`, manager `+99891227200`, cashier `+99891227300`. The login page does not list accounts; this page does.

Opening another role's page shows "No access" with a button back to the account's own home. An account with several
roles has "Switch workspace" in its menu.

## What the demo data contains

- 60 deals from 30 Tashkent businesses; **3 deals end within two days** (the "Ending soon" filter).
- **2 deals and 1 business (Yashil Bog Kafe) wait for moderation.**
- **1 open dispute** on a Lotus Spa redemption ("I did not use this voucher").
- **Lotus Spa payouts:** two paid (one reconciled, one sent) and one waiting for the second finance approval.
- **Cashier Bekzod** already redeemed five vouchers today.
- Reviews in the buyer's language, most with a reply from the business.

## A 10-minute demo script

1. **Buyer (3 min).** Open `/uz` as a guest. Type "Amir Temur" in the search: street addresses are suggested; each
   card shows its address. Open a deal, pick an option, set quantity 2, press "Hozir xarid qilish". Sign in as Aziz:
   checkout still has the same deal, option and quantity. Pay with "Muvaffaqiyatli", open "Vaucherlarim" and show
   the QR code (it refreshes every 30 seconds).
2. **Cashier (2 min).** In another browser sign in as Bekzod. The cashier sees only Dealo Kassa. Type the voucher code
   (or scan the QR on a phone), confirm: a green check fills the screen. Show "Bekor qilish" on the new row (works
   for 10 minutes). Type a wrong code to show the red error.
3. **Business owner (2 min).** Sign in as Dilnoza. Show the dashboard (sales chart with values and dates), the dark
   sidebar with the "Vaucher ishlatish" button, deals, the calculator and payouts.
4. **Moderator (1 min).** Sign in as Madina. Approve one deal and reject the other with a reason; open Businesses and
   show Yashil Bog Kafe waiting for review.
5. **Finance (1 min).** Sign in as Shahnoza and approve the waiting Lotus Spa payout. Farhod prepared it, so he could
   not approve it himself (four-eyes rule).
6. **Superadmin (1 min).** Sign in as Saida: overview totals, fraud signals, the audit log of everything above.

Tip: use a separate browser window (or a private window) per role, so sessions do not mix.
