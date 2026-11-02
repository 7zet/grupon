# Dealo bilan taqqoslash: nima qo'shiladi, nima olib tashlanadi

Bu fayl `ibragimoffline/Dealo` reposi (PRD v1.1 va dizayn hujjatlari) to'liq o'qib chiqilgandan keyin yozildi. Kod yozilmagan, faqat tahlil. Har bir band Dealo hujjatining bo'limiga havola qiladi (`PRD §…`, `GAP-…`, `ADR-…`).

## 1. Xulosa

- **Dealo** — hozircha faqat spetsifikatsiya: PRD v1.1, 19 ta bo'lim, 8 ta ADR, 18 ta dizayn deliverable (DD-00…DD-18). Kod yozilmagan ("kod muzlatilgan"). Stek: Django monolit, PostgreSQL + PostGIS, Celery, aiogram bot, bitta Next.js.
- **Bizning loyiha** — ishlaydigan kod: Go API (chi, pgx, sqlc), Next.js web, Playwright testlar. Mavzu: Groupon klonining Toshkent/Uzbekistan ko'rinishi. Biznes modeli boshqacha: bizda savatcha, ko'p shahar, mahsulot (goods) va sovg'a bor, Dealo'da esa yo'q.
- **Asosiy farq:** Dealo — **agent modeli, oldindan to'lov, ledger, breakage, 4-eyes payout** bilan moliyaviy platforma. Bizda bu qismlar yo'q (B6 olib tashlangan edi). Shu sababli eng katta ish — moliya yadrosini qo'shish.
- **Tavsiya:** backend tilini o'zgartirmaslik (Go, sizning talabingiz). Dealo'ning hamma talabini Go'da bajarish mumkin. Django'ga o'tish sekinlashtiradi va Go'ning afzalligini yo'qotadi.

## 2. Asosiy mahsulot farqlari

