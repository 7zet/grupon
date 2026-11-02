# Dealo va Grupon loyihasi: to'liq tahlil va farqlar

Sana: 2026-11-02. Maqsad: Dealo PRD v1.1 va ADR'lar hamda hozirgi loyihamizni (Grupon, Go + Next.js)
element-bo'yicha solishtirish, har bir farqni nomlash va holatini belgilash.

## 0. Usul va chegaralar

- **Dealo manbasi:** `ibragimoffline/Dealo` repozitoriysi (klonlangan, commit `HEAD`, 2026-11-02).
  O'qilganlar: `docs/prd/00…18` (hammasi), `docs/decisions/ADR-0001…0008`, `docs/analysis/prd-gaps.md`,
  `docs/design/backlog.md`, `docs/setup/tech-stack.md`, `docs/process/approvals.md`, manifestlar.
  O'qilmaganlar: `.claude/agents/*` (agent ta'riflari, mahsulot emas) va `docs/process/agents.md`,
  `workflow.md`, `docs/README.md` (jarayon hujjatlari; mahsulotga ta'siri yo'q).
- **Bizning loyiha:** kontrakt (`packages/api-spec/openapi.yaml`, 90 ta operatsiya), 12 ta migratsiya,
  Go paketlari, web sahifalari, testlar. Har bir "bor/yo'q" da'vo kod yoki `rg`/`grep` natijasi bilan
  tekshirilgan. Tekshirilmagan narsalar **❔** belgisi bilan ko'rsatilgan.
- **Holat belgilari:** ✅ bor (tekshirilgan) · 🟡 qisman · ❌ yo'q · ❔ tekshirilmagan · ➖ bizda bor, Dealo'da yo'q.

## 1. Asosiy xulosa

1. **Dealo — ishlaydigan dastur emas.** Repozitoriy PRD, ADR va dizayn rejasi to'plami. Kod "muzlatilgan"
   (`CLAUDE.md`: foydalanuvchi ruxsatisiz kod yozilmaydi). Dizayn deliverable'lari (DD-00…DD-18) hammasi
   **⬜ boshlanmagan**. Demak "Dealo bilan bir xil" degani — Dealo spetsifikatsiyasiga mos bo'lish, boshqa
   kodni nusxalash emas.
2. **Stek farqi (qasddan):** Dealo — Django + DRF, aiogram bot, Django Admin. Bizda — Go API, Go bot, Next.js
   admin. Bu farq foydalanuvchi qarori bilan saqlanadi. Xatti-harakat (biznes qoidalari, holatlar, ekranlar,
   ma'lumotlar) bo'yicha solishtirish shu hujjatning asosiy vazifasi.
3. **Bizda eng kuchli qismlar:** moliya yadrosi (double-entry ledger, append-only triggerlar, 4-eyes payout,
   refund, breakage, reconciliation, dispute), redeem atomikligi, telefon OTP, merchant onboarding va hujjatlar,
   admin tasdiqlash oqimi, contract-first kod generatsiyasi.
4. **Bizda eng katta bo'shliqlar:** PSP integratsiyasi (faqat mock), SMS va Telegram (faqat log/yo'q),
   merchant panelning 7 ta bo'limi (Orders, Customers, Analytics, Payouts, Reviews, Settings, Staff),
   Scanner PWA, support tizimi, Deal Studio'ning narx dalili/kvota/blackout/kalkulyator qismi, SEO (sitemap,
   robots), bank rekvizitini o'zgartirish himoyasi, MXIK/QQS, kategoriya flaglari.
5. **Qiymat jihatdan xavfli farqlar (tuzatish shart):**
   - Kategoriyalar nomi Dealo'nikidan farq qiladi (bizda `coupons`, `gifts`, `local`, `seasonal-deals`, `goods`,
     `auto-and-home`, `travel`; Dealo'da pilot 4 tasi va R1b/R2 ro'yxati).
   - Zaxira kod: PRD 10 belgi (Crockford base32); bizda 8 belgi (`XXXX-XXXX`).
   - Bron ushlab turish (hold): PRD 15 daqiqa; bizda 10 daqiqa (`ReservationTTL`).
   - Deal, buyurtma va vaucher holatlari PRD'dagidan sezilarli farq qiladi (§5.4 ga qarang).
   - Payout holatlari: PRD `draft → pending_approval → approved → sent → reconciled`; bizda
     `draft/pending_approval/approved/sent/reconciled/failed` (shu mos), lekin eski `payouts` jadvali
     (`pending/processing/paid`) hali bor.

---

## 2. Dealo: to'liq xarita

### 2.1 Mahsulot chegarasi (00-overview)
- Toshkent, 4 pilot kategoriya (Beauty & Wellness, Food & Drink, Things to Do, Health & Fitness).
- Logistika **yo'q** (BR-14): `Shipment`, `ShippingAddress`, `delivery_fee`, kuryer, ombor — hech qachon yo'q.
- Scope tashqarisi: yetkazish, onlayn do'kon, multi-merchant savatcha, platforma hamyoni, AI narx, obuna,
  reklama auksioni, avia/transport/tibbiy xizmat.
- Kanallar R1: responsive web (SEO), Telegram Mini App + bot, merchant web + Scanner PWA, ichki admin.
- Valyuta/til: UZS tiyinda `bigint`; uz-Latn va ru asosiy, uz-Cyrl avtomatik transliteratsiya.
- Hosting: O'zbekiston hududi (PII lokalizatsiyasi, ADR-0007).

### 2.2 Bozor va gipotezalar (01)
- Raqobat tahlili R0 da to'ldiriladi (Telegram kanallar, Instagram, bank keshbeki, marketplace'lar).
- H1–H5: concierge konversiya ≥2%; merchant LOI ≥15; redemption ≥70% (60 kun); repeat ≥20% (90 kun);
  contribution > 0.

### 2.3 Rollar (02-rbac)
Rollar: Mehmon, Xaridor, Merchant owner, Merchant manager, Kassir/scanner, Moderator, Support agent,
Finance operator, Finance approver, Trust & fraud analyst, Superadmin, Analyst. Har bir rol scope bilan
(platforma / merchant / filial). Serverda tekshiruv. 4-eyes moliyada. RBAC matritsasi alohida DD-01.

### 2.4 Kategoriyalar (03)
- 11 kategoriya: Beauty & Wellness, Food & Drink, Things to Do, Health & Fitness (R1a); Automotive,
  Education (R1b); Events, Home Services (R2); Travel (R3); Pickup Goods (R3); Medical (yopiq).
- Feature flag'lar (kategoriya × shahar): `searchable`, `deal_creation_enabled`, `publish_enabled`,
  `purchasable`, `bookable`, `redeemable`, `ads_enabled`.
- Taqiqlangan takliflar ro'yxati (alkogol, tamaki, qimor, retsept dorilari va h.k.).

### 2.5 Modullar (04): M01–M21
R1a: M01 Identity, M02 Location, M03 Taxonomy, M04 Merchant onboarding, M05 Deal Studio, M06 Catalog &
Search, M07 Checkout, M08 Payments, M09 Vouchers, M10 Redemption & Scanner, M11 Ledger & Settlement,
M12 Support & Disputes, M13 Trust & Fraud, M14 Notifications, M15 Admin, M16 Governance.
Keyinroq: M17 Reviews (R1b/R2), M18 Booking (R2), M19 Mobil ilova (R2), M20 CRM/referral (R2), M21 Gift/Loyalty/Ads/Public API (R3).

