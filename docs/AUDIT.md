# Loyiha auditi — Dealo talablari bo'yicha

> To'ldiruvchi: Claude Code. Sana: 2026-11-05. Tekshirilgan commit: 7c47bc7 (branch feat/stage-3; working tree has uncommitted Stage 3 Telegram code, which was also read).

## Claude Code uchun ko'rsatma (read this first)

You are an independent reviewer. You did NOT write this code. Your job is to check the project
against `docs/DEALO-REQUIREMENTS.md` and fill in THIS file. Be strict and factual.

Rules:
1. READ-ONLY. Do not change any code, config, tests or other docs. The only file you write is `docs/AUDIT.md`.
2. Do NOT spawn subagents. Do all work yourself in this session.
3. Do not scan the whole repo blindly. For each ID, search with `rg` for the relevant names,
   then read only the needed lines. Skip node_modules, generated code, lockfiles, .next.
4. Judge by CODE and TESTS, not by what ROADMAP/STATUS/other docs claim. If a doc claims something
   the code does not do, write it in section "B. Doc vs code contradictions".
5. You may run existing tests (go test, vitest) if the environment allows it. Do not start servers
   against production, do not call the live site, do not use real secrets.
6. Status values:
   - ✅ done — implemented AND (for rules) covered by a test or clearly enforced in code
   - 🟡 partial — some of it exists; write exactly what is missing
   - ❌ missing — not implemented
   - ➖ excluded — marked ➖ in the requirements (deliberate); only check it is listed in docs/HANDOVER.md
   - ❔ unknown — you could not verify; write what would prove it
7. Evidence = `path/to/file.go:123` or a test name. Note = max 1 line.
8. For each DEV-xx item: check that the deviation is real and documented in docs/HANDOVER.md.
9. At the end, fill in sections A–E and the summary counts.
10. Write in Uzbek (Latin) or English — short and concrete.

---

## Natijalar jadvali