| Mavzu | Dealo (PRD v1.1) | Bizda hozir | Qaror |
|---|---|---|---|
| Biznes modeli | Oldindan to'lov, platforma agent, pul merchantga vaucher ishlatilgandan keyin | Savdo bor, lekin pul oqimi modeli yo'q (mock to'lov) | **Qo'shish** (moliya yadrosi) |
| Buyurtma | Faqat "Hozir sotib olish", bitta variant, bitta merchant | Savatcha (cart), ko'p mahsulot, guest + server cart | **Savatchani olib tashlash** |
| Tovar | Logistika taqiqlangan (BR-14). Faqat "do'konda olib ketish" (R3) | `product` turi, `shipping_address`, goods checkout | **Olib tashlash** |
| Sovg'a | Yo'q (R3) | Gift maydonlari (rejada, C2 olib tashlangan) | Kodda bo'lsa olib tashlash |
| Valyuta | Faqat UZS, tiyinda `bigint` | UZS + USD ko'rsatish (≈) | **USD ni olib tashlash** |
| Shaharlar | Toshkent (R1–R2); 2-shahar R3 | 3 shahar seed qilingan | Faqat Toshkent, qolganlari flag bilan yopiq |
| Kategoriyalar | 4 pilot: Beauty & Wellness, Food & Drink, Things to Do, Health & Fitness | 10 kategoriya seed | Pilotga qisqartirish, qolganlari flag bilan |
| Kanal | Responsive web + Telegram Mini App + bot + merchant PWA + admin | Faqat web | **Mini App va bot qo'shish** |
| Login | Telefon OTP (SMS + Telegram), 2FA (TOTP) moliya/admin uchun | Email/parol + Google OAuth | **Almashtirish** (telefon OTP) |
| Merchant onboarding | STIR, MChJ/YaTT, soliq rejimi, bank rekvizit (MFO), oferta e-qabul, hujjatlar | Nomi, telefon, manzil, kategoriya | **Kengaytirish** |
| Deal | Versiyalangan (immutable), narx dalili, MXIK, QQS, kvota, limitlar, blackout | Versiyasiz, narx dalilisiz | **Kengaytirish** |
| Vaucher | single_use / multi_use, zaxira kod, QR 30 s aylanadi, token HMAC | Bir martalik, QR statik, HMAC yo'q | **Kengaytirish** |
| Sharh | R1b da oddiy reyting; to'liq R2 | Sharh yozish, shikoyat (C1) | **Qisqartirish** (R1b ga) |
| Promo kod | R1b | Bor (kodda) | R1a dan olib tashlash |
| Saqlangan deal (wishlist) | R2 | "Saved" sahifasi | **Olib tashlash** |
| Kolleksiya, bosh sahifa bo'limlari, banner, seasonal | Yo'q | Bor (seed + sahifalar) | **Olib tashlash** yoki R2 |
| Qidiruv | Lotin/kirill transliteratsiya, pg_trgm, FTS, geo | pg_trgm + FTS bor, transliteratsiya yo'q | **Transliteratsiya qo'shish** |
| Xarita | Yandex yoki 2GIS (filial koordinatalari, PostGIS point) | Lat/lng ustunlari, xarita yo'q | Keyinroq |

## 3. Qo'shiladigan funksiyalar (Dealo talabi, bizda yo'q)

### 3.1 Moliya yadrosi (eng muhim, R1a blokeri)

1. **Append-only double-entry ledger** — `ledger_accounts`, `journal_entries`, `journal_lines`. UPDATE/DELETE trigger bilan taqiqlangan. Har bir entry `Σdebit = Σkredit` (DB constraint). Tuzatish faqat storno. (`ADR-0005`, `BR-09`, `BR-10`, `AT-08`)
2. **Hisoblar rejasi** (`PRD §8.3`): `psp_receivable`, `bank:operating`, `customer_prepaid:{merchant}`, `merchant_payable:{merchant}`, `merchant_reserve`, `refunds_payable`, `vat_payable`, `commission_revenue`, `breakage_revenue`, `psp_fees`, `merchant_receivable`.
3. **Provodkalar katalogi** (DD-04): har bir hodisa uchun debit/kredit, Variant A/B/C uchun alohida posting strategiyasi (konfiguratsiya), hisoblar rejasi umumiy.
4. **Settlement va payout** (`FR-07`): payable har redeemda hisoblanadi; 3 kunlik dispute oynasi; haftalik payout jadvali; payout holatlari `draft → pending_approval → approved → sent → reconciled` (`failed → draft`).
5. **4-eyes** (`AT-18`): payout tayyorlovchi ≠ tasdiqlovchi. Ikki rol: finance operator, finance approver. Refund ham shu qoidaga bo'ysunadi (istisno: avtomatik 14 kunlik refund, `GAP-22`).
6. **Bank rekvizit** (`FR-01.5`, `BR-11`, `AT-17`): rekvizit o'zgarsa owner 2FA → platforma tasdig'i → 72 soat payout hold → eski raqamga xabar. Alohida `bank_accounts` jadvali, tarix bilan.
7. **Reconciliation** (`§8.6`, `NFR-11`): har kuni 06:00 — PSP hisoboti ↔ payments ↔ ledger; har payoutdan keyin bank ↔ payouts ↔ ledger; oylik `customer_prepaid` invariant tekshiruvi. Nomuvofiqlik — alert.
8. **Breakage** (`§8.5`): muddati o'tgan vaucher uchun chegirma yo'qoladi, pul 90 kun davomida qaytariladi; keyin `customer_prepaid_expired` (GAP-02). Oylik breakage hisoboti.
9. **Refund oqimi** (`FR-06`, `AT-11`): 14 kun ichida ishlatilmagan — avtomatik to'liq refund; keyingi holatlar — support + finance; faqat asl to'lov usuliga.
10. **Fiskal** (`§8.2`): har bir variant uchun MXIK, paket kodi, QQS stavkasi; PSP fiskal cheki (merchant STIR nomidan); refund cheki; komissiya bo'yicha oylik EDO hisob-faktura (R2).

### 3.2 Vaucher va redeem

11. **Vaucher turlari**: `single_use` va `multi_use` (`redemptions_allowed`, `redemptions_used`), `redeemed` faqat oxirgi ishlatishda (`FR-04.4`, `BR-03`, `AT-15`).
12. **Token xavfsizligi** (`FR-04.1–2`): token 128-bit tasodifiy, DB'da faqat HMAC-hash. Zaxira kod 10 belgi (Crockford base32), faqat "ko'rsatish" bosilganda.
13. **QR aylanishi** (`FR-04.3`): web va Mini App'da QR har 30 soniyada yangilanadi (token + vaqt imzosi).
14. **Redeem reversal** (`FR-05.5`, `GAP-05`): kassir 10 daqiqada bekor qilishi mumkin; keyin faqat support. Vaucher holati qayta hisoblanadi, ledger storno.
15. **Redeem qoidalari** (`§9.2`): rate-limit (kassir 30/daqiqa), `SELECT … FOR UPDATE`, filial ruxsati, kunlik limit, Idempotency-Key, parallel so'rov — 409 (bizda shu bor, faqat kunlik limit va filial ruxsati tekshirilishi kerak).
16. **Xaridorga xabar** (`FR-05.4`): redeemdan keyin darhol "Siz emasmisiz?" + bir bosishda shikoyat.
17. **Eslatma** (`FR-04.5`): muddat tugashidan 7 va 1 kun oldin.
18. **Vaqt zonasi** (`AT-12`): muddat `Asia/Tashkent` kun oxirida (23:59:59.999), UTC da saqlanadi.

### 3.3 Checkout va inventar

19. **Inventory hold**: 15 daqiqa, `held + sold ≤ capacity` (DB CHECK), atomik `UPDATE … WHERE` (bizda bor, lekin hold/expiry jadvali kerak).
20. **Order holatlari**: `created → awaiting_payment → paid → completed`; `expired / failed / cancelled`; `paid → partially_refunded / refunded`.
21. **Kech to'lov** (`FR-03.5`, `AT-13`): hold tugagan, to'lov keldi — inventory bo'sh bo'lsa vaucher, bo'lmasa avtomatik refund.
22. **Narx snapshot** (`BR-06`, `AT-06`): order sotib olingan deal versiyasi bo'yicha o'zgarmas.
23. **Checkout ekrani** (`FR-03.2`): yakuniy narx, asl narx, muddat, filiallar, bron talabi, refund qoidasi, shartlarni belgilash.
24. **Outbox** (`ADR-0004`): har bir tashqi ta'sir (SMS, Telegram, PSP so'rovi, analitika) biznes yozuvi bilan bitta tranzaksiyada `outbox` jadvaliga; worker yuboradi; 10 urinishdan keyin dead-letter.
25. **Webhook** (`ADR-0004`): imzo, `provider_event_id` dedup, async qayta ishlash, PSP'dan holatni qayta so'rash.
26. **Idempotency-Key 24 soat** (`§10.3`): bizda bor, lekin hamma yozish endpointlarida majburiy bo'lishi va 24 soat saqlanishi kerak.
27. **State transitions jurnali** (`§9`): har bir holat o'tishi `state_transitions` ga yoziladi; ruxsat etilmagan o'tish — 422.