### 2.6 Funksional talablar (05): FR-01…FR-09
Batafsil farqlar §3 da. Qisqacha:
- FR-01 merchant onboarding (ariza maydonlari, hujjatlar, holatlar, oferta, bank rekvizit o'zgarishi, xodim taklifi).
- FR-02 Deal Studio (maydonlar, variant, narx dalili, limitlar, shart shabloni, kalkulyator, holatlar).
- FR-03 checkout (1 variant, hold 15 daqiqa, kech to'lov AT-13, bitta order = bitta merchant).
- FR-04 vaucher (128-bit token, HMAC, QR 30 s, zaxira kod 10 belgi, multi-use, muddat eslatmasi).
- FR-05 redemption (Scanner PWA, atomik tekshiruv, maskalangan ism, "Siz emasmisiz?" xabari, 10 daqiqa bekor qilish).
- FR-06 refund (qoidalar jadvali: 14 kun avtomatik, 14 kundan keyin support+finance, merchant yopilishi, dispute, breakage, multi-use qisman).
- FR-07 settlement (haftalik payout, 3 kunlik dispute oynasi, payout paketi, merchant hisoboti, manfiy balans).
- FR-08 support (Telegram bot asosiy, ticket, SLA 2 soat/48 soat, dispute payout hold).
- FR-09 discovery (bosh sahifa bloklari, transliteratsiya qidiruvi, filtrlar, SEO landing, marketing rozilik).

### 2.7 Biznes qoidalari (06): BR-01…BR-15
Har biri DB yoki tranzaksiya yoki test bilan ta'minlanishi kerak (BR-01 unique seq, BR-03 redemptions ≤ allowed,
BR-05 sold+held ≤ capacity, BR-09 ledger append-only, BR-10 balanslangan, BR-11 72 soat hold, BR-12 kategoriya gate,
BR-13 bitta merchant, BR-14 logistika yo'q, BR-15 dispute payout hold, …).

### 2.8 Pul oqimi (07)
- Variant A (agent + redemption-based, tavsiya), B (PSP split), C (darhol to'lov + rezerv). Ledger uchalasini qo'llashi kerak.
- Fiskal: PSP fiskal cheki merchant STIR bilan; MXIK, paket kodi, QQS; refund cheki; EDO hisob-faktura.
- Hisoblar rejasi: psp_receivable, bank:operating, customer_prepaid, merchant_payable, merchant_reserve,
  refunds_payable, vat_payable, commission_revenue, breakage_revenue, psp_fees, merchant_receivable.
- Provodka namunasi (100 000 so'm, 15% komissiya, 2% PSP). Namunada yo'q: multi-use, reversal, dispute,
  chargeback, breakage, QQS, variant B/C, kech to'lov refundi.
- Breakage: chegirma yo'qoladi, to'lov 90 kun qaytariladi, keyin yurist xulosasi bo'yicha.
- Reconciliation: kunlik 06:00 (PSP ↔ payments ↔ ledger), payoutdan keyin bank ko'chirmasi, oylik invariant.

### 2.9 Holat mashinalari (08)
- Merchant: `draft → submitted → in_review → approved | rejected`; `approved ⇄ suspended → terminated`.
- Deal: `draft → in_review → approved → scheduled → active ⇄ paused → ended → archived`; `in_review → changes_requested → draft`.
- Order: `created → awaiting_payment → paid → completed`; `awaiting_payment → expired | failed | cancelled`; `paid → partially_refunded | refunded`.
- Payment: `created → pending → succeeded | failed`; `succeeded → chargeback_open → chargeback_won | chargeback_lost`.
- Refund: `requested → approved | rejected`; `approved → processing → succeeded | failed`.
- Voucher: `issued → active → partially_redeemed → redeemed`; `active → expired | refunded | voided`; flag'lar `fraud_hold`, `dispute_open`.
- Redemption: `confirmed → reversed` (10 daqiqa yoki support).
- Booking (R2): `held → confirmed → completed | no_show | cancelled`.
- Payout: `draft → pending_approval → approved → sent → reconciled`; `sent → failed → draft`.
- Dispute: `open → awaiting_evidence → under_review → resolved_customer | resolved_merchant → closed`.
- Har o'tish servis funksiyasi orqali, `state_transitions` jurnaliga, outbox hodisasi bilan; ruxsatsiz o'tish = 422.

### 2.10 Ma'lumotlar modeli va API (10)
- Obyektlar: User/UserRole, Merchant, BankAccount (alohida, tarixli), Branch (PostGIS point), Category (JSON Schema),
  Deal, **DealVersion** (immutable), DealVariant (MXIK, paket kodi, QQS), Inventory (branch nullable, CHECK held+sold≤capacity),
  Order/OrderItem, Payment, Refund, Voucher (token_hmac, backup_code_hmac), Redemption (reversed_at, device, geo),
  LedgerAccount/JournalEntry/JournalLine, Payout/PayoutItem, Ticket/Dispute, FraudSignal, Outbox/AuditLog/StateTransition.
- R1a endpointlar (§11.2): `/v1/auth/otp`, `/v1/auth/otp/verify`, `/v1/auth/telegram`, `/v1/deals` (cursor), `/v1/deals/{slug}`,
  `/v1/orders`, `/v1/orders/{id}/payment`, `/v1/webhooks/{psp}`, `/v1/me/vouchers`, `/v1/me/vouchers/{id}/qr`,
  `/v1/me/refunds`, `/v1/merchant/redemptions`, `/v1/merchant/redemptions/{id}/reverse`, `/v1/merchant/deals`,
  `/v1/merchant/deals/{id}/submit`, `/v1/merchant/statements`, `/v1/admin/deals/{id}/decision`,
  `/v1/admin/payouts/{id}/approve`.
- Standartlar: OpenAPI 3.1, RFC 9457 problem+json, 400/401/403/404/409/422/429, `X-Request-Id`, URL versiyalash.
- Eslatma (GAP-20): merchant onboarding, bank rekvizit, xodim taklifi, "men emasman", support ticket, payout
  tayyorlash, refund tasdig'i, kvota boshqaruvi — PRD endpoint ro'yxatida yo'q; DD-05 da qo'shilishi kerak.

### 2.11 Arxitektura (09) va ADR'lar
- Modular monolith (ADR-0001): Django app'lar: identity, merchants, catalog, pricing, inventory, orders,
  payments, vouchers, redemptions, ledger, settlement, support, trust, notifications, audit, growth.
- PostgreSQL 16 + PostGIS + pg_trgm + unaccent (ADR-0002). Qidiruv: `search_text` + pg_trgm + ruscha FTS + geo.
- Pul tiyinda `bigint`, ID UUIDv7, vaqt UTC, chegaralar `Asia/Tashkent` (ADR-0003).
- Transactional outbox, Idempotency-Key 24 soat, webhook dedup, dead-letter 10 urinishdan keyin (ADR-0004).
- Append-only ledger, DB triggerlar, storno (ADR-0005).
- Bitta Next.js kod bazasi: customer web, Mini App, merchant PWA/Scanner (ADR-0006).
- PII O'zbekistonda; CDN faqat statik; SaaS faqat self-hosted (ADR-0007).
- S3: lokal SeaweedFS, zaxira Garage; kod faqat S3 API (ADR-0008).

### 2.12 NFR (12)
NFR-01 99.5% (R1a); NFR-02 qidiruv p95 ≤800 ms, deal ≤400 ms; NFR-03 order p95 ≤1 s; NFR-04 redeem p95 ≤700 ms,
100 parallel → aynan 1 muvaffaqiyat; NFR-05 LCP ≤2.5 s; NFR-06 RPO 15 daqiqa, RTO 4 soat; NFR-07 500 ko'ruvchi,
30 checkout/daqiqa; NFR-08 uz-Latn, ru, uz-Cyrl; NFR-09 WCAG 2.2 AA; NFR-10 audit 5 yil, log 90 kun;
NFR-11 kunlik reconciliation, 1 soatda alert; NFR-12 domen yadrosi test qamrovi ≥90%.

### 2.13 Acceptance testlar (16): AT-01…AT-18
AT-01 webhook takrori; AT-02 parallel redeem (100 skan → 1×200, 99×409); AT-03 inventory parallel; AT-04 begona merchant (403);
AT-05 expired/refunded skan (422); AT-06 narx snapshot; AT-07 muvaffaqiyatsiz to'lov → hold qaytadi; AT-08 reconciliation;
AT-09 kassir payout 403; AT-10 yopiq kategoriya; AT-11 14 kun refund; AT-12 Asia/Tashkent 23:59; AT-13 kech to'lov;
AT-14 collusion (1 daqiqada 8 skan); AT-15 multi-use parallel; AT-16 dispute payout hold; AT-17 bank rekvizit hold; AT-18 4-eyes.

### 2.14 Xavf, gap va ochiq qarorlar
- **Risklar** (18 §19.1): pul varianti yuridik rad etilishi; merchant chegirmadan zarar; oldindan to'lashga ishonchsizlik;
  collusion; navbat ortiqcha yuklanishi; past redemption; yuqori CAC; PSP kechikishi; hosting ishonchliligi; Telegram bog'liqligi.
- **GAP (analysis/prd-gaps.md):** 27 ta. 🔴 bloklovchi 7 ta: GAP-02 (customer_prepaid invariant), GAP-07 (multi-use
  summa taqsimoti), GAP-08 (redeem reversal provodkasi), GAP-09 (ishlatilgandan keyingi refund provodkasi), GAP-03
  (expired → paid), GAP-04 (partially_redeemed → expired/refunded), GAP-05 (reversal vaucher holati).
  🟠 15 ta (masalan GAP-10 chargeback, GAP-11 QQS, GAP-13 variant B/C, GAP-16 filial kvotasi, GAP-17 muddat formulasi,
  GAP-18 zaxira kod xavfi, GAP-21 owner 2FA, GAP-22 avtomatik refund istisnosi, GAP-19/20 AT va endpoint qamrovi).
  🟢 qolgan tahririy.
- **Ochiq qarorlar D1…D10:** pul varianti (D1), PSP (D2), fiskal sxema (D3), breakage matni (D4), hosting (D5),
  refund muddatlari 14/90 (D6), komissiya stavkalari (D7), payout chastotasi va dispute oynasi (D8), brend (D9),
  mobil ilova (D10). Hammasi ⏳.
- **Yuridik gate'lar G1…G9:** hammasi ⏳ (pul oqimi, fiskal, e-tijorat ro'yxat, PII lokalizatsiya, iste'molchi huquqi,
  merchant oferta, reklama, brend, regulyatsiyalangan kategoriyalar). R1a launch va birinchi merchant shulardan o'tishi kerak.
- **Relizlar (15):** R0 (6 hafta) → R0.5 concierge → R1a (12 hafta, launch gate AT-01…18 + pentest + 7 kun reconciliation)
  → R1b (8 hafta) → R2 (~4 oy) → R3. Jamoa: PM, dizayner, 2 backend, 1–2 frontend, QA, DevOps 0.5, BD 2, support, yurist.
- **Dizayn backlog (design/backlog.md):** DD-00…DD-18, hammasi ⬜. P0: DD-00, DD-14 (1-to'lqin), DD-01, DD-02, DD-03, DD-04, DD-05.

---

## 3. Bizning loyiha: holat inventari

| Qatlam | Holat | Dalil |
|---|---|---|
| Repozitoriy | Monorepo (pnpm + turbo), branch `feat/merchant-and-admin-flows`, GitHub'ga push qilingan | `git status`, `git log` |
| Kontrakt | 90 operatsiya, OpenAPI, TS va Go kodi generatsiya qilinadi | `packages/api-spec/openapi.yaml` |
| Ma'lumotlar bazasi | 12 migratsiya (00001…00012), PostgreSQL, pg_trgm va unaccent yoqilgan, PostGIS yo'q | `services/api/migrations` |
| Go API | 27 paket: app/api, app/worker, domen (catalog, commerce, finance, identity, ledger, merchant, moderation, outbox, redemption, reviews), http, store, platform | `services/api/internal` |
| Worker | Bron muddati, vaucher muddati, deal o'tishlari, breakage, reconciliation, outbox | `internal/app/worker` |
| Web | Next.js App Router, uz/ru/en, ~40 sahifa (public, account, checkout, merchant panel, admin) | `apps/web/src/app` |
| Admin | Web ichida `/admin/*` (moderation, merchants, reports, audit); alohida `apps/admin` yo'q | `apps/web/src/app/[locale]/admin` |
| Testlar | 36 ta Go test fayli (store testlari PostgreSQL'da), 7 ta web test fayli, 3 ta Playwright spec (ishga tushirilmagan) | `services/api`, `apps/web`, `e2e/` |
| Infra | docker-compose.yml bor; production Dockerfile yo'q; CI workflow bor | `infra/`, `.github/workflows` |
| Hujjat | CLAUDE.md, STATUS, DECISIONS, ARCHITECTURE, ENVIRONMENT, DEALO-GAP, shu fayl | `docs/` |

---

## 4. Bajarilish darajasi (taxminiy, modul bo'yicha)

| Dealo modul | Bizda | Izoh |
|---|---|---|
| M01 Identity | 🟡 | Telefon OTP ✅; SMS provayder ❌ (log); Telegram OTP/initData ❌; owner 2FA ❌; TOTP admin/moliya ❌ |
| M02 Location | 🟡 | Toshkent ✅ (boshqa shaharlar yopiq); PostGIS ❌; filial koordinatalari ✅ |
| M03 Taxonomy | 🟡 | 4 pilot kategoriya bor, nomlar farq; feature flag'lar ❌ |
| M04 Merchant onboarding | 🟡 | Rekvizit, STIR, MFO, hisob, oferta, hujjatlar, holatlar ✅; xodim taklifi ❌ (web); bank o'zgarishi himoyasi ❌ |
| M05 Deal Studio | 🟡 | Draft/variant/submit ✅; narx dalili ❌; kvota (faqat stock) 🟡; blackout ❌; shart shabloni ❌; kalkulyator ❌; versiyalash ❌ |
| M06 Catalog & Search | 🟡 | pg_trgm ✅; transliteratsiya ❌; geo saralash ❔; SEO: JSON-LD va alternates ✅, sitemap ❌, robots ❌ |
| M07 Checkout | 🟡 | "Hozir sotib olish" ✅; idempotency ✅; hold 10 daqiqa (PRD 15); kech to'lov AT-13 ❔ |
| M08 Payments | ❌ (haqiqiy) | Faqat mock to'lov va webhook; PSP ❌; fiskal ❌; chargeback ❌ |
| M09 Vouchers | 🟡 | single/multi (redemptions_allowed) ✅; QR 30 s ✅; zaxira HMAC ✅; zaxira 8 belgi ❌ (10 kerak); muddat eslatma ❌ |
| M10 Redemption & Scanner | 🟡 | Atomik redeem ✅; web skaner (zxing) ✅; PWA ❌; "Siz emasmisiz?" ❌; kassir rate limit ❌; 10 daqiqa bekor ✅ |
| M11 Ledger & Settlement | 🟡 | Double-entry ✅; append-only ✅; 4-eyes ✅; 3 kunlik oyna ✅; haftalik jadval ❌ (qo'lda); merchant hisobot ❌; manfiy balans ❌ |
| M12 Support & Disputes | 🟡 | Refund/dispute/resolve endpointlar ✅; ticket tizimi ❌; SLA ❌; Telegram support ❌ |
| M13 Trust & Fraud | 🟡 | Burst (5/daq) ✅; off-hours ✅; masofa 500 m ✅; payout hold ✅; refund abuse (30 kunda >3) ❌; yangi akkaunt limiti ❌ |
| M14 Notifications | ❌ | SMS ❌; Telegram ❌; email (buyurtma tasdig'i) 🟡; outbox → log |
| M15 Admin | 🟡 | Moderatsiya ✅; merchant review ✅; audit ✅; reports ✅; finance UI ❌ (API bor); categories/cities/flags UI ❌ |
| M16 Governance | 🟡 | Audit ✅; terms/privacy sahifalari ✅; ma'lumot o'chirish (DeleteAccount) 🟡; PII maskalash ❔ |
| M17 Reviews | 🟡 | Sharh, reyting, merchant javobi, shikoyat ✅ (R1b vs R2 farqi bor) |
| M18 Booking | ❌ | R2 |
| M19 Mobil | ❌ | Mini App ❌ |
| M20 CRM/Referral | ❌ | R2 |
| M21 Gift/Loyalty/Ads | ➖ | Olib tashlangan (gift); reklama ❌ |

---

## 5. Batafsil solishtirish

### 5.1 Biznes model va yo'nalish

| Element | Dealo | Bizda | Holat | Harakat |
|---|---|---|---|---|
| Oldindan to'lov, agent modeli | Qaror: variant A (tavsiya), D1 ochiq | Mock to'lov, pul modeli yo'q | ❌ | D1 yopilishi, PSP shartnomasi |
| Logistika yo'q (BR-14) | Qat'iy | Savatcha, goods, shipping olib tashlangan | ✅ | Kodda `shipping`/`product` qolmaganini doimiy tekshirish |
| Multi-merchant savatcha | Yo'q (R1) | Olib tashlangan | ✅ | — |
| Gift | R3 | Olib tashlangan | ✅ | — |
| Loyalty, reklama, obuna | R3+ | Yo'q | ✅ | — |
| Komissiya | 10–20% kategoriya bo'yicha | 1500 bps (15%) default | 🟡 | Kategoriya bo'yicha stavka (D7) |
| Breakage | 90 kun qaytarish, keyin yurist | 90 kun (DECISIONS) | 🟡 | Yurist matni (D4) |

### 5.2 Ma'lumotlar modeli

| Dealo obyekt | Bizda | Holat | Farq |
|---|---|---|---|
| User (telegram_id, status) | `users` (phone, email, password_hash, google_sub) | 🟡 | `telegram_id` ❌; `password_hash`/`google_sub` ➖ (olib tashlash kerak, ustunlar qolgan) |
| UserRole (scope_type, scope_id) | `role_assignments` (role, merchant_id) | 🟡 | Filial scope ❌; `scope_type` ❌ |
| Merchant (tin, legal_form, tax_regime, vat_payer, contract_version) | `merchants` (stir, org_form, tax_regime, contract_version) | 🟡 | `vat_payer` ❌ (tax_regime bor); `status` = active/suspended va `review_status` ikki ustun (ikki manba) |
| BankAccount (alohida, tarixli) | `merchants.bank_mfo`, `bank_account` (ustunlar) | 🟡 | Alohida jadval va tarix ❌; verified_by ❌; hold_until ❌ |
| Branch (PostGIS point, district, opening_hours) | `merchant_locations` (lat, lng numeric, opening_hours jsonb) | 🟡 | PostGIS ❌; district ❌ |
| Category (attribute_schema JSON Schema, flags) | `categories` (name, slug, subcategories) | 🟡 | JSON Schema ❌; flags ❌ |
| Deal + DealVersion (immutable) | `deals` (versiya yo'q) | ❌ | Versiyalash ❌; price evidence ❌ |
| DealVariant (MXIK, paket, vat_rate, voucher_type, redemptions_allowed, valid_until) | `deal_options` (title, original_price, deal_price, stock_*, redemptions_allowed) | 🟡 | MXIK ❌; paket kodi ❌; vat_rate ❌; voucher_type ❌ (faqat redemptions_allowed); valid_until (per option) ❌ |
| Inventory (variant, branch nullable, capacity, held, sold, CHECK) | `deal_options` (stock_total, stock_sold, stock_reserved, CHECK) | 🟡 | Filial inventory ❌; CHECK `held+sold ≤ capacity` ❔ (tekshirilmagan) |
| Order, OrderItem (snapshot, hold_expires_at) | `orders` (status, idempotency_key, request_hash), `order_items` (price snapshot) | 🟡 | `hold_expires_at` ❔; holatlar farq (§5.4) |
| Payment (psp, psp_txn_id, fiscal_receipt_ref) | `payments` (provider enum: mock, payme, click, uzum) | 🟡 | PSP adapter ❌; fiscal ref ❌ |
| Refund | `refunds` (requested/approved/... holatlar) | ✅ | requested_by/approved_by 4-eyes ✅ |
| Voucher (token_hmac, backup_code_hmac, redemptions) | `vouchers` (code, qr_token, zaxira_code_hmac, redemptions_allowed/used) | 🟡 | token 128-bit random ❌ (bizda HMAC(id) bilan imzolangan QR); qr_token DB'da saqlanadi |
| Redemption (reversed_at, device, geo) | `redemptions` (geo_lat, geo_lng, reversed) | 🟡 | device_info ❌ |
| Ledger (LedgerAccount, JournalEntry, JournalLine) | `ledger_accounts`, `journal_entries`, `journal_lines` | ✅ | Hisob nomlari PRD §8.3 dan farq (`commission_revenue` va h.k. tekshirilmagan) ❔ |
| Payout, PayoutItem (prepared_by, approved_by, bank_doc_ref) | `payout_runs` (prepared_by, approved_by CHECK) | ✅ | `payout_items` ❔; eski `payouts` jadvali ➖ (ishlatilmaydi, olib tashlash) |
| Ticket, Dispute | `disputes` ✅; `tickets` ❌ | 🟡 | Ticket ❌ |
| FraudSignal | `fraud_signals` (rule, detail) | ✅ | score ❌; reviewed_by ❌ |
| Outbox, AuditLog, StateTransition | `outbox`, `audit_logs` ✅; `state_transitions` ❌ | 🟡 | Holat o'tish jurnali ❌ (audit faqat bir qismi) |
| Phone OTP | `phone_otps` | ✅ | — |
| Auth tokens (verify/reset) | `auth_tokens` (migratsiya 00002) | ➖ | Email flow olib tashlangan; jadval hali bor (00009 da drop qilingan — tekshirish kerak ❔) |
| Savat, saved, promo, kolleksiya, banner | Olib tashlangan | ✅ | Migratsiya 00009 |
| Homepage section (type: collection, banner) | `homepage_sections` (type enum hali collection/banner) | 🟡 | Enum qiymatlari olib tashlanmagan ➖ |
| Exchange rate, USD | `currency_code` enum `'UZS','USD'` hali bor | 🟡 | USD enum qiymati ➖ (olib tashlash) |

### 5.3 Merchant onboarding (FR-01)

| Talab | Bizda | Holat |
|---|---|---|
| STIR, yuridik nomi, tashkiliy shakl (MChJ/YaTT) | ✅ | ✅ |
| Soliq rejimi (QQS/aylanma) | ✅ | ✅ |
| Mas'ul shaxs, telefon | ✅ | ✅ |
| Filiallar | Ro'yxatdan o'tishda bitta manzil; boshqa filiallar `merchant/locations` (CRUD) | 🟡 (ro'yxatdan o'tishda bitta) |
| Bank rekvizit (hisob, MFO) | ✅ (validatsiya: 5 va 20 raqam) | ✅ |
| Hujjatlar: guvohnoma, litsenziya, filial fotosi, versiyalash | Guvohnoma va bank xati ✅ (versiyasiz); litsenziya ❌; filial fotosi ❌ (media orqali bor, ulanmagan); versiyalash ❌ | 🟡 |
| Holatlar: draft → submitted → in_review → approved/rejected; approved → suspended → approved/terminated | `review_status`: draft, submitted, in_review, approved, rejected, suspended, terminated (CHECK) — o'tish qoidalari: submitted→in_review→approved/rejected ✅; suspended/terminated o'tishlari ❌ (API yo'q) | 🟡 |
| Oferta elektron qabul (vaqt, IP, versiya, shaxs) | Vaqt, IP, versiya ✅; "qabul qilgan shaxs" — owner (user_id) ❔ | 🟡 |
| Bank o'zgarishi: owner 2FA → platforma tasdig'i → 72 soat payout hold → eski raqamga xabar | ❌ (hech biri yo'q) | ❌ |
| Owner xodimlarni telefon orqali taklif qiladi, filialga biriktiradi | `merchant/staff` endpoint bor; telefon taklifi ❔; web sahifa ❌ | 🟡 |
| Rad etish sababi merchantga ko'rinadi | Sabab saqlanadi; merchant API qaytarmaydi | 🟡 |

### 5.4 Holat mashinalari — farqlar (eng muhim qism)

| Obyekt | Dealo holatlari | Bizda (kod) | Farq va tuzatish |
|---|---|---|---|
| Merchant | draft, submitted, in_review, approved, rejected, suspended, terminated | `review_status` shu qiymatlar; `status` = active/suspended | Ikki ustun; `status` ni `review_status` ga birlashtirish kerak |
| Deal | draft, in_review, approved, scheduled, active, paused, ended, archived, changes_requested | `deal_status` enum: draft, pending_review, rejected, scheduled, active, paused, sold_out, expired | **Farq:** `in_review`↔`pending_review`; `approved` yo'q; `ended`/`archived` yo'q; `changes_requested` yo'q; `sold_out`, `rejected` (Dealo'da `changes_requested` bor) ➖ |
| Order | created, awaiting_payment, paid, completed, expired, failed, cancelled, partially_refunded, refunded | `order_status` enum: pending_payment, paid, failed, cancelled, refunded, partially_refunded | **Farq:** `awaiting_payment`↔`pending_payment`; `created`, `completed`, `expired` yo'q |
| Payment | created, pending, succeeded, failed, chargeback_open, chargeback_won, chargeback_lost | `payment_status`: created, pending, succeeded, failed, refunded | **Farq:** chargeback holatlari yo'q; `refunded` Dealo'da alohida Refund obyekti |
| Refund | requested, approved, rejected, processing, succeeded, failed | `refunds.status` CHECK: requested, approved, processing, succeeded, rejected, failed | ✅ (mos) |
| Voucher | issued, active, partially_redeemed, redeemed, expired, refunded, voided; flag'lar fraud_hold, dispute_open | `voucher_status`: active, redeemed, expired, refunded, cancelled | **Farq:** `issued`, `partially_redeemed` yo'q; `voided` ↔ `cancelled`; flag'lar (`fraud_hold`, `dispute_open`) yo'q |
| Redemption | confirmed, reversed | `redemptions.reversed_at` (ikki holat) | ✅ (mantiqan mos) |
| Payout | draft, pending_approval, approved, sent, reconciled, failed | `payout_runs.status` CHECK: draft, pending_approval, approved, sent, reconciled, failed | ✅ (mos) — eski `payout_status` (pending/processing/paid) ➖ olib tashlash |
| Dispute | open, awaiting_evidence, under_review, resolved_customer, resolved_merchant, closed | `disputes.status`: open, resolved_customer, resolved_merchant | **Farq:** `awaiting_evidence`, `under_review`, `closed` yo'q |
| Booking | R2 | Yo'q | R2 |

**Ochiq holat-mashina gap'lari (Dealo o'zi):** GAP-03 (expired→paid), GAP-04 (partially_redeemed→expired/refunded),
GAP-05 (reversal qaytishi), GAP-06 (issued ma'nosi), GAP-14 (order completed), GAP-15 (dispute muddati).
Bizda ham shu savollar ochiq; ulardan hech biri hal qilinmagan.

### 5.5 Deal Studio (FR-02)

| Talab | Bizda | Holat |
|---|---|---|
| Kategoriya, sarlavha, tavsif (uz majburiy, ru ixtiyoriy, uz-Cyrl avtomatik) | Sarlavha/tavsif uz/ru/en; uz-Cyrl ❌ | 🟡 |
| Rasmlar (min 3) | Media library, deal'da min 3 (DealInput: minItems 3) | ✅ |
| Variantlar (original, deal price, voucher_type, redemptions_allowed) | ✅ (voucher_type ❌, redemptions_allowed ✅) | 🟡 |
| MXIK, paket kodi, QQS stavkasi | ❌ | ❌ |
| Narx dalili (menyu/prays/chek fotosi), 70% dan yuqori chegirma tekshiruvi | ❌ | ❌ |
| Global kvota, filial kvotasi | Global stock ✅; filial kvotasi ❌ | 🟡 |
| Kunlik redemption limiti | ❌ | ❌ |
| Bitta xaridor limiti | `max_per_user` ✅ | ✅ |
| Blackout kunlar/soatlar | ❌ | ❌ |
| Shartlar shabloni (kategoriya bo'yicha) | ❌ (terms maydoni qo'lda) | ❌ |
| Kampaniya kalkulyatori (sof tushum, manfiy ogohlantirish) | ❌ | ❌ |
| Holat oqimi (draft → in_review → approved → …; changes_requested) | draft → pending_review → active/scheduled/rejected (§5.4) | 🟡 |
| Faol deal narx/shart o'zgarishi → qayta moderatsiya | `review_fingerprint` orqali (qayta moderatsiya mantig'i bor) ❔ | 🟡 |
| Deal wizard 6 qadam (basics, options, media, details, locations, review) | `merchant/(panel)/deals/new` sahifa bor; qadamlar va jonli preview ❔ | 🟡 |

### 5.6 Checkout (FR-03)

| Talab | Bizda | Holat |
|---|---|---|
| Faqat "Hozir sotib olish", 1 variant, N dona | ✅ (savat olib tashlangan) | ✅ |
| Checkout: narx, asl narx, muddat, filiallar, bron talabi, refund qoidasi, shartlarni belgilash | Narx va muddat ✅; filiallar va bron talabi ❔; "shartlarni belgilab tasdiqlash" checkbox ❔ | 🟡 |
| Server narxni qayta hisoblaydi, deal versiyasi snapshot | Narx snapshot ✅ (order_items); versiya ❌ | 🟡 |
| Hold 15 daqiqa | 10 daqiqa (`ReservationTTL`) | ❌ (farq) |
| Hold tugaganda order expired, inventory qaytadi | Reservation worker ✅; order holati `expired` yo'q (pending_payment/failed) | 🟡 |
| Kech to'lov: inventory bo'lsa vaucher, bo'lmasa avtomatik refund (AT-13) | ❔ (tekshirilmagan) | ❔ |
| Bitta order = bitta merchant = bitta to'lov (BR-13) | Savat yo'q, bitta deal; kod darajasida ❔ | 🟡 |
| Telefon orqali kirish checkout'da | ✅ | ✅ |
| Idempotency-Key | ✅ (orders) | ✅ |

### 5.7 Vaucher (FR-04)

| Talab | Bizda | Holat |
|---|---|---|
| Har bir dona uchun bitta voucher | ✅ (issueVoucher per unit) | ✅ |
| Token 128-bit tasodifiy, DB'da HMAC | QR: `gv2:<id>:<oyna>` HMAC bilan imzolangan; DB'da faqat zaxira HMAC; QR token DB'da (seed) | 🟡 (farq: token id dan olingan, tasodifiy emas) |
| QR ichida opaque token, PII yo'q | ✅ (id bor, PII yo'q) | ✅ |
| Zaxira kod 10 belgi (Crockford base32) | 8 belgi, `XXXX-XXXX` (alfavit: 23456789ABCDEFGHJKMNPQRSTUVWXYZ — Crockford'ga yaqin) | 🟡 (uzunlik farq) |
| QR har 30 s yangilanadi (token + vaqt imzosi) | ✅ (oyna 30 s, oldingi oyna qabul) | ✅ |
| Web'da QR yangilanishi | Voucher sahifasi qayta so'rashi kerak (web ulanmagan) ❔ | 🟡 |
| multi_use, redemptions_allowed, har skanerda +1, oxirgisida redeemed | ✅ | ✅ |
| Muddat tugashidan 7 va 1 kun oldin eslatma | ❌ | ❌ |
| Muddat formulasi (GAP-17): min(valid_until, purchased+N kun), 23:59:59 Tashkent | valid_days yoki valid_until (deal darajasida) ✅; Tashkent kun oxiri ❔ | 🟡 |

### 5.8 Redemption va scanner (FR-05)

| Talab | Bizda | Holat |
|---|---|---|
| Scanner PWA (kamera QR) | Web'da voucher-redeem-panel (zxing) bor; PWA manifest/offline ❌ | 🟡 |
| Zaxira kod kiritish | ✅ (code) | ✅ |
| Bitta tranzaksiyada tekshiruv (token, holat, muddat, merchant/filial, to'lov, flag, kunlik limit) | Holat, muddat, merchant, filial ✅; to'lov holati ❔; flag ❌; kunlik limit ❌ | 🟡 |
| Atomik redeem, parallel 409 | `FOR UPDATE` ✅; test (AT-02) ❔ | 🟡 |
| Kassir natijasi: deal, variant, maskalangan ism, qolgan tashriflar | ❔ (tekshirilmagan) | ❔ |
| Xaridorga darhol xabar "Siz emasmisiz?" + shikoyat tugmasi | ❌ (xabar matni yo'q) | ❌ |
| 10 daqiqa ichida redeem bekor qilish | ✅ (ReversalWindow 10 min, faqat kassir o'zi; owner/manager istalgan vaqt) | ✅ (Dealo: kassir 10 daqiqa, undan keyin support — bizda owner ham istalgan vaqt: farq) |
| Kassir rate limit (30/daqiqa) | ❌ | ❌ |
| Offline yo'q | Offline yo'q; zaxira kod orqali telefon | ✅ |
| Geo/device yozish | geo ✅ (kassir ruxsat etilsa); device ❌ | 🟡 |

### 5.9 Refund (FR-06)

| Qoida | Bizda | Holat |
|---|---|---|
| 14 kun ichida ishlatilmagan → avtomatik | ✅ (`RefundWindow`) | ✅ |
| 14 kundan keyin → xaridor so'rovi → support + finance | `requested` → `approve` (support/finance) ✅ | ✅ |
| Merchant yopilgan/xizmat ko'rsatmagan → to'liq, merchantdan undiriladi | ❌ (alohida jarayon yo'q) | ❌ |
| Ishlatilgan, sifat shikoyati → dispute → finance approver | Dispute ✅; finance approver ✅ | 🟡 |
| Muddati o'tgan → breakage | ✅ (breakage job) | ✅ |
| Multi-use qisman → qolgan tashriflar ulushi | ❌ (GAP-07 jadval yo'q) | ❌ |
| Refund faqat asl to'lov usuliga | Mock; PSP yo'q | ❌ |
| Refund abuse (30 kunda >3) | ❌ | ❌ |
| Avtomatik refund 4-eyes istisnosi (GAP-22) | Avtomatik = tizim; istisno qoidasi ❔ | 🟡 |

### 5.10 Settlement va payout (FR-07)

| Talab | Bizda | Holat |
|---|---|---|
| Payable har redeemda hisoblanadi | ✅ (Redeemed postings) | ✅ |
| Haftalik payout jadvali | Qo'lda (`PreparePayout`); jadval ❌ | ❌ |
| 3 kunlik dispute oynasi | ✅ (PayoutDisputeWindow) | ✅ |
| Payout paketi: tayyorlash → tasdiqlash → bank → hujjat raqami → reconciliation | ✅ (bank_doc_ref, reconcile) | ✅ |
| Merchant hisobot (redeemlar, komissiya, ushlab qolish, refund) | ❌ (statements endpoint PRD'da, bizda yo'q) | ❌ |
| Manfiy balans keyingi payoutdan ushlab qolinadi; 60 kun → undirish | ❌ | ❌ |
| Avtomatik bank API (R2) | ❌ (R2 dan) | ➖ kelajak |
| Breakage hisobot (soni, summa, kategoriya) | ❌ | ❌ |

### 5.11 Support va dispute (FR-08)

| Talab | Bizda | Holat |
|---|---|---|
| Telegram bot asosiy kanal | Yo'q | ❌ |
| Web forma, telefon | Yo'q | ❌ |
| Ticket: kategoriya, bog'liq order/voucher, dalil fayllari, SLA | Yo'q (faqat dispute) | ❌ |
| SLA: birinchi javob ≤ 2 soat (09–21), hal ≤ 48 soat | Yo'q | ❌ |
| Dispute ochilganda payout hold | ✅ (open dispute → PayableRedemptions chiqaradi) | ✅ |
| Dispute ochish muddati (GAP-15) | ❌ | ❌ |

### 5.12 Discovery va SEO (FR-09)

| Talab | Bizda | Holat |
|---|---|---|
| Bosh sahifa: tuman tanlash, kategoriyalar, "yaqin atrofda", "tugab bormoqda" | Bosh sahifa: featured/popular bo'limlari; kategoriya ✅; tuman ❌; "yaqin" ❌; "tugab bormoqda" (urgency — real data) ❔ | 🟡 |
| Qidiruv lotin va kirilda bir xil | Lotin ✅ (unaccent/pg_trgm); kirill ↔ lotin transliteratsiya ❌ | 🟡 |
| Filtrlar: kategoriya, tuman, narx, chegirma, "bugun ishlatsa bo'ladi" | Kategoriya/narx/sort ✅ (listing); tuman ❌; chegirma ❔; "bugun" ❌ | 🟡 |
| SEO: kategoriya × tuman landing, SSR, schema.org/Offer | SSR ✅; JSON-LD ✅ (deal page); alternates ✅; kategoriya × tuman ❌; sitemap ❌; robots ❌ | 🟡 |
| Marketing rozilik, obuna chiqish | ❌ | ❌ |

### 5.13 Trust va fraud (§12.1)

| Ssenariy | Dealo nazorati | Bizda | Holat |
|---|---|---|---|
| Merchant xaridorsiz redeem | Xabar + "Siz emasmisiz?", 3 kun oyna, skaner geo/vaqt | Oyna ✅, geo ✅ (yozish), xabar ❌ | 🟡 |
| Ish vaqtidan tashqari / 1 daqiqada >5 redeem / filialdan >500 m | Signal → analyst | Burst ✅ (>5/daq, hold), off-hours ✅ (07–23 UZT), masofa ✅ (500 m) — signal qoidalari hold qo'yadi (burst) | ✅ |
| QR skrinshot sotish | Aylanuvchi QR, egasi ismi kassirga | QR aylanadi ✅; ism ko'rsatish ❔ | 🟡 |
| Zaxira kod topish | 10 belgi, kassir 30/daq, 10 xatoda blok | 8 belgi; rate limit ❌ | 🟡 |
| O'g'irlangan karta | PSP 3DS/OTP, yangi akkaunt limiti | PSP yo'q | ❌ |
| Refund suiiste'moli | 30 kunda >3 | ❌ | ❌ |
| Soxta chegirma | Narx dalili, moderatsiya, shikoyat | Moderatsiya ✅; narx dalili ❌ | 🟡 |
| Bank rekvizit o'g'irlash | 2FA, 72 soat hold, eski raqamga xabar | ❌ | ❌ |
| Ichki xodim | 4-eyes, audit, eng kam huquq | 4-eyes ✅, audit ✅, RBAC enum ✅ | ✅ |

### 5.14 Xavfsizlik va RBAC (§12.2, 02)

| Talab | Bizda | Holat |
|---|---|---|
| OWASP ASVS L2 (moliya/admin/merchant) | Rate limit (auth) ✅; problem+json ✅; ASVS mapping ❌ | 🟡 |
| Admin va moliya: 2FA TOTP, IP allowlist, sessiya 8 soat | ❌ (2FA yo'q; sessiya TTL tekshirilmagan) | ❌ |
| Xaridor: httpOnly cookie; Telegram initData | httpOnly cookie ✅; initData ❌ | 🟡 |
| PII maskalash (telefon) | ❔ | ❔ |
| Hujjatlar shifrlangan bucket | Hujjatlar maxfiy papkada ✅ (shifrlash ❌) | 🟡 |
| Tashqi pentest | ❌ | ❌ |
| CI: SCA va sir qidiruvi | CI workflow bor ❔ (tekshirilmagan) | ❔ |
| Incident jarayoni | ❌ | ❌ |
| RBAC matritsasi (DD-01) | Hujjat yo'q; kod ruxsatlari bor | 🟡 |

### 5.15 Notifications va integratsiyalar (§10.6)

| Kanal | Dealo R1a | Bizda | Holat |
|---|---|---|---|
| To'lov (PSP: Payme/Click/Uzum) | 1 PSP | Mock (enum'da payme/click/uzum bor, adapter yo'q) | ❌ |
| Fiskal (PSP orqali) | Kerak | ❌ | ❌ |
| SMS (mahalliy provayder) | R1a | `LogSender` (faqat development); production provayder ❌ | ❌ |
| Telegram Bot API, Mini App | R1a | ❌ | ❌ |
| Xarita (Yandex/2GIS) | R1a (geokodlash) | ❌ (koordinatalar qo'lda) | ❌ |
| EDO | R2 | ❌ | ➖ kelajak |
| Email | Bildirishnomalar R1a | SMTP sender ✅; buyurtma emaillari ✅ (commerce emails) | ✅ (qisman) |
| Outbox → worker | ✅ (outbox relay, log publisher) | Broker ❌ (log) | 🟡 |

### 5.16 Admin (M15)

| Ekran | Dealo | Bizda | Holat |
|---|---|---|---|
| Moderatsiya deal | ✅ | `/admin/moderation` ✅ | ✅ |
| Merchant review | Merchant KYC | `/admin/merchants` ✅ | ✅ |
| Hisobotlar (review reports) | ✅ | `/admin/reports` ✅ | ✅ |
| Audit log | ✅ | `/admin/audit` ✅ | ✅ |
| Moliya: payout tayyorlash/tasdiqlash, refund, reconcile | Django admin view'lari | API bor, UI yo'q | 🟡 |
| Support: ticket, refund taklifi, vaucher bloklash | ✅ | Yo'q | ❌ |
| Users, businesses, deals & coupons, orders & transactions | ✅ (D2) | Yo'q | ❌ |
| Categories, cities, flags, collections, banners, homepage builder, exchange rate | ✅ (D3) | Yo'q (ko'p narsa olib tashlangan) | ❌ / ➖ |
| Roles & permissions (admin, moderator, support, finance, superadmin) | ✅ (D4) | Rollar DB'da; UI ❌ | ❌ |
| System settings | ✅ (D4) | ❌ | ❌ |
| Analyst (agregat analitika, PII maskalangan) | Rol | ❌ | ❌ |

### 5.17 Merchant panel (CLAUDE.md menyu bilan, Dealo §5)

Dealo merchant portali: `Overview, My Business, My Deals, Create Deal, Orders & Sales, Voucher Redemption, Customers, Analytics & Reports, Payouts & Finance, Reviews, Media Library, Settings` (loyiha menyusi).

| Bo'lim | Bizda sahifa | Holat |
|---|---|---|
| Overview | `merchant/(panel)/dashboard` | ✅ (metrikalar, chart — ❔) |
| My Business | `merchant/(panel)/business` | ✅ |
| My Deals | `merchant/(panel)/deals` | ✅ |
| Create Deal | `merchant/(panel)/deals/new` | 🟡 |
| Orders & Sales | Yo'q | ❌ |
| Voucher Redemption | `merchant/(panel)/redeem` | 🟡 |
| Customers | Yo'q (API: `merchant/customers` bor) | ❌ |
| Analytics & Reports | Yo'q (API: `merchant/analytics`) | ❌ |
| Payouts & Finance | Yo'q (API: `merchant/payouts`) | ❌ |
| Reviews | Yo'q (API: `merchant/reviews`) | ❌ |
| Media Library | `merchant/(panel)/media` | ✅ |
| Settings | Yo'q (API: `merchant/settings`) | ❌ |
| Staff | Yo'q (API: `merchant/staff`) | ❌ |
| Holat sahifasi (tekshiruv) | `merchant/(panel)/review` | ✅ (yangi) |
| Hujjatlar | Review sahifasida yuklash ✅ | ✅ |

### 5.18 Xatti-harakat va UX

| Element | Dealo | Bizda | Holat |
|---|---|---|---|
| Til | uz-Latn, ru; uz-Cyrl avtomatik | uz, ru, en (uz-Cyrl ❌) | 🟡 |
| Valyuta | UZS tiyinda | UZS tiyinda ✅ (USD enum qiymati qolgan) | ✅ |
| Vaqt zonasi | Asia/Tashkent | Shu zona: fraud (FixedZone), worker; to'liq ❔ | 🟡 |
| Dark mode | Yo'q (out of scope) | Yo'q | ✅ |
| Animatsiya (checkout, merchant forma, admin) | Taqiqlangan (CLAUDE.md) | Qoidaga rioya ❔ | ❔ |
| Urgency matnlari (real data) | Qat'iy | ❔ | ❔ |
| Deal rangi (qizil faqat chegirma/Buy/urgency) | Qat'iy | Tokenlar bor; ❔ | ❔ |
| Performance | LCP ≤2.5 s | Lighthouse: perf 98–100 (production build, hujjat) | ✅ (eski o'lchov) |
| Accessibility WCAG 2.2 AA | Asosiy oqimlar | ❔ (tekshirilmagan) | ❔ |

### 5.19 Testlar va sifat (16, 12)

| Talab | Bizda | Holat |
|---|---|---|
| Concurrency: 100 buyer, 10 stock → 10 | ✅ (`TestOverSellingIsImpossible`) | ✅ |
| Ledger balanslangan | ✅ (`TestLedgerStaysBalancedAndReconciles`) | ✅ |
| 4-eyes payout | ✅ (`TestPayoutNeedsADifferentApprover`) | ✅ (AT-18) |
| Breakage | ✅ | ✅ |
| Refund 14 kun avtomatik | ✅ | ✅ (AT-11 yarmi) |
| Burst fraud | ✅ (`TestRedemptionBurstHoldsMerchantPayouts`) | ✅ (AT-14 yarmi) |
| Outbox | ✅ | ✅ |
| Parallel redeem 100 → 1 | ❔ (commerce test bor, redeem testi tekshirilmagan) | ❔ (AT-02) |
| Webhook takrori (AT-01) | ❔ | ❔ |
| Begona merchant 403 (AT-04) | ❔ (kod bor, test ❔) | ❔ |
| Expired/refunded skan 422 (AT-05) | ❔ | ❔ |
| Narx snapshot (AT-06) | ❔ | ❔ |
| Failed to'lov hold (AT-07) | ❔ | ❔ |
| Reconciliation alert (AT-08) | ❔ (worker reconcile bor, alert ❌) | 🟡 |
| Kassir payout 403 (AT-09) | ❔ (`TestModeratorCannotReadFinanceOrAuditWrites` bor — moderator, kassir emas) | 🟡 |
| Yopiq kategoriya (AT-10) | ❌ (flag yo'q) | ❌ |
| Vaqt zonasi (AT-12) | ❌ | ❌ |
| Kech to'lov (AT-13) | ❌ | ❌ |
| Multi-use parallel (AT-15) | ❌ | ❌ |
| Dispute hold (AT-16) | ✅ (PayableRedemptions dispute filtri) | ✅ |
| Bank rekvizit hold (AT-17) | ❌ | ❌ |
| Test qamrovi ≥90% (NFR-12) | ❔ (o'lchanmagan) | ❔ |
| E2E (Playwright) | 3 spec yozilgan, oxirgi ishga tushirish eski (moliya va olib tashlashdan oldin) | 🟡 |
| k6 flash-sale | ❌ | ❌ |
| Lighthouse CI | Konfiguratsiya bor, natija hujjatda | 🟡 |

### 5.20 NFR (12)

| Kod | Talab | Bizda | Holat |
|---|---|---|---|
| NFR-01 | 99.5% | Production yo'q | ❌ (hosting D5) |
| NFR-02 | Qidiruv p95 ≤800 ms | ❔ | ❔ |
| NFR-03 | Order p95 ≤1 s | ❔ | ❔ |
| NFR-04 | Redeem p95 ≤700 ms | ❔ | ❔ |
| NFR-05 | LCP ≤2.5 s | Lighthouse eski o'lchovda ✅ | ✅ |
| NFR-06 | RPO 15 daq, RTO 4 soat | Backup ❌ | ❌ |
| NFR-07 | 500 ko'ruvchi, 30 checkout/daq | Yuklama testi ❌ | ❌ |
| NFR-08 | uz-Latn, ru, uz-Cyrl | uz-Cyrl ❌ | 🟡 |
| NFR-09 | WCAG 2.2 AA | ❔ | ❔ |
| NFR-10 | Audit 5 yil, log 90 kun | Audit jadvali ✅; retention siyosati ❌ | 🟡 |
| NFR-11 | Kunlik reconciliation, 1 soat alert | Worker har tick'da ishlaydi; alert ❌ | 🟡 |
| NFR-12 | Qamrov ≥90% | ❔ | ❔ |

---

## 6. Farqlar ro'yxati (bitta joyda)

### 6.1 Dealo'da bor, bizda yo'q (tuzatish kerak)
1. Haqiqiy PSP integratsiyasi (Payme/Click/Uzum), webhook imzo tekshiruvi, PSP status so'rovi.
2. Fiskal chek (PSP orqali), MXIK, paket kodi, QQS stavkasi (variant va chek darajasida), refund cheki.
3. SMS provayder, Telegram bot (support, OTP, bildirishnoma), Telegram Mini App va initData login.
4. Telegram OTP kanali; owner va admin uchun TOTP 2FA; bank rekvizit o'zgarishida 2FA + tasdiq + 72 soat hold.
5. Merchant onboarding: litsenziya hujjati, filial fotosi, hujjat versiyalash, suspended/terminated o'tishlari,
   xodim taklifi (telefon orqali) va filialga biriktirish UI, merchant rad etish sababini ko'rishi.
6. Deal Studio: versiyalangan deal (immutable), narx dalili, 70% qo'shimcha tekshiruv, filial kvotasi,
   kunlik redemption limiti, blackout, shart shabloni (kategoriya), kampaniya kalkulyatori, changes_requested holati.
7. Kategoriya flaglari (`searchable`, `publish_enabled`, …) va JSON Schema atributlari (DD-17).
8. Qidiruv: kirill ↔ lotin transliteratsiya, geo (PostGIS), tuman filtri, "bugun ishlatsa bo'ladi".
9. SEO: kategoriya × tuman landing sahifalari, sitemap, robots.txt, hreflang (uz-Latn/ru).
10. Vaucher: 128-bit tasodifiy token (HMAC bilan saqlash), zaxira kod 10 belgi, muddat eslatmalari (7 va 1 kun),
    `partially_redeemed` va `voided`, flag'lar (`fraud_hold`, `dispute_open`), voucher_type maydoni.
11. Redemption: "Siz emasmisiz?" xabari va shikoyat tugmasi, kassir rate limit (30/daq), xaridor maskalangan ismi,
    kunlik limit, qurilma yozuvi, to'lov holati tekshiruvi, reversal'dan keyin vaucher holati qaytishi (GAP-05).
12. Settlement: haftalik payout jadvali, merchant hisoboti (statements), manfiy balans ushlab qolish, 60 kun undirish,
    breakage hisoboti.
13. Support: ticket (kategoriya, bog'lanish, dalil fayllari), SLA taymer (2 soat / 48 soat), Telegram support.
14. Trust: refund suiiste'moli (30 kunda >3), yangi akkaunt limiti, zaxira kod brute-force blok, skor (score) va reviewed_by.
15. Admin: moliya UI (payout, refund, reconcile), support UI, users/businesses/deals/orders/transactions, categories,
    cities, flags, roles & permissions UI, system settings, analyst rol.
16. Analytics eventlar (DD-08) va KPI hisoblash (17).
17. Holat jurnali (`state_transitions`) — har bir o'tish yoziladi.
18. Chargeback holatlari (payment) va provodkalari (DD-04).
19. Provodkalar katalogi (DD-04): multi-use summa taqsimoti (GAP-07), reversal (GAP-08), dispute refund (GAP-09),
    PSP xarajati (GAP-12), kech to'lov (AT-13), QQS (GAP-11), variant B/C (GAP-13).
20. Data residency (PII O'zbekistonda), hosting (D5), backup/DR (RPO/RTO), production Dockerfile, monitoring.
21. Testlar: AT-01…AT-18 ning ko'pi (§5.19 da ❔ va ❌ belgili), AT-19+ (DD-11), k6, pentest, ASVS.
22. Mobil: Mini App (R1), Scanner PWA (R1), merchant PWA (R1), offline yo'q (ataylab).

### 6.2 Bizda bor, Dealo'da yo'q yoki farq qiladi (tekshirish/olib tashlash kerak)
1. **Kategoriya nomlari:** `coupons`, `gifts`, `local`, `seasonal-deals`, `goods`, `auto-and-home`, `travel`.
   Dealo: 4 pilot (Beauty & Wellness, Food & Drink, Things to Do, Health & Fitness), R1b (Automotive, Education),
   R2 (Events, Home Services), R3 (Travel, Pickup Goods), yopiq (Medical). **Seed ma'lumotini Dealo bo'yicha qayta qurish kerak.**
2. **Zaxira kod:** 8 belgi (`XXXX-XXXX`) — PRD 10 belgi.
3. **Hold:** 10 daqiqa — PRD 15 daqiqa.
4. **Deal holatlari:** `pending_review`, `rejected`, `sold_out` (PRD: `in_review`, `approved`, `changes_requested`, `ended`, `archived`).
5. **Order holatlari:** `pending_payment`, `failed` (PRD: `awaiting_payment`, `expired`, `completed`, `created`).
6. **Voucher holati:** `cancelled` (PRD: `voided`); `partially_redeemed` yo'q.
7. **Dispute holatlari:** `resolved_customer/merchant` bor; `awaiting_evidence`, `under_review`, `closed` yo'q.
8. **Eski ustunlar va jadvallar:** `users.password_hash`, `users.google_sub` (email/Google olib tashlangan),
   `payouts` jadvali (ishlatilmaydi), `auth_tokens` (migratsiya 00009 da drop qilingan ❔ — tekshirish),
   `currency_code` enumida `USD`, `homepage_section_type`da `collection` va `banner`, `deal_kind` (drop qilingan — tekshirish),
   `merchants.status` (`review_status` bilan ikki manba).
9. **Redeem reversal:** owner/manager istalgan vaqtda bekor qila oladi (PRD: kassir 10 daqiqa, undan keyin support).
10. **Merchant holati:** ikki ustun (`status`, `review_status`).
11. **Email:** bizda buyurtma emaillari bor; PRD R1a da email ixtiyoriy profil (GAP-24) — bizda profil email majburiy placeholder.
12. **Google OAuth:** olib tashlangan (PRD'da yo'q) — ✅ to'g'ri.
13. **Savat, gift, goods, shipping, USD, saved, kolleksiya, banner, promo:** olib tashlangan — ✅ to'g'ri (PRD bilan mos).
14. **Dealo'da bo'lmagan merchant panel ekranlari:** yo'q, bizda yo'q. (Farq emas.)
15. **Stek:** Go, Next.js admin va Go bot — foydalanuvchi qarori (Dealo: Django). Ataylab farq.

### 6.3 Ziddiyatlar va ikki xil talqinlar
- **Deal holati `pending_review` vs `in_review`:** nomlash farqi, xatti-harakat yaqin.
- **Order `pending_payment` vs `awaiting_payment`:** nomlash.
- **Merchant holati (PRD `approved ⇄ suspended → terminated`):** bizda `review_status` da suspended/terminated bor, lekin o'tish API yo'q.
- **PRD ichidagi qarama-qarshiliklar (GAP-02…09):** biz ham shu savollarga javob bermadik — dizayn qarori kerak.
- **Dealo'da `customer_prepaid` invariant (GAP-02):** bizning ledger'da ham shu muammo bor, chunki muddati o'tgan
  summa `customer_prepaid` da qoladi.
- **Kech to'lov (AT-13) va hold:** PRD expired→paid ruxsat beradi (GAP-03); bizda bu o'tish yo'q.

---

## 7. Ustuvor tartib (tavsiya)

Ustuvorlik: 1 = pul va xavfsizlik (launch'ni bloklaydi), 2 = merchant va xaridor oqimi, 3 = kanallar, 4 = qolgan.

1. **Dealo GAP'larini yopish (DD-00)** — 7 bloklovchi GAP. Bizning kod bu savollarga javob berishi kerak
   (GAP-02, 03, 04, 05, 07, 08, 09). Yuristdan oldin ichki qaror.
2. **Provodkalar katalogi (DD-04)** va ledger'ni shu bo'yicha to'ldirish (multi-use, reversal, dispute refund).
3. **Holat mashinalari (DD-03)** — deal, order, voucher, dispute holatlarini Dealo bo'yicha tekislash (nomlar va yangi holatlar).
4. **Zaxira kod (10 belgi) va QR token (128-bit)** — kontrakt va migratsiya.
5. **Hold 15 daqiqa** — bitta konstanta va test (AT-07).
6. **PSP adapteri (1 ta)** — sandbox, webhook imzo, idempotency, fiskal sinov. Yuridik G1/G2 kutilmoqda.
7. **Merchant panel qolgan bo'limlari** (Orders, Payouts, Reviews, Settings, Staff, Customers, Analytics) — UI.
8. **Deal Studio** (narx dalili, kvota, blackout, shablon, kalkulyator, versiyalash).
9. **Redemption:** "Siz emasmisiz?", kassir rate limit, maskalangan ism, Scanner PWA.
10. **Support tizimi** (ticket, SLA) va Telegram bot.
11. **Admin moliya UI** va roles/permissions.
12. **SEO:** sitemap, robots, kategoriya × tuman landing, transliteratsiya qidiruvi.
13. **Testlar:** AT-01…AT-18 har biri uchun test (❔ va ❌ belgilari), concurrency, E2E qayta ishga tushirish.
14. **Infra:** production Dockerfile, backup, monitoring, hosting (D5), data residency (ADR-0007).
15. **Yuridik gate'lar G1–G9** — launch'dan oldin (kod emas, lekin rejada).

---

## 8. Ochiq savollar (foydalanuvchi qarori kerak)

1. Kategoriyalarni Dealo bo'yicha qayta nomlash va seed'ni qayta qurish tasdiqlansinmi?
2. Zaxira kod 10 belgi va hold 15 daqiqa — PRD bo'yicha o'zgartirilsinmi?
3. Deal holatlarini Dealo bo'yicha to'liq qayta qurish (migratsiya) tasdiqlansinmi, yoki nomlar bilan moslashtirish yetarlimi?
4. Telegram kanallari (bot, Mini App) R1a da majburiymi, yoki birinchi navbatda web + merchant panel?
5. PSP tanlovi (Payme, Click, Uzum) — D2 qarori.
6. Pul oqimi varianti (A/B/C) — D1; bu butun ledger dizaynini belgilaydi.
7. Stek ataylab farq (Go/Next admin/Go bot) — shu qaror qolsinmi (oldingi qarorga muvofiq: ha).

---

## 9. Eslatma

- Bu hujjat faktlarga asoslangan; ❔ belgili bandlar tekshirilmagan — ular keyingi tekshiruvda yopiladi.
- Yuridik va soliq talqinlari (G1–G9, D3, D4, D6) yuristlar tomonidan tasdiqlanadi; kod ularni hal qilmaydi.
- Dealo hujjatlari "KOD MUZLATILGAN" holatda; ularni o'zgartirish uchun PRD egasi (@Ulugbek) yoki foydalanuvchi qarori kerak.