| ID | Talab (qisqa) | Holat | Dalil (file:line / test) | Izoh |
|---|---|---|---|---|
| **0. Loyiha doirasi va ataylab farqlar** | | | | |
| DEV-01 | Stek: Dealo'da Django + DRF + aiogram + Django Admin; bizda Go API/… | ✅ | HANDOVER.md DEV-01 | v1.0: recorded in HANDOVER |
| DEV-02 | To'lov: real PSP (Payme/Click/Uzum), fiskal chek, MXIK, paket kodi,… | ✅ | payments/provider.go; payments/mock.go; payments/manual.go; HANDOVER.md | Mock + manual confirm + PSP port exist; documented |
| DEV-03 | Hosting: Vercel + Render + Neon (O'zbekistondan tashqarida). Faqat… | ✅ | HANDOVER.md DEV-03, docs/SHARE.md | v1.0: no hosting; local demo + ngrok; documented |
| DEV-04 | Yuridik gate'lar G1–G9 qo'llanilmaydi. | ✅ | HANDOVER.md 'Legal, regulatory... launch gates' | Documented |
| DEV-05 | Support: ticket tizimi va SLA yo'q; faqat kontaktlar (telefon, Tele… | ✅ | HANDOVER.md DEV-05 | v1.0: recorded in HANDOVER |
| DEV-06 | Telegram Mini App: to'liq initData login yo'q; faqat botdagi "Saytn… | ✅ | HANDOVER.md DEV-06; tag archive/telegram-wip | v1.0: Telegram dropped, recorded |
| DEV-07 | Kategoriya flag'lari faqat kategoriya darajasida (Dealo'da kategori… | ✅ | HANDOVER.md DEV-07 | v1.0: recorded in HANDOVER |
| DEV-08 | Tillar: uz, ru, en (Dealo: uz-Latn, ru, uz-Cyrl avtomatik). uz-Cyrl… | ✅ | HANDOVER.md DEV-08 | v1.0: recorded in HANDOVER |
| DEV-09 | DEMO_MODE: kirish kodi sahifada ko'rinadi, test to'lov tugmalari, "… | 🟡 | config.go:129,403; app.go:197; site-shell.tsx:48; checkout-form.tsx:159 | Gated by DEMO_MODE (OTP echo, mock pay, badge); not in HANDOVER; see A-4 (log sender) |
| DEV-10 | Sharhlar (Dealo'da R1b/R2) — demo'da bor. | ✅ | HANDOVER.md DEV-10 | v1.0: recorded in HANDOVER |
| DEV-11 | Booking (R2), CRM/referral, reklama, loyalty, sovg'a (R2–R3) — yo'q. | ✅ | HANDOVER.md DEV-11 | v1.0: recorded in HANDOVER |
| DEV-12 | Docker fayllari bor, lekin "ixtiyoriy, tekshirilmagan". | ✅ | services/api/Dockerfile; apps/web/Dockerfile; docker-compose.prod.yml; HANDOVER.md | Documented unverified; docker daemon not running here either; see B-3 |
| **1. Doimiy tamoyillar** | | | | |
| P-01 | Logistika yo'q: Shipment, ShippingAddress, delivery_fee, kuryer, om… | ✅ | migrations/00009_dealo_removals.sql:17; rg shipment/courier = 0 in code | Only unused i18n key 'shippingTitle' left (D) |
| P-02 | Pul faqat UZS, tiyinda butun son (int64/bigint); float hech qayerda… | ✅ | 00001:166,220,251 (bigint); packages/ui/src/money.ts | float only for geo/rating, not money |
| P-03 | ID'lar UUIDv7; vaqt timestamptz UTC'da; kun chegaralari (muddat, ku… | ✅ | 00001:9 uuidv7(); timestamptz everywhere; commerce/model.go:90; voucher_window_test.go | No daily limits exist yet (FR-02.4) |
| P-04 | Ledger ikki yoqlama va faqat qo'shiladigan (append-only); tuzatish… | ✅ | 00007_finance_core.sql:61-94; ledger/ledger.go:128 (Reversed) | Triggers block UPDATE/DELETE; storno only |
| P-05 | Bitta Order = bitta merchant = bitta to'lov; savatcha yo'q, faqat "… | 🟡 | commerce/orders.go:22 (MaxOrderItems=100); openapi.yaml:1036; web stores/cart.ts | One payment per order, but order may hold 100 items of any merchants; client cart exists (A-1) |
| P-06 | Karta ma'lumotlari platformada saqlanmaydi (BR-07). | ✅ | rg card_number/cvv/pan = 0; payments/provider.go | No card data handled |
| P-07 | Sayt Telegram'siz va real SMS'siz ham to'liq ishlaydi. | 🟡 | app.go:172-174; config.go:267-275 | Works with DEMO_MODE echo; without it no SMS provider exists, so nobody can sign in |
| P-08 | Shaxsiy ma'lumotlar O'zbekistonda saqlanadi. | ➖ | - | Excluded; HANDOVER.md does not state it explicitly (B-9) |
| **2. Rollar va ruxsatlar (RBAC)** | | | | |
| R-01 | Mehmon: qidiruv, deal va shartlarni ko'radi; sotib olish uchun tele… | ✅ | web checkout/page.tsx:11 (redirect to login); commerce_handlers auth | Browse public; buying needs phone OTP |
| R-02 | Xaridor: faqat o'z buyurtma, vaucher, refund va sharhlarini ko'radi… | ✅ | commerce/orders.go GetOrder ownership; finance.go:213 ErrNotOwner; migration 00013 | Own data only; 'not me' via notifications |
| R-03 | Merchant owner: filiallar, xodimlar, deallar, hisobotlar, bank rekv… | ✅ | merchant/staff.go:73 (owner only); workspace_test.go | Owner manages staff/branches/docs; bank details via registration/settings |
| R-04 | Merchant manager: deal draft, kvota, statistika; bank rekviziti va… | ✅ | merchant/staff.go:55-73; workspace_test.go TestCashierCanReadButNotEdit | Manager cannot manage staff |
| R-05 | Kassir: faqat QR skanerlash, zaxira kod, o'z redeem tarixi; narx, r… | ✅ | redemption.go:143-160; at_gates_test.go TestMerchantRedemptionHistoryScopesCashierAndMasksBuyer | Cashier scan + own history only |
| R-06 | Moderator: merchant KYC, deal moderatsiyasi, kontent shikoyatlari;… | ✅ | moderation/moderation.go:28; admin_merchant_review_handlers.go:36; admin_audit_test.go | Moderator has no finance access (DB test, skipped here) |
| R-07 | Support: refund taklifi, vaucherni vaqtincha bloklash; refund/payou… | 🟡 | finance.go:392-394,339-343 (supportOrFinance → ApproveRefund) | Support can approve refunds alone; no endpoint to temporarily block a voucher (A-3) |
| R-08 | Finance operator: reconciliation, payout tayyorlash, refund ijrosi;… | 🟡 | finance.go:400-406; finance.go:151-179 | Operator and approver are the same 'finance' role; split only by user id |
| R-09 | Finance approver: payout va refund tasdig'i (2-imzo); ledger yozuvi… | ✅ | finance.go:404; 00007:61-70 | Approver cannot change ledger (triggers); same role as operator |
| R-10 | Trust & fraud analyst: fraud navbati, payout hold, anomaliyalar. (A… | ✅ | web (admin)/admin/fraud/page.tsx; lib/roles.ts ADMIN_SECTIONS.fraud | v1.0: fraud signals page for moderator/superadmin; burst rule holds payouts |
| R-11 | Superadmin: rollar, flag'lar; har bir harakati audit qilinadi. | 🟡 | totp.go:27; admin_audit_test.go TestEveryStaffWriteLeavesAnAuditRow | No endpoint to manage roles or flags (ADM-06) |
| R-12 | Barcha ruxsatlar serverda tekshiriladi (UI'da yashirish yetarli emas). | ✅ | service-level checks: finance.go, redemption.go:371-383, moderation.go; handlers map to 403 | UI hiding is not the only guard |
| R-13 | Moliyaviy amallar 4-eyes: tayyorlovchi ≠ tasdiqlovchi. | 🟡 | 00007:~74 CHECK approved_by<>prepared_by; finance.go:187; refunds CHECK 00007 | Payout enforced (DB+domain). Refund: only requester≠approver; MarkRefundPaid has no preparer check |
| **3. Kategoriyalar** | | | | |
| C-01 | Ochiq pilot kategoriyalar: Beauty & Wellness, Food & Drink, Things… | ✅ | seed/data.json (first 4 active); seed.go:172 | Seeded active |
| C-02 | Qolganlari (Automotive, Education, Events, Home Services, Travel, P… | ✅ | seed/data.json: 7 categories active:false | Seeded closed; redeemable=true |
| C-03 | Flag'lar: searchable, deal_creation_enabled, publish_enabled, purch… | ✅ | 00025; at_gates_test.go TestCategory{Searchable,PublishEnabled,Purchasable,Redeemable}* | Enforced server-side; bookable/ads_enabled not present; no UI to change flags |
| C-04 | Kategoriya yopilsa, mavjud vaucherlar redeemable bo'lib qoladi. | ✅ | seed.go:172 (redeemable=true); at_gates_test.go TestCategoryRedeemableBlocksOnlyWhenFalse |  |
| C-05 | Yopiq kategoriyada deal yuborish aniq sabab bilan rad etiladi (BR-1… | ✅ | merchant/deal.go:109,323; merchant_deal_repository.go:51; at_gates_test.go TestAT10 | code category_closed |
| C-06 | Taqiqlangan takliflar (alkogol, tamaki, qimor, retsept dorilari, li… | ❌ | rg alkogol/tobacco/prohibited in code/i18n = 0 | No prohibited-offer rules anywhere (moderation has no checklist) |
| C-07 | Kategoriya × shahar flag'lari. | ➖ | - | Excluded (DEV-07); not in HANDOVER |
| **4. FR-01 Merchant onboarding** | | | | |
| FR-01.1 | Ariza: STIR, yuridik nom, tashkiliy shakl (MChJ/YaTT), soliq rejimi… | ✅ | migrations/00010 (stir, org_form, tax_regime, bank_mfo, bank_account, responsible_name); register_test.go |  |
| FR-01.2 | Hujjatlar yuklanadi va versiyalanadi: guvohnoma, kerak bo'lsa litse… | 🟡 | 00012_merchant_documents.sql (UNIQUE(merchant_id,kind)); documents.go; storage/private.go | Replace, not versioned; kinds only certificate/bank_letter/price_proof (no license, branch photo) |
| FR-01.3 | Holatlar: draft → submitted → in_review → approved / rejected; appr… | 🟡 | merchant/review.go:77-96; 00010 CHECK includes suspended/terminated | suspended/terminated states exist but no transition/endpoint |
| FR-01.4 | Oferta elektron qabul: vaqt, IP, versiya va qabul qilgan shaxs saql… | 🟡 | 00010: contract_version, contract_accepted_at, contract_accepted_ip | Accepting person not stored explicitly |
| FR-01.5 | Bank rekviziti o'zgarishi: owner 2FA → platforma tasdig'i → 72 soat… | ➖ | - | Excluded; HANDOVER.md 'Bank-account change protection' |
| FR-01.6 | Owner xodimlarni telefon orqali taklif qiladi va filialga biriktiradi. | 🟡 | merchant/staff.go:72-90 | Invites only already-registered users by phone; no branch assignment seen |
| FR-01.7 | Rad etish sababi merchantga ko'rinadi. | ✅ | (business)/merchant/deals/page.tsx:38 (rejection_reason); openapi Deal.rejection_reason | v1.0: reason shown to the business on its deals list |
| **5. FR-02 Deal Studio** | | | | |
| FR-02.1 | Maydonlar: kategoriya, sarlavha, tavsif (uz majburiy), kamida 3 ras… | ✅ | merchant/deal.go:98-170 (3 images, highlights, dates, branches, terms) | Stricter: uz+ru+en all required |
| FR-02.2 | Variant: original_price, deal_price, voucher_type (single/multi), r… | ➖ | merchant/deal.go:154-160 (prices, redemptions_allowed) | Excluded part (MXIK/QQS); core variant fields exist |
| FR-02.3 | Narx dalili (menyu/prays/chek) majburiy; chegirma 70% dan yuqori bo… | 🟡 | 00017_price_proof.sql (optional doc) | Price proof optional; no >70% extra check |
| FR-02.4 | Limitlar: global kvota, filial kvotasi, bitta xaridor limiti, kunli… | 🟡 | deal.go:143 max_per_user; deal_options.stock_total | Global quota + per-buyer only; no branch quota, daily limit, blackout |
| FR-02.5 | Kategoriya bo'yicha shartlar shabloni (bron kerakmi, bir tashrifda… | ❌ | terms is free text; booking_required boolean only | No per-category terms templates |
| FR-02.6 | Kampaniya kalkulyatori: sof tushum; manfiy bo'lsa ogohlantirish. | 🟡 | web components/campaign-calculator.tsx; merchant/(panel)/calculator | No negative-net warning |
| FR-02.7 | Holatlar: draft → in_review → approved → scheduled → active ⇄ pause… | 🟡 | merchant/deal.go:21-44; moderation.go:124-144; store/deal_jobs.go | paused/archived have no endpoint (seed only) |
| FR-02.8 | Faol dealda narx/shart/variant o'zgarishi → qayta moderatsiya; faqa… | ✅ | merchant/deal.go ReviewFingerprint + Update (status→in_review); deal_test.go | Stock excluded from fingerprint |
| **6. FR-03 Checkout** | | | | |
| FR-03.1 | Faqat "Hozir sotib olish": bitta variant, 1–N dona (N ≤ bitta xarid… | 🟡 | orders.go:22; openapi.yaml:1036; MaxPerUser check orders.go:~103 | API allows many items/variants (A-1) |
| FR-03.2 | To'lovdan oldin ko'rsatiladi: yakuniy narx, asl narx, ishlatish mud… | ❔ | web components/checkout-form.tsx (terms checkbox :13-16) | Could not confirm refund rule/branches shown; needs browser walk-through |
| FR-03.3 | Server narxni qayta hisoblaydi; Order'da narx va shartlarning o'zga… | ✅ | orders.go:~120-160; migrations 00022/00023; at_gates_test.go TestAT06 |  |
| FR-03.4 | Inventory hold 15 daqiqa; tugasa Order expired, inventory qaytadi. | ✅ | commerce/model.go:14; jobs.go:17; at_payments_test.go TestAT07 | 15 min (manual mode 24h by config) |
| FR-03.5 | Kech to'lov: inventory bo'lsa vaucher chiqariladi, bo'lmasa avtomat… | ✅ | migrations/00021; at_payments_test.go TestAT13 | Notification to buyer not sent |
| FR-03.6 | POST /orders Idempotency-Key bilan; takror so'rov ikkinchi buyurtma… | ✅ | pricing.go:44-70 (fingerprint); migration 00003 |  |
| FR-03.7 | Login faqat checkout'da so'raladi; kirgandan keyin xaridor o'sha de… | ✅ | checkout/page.tsx:11 ?next=; lib/safe-next.ts + test |  |
| **7. FR-04 Vaucher** | | | | |
| FR-04.1 | Har bir sotilgan dona uchun bitta vaucher (BR-01: UNIQUE(order_item… | 🟡 | 00001:258 vouchers; only INDEX on order_item_id (no seq, no UNIQUE) | Idempotency relies on payment-state change; no DB unique (A-7) |
| FR-04.2 | Token 128-bit tasodifiy, DB'da faqat HMAC-hash. (Boshqacha yechim b… | 🟡 | commerce/qr.go:12-90 ('gv2:<voucher id>:<window>' HMAC-SHA256) | Deviation: signed rotating token containing voucher UUID, not 128-bit random + hash; sound but not documented as deviation |
| FR-04.3 | QR ichida faqat opaque token, PII yo'q; zaxira kod 10 belgi (Crockf… | 🟡 | qr.go:104-114; pricing.go:80 | No PII; 10-char Crockford OK, but derived from voucher id+secret (not stored) |
| FR-04.4 | QR har 30 soniyada yangilanadi (web'da ham). | ✅ | web components/voucher-qr.tsx qrRefreshDelay; voucher-qr.test.ts | v1.0: QR refetched before the server expiry (30 s window) |
| FR-04.5 | multi_use: har skanerda redemptions_used + 1; oxirgisida redeemed. | ✅ | queries/redemption_ledger.sql:16-21; at_redeem_test.go |  |
| FR-04.6 | Muddat tugashidan 7 va 1 kun oldin eslatma. | ❌ | rg reminder/7 days = 0 | No expiry reminders |
| FR-04.7 | Muddat: min(valid_until, xarid + N kun), Asia/Tashkent 23:59:59 da… | ✅ | commerce/model.go:89-110; voucher_window_test.go |  |
| FR-04.8 | To'lov server tomonda tasdiqlanmaguncha vaucher active bo'lmaydi (B… | ✅ | commerce/payments.go (vouchers issued on webhook success); at_payments_test.go TestAT01 |  |
| **8. FR-05 Redemption** | | | | |
| FR-05.1 | Kassir QR'ni kamera bilan skanerlaydi yoki zaxira kodni kiritadi (S… | ✅ | web voucher-redeem-panel.tsx (zxing camera + code); public/scanner-sw.js |  |
| FR-05.2 | Bitta tranzaksiyada tekshiruv: token, holat, muddat, merchant va fi… | 🟡 | redemption.go:211-300; redemption_ledger.sql:3 fraud_hold,dispute_open | No deal-branch check (BR-04), no daily limit |
| FR-05.3 | Kassirga natija: deal, variant, xaridorning maskalangan ismi ("Alis… | ✅ | merchant/panel.go:215 MaskName; redemption.go:161 |  |
| FR-05.4 | Redeemdan keyin darhol xaridorga xabar "…ishlatildi. Siz emasmisiz?… | 🟡 | migrations/00013_notifications.sql; buyer-notifications.tsx | In-app only (no Telegram/SMS) |
| FR-05.5 | Kassir 10 daqiqa ichida redeemni bekor qila oladi; keyin faqat supp… | ✅ | redemption.go:42 ReversalWindow=10m; redemption_test.go TestCashierCanReverseOnlyWithinTheWindow; ledger.Reversed |  |
| FR-05.6 | Offline redeem yo'q; internet bo'lmasa aniq xabar. | ✅ | voucher-redeem-panel.tsx:72-74; public/offline.html |  |
| FR-05.7 | Kassir uchun rate limit 30/daqiqa; 10 xato urinishdan keyin blok. | ✅ | redemption.go:199-224 | 30/min, 10 fails → 15 min block |
| FR-05.8 | Parallel ikkinchi redeem → 409; begona merchant → 403; expired/refu… | ✅ | merchant_redemption_handlers.go:111-134; TestAT05... |  |
| **9. FR-06 Refund** | | | | |
| FR-06.1 | Ishlatilmagan, xariddan 14 kun ichida → avtomatik to'liq refund. | ✅ | finance.go:224; finance_test.go TestUnusedVoucherRefundsAutomaticallyWithinFourteenDays |  |
| FR-06.2 | Ishlatilmagan, 14 kundan keyin, muddat ichida → so'rov, support + f… | 🟡 | finance.go:339-343 | Support alone approves; no second (finance) approval |
| FR-06.3 | Merchant xizmat ko'rsatmadi/yopildi → to'liq refund, merchantdan un… | ❌ | no merchant-fault refund / merchant recovery flow | Only dispute resolution exists |
| FR-06.4 | Ishlatilgan, sifat shikoyati → dispute oqimi, finance approver. | 🟡 | finance.go:346-376; finance_test.go dispute tests | Customer win creates refund that support can approve alone |
| FR-06.5 | Muddati o'tgan → breakage siyosati (90 kun qaytarish huquqi). | 🟡 | finance.go:19,262; ledger.Expired; TestBreakageMovesExpiredValue | Expired voucher cannot be refunded during the 90 days (RequestRefund needs status active) |
| FR-06.6 | Multi-use qisman ishlatilgan → qolgan tashriflar ulushi (GAP-07). | ❌ | finance.go:222 (RedemptionsUsed>0 → ErrNotRefundable) | Partly used multi-use cannot be refunded |
| FR-06.7 | Refund faqat asl to'lov usuliga. | 🟡 | payments/provider.go Refund(); MarkRefundPaid manual | No real PSP; method not enforced |
| FR-06.8 | Refund suiiste'moli: 30 kunda > 3 refund → qo'lda ko'rib chiqish. | ❌ | rg 'refunds in 30' = 0 | No abuse rule |
| **10. FR-07 Settlement va payout** | | | | |
| FR-07.1 | Merchant majburiyati har bir redeemda hisoblanadi. | ✅ | redemption_ledger_repository.go:70; ledger.Redeemed |  |
| FR-07.2 | Payout haftada bir marta; redeemdan keyin 3 kunlik dispute oynasi o… | 🟡 | finance.go:16,164; queries/finance.sql:1-10 | 3-day window OK; weekly cadence not scheduled (manual per merchant) |
| FR-07.3 | Payout paketi: tayyorlash → tasdiqlash (boshqa odam) → yuborildi →… | 🟡 | finance.go:151-196; finance.sql ApprovePayoutRun | approve → 'sent' directly; 'draft'/'approved'/'failed→draft' unused |
| FR-07.4 | Merchant har payout bo'yicha hisobot oladi: redeemlar, komissiya, u… | 🟡 | queries/merchant_panel.sql:1-11; web statements-download.tsx | Per-payout redemptions yes; holds/refunds breakdown not verified |
| FR-07.5 | Manfiy balans keyingi payoutdan ushlab qolinadi; 60 kundan oshsa un… | ❌ | queries/finance.sql PayableRedemptions ignores reversals after payout | No negative-balance carry/recovery |
| FR-07.6 | Ochiq dispute summasi payoutga kirmaydi (BR-15). | ✅ | queries/finance.sql:9; at_gates_test.go TestAT16 |  |
| **11. FR-08 Support** | | | | |
| FR-08.1 | Support kontaktlari (telefon, Telegram, email) sozlamadan o'qiladi… | ✅ | web lib/support.ts; components/support-contacts.tsx; config.go:264 | Env-driven, hidden when empty |
| FR-08.2 | Dispute holatlari: open → awaiting_evidence → under_review → resolv… | ✅ | 00019 CHECK; finance.go:346-376; TestDisputeAllowedTransitionsAuditAndPayoutHold |  |
| FR-08.3 | Dispute ochilganda merchant payouti shu summa bo'yicha hold. | ✅ | finance.sql:9; at_gates_test.go TestAT16, finance_test.go |  |
| FR-08.4 | Ticket tizimi, dalil fayllari, SLA (2 soat / 48 soat). | ➖ | - | Excluded (DEV-05); not in HANDOVER |
| **12. FR-09 Discovery** | | | | |
| FR-09.1 | Bosh sahifa: kategoriyalar, "tugab bormoqda", tavsiyalar; (tuman ta… | 🟡 | web [locale]/page.tsx:132; homepage sections | No district picker / nearby |
| FR-09.2 | Qidiruv xatolarga chidamli; lotin va kirilda bir xil natija. | 🟡 | queries/catalog.sql:114-117 (pg_trgm word_similarity + unaccent) | Typo-tolerant; no Cyrillic↔Latin transliteration for uz |
| FR-09.3 | Filtrlar: kategoriya, tuman, narx, chegirma, "bugun ishlatsa bo'ladi". | 🟡 | openapi.yaml:52 | Category, price, discount, rating, ending soon; no district, no 'usable today' |
| FR-09.4 | SEO: server-side render, schema.org/Offer, sitemap, robots, kategor… | 🟡 | web deal/[slug]/page.tsx:56-78 (JSON-LD Offer); [segment]/[category] | No sitemap.ts / robots.ts (ROADMAP Stage 5) |
| FR-09.5 | Marketing xabarlari faqat alohida rozilik bilan; obunadan chiqish. | ❌ | rg marketing/consent/unsubscribe = 0 | No marketing messages and no consent model |
| FR-09.6 | Urgency matnlari ("N ta qoldi", "tugaydi") faqat real ma'lumotdan. | ✅ | purchase-panel.tsx:27,85; deal-card.tsx:68 | Uses real stock / dates |
| **13. Holat mashinalari** | | | | |
| S-01 | Merchant: draft → submitted → in_review → approved / rejected; appr… | 🟡 | 00010 CHECK; review.go | suspended/terminated never reachable |
| S-02 | Deal: draft → in_review → approved → scheduled → active ⇄ paused →… | 🟡 | 00019; deal.go; deal_jobs.go | paused/archived never reachable via API |
| S-03 | Order: created → awaiting_payment → paid → completed; awaiting_paym… | 🟡 | 00019; commerce/model.go:28-31 | completed/partially_refunded/refunded constants only; not set by code |
| S-04 | Payment: created → pending → succeeded / failed; succeeded → charge… | 🟡 | 00019 enum; payments.go | chargeback_* states exist, no transitions |
| S-05 | Refund: requested → approved / rejected; approved → processing → su… | 🟡 | 00007:116; finance.go:339; ApproveRequestedRefund | processing/failed/rejected unused |
| S-06 | Voucher: issued → active → partially_redeemed → redeemed; active →… | 🟡 | redemption_ledger.sql:16; 00019 flags | 'issued' unused (vouchers born active); 'voided' no way to set |
| S-07 | Redemption: confirmed → reversed. | ✅ | 00007 redemptions.reversed_at; redemption.go |  |
| S-08 | Payout: draft → pending_approval → approved → sent → reconciled; se… | 🟡 | 00007:~72 CHECK; finance.sql | See FR-07.3 |
| S-09 | Dispute: open → awaiting_evidence → under_review → resolved_custome… | ✅ | finance_repository.go:396-430; TestDisputeForbiddenTransitionAndRBAC | 422 invalid_dispute_transition |
| S-10 | Ruxsatsiz o'tish 422 qaytaradi va holat o'zgarmaydi. | 🟡 | finance_handlers.go:239-262 | Disputes → 422; payout/refund/merchant wrong state → 409 |
| S-11 | Har bir o'tish jurnalga yoziladi (kim, qachon, eski → yangi). | 🟡 | audit_logs writes in moderation/review/dispute/finance repos | No generic state_transitions; order/payment/voucher moves not logged |
| **14. Biznes qoidalari (BR)** | | | | |
| BR-01 | Bitta to'lov vaucherlarni ikki marta chiqarmaydi (unique + idempote… | 🟡 | at_payments_test.go TestAT01 (5 webhooks → 1 voucher) | No UNIQUE on vouchers (A-7); idempotency by payment state |
| BR-02 | To'lov server tomonda tasdiqlanmaguncha vaucher active emas. | ✅ | payments.go (issue on verified webhook); TestAT01/AT07 |  |
| BR-03 | redemptions_used ≤ redemptions_allowed (FOR UPDATE + CHECK). | ✅ | 00007:10-12 CHECK; redemption_ledger.sql:11,35 FOR UPDATE; TestAT15 |  |
| BR-04 | Vaucher faqat dealga biriktirilgan filialda ishlatiladi. | 🟡 | redemption.go:274 LocationOwned (merchant's own branch only) | Does not check branch is attached to the deal (A-5) |
| BR-05 | sold + held ≤ capacity (atomik shartli UPDATE). | ✅ | 00001:170 CHECK; queries/commerce.sql:3 conditional UPDATE; TestOverSellingIsImpossible |  |
| BR-06 | Order shartlari sotib olingan versiya bo'yicha o'zgarmaydi. | ✅ | 00022/00023; TestAT06ChangedPriceDoesNotChangeAPlacedOrder |  |
| BR-07 | Karta ma'lumotlari saqlanmaydi. | ✅ | rg card data = 0 |  |
| BR-08 | Refunded/voided vaucher ishlatilmaydi. | ✅ | redemption.go unusable(); TestAT05... |  |
| BR-09 | Moliyaviy holat faqat ledger orqali o'zgaradi; ledger'da UPDATE/DEL… | ✅ | 00007:61-70 |  |
| BR-10 | Har bir journal entry balanslangan (debit = kredit), DB cheklovi bi… | ✅ | 00007:75-94 deferred constraint trigger; ledger_test.go |  |
| BR-11 | Bank rekviziti o'zgarsa 72 soat payout hold. | ➖ | - | Excluded (FR-01.5) |
| BR-12 | Yopiq kategoriyada publish yo'q. | ✅ | TestCategoryPublishEnabledBlocksModerationApproval; deal.go:109 |  |
| BR-13 | Bitta Order = bitta merchant. | 🟡 | ledger per item merchant; orders.go no merchant check | Not enforced (A-1) |
| BR-14 | Jismoniy yetkazish obyekti yo'q. | ✅ | 00009 drops shipping_address |  |
| BR-15 | Ochiq dispute summasi payoutga kirmaydi. | ✅ | finance.sql:9 |  |
| **15. Ledger va pul oqimi** | | | | |
| L-01 | Hisoblar: psp_receivable, bank:operating, customer_prepaid:{merchan… | 🟡 | ledger/ledger.go:12-21 | Missing merchant_reserve, vat_payable, merchant_receivable; breakage_revenue unused (customer_prepaid_expired used); no mapping table in docs |
| L-02 | To'lov tasdiqlandi: psp_receivable → customer_prepaid. | ✅ | ledger.PaymentReceived | Posted per order item |
| L-03 | Vaucher ishlatildi: customer_prepaid → merchant_payable + commissio… | ✅ | ledger.Redeemed |  |
| L-04 | Payout yuborildi: merchant_payable → bank:operating. | ✅ | ledger.PayoutSent |  |
| L-05 | Refund (ishlatilmagan): customer_prepaid → refunds_payable → bank:o… | ✅ | ledger.RefundApproved + RefundPaid; TestUnusedVoucherRefunds... |  |
| L-06 | Redeem reversal — teskari yozuv; payout yuborilgan bo'lsa merchant_… | 🟡 | ledger.Reversed | No merchant_receivable; reversal after payout leaves merchant_payable negative |
| L-07 | Ishlatilgandan keyingi refund (dispute) provodkasi (GAP-09). | ✅ | ledger.RefundFromMerchant; finance_repository.go:384; TestDisputeCustomerResolutionKeepsRefundBehavior |  |
| L-08 | Multi-use: har redeem floor(narx/N), qoldiq oxirgisiga; komissiya h… | 🟡 | ledger.Split (:80); redemption_ledger_repository.go:70 | Price split OK; commission floored per share, remainder not carried to last visit |
| L-09 | Breakage: muddat tugagach chegirma yo'qoladi, to'langan pul 90 kun… | ✅ | finance.go:19,262; TestBreakageMovesExpiredValue |  |
| L-10 | Reconciliation: kunlik; payout'dan keyin; oylik invariant "customer… | 🟡 | finance.go Reconcile; worker/app.go:108 | Runs each sweep; no post-payout/monthly report |
| L-11 | Komissiya sozlamada (bps), kategoriya bo'yicha bo'lishi mumkin; qat… | 🟡 | merchants.commission_bps; merchant_handlers.go:77,114; workspace:244 | Per merchant; default 1500 hardcoded; not per category |
| L-12 | Pul oqimi varianti (A/B/C), refund muddatlari, payout chastotasi ko… | 🟡 | finance.go:16-20 constants; ManualPaymentHold env | Windows are Go constants, not config |
| **16. Xavfsizlik va antifraud** | | | | |
| SEC-01 | Admin va moliya rollari uchun 2FA (TOTP) majburiy; sessiya ≤ 8 soat. | ✅ | totp.go:27; service.go:249-250 (8h cap); totp_test.go |  |
| SEC-02 | Xaridor sessiyasi httpOnly cookie; refresh token rotatsiyasi. | ✅ | auth_handlers.go:274-276; TestConcurrentRefreshAllowsExactlyOneRotation | SameSite=Lax; CSRF token not implemented |
| SEC-03 | OTP: 60 s cooldown, telefon va IP limiti; 5 xato urinishdan keyin 1… | 🟡 | otp.go:20-24; service.go:287-305; app.go:160-163 | Cooldown OK (0 in dev); limits fail open without Redis; no 15-min lock (new code after cooldown) |
| SEC-04 | Telefon raqami interfeysda maskalangan holda ko'rsatiladi. | ✅ | web lib/phone.ts maskPhone (+ test); account-menu | v1.0: phone masked in the UI |
| SEC-05 | Merchant hujjatlari maxfiy saqlanadi, faqat vakolatli rollarga ochi… | ✅ | storage/private.go; merchant_document_handlers.go:85-92 | Private attachment; but 2 files in git (C-5) |
| SEC-06 | Fraud qoidalari: ish vaqtidan tashqari redeem, 1 daqiqada > 5 redee… | 🟡 | redemption_ledger_repository.go:103-132; redemption_fraud.go:38-62 | Burst → hold OK; off_hours/distance only record a signal |
| SEC-07 | Yangi akkaunt uchun birinchi kunda xarid limiti. | ❌ | - | No new-account first-day limit |
| SEC-08 | Har bir admin va sezgir merchant amali audit jurnaliga yoziladi. | ✅ | admin_audit_test.go; audit_logs | DB test (skipped here) |
| SEC-09 | CI'da bog'liqliklar skaneri va sir (secret) qidiruvi. | ✅ | .github/workflows/ci.yml security job (govulncheck, pnpm audit, gitleaks) | v1.0: runs on every CI run |
| SEC-10 | Repo'da maxfiy kalit, parol, token yo'q; prod kalitlari dev kalitla… | 🟡 | rg secrets = none; .env.example dev values; git ls-files .data | No real keys; merchant docs tracked; OTP logged |
| SEC-11 | IP allowlist/VPN admin uchun, tashqi pentest. | ➖ | - | Excluded; not explicit in HANDOVER |
| **17. API standartlari** | | | | |
| API-01 | OpenAPI 3.1 yagona manba; Go va TS kodi undan generatsiya qilinadi. | ✅ | packages/api-spec/openapi.yaml (3.1.0); generate.go; ci.yml:80-120 | Telegram webhook outside spec |
| API-02 | Xatolar RFC 9457 problem+json, barqaror code bilan; 400/401/403/404… | ✅ | catalog_handlers.go:271; problem.go | problem+json with code |
| API-03 | Har bir javobda X-Request-Id; URL orqali versiyalash (/v1). | ✅ | router.go:18,81; /api/v1 paths |  |
| API-04 | Deal ro'yxati pagination bilan (Dealo: cursor). | 🟡 | openapi.yaml:846,877 | page/page_size, not cursor |
| API-05 | R1a endpointlari bor: auth OTP, deals, orders, payment, webhook, me… | 🟡 | openapi.yaml paths | Present: auth, deals, orders, pay, webhook, me/vouchers, redeem/reverse, deals/submit, payouts, payout approve. Missing/renamed: me/vouchers/{id}/qr, me/refunds, merchant/statements, admin/deals/{id}/decision (approve/reject) |
| API-06 | Webhook: imzo tekshiruvi, provider_event_id bo'yicha dedup, takror… | 🟡 | payments/mock.go:62-105; commerce_handlers.go:241 | HMAC OK; dedup by payment state (no provider_event_id) |
| API-07 | Outbox: hodisalar bir marta yuboriladi; Idempotency-Key 24 soat saq… | 🟡 | outbox/outbox.go; worker/app.go:48,68 | At-least-once, log-only publisher; no 24h Idempotency-Key expiry |
| **18. Bildirishnomalar va Telegram** | | | | |
| T-01 | Kirish kodi SMS va Telegram orqali yuboriladi; SMS provayderi sozla… | ➖ | HANDOVER.md DEV-06 | Telegram dropped by decision; SMS log adapter + Eskiz adapter exist |
| T-02 | Telegram bot: /start (til tanlash), /help, /support; token bo'lmasa… | 🟡 | telegram/bot.go:46-57; bot.go:22 | /start,/help,/support work, token-less = off; /start does not store a language (uncommitted, no tests) |
| T-03 | Hisobni Telegram'ga ulash (bir martalik havola) va uzish. | ➖ | HANDOVER.md DEV-06 | Telegram dropped by decision |
| T-04 | Xabarlar: vaucher chiqarildi; vaucher ishlatildi + "Bu men emasman"… | 🟡 | migrations/00013 (kind only voucher_redeemed) | In-app only; no Telegram messages, no 'issued' message |
| T-05 | Muddat eslatmalari (7 va 1 kun), takrorlanmaydi. | ➖ | HANDOVER.md DEV-06 | Telegram dropped by decision (reminders: see FR-04.6) |
| T-06 | Botda "Saytni ochish" tugmasi. | ➖ | HANDOVER.md DEV-06 | Telegram dropped by decision |
| T-07 | Webhook Telegram secret-token sarlavhasi bilan tekshiriladi. | ✅ | bot.go:38-42; telegram_routes.go:29 | Constant-time check; but see C-7 (empty secret in polling mode); no tests |
| T-08 | SMS xarajatini himoya qilish: kunlik limitlar (IP va umumiy). | ✅ | config.go:282-286 SMS_DAILY_CAP_PER_IP / SMS_DAILY_CAP_GLOBAL | v1.0: daily caps per IP and global |
| T-09 | Telegram Mini App initData login. | ➖ | - | Excluded (DEV-06); not in HANDOVER |
| **19. Admin (M15)** | | | | |
| ADM-01 | Deal moderatsiyasi: tasdiqlash, sabab bilan rad etish, changes_requ… | ✅ | moderation.go:124-144; web admin/moderation |  |
| ADM-02 | Merchant KYC: hujjatlarni ko'rish, tasdiqlash/rad etish. | ✅ | review.go; merchant_document_handlers.go; web admin/merchants |  |
| ADM-03 | Moliya: payout tayyorlash/tasdiqlash/reconcile, refund tasdiqlash,… | ✅ | finance_handlers.go; web admin/finance + finance-actions.tsx |  |
| ADM-04 | Sharh shikoyatlari va audit jurnali ko'rinishi. | ✅ | admin_reports_handlers.go; web admin/reports, admin/audit |  |
| ADM-05 | Qo'lda to'lov tasdiqlash (pilot rejim), audit bilan. | ✅ | openapi.yaml:284; commerce_handlers.go:200; TestManualPaymentConfirmationIsIdempotentAndAudited |  |
| ADM-06 | Foydalanuvchi/merchant bloklash, kategoriya va shaharlarni boshqarish. | 🟡 | admin merchants suspend/terminate/reinstate; categories/cities via seed | v1.0: business blocking exists; category/city/user admin screens not built |
| **20. Nofunksional talablar (demo uchun moslashtirilgan)** | | | | |
| NFR-02 | Qidiruv/deal API tez (Dealo: p95 ≤ 800/400 ms). | ❔ | platform/cache/redis.go | No benchmark; run k6/hey against seeded API |
| NFR-04 | Redeem: 100 parallel urinish → aynan 1 muvaffaqiyat. | ✅ | at_redeem_test.go TestAT02 | DB test (skipped here) |
| NFR-05 | Mobil LCP ≤ 2.5 s. | ❔ | apps/web/lighthouserc.cjs (desktop preset, not in CI) | Run Lighthouse mobile |
| NFR-08 | Tillar to'liq tarjima qilingan (UI'da tarjimasiz kalit yo'q). | 🟡 | all 3 catalogs have identical keys (checked); messages.test.ts checks only top-level namespaces | Dead 'shipping' strings; Telegram bot replies Uzbek only |
| NFR-09 | Asosiy oqimlar WCAG 2.2 AA (klaviatura, kontrast, label'lar). | ❔ | no axe in tests/CI | Run axe/Lighthouse a11y |
| NFR-10 | Audit jurnali o'zgarmas. | ✅ | migrations/00032_audit_logs_append_only.sql (trigger) | v1.0: audit_logs append-only like the ledger |
| NFR-12 | Domen yadrosi (to'lov, vaucher, ledger) testlar bilan qoplangan (De… | 🟡 | go test ./...: domain pkgs pass; 38 store tests skipped (no Docker) | finance/telegram pkgs have no unit tests; coverage not measured |
| NFR-13 | Telefon (390 px) va kompyuter (1440 px) ko'rinishida asosiy sahifal… | 🟡 | e2e/public.spec.mjs:22-27 (390 & desktop); merchant.spec.mjs:160 | Only a few pages; advisory in CI |
| **21. Qabul sinovlari (AT)** | | | | |
| AT-01 | Bir xil success webhook 5 marta → 1 vaucher to'plami, 1 journal entry. | 🟡 | at_payments_test.go TestAT01 | Asserts 1 voucher, order paid; journal entries not counted |
| AT-02 | Bitta single-use vaucherga 100 parallel skan → 1 × 200, 99 × 409. | ✅ | at_redeem_test.go TestAT02 | Counts 1 success; 409 codes at domain level only |
| AT-03 | capacity 20, sold 19, 10 parallel order → ko'pi bilan 1 hold. | ✅ | commerce_concurrency_test.go TestOverSellingIsImpossible |  |
| AT-04 | B merchant kassiri A vaucherini skanerlaydi → 403, holat o'zgarmayd… | ✅ | redemption_test.go TestForeignMerchantScanIsRefusedAndAudited (passes) |  |
| AT-05 | Expired/refunded vaucher skani → 422 aniq sabab bilan, ledger yozuv… | ✅ | merchant_redemption_handlers_test.go TestAT05... |  |
| AT-06 | Merchant narxni o'zgartiradi → avvalgi buyurtma va vaucher eski sha… | ✅ | at_gates_test.go TestAT06... |  |
| AT-07 | To'lov failed, hold tugaydi → inventory qaytadi, vaucher yo'q, orde… | ✅ | at_payments_test.go TestAT07... |  |
| AT-08 | PSP hisobotida mahalliy yo'q to'lov → alert. | ➖ | - | Excluded; HANDOVER 'reconciliation' |
| AT-09 | Kassir payout tasdiqlashga urinadi → 403 + audit. | 🟡 | at_gates_test.go TestAT09 (prepare path only); finance.go:171-173 | Approve-by-cashier returns 403 but writes no audit row |
| AT-10 | Yopiq kategoriyada deal yuborish → rad, aniq sabab. | ✅ | at_gates_test.go TestAT10... |  |
| AT-11 | 14 kun ichidagi ishlatilmagan vaucher refund → refunded, teskari pr… | 🟡 | finance_test.go TestUnusedVoucherRefunds... | No refund notification to buyer |
| AT-12 | Vaucher 23:59 Toshkentda tugaydi → 23:58 da 200, 00:01 da 422. | ✅ | voucher_window_test.go; redemption.go:346 |  |
| AT-13 | Hold tugagan, inventory tugagan, PSP success → avtomatik refund, va… | ✅ | at_payments_test.go TestAT13... |  |
| AT-14 | Merchant 1 daqiqada 8 vaucher skanerlaydi → FraudSignal, summa payo… | ✅ | at_fraud_test.go TestAT14... | Holds all merchant payouts, not just the burst |
| AT-15 | 10 tashriflik vaucher, 9 tasi ishlatilgan, 2 parallel skan → 1 × 20… | ✅ | at_redeem_test.go TestAT15... |  |
| AT-16 | "Men emasman" bosilgan → summa payoutga kirmaydi. | ✅ | at_gates_test.go TestAT16... |  |
| AT-17 | Bank rekviziti yangilandi, 72 soat ichida payout → bloklanadi. | ➖ | - | Excluded; HANDOVER bank-change item |
| AT-18 | Payout'ni tayyorlagan odam o'zi tasdiqlaydi → 403 + audit. | 🟡 | finance.go:187-189; 00007 CHECK; finance_test.go TestPayoutNeedsADifferentApprover | 403 self_approval but no audit row |

---

## A. Topilgan xatolar (bugs)

| # | Qayerda | Muammo | Jiddiylik (yuqori/o'rta/past) |
|---|---|---|---|
| 1 | commerce/orders.go:22, openapi.yaml:1036, web stores/cart.ts | `POST /orders` accepts up to 100 items of any variants and merchants. Breaks "one order = one merchant = one variant" (P-05, BR-13, FR-03.1) | yuqori |
| 2 | web `coupons/[voucherId]/page.tsx:31`, commerce/qr.go:12-22 | Voucher QR is rendered once on the server; token is valid only ~60 s, then the cashier gets `qr_expired`. The 30-s refresh (FR-04.4) was never wired in the web | yuqori |
| 3 | finance/finance.go:392-394, :339-343 | Support role alone approves refunds (also after 14 days and dispute refunds). Requirement: support+finance, nobody approves alone (R-07, FR-06.2) | yuqori |
| 4 | app/api/app.go:172-174, platform/sms/sms.go | `LogSender` is wired in every environment when `SMS_PROVIDER=log` (the default). It logs phone + OTP; with `DEMO_MODE=false` nobody can receive a code | o'rta |
| 5 | domain/redemption/redemption.go:274 | Redeem checks the branch belongs to the merchant, not that it is attached to the deal (BR-04) | o'rta |
| 6 | finance/finance.go:171-190 | Payout approval by a wrong role or by the preparer returns 403 but writes no audit row (AT-09, AT-18 ask for "403 + audit"); only `payout_prepare_denied` is audited | o'rta |
| 7 | migrations/00001:258-266 | `vouchers` has no `seq` and no `UNIQUE(order_item_id, seq)`; double issue is prevented only by payment-state logic (BR-01, FR-04.1) | o'rta |
| 8 | migrations/00011, openapi.yaml | Merchant rejection reason is stored but never returned or shown to the merchant (FR-01.7) | o'rta |
| 9 | finance/finance.go:222 | Refund of a partly used multi-use voucher is refused; expired vouchers cannot be refunded in the 90-day window (FR-06.5, FR-06.6) | o'rta |
| 10 | queries/finance.sql PayableRedemptions, ledger.Reversed | Redeem reversal after payout leaves `merchant_payable` negative; nothing deducts it from the next payout (FR-07.5, L-06) | o'rta |
| 11 | redemption_fraud.go:38-62 | `off_hours` and `distance` signals only insert rows; no payout hold and no screen shows `fraud_signals` (SEC-06, R-10) | o'rta |
| 12 | migrations/00001:325 | `audit_logs` is mutable (no trigger) (NFR-10) | o'rta |
| 13 | telegram/bot.go:46-57 | `/start` offers languages as plain text only; nothing is stored or switchable | past |
| 14 | merchant_handlers.go:77,114; merchant_workspace_handlers.go:244 | Default commission 1500 bps is hardcoded in several places (L-11) | past |
| 15 | web account/profile/page.tsx:19 | Phone is shown in full (SEC-04) | past |
| 16 | domain/commerce/model.go:28-31, 00019 | Order `completed/partially_refunded/refunded`, voucher `issued/voided`, payment `chargeback_*` and merchant `suspended/terminated`, deal `paused/archived` are in the enums but no code moves into them | past |

## B. Hujjat va kod o'rtasidagi ziddiyatlar

| # | Hujjat (fayl) nima deydi | Kod aslida nima qiladi |
|---|---|---|
| 1 | commerce/qr.go:13 comment: "The voucher page refreshes at this rate"; ARCHITECTURE/ACCEPTANCE imply working rotation | Web page renders the QR once (DECISIONS.md:210 says "web side not connected yet" — still true) |
| 2 | platform/sms/sms.go:5-6 and app.go:169: "production runs without a sender, so codes cannot be issued" | `loadSMS` defaults to `log` in all envs and `newIdentityService` always wires `LogSender` for it (config.go:267-275, app.go:172) |
| 3 | ROADMAP.md Stage 2 finish line: "full Docker smoke/backup-restore run is verified" | HANDOVER.md: Docker files are "optional and not verified"; docker daemon is not running in this environment either |
| 4 | ACCEPTANCE.md AT-09 ✅ "Cashier cannot approve payout" | Test covers only *prepare* denial; approve denial and AT-18 self-approval have no audit row |
| 5 | ACCEPTANCE.md AT-01 ✅ (also says "ledger lines are not checked yet") | Spec requires 1 journal entry; test counts vouchers/status only |
| 6 | ACCEPTANCE.md header: "16 ✅" — tests "pass" | 38 DB-backed store tests are skipped without Docker/TEST_DATABASE_URL; they could not be re-run in this audit (CI is the only proof) |
| 7 | DECISIONS.md:207 "NOT YET DONE": QR rotation, zaxira code, multi-use, reconciled status, refund approval endpoint, off-hours/distance rules | All of these now exist in code (list is stale) |
| 8 | DECISIONS.md:140,148: argon2id passwords, Google sign-in | Removed by migration 00018; sign-in is phone OTP only |
| 9 | DECISIONS.md:211: backup code is 8 chars `XXXX-XXXX` | Backup code is 10 chars `XXXXX-XXXXX` (qr.go:104-114, pricing.go:80) |
| 10 | HANDOVER.md "Not included" lists PSP, fiscal, bank change, legal, pilot, Docker | Does not list DEV-01/03/05/06/07/08 (stack, hosting/PII, tickets, Mini App, category×city, languages) that the requirements say must be recorded (ROADMAP Stage 3 task 7 also asks for Mini App) |
| 11 | ROADMAP.md Stage 3: "no implementation task complete yet" and task 2 promises language choice | Bot, webhook route and config already exist in the uncommitted working tree; language choice is not implemented |
| 12 | ROADMAP.md Stage 2: native E2E runs in CI "as advisory" | A failing purchase/merchant E2E is accepted in CI, so Stage 1 "Done" is not enforced by CI |

## C. Xavfsizlik bo'yicha kuzatuvlar

| # | Muammo | Qayerda | Tavsiya |
|---|---|---|---|
| 1 | OTP code and phone are written to logs by `LogSender` in any environment | app.go:172-174; sms/sms.go | Only wire it when `DEMO_MODE=true` or `APP_ENV!=production`; fail start otherwise |
| 2 | Rate limiter fails open (Redis missing or down) — OTP/verify limits silently off | identity/service.go:295-298; ENVIRONMENT.md REDIS_URL | Fail closed for OTP routes or require Redis in production |
| 3 | Support alone can approve refunds (money out) | finance.go:392-394 | Require finance second approval |
| 4 | Denied/self approvals are not audited | finance.go:171-190 | Call `RecordDenied` on every refusal |
| 5 | Two merchant documents are tracked in git although `/.data/` is ignored | `.data/documents/*.pdf, *.png` | `git rm --cached`, rotate if real; keep `.data` untracked |
| 6 | CSRF double-submit not implemented; only `SameSite=Lax` | DECISIONS.md:152; auth_handlers.go:276 | Add token or Origin check for cookie-authenticated mutations |
| 7 | Telegram webhook route is mounted whenever the bot exists; in polling mode `TELEGRAM_WEBHOOK_SECRET` may be empty, so `"" == ""` passes and anyone can POST updates that make the bot message arbitrary chat ids | telegram_routes.go:17-30; bot.go:38-42; config.go:254-264 | Refuse empty secret in `Webhook()`; mount route only in webhook mode |
| 8 | Backup codes are derived from voucher id + `VOUCHER_QR_SECRET`; secret leak reveals every code | qr.go:104-114 | Accept as demo risk or store random codes hashed |
| 9 | No dependency or secret scanning in CI | ci.yml | Add govulncheck, `pnpm audit`, gitleaks |
| 10 | Fraud signals are invisible; only burst rule blocks payouts | redemption_fraud.go | Show queue to a fraud/finance role and hold on repeats |
| 11 | Full phone number shown in profile | account/profile/page.tsx:19 | Mask in UI |
| 12 | Audit log is mutable | 00001:325 | Add BEFORE UPDATE/DELETE trigger like the ledger |

## D. Demo sifati (UI, demo ma'lumotlar, tarjimalar, mobil ko'rinish)

- Could not run: DB-backed Go tests (38 skipped, no Docker), Playwright E2E, Lighthouse, live UI. `go test ./...` passes (DB tests skipped); `vitest` 6/6 pass. UI items below come from code reading only.
- Live demo risk: the voucher QR goes stale after about a minute (A-2). Fix before any cashier walkthrough.
- Production demo needs `DEMO_MODE=true` to sign in at all (no SMS/Telegram delivery); the dev code is shown on the page.
- i18n: uz/ru/en catalogs have identical key sets (checked with a script). Leftover unused `shippingTitle` strings in all three. The Telegram bot answers only in Uzbek. The test only compares top-level namespaces.
- Mobile: E2E covers 390 px for the deal first screen and one merchant flow only; no accessibility checks (axe) exist.
- Missing visible features: district filter / nearby, "usable today" filter, sitemap/robots, merchant rejection reason, deal pause/archive, refund notification, expiry reminders.
- Category/city/user management exists only through seed/DB (ADM-06), so an admin cannot show it in a demo.
- Calculator shows totals but no negative-margin warning. Terms are free text with no category templates.
- Telegram Stage 3 is mostly not built: only `/start`, `/help`, `/support` exist, uncommitted and untested.

## E. Eng muhim 10 ta tavsiya (muhimlik tartibida)

1. Enforce one order = one merchant = one variant in `POST /orders` and drop the client cart (A-1).
2. Make the web voucher page refresh its QR every 30 s using the token expiry (A-2).
3. Refund approval: support proposes, finance approves; add the >3 refunds/30 days manual-review flag; allow refund of partly used and recently expired vouchers (A-3, A-9).
4. Audit every denied or self-approved finance action; extend AT-09/AT-18 tests to the approve path (A-6).
5. Wire `LogSender` only in demo/dev, require a real provider otherwise; fix the empty Telegram webhook secret hole; add daily SMS caps (A-4, C-1, C-7).
6. Add `seq` + `UNIQUE(order_item_id, seq)` to vouchers and check the deal's branches on redeem (A-5, A-7).
7. Show the merchant rejection reason; make price proof required and add the >70% discount check; implement pause/archive/suspend transitions (A-8, A-16).
8. Make `audit_logs` append-only, add a state-transition log for order/payment/voucher, and remove tracked `.data/documents` from git (A-12, C-5).
9. Finish Stage 3 Telegram (linking, outbox notifications, 7/1-day reminders, Open-site button) with tests, and update HANDOVER.md with every deviation DEV-01…12 (B-10, B-11).
10. CI hardening: govulncheck/audit/gitleaks, mobile Lighthouse + axe, make the E2E blocking, and refresh stale docs (DECISIONS 207/140/211, ROADMAP Docker line) (C-9, B-3, B-7).

---

## Xulosa

| Holat | Soni |
|---|---|
| ✅ | 90 |
| 🟡 | 74 |
| ❌ | 21 |
| ➖ | 10 |
| ❔ | 4 |

Jami: 199 band.

Bir jumlada umumiy baho: Core money, voucher, redemption and ledger logic is solid and well tested (DB tests could not be re-run here), but the cart/multi-merchant order hole, the stale voucher QR, support-only refund approval, and the nearly empty Telegram stage mean it is not yet a faithful Dealo demo.

## v1.0 recheck (2026-11-11)

Rechecked against the code on branch `feat/v1-polish`. Only rows with new evidence were changed; each changed row says
"v1.0:" in its note. Rows not mentioned below were not re-verified and keep their audit status.

| Status | Count |
|---|---|
| ✅ done | 105 |
| 🟡 partial | 67 |
| ❌ not built | 9 |
| ➖ excluded by decision | 14 |
| ❔ not checkable in code | 4 |
| **Total** | **199** |

Changed in v1.0: R-10, FR-01.7, FR-04.4, SEC-04, SEC-09, T-08, NFR-10 became ✅; ADM-06 became 🟡; T-01, T-03, T-05, T-06
became ➖ (Telegram dropped); DEV-01, DEV-03, DEV-05, DEV-06, DEV-07, DEV-08, DEV-10, DEV-11 became ✅ (recorded in
HANDOVER.md). Still ❌: C-06 prohibited-offer rules, FR-02.5 category term templates, FR-04.6 expiry reminders, FR-06.3 and
FR-06.6 refund cases, FR-06.8 refund abuse rule, FR-07.5 negative balance recovery, FR-09.5 marketing consent, SEC-07 new
account limit. They are listed under "Not yet built" in HANDOVER.md.