### 3.4 Merchant

28. **Deal versiyalash** (`FR-02.7`, `DealVersion`): faol dealdagi narx/shart o'zgarishi yangi versiya va qayta moderatsiya.
29. **Holatlar**: `draft → in_review → approved → scheduled → active ⇄ paused → ended → archived`; `in_review → changes_requested → draft`.
30. **Narx dalili** (`FR-02.3`): asl narx uchun menyu/prays/30 kunlik chek fotosi; 70% dan yuqori chegirma — qo'shimcha tekshiruv.
31. **Kvota va limitlar** (`FR-02.4`, `GAP-16`): global kvota, filial kvotasi, xaridorga limit, kunlik redemption limiti, blackout kunlar/soatlar.
32. **Shartlar shabloni** (`FR-02.5`): kategoriya bo'yicha tayyor shart matnlari.
33. **Kampaniya kalkulyatori** (`FR-02.6`): merchant deal narxi, komissiya, PSP xarajati, tannarx kiritadi; manfiy sof tushum — ogohlantirish.
34. **Kategoriya flaglari** (`§4`): `searchable`, `deal_creation_enabled`, `publish_enabled`, `purchasable`, `bookable`, `redeemable`, `ads_enabled` — kategoriya va shahar kesimida. Yopilsa mavjud vaucher `redeemable` bo'lib qoladi.
35. **Kategoriya atributlari** (`DD-17`): JSON Schema orqali (masalan usta/slot, yosh, guruh soni).
36. **Merchant holatlari**: `draft → submitted → in_review → approved / rejected`; `approved ⇄ suspended → terminated`.
37. **Oferta e-qabul** (`FR-01.4`): qabul vaqti, IP, versiya, qabul qilgan shaxs saqlanadi.
38. **Xodim taklifi telefon orqali** (`FR-01.6`) va filialga biriktirish.

### 3.5 Xavfsizlik va antifraud

39. **Telefon OTP** (`§11.2`): cooldown 60 s, telefon va IP limit, 5 urinish, keyin 15 daqiqa blok. Yuborish: SMS yoki Telegram.
40. **2FA (TOTP)** — admin, finance, owner uchun majburiy (`§12.2`), IP allowlist yoki VPN, admin sessiya 8 soat.
41. **Fraud signal qoidalari** (`§12.1`, `DD-09`): ish vaqtidan tashqari va tez redeemlar (>5/daqiqa, filialdan >500 m), zaxira kod brute-force (10 xatoda blok), 30 kunda >3 refund, 1 daqiqada 8 vaucher (`AT-14`). `FraudSignal` jadvali, analyst navbati, payout hold.
42. **Dispute** (`FR-08`, `BR-15`, `AT-16`): ticket, SLA (birinchi javob ≤ 2 soat, 09:00–21:00), dispute ochilsa merchant payouti shu summaga hold.
43. **Audit log** (`NFR-10`): moliya/kirish audit o'zgarmas, 5 yil; boshqa loglar 90 kun.
44. **PII maskalash** loglarda; telefon raqami maskalangan ko'rsatiladi; hujjatlar shifrlangan bucket'da.
45. **Data residency** (`ADR-0007`): barcha PII O'zbekistonda; tashqi SaaS faqat self-hosted yoki PII'siz.

### 3.6 Kanallar va boshqa

46. **Telegram Mini App** (`ADR-0006`): web kodining alohida layout'i, `initData` orqali login. Bu bizning Next.js'ga mos.
47. **Telegram bot** (`ADR-0006`): alohida servis; Dealo'da aiogram (Python). Bizda Go'da yozish mumkin (`telebot` yoki Bot API to'g'ridan-to'g'ri).
48. **Scanner PWA** (`FR-05.1`): kamera orqali QR, zaxira kod kiritish. Offline rejim yo'q. Bizda `voucher-redeem-panel` bor, PWA emas.
49. **Transliteratsiya** (`FR-09.2`, `§10.5`): o'zbek lotin/kirill, `o‘`/`o'`/`g‘`/`g'` variantlari, `search_text` ustuni.
50. **Qidiruv filtrlari** (`FR-09.3`): kategoriya, tuman, narx, chegirma, "bugun ishlatsa bo'ladi".
51. **Tuman va "yaqin atrofda"** (`FR-09.1`): PostGIS yoki Haversine (Go'da bizning SQL'da).
52. **SEO** (`FR-09.4`): kategoriya × tuman landing sahifalari, `schema.org/Offer`. (Bizda C4 olib tashlangan edi — qaytarish kerak.)
53. **Marketing rozilik** (`FR-09.5`): alohida rozilik, har xabarda obunadan chiqish.
54. **Monitoring**: OpenTelemetry, Prometheus, Grafana, Loki, Sentry (self-hosted).
55. **Lokal servislar** (`infra/docker-compose.yml`): PostgreSQL 16 + PostGIS, Redis 7.4, SeaweedFS (S3). Bizda Docker yo'q edi — buni o'rnatish kerak.
56. **S3 saqlash**: SeaweedFS (lokal), production D5 qaroriga qarab. Bizda hozir lokal disk adapteri (kodda `storage.Store` interfeysi bor — S3 adapter qo'shish osonroq).

## 4. Olib tashlanadigan yoki o'zgartiriladigan funksiyalar

| Funksiya | Qayerda | Sabab | Harakat |
|---|---|---|---|
| Savatcha (cart) | `stores/cart.ts`, `cart/page.tsx`, `cart` API, purchase-panel | PRD §6 "savatcha yo'q", faqat "Hozir sotib olish" | Olib tashlash, "Hozir sotib olish" qoldirish |
| Goods (`kind: product`) | `catalog.sql`, `checkout-form.tsx`, `needsShipping` | BR-14: logistika taqiqlangan | Olib tashlash |
| `shipping_address` | `CreateOrder` (OpenAPI), checkout | BR-14 | Olib tashlash |
| Sovg'a maydonlari | `CartItemInput.gift_*`, checkout | R3 | Olib tashlash |
| USD hint | `packages/ui/src/money.ts`, `formatMoney` | "Valyuta: UZS" | USD olib tashlash |
| Email/parol login, Google OAuth | `identity`, `auth-form.tsx`, `google` paket | PRD: telefon OTP | OTP bilan almashtirish; Google — R1b yoki yo'q |
| Saved (wishlist) | `account/saved`, `saved_deals` | R2 | Olib tashlash |
| Kolleksiya, bosh sahifa bo'limlari, banner, seasonal | `collections`, `homepage_sections`, `banners`, `seasonal-deals` | PRD'da yo'q | Olib tashlash yoki R2 |
| Promo kodlar | `promo_codes`, `CountActivePromoCodes` | R1b | R1a dan chiqarish (kod qoladi, o'chirilgan) |
| Ko'p shahar | seed 3 shahar, `city` filtrlari | Toshkent (R1–R2) | Faqat Toshkent; boshqalari `searchable=false` |
| Ko'p kategoriya (10) | seed | 4 pilot | 4 tasini qoldirish, qolganlari flag bilan yopiq |
| Sharh yozish va shikoyat | `review` flow (C1) | R1b (oddiy reyting), R2 (to'liq) | R1a da reyting ko'rsatish yetarli; yozish — R1b |
| `expired` tekshiruvi ko'p joyda | `ListDealSummaryRows` va h.k. | Dealo'da deal holatlari boshqacha | Holat mashinasini Dealo bo'yicha qayta yozish |
| Cursor bo'lmagan pagination | `GET /deals` (page-based) | PRD: cursor pagination | Cursor'ga o'tkazish |
| Deal `kind` | voucher/product | Faqat voucher (R1a) | Enum'ni `voucher`ga qisqartirish |

## 5. Saqlanadigan kuchli tomonlar (Go va Next.js)

Siz aytgandek, tezlik va Go kuchi saqlanadi:

- **Go API** (chi, pgx, sqlc): Dealo Django/DRF ga nisbatan ancha tez va xotira tejamkor. Bizda p95 9–18 ms (o'lchangan).
- **Atomik tranzaksiyalar**: Go'da `pgx` tranzaksiyalari bilan moliyaviy yozuvlar bitta tranzaksiyada — Dealo'ning ADR-0001 talabiga to'liq mos.
- **Conditional UPDATE**: sotuv parallelligi (100 xaridor → 10 dona) allaqachon isbotlangan (A6). Bu `BR-05`, `AT-03` ning to'g'ridan-to'g'ri qoplanishi.
- **Redeem parallelligi**: bizda `UPDATE … WHERE status='active'` bilan bitta muvaffaqiyat (B5). `AT-02` ga mos.
- **Next.js bitta kod bazasi** (ADR-0006): bizda shu bor. Mini App uchun alohida layout qo'shish oson.
- **Problem+json, Idempotency-Key, X-Request-Id**: bizda bor (`router.go`), Dealo talabiga mos.
- **OpenAPI yagona manba**: bizda ham (`packages/api-spec/openapi.yaml` → Go va TS generatsiya). Dealo'ning DD-05 ga mos.
- **Worker**: bizda `cmd/worker` bor (Celery o'rnida). Outbox poller, hold expiry, reconciliation shu worker'ga qo'shiladi.

## 6. Qaror talab qiladigan masalalar (foydalanuvchi)

Bu qarorlar kodga ta'sir qiladi, shuning uchun avval siz hal qilasiz:

1. **Pul oqimi varianti** (`D1`): A (agent + redemption-based, tavsiya), B (PSP split), C (darhol to'lov + rezerv). Ledger uchala variantni qo'llashi kerak.
2. **PSP tanlovi** (`D2`): Payme, Click yoki Uzum. Sandbox va webhook formati shunga bog'liq.
3. **Hosting** (`D5`): O'zbekistondagi data-markaz yoki bulut. S3 va Postgres shunga bog'liq.
4. **Pul birligi**: Dealo — tiyin (`bigint`). Bizda hozir so'm (UZS minor unit). Bu API va bazadagi barcha summalarni ×100 qiladi. Tasdiqlang.
5. **Telegram bot tili**: Go (bizning stek) yoki aiogram (Python, Dealo'dagi kabi). Tavsiya: Go.
6. **Admin**: Dealo — Django Admin (R1), Next.js (R2). Bizda Next.js admin sahifalari bor. Tavsiya: Next.js saqlash (R1 da ham ishlaydi).
7. **Login**: telefon OTP (Dealo) — SMS provayder kerak (`SMS` mahalliy provayder). Email/parol qoladimi? Tavsiya: telefon asosiy, email profil maydoni (GAP-24).
8. **Reviews**: R1b ga tushirish. Yozish oqimini saqlaymizmi?
9. **Promo kodlar**: R1b ga tushirish — hozircha o'chiramizmi?
10. **Kategoriyalar**: 4 pilot (Beauty & Wellness, Food & Drink, Things to Do, Health & Fitness). Qolgan 6 tasi flag bilan yopiladimi yoki o'chiriladimi?
11. **Shaharlar**: faqat Toshkent. Qolgan 2 ta shahar o'chiriladimi yoki `searchable=false` bilan qoladimi?

## 7. Ustuvorlik tartibi (qadam-baqadam)

Dealo'ning R1a qabul mezonlari (AT-01…AT-18) asosida, bizning kodga qo'llash tartibi:

**Bosqich 1 — Moliya yadrosi (blokerlar):**
1. Ledger jadvallari, append-only trigger, Σdebit=Σkredit constraint (BR-09, BR-10, ADR-0005).
2. Hisoblar rejasi va provodkalar (DD-04 ga qarab).
3. Vaucher holat mashinasi (`active → redeemed → …`), `redemptions_used/allowed`, `SELECT FOR UPDATE` (BR-03).
4. Redeem reversal (10 daqiqa), storno (GAP-05).
5. Kech to'lov (AT-13), hold expiry (AT-07), idempotency 24 soat.

**Bosqich 2 — Pul chiqishi va nazorat:**
6. Settlement: payable har redeemda, 3 kunlik dispute oynasi, haftalik payout.
7. Payout 4-eyes (AT-18) va payout holatlari.
8. Bank rekvizit hold 72 soat (AT-17), merchant bank_accounts tarixi.
9. Reconciliation: kunlik, payout'dan keyin, oylik invariant (AT-08).
10. Breakage (§8.5) va refund oqimi (FR-06, AT-11).

**Bosqich 3 — Xavfsizlik va antifraud:**
11. Telefon OTP, 2FA (TOTP) admin/finance/owner, RBAC matritsasi (AT-04, AT-09).
12. Token HMAC, zaxira kod, QR 30 s aylanishi.
13. Fraud signal qoidalari (AT-14), payout hold (AT-16).
14. Audit log o'zgarmas (NFR-10).

**Bosqich 4 — Merchant va deal:**
15. Merchant onboarding (STIR, bank, oferta e-qabul), holatlar (`approved ⇄ suspended`).
16. Deal versiyalash, narx dalili, kvota, limitlar, blackout.
17. Kategoriya flaglari va JSON Schema atributlari.
18. Kalkulyator (FR-02.6).

**Bosqich 5 — Kanallar:**
19. Telegram Mini App layout + initData login.
20. Telegram bot (OTP va bildirishnomalar).
21. Scanner PWA (kamera QR).
22. Transliteratsiya (lotin/kirill) va qidiruv.

**Bosqich 6 — Tozalash (olib tashlash):**
23. Savatcha, goods, sovg'a, USD, saved, kolleksiya, banner, promo, ko'p shahar, ko'p kategoriya.

**Bosqich 7 — Yuridik gate (kod emas, lekin production'dan oldin):**
24. G1 (pul oqimi), G2 (fiskal), G3 (e-tijorat ro'yxat), G4 (PII lokalizatsiya), G5 (iste'molchi huquqlari), G6 (merchant oferta), G7 (reklama), G8 (brend — "Dealo" nomi), G9 (regulyatsiyalangan kategoriyalar).

## 8. Eslatma: nima kodga emas, hujjatga kerak

Dealo hujjatlari bo'yicha, kod yozishdan oldin tayyor bo'lishi kerak (ular Dealo'da DD-00…DD-18):
- PRD gap'larini yopish (27 ta GAP, 7 tasi 🔴 bloklovchi): `GAP-02, 07, 08, 09, 03, 04, 05`.
- RBAC matritsasi (DD-01).
- ERD (DD-02) va holat mashinalari (DD-03).
- Provodkalar katalogi (DD-04).
- OpenAPI (DD-05) — bizda bor, lekin Dealo endpointlari bilan to'ldirish kerak.
- Yuridik savollar paketi (DD-14).

## 9. Xulosa jadvali

| Kategoriya | Soni (taxminiy) |
|---|---|
| Qo'shiladigan asosiy modullar | 10 (ledger, settlement, payout, refund, breakage, reconciliation, fraud, dispute, OTP, bank hold) |
| Qo'shiladigan kichik funksiyalar | ~45 (yuqoridagi ro'yxat) |
| Olib tashlanadigan / qisqartiriladigan | ~15 (savatcha, goods, sovg'a, USD, saved, kolleksiya, banner, promo, ko'p shahar, ko'p kategoriya, va h.k.) |
| Saqlanadigan kuchli tomonlar | Go API, pgx tranzaksiyalar, conditional UPDATE, Next.js, OpenAPI, worker, problem+json |
| Foydalanuvchi qarori kerak | 11 ta (6-bo'lim) |

> Bu hujjat Dealo'ning spetsifikatsiyasiga asoslangan. Yuridik va soliq talqinlari (G1–G9) yuristlar tomonidan tasdiqlanishi kerak, kod ularni hal qilmaydi.
