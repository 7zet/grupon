# Grupon — butun ish hisoboti

> Sana: 2026-11-03. Branch: `feat/merchant-and-admin-flows`. Oxirgi commit: `a6e6618`.
> Bu fayl qilingan barcha ishlarning bitta joydagi xulosasi. Batafsil qarorlar `docs/DECISIONS.md`da,
> holat `docs/STATUS.md`da, Dealo bilan solishtirish `docs/ANALYSIS-DEALO-VS-GRUPON.md`da.

## 1. Loyiha qisqacha

Toshkent uchun mahalliy deal bozori. Xaridor vaucher sotib oladi, merchant filialida QR yoki kod orqali
ishlatadi, admin moderatsiya va to'lovlarni boshqaradi. Stek: Go (API, worker, bot), Next.js (web),
PostgreSQL, Redis (hozircha ishlatilmaydi), S3 o'xshash saqlash. Pul — tiyin (int64), faqat UZS.

Asosiy qoidalar: kontrakt birinchi (OpenAPI → Go va TS kod), ledger ikki yoqlama va faqat qo'shiladigan,
har bir sezgir amal audit yozuvi bilan, zaxira yozish bitta shartli UPDATE bilan.

## 2. Commit'lar

| Commit | Mazmun |
|---|---|
| `5898a11` | Telefon OTP bilan kirish |
| `f49b2a2` | Moliya yadrosi: ikki yoqlama ledger, redemption, payout, refund, dispute |
| `79390ed` | Moliya davomi: aylanuvchi QR, zaxira kod, bir necha marta ishlatiladigan variantlar, outbox, reconciliation, refund tasdiqlash, fraud qoidalari |
| `c01209a` | Dealo bo'yicha olib tashlashlar: savat, saqlanganlar, promo, kolleksiyalar, sovg'alar, tovarlar, yetkazish, valyuta kursi, email auth |
| `52c5d1f` | Merchant onboarding va ko'rib chiqish, biznes hujjatlari, seed haqiqiy akkauntlarni saqlaydi |
| `a6e6618` | PRD holat mashinalari, pilot kategoriyalar, merchant panel, tozalash (bu hisobotning asosiy qismi) |

Commit'ga qo'shilmagan: `Grupon_hisobot.pdf` (egasining shaxsiy fayli) va `dealo/` papkasi hamda `dealo.zip`
(Dealo nusxasi; ularni `.gitignore`ga qo'shish yoki o'chirish hali hal qilinmagan).

## 3. Moliya yadrosi

- **Double-entry ledger**: `ledger_accounts`, `journal_entries`, `journal_lines`. Triggerlar bilan faqat
  qo'shish; tuzatish faqat teskari yozuv bilan. `ledger.Split` ulushlarni pastga yaxlitlaydi, qoldiq oxirgisiga.
  Komissiya bps'da. `PostEntry` idempotent.
- **Redemption**: bitta tranzaksiyada qulflash, ishlatish, yozuv, audit. Takroriy ishlatish mumkin emas.
  Redemption raqami (`NextRedemptionSequence`) teskari yozuvlarni ham hisobga oladi, shu sababli 500 xatosi
  tuzatildi.
- **Reversal (teskari qaytarish)**: kassir/egasi/menejer uchun 10 daqiqa oynasi. Oyna o'tgach faqat support
  (`SupportReverse`).
- **Payout**: `draft → pending_approval → approved → sent → reconciled`. Ikki kishilik tasdiqlash (4-eyes):
  tayyorlagan va tasdiqlagan bir xil bo'lmaydi.
- **Refund**: so'rov → tasdiq → paid. Vaucher ishlatilmagan bo'lsa avtomatik; ishlatilgan bo'lsa tasdiq kerak.
- **Breakage**: muddati o'tgan vaucherning qolgan qiymati; 90 kun qaytariladi, keyin yurist xulosasi.
- **Reconciliation**: kunlik solishtirish, `MarkPayoutReconciled`.
- **Dispute**: xaridor "Siz emasmisiz?" deb ochadi, ochiq dispute merchant payout'ini ushlab turadi (AT-16).
- **Outbox**: hodisalar bir marta yuboriladi.
- **Fraud**: `fraud_signals` jadvali; skanerlash cheklovi 30 so'rov/daqiqa, 10 xato so'rovdan keyin 15 daqiqa blok.

## 4. Vaucher va to'lov

- Zaxira kod: 10 belgi, Crockford base32, `XXXXX-XXXXX` ko'rinishida. Inson kodi 8 belgi.
- QR har 30 soniyada yangilanadi (`qr_token_expires_at`).
- Bron ushlab turish: 15 daqiqa (PRD FR-03.4).
- Vaucher muddati: `min(valid_until, xarid + N kun)`, kun oxiri Asia/Tashkent bo'yicha 23:59.
- Mock to'lov: `POST /payments/mock/{id}/complete`, natija `succeeded` yoki `failed`. Haqiqiy PSP (Payme, Click,
  Uzum) hali yo'q, PRD bo'yicha "test'dan keyin".
- Kech to'lov (buyurtma muddati o'tgach) bo'lsa provayder qaytarishi chaqiriladi; alohida `refund` yozuvi
  hali yaratilmaydi (ochiq).

## 5. Holat mashinalari (PRD 08 bo'yicha)

Migratsiyalar 00019 va 00020. Qiymatlar avval mapping qilingan, shu sababli hech bir qator yo'qolmagan.

| Obyekt | Holatlar | Eslatma |
|---|---|---|
| Deal | draft, in_review, approved, scheduled, active, paused, ended, archived, changes_requested | `sold_out` holat emas: faol deal zaxirasi 0 bo'lsa ko'rinadi |
| Buyurtma | created, awaiting_payment, paid, completed, expired, failed, cancelled, partially_refunded, refunded | `created` faqat tranzaksiya ichida; hold tugasa `expired` |
| To'lov | created, pending, succeeded, failed, chargeback_open, chargeback_won, chargeback_lost | Qaytarish alohida obyekt; to'lov `refunded` emas |
| Vaucher | issued, active, partially_redeemed, redeemed, expired, refunded, voided | `issued` hozircha ishlatilmaydi; `active` to'g'ridan-to'g'ri. Flaglar: `fraud_hold`, `dispute_open` |
| Dispute | open, awaiting_evidence, under_review, resolved_customer, resolved_merchant, closed | Kodda faqat `open` va natija o'tishlari bor |
| Redemption | confirmed, reversed | Mos |
| Refund | requested, approved, processing, succeeded, rejected, failed | Mos |
| Payout | draft, pending_approval, approved, sent, reconciled, failed | Mos |
| Merchant | draft, submitted, in_review, approved, rejected, suspended, terminated | Yagona `review_status`; eski `status` ustuni olib tashlandi |

Qoidalar: deal tasdiqlash ikki bosqichli (`in_review → approved → scheduled|active`), ikkala qadam audit
qilinadi. Ko'p marta ishlatiladigan vaucher har ishlatishdan keyin `partially_redeemed`, oxirgisidan keyin
`redeemed`. Dispute ochilganda `dispute_open` o'rnatiladi, yopilganda tozalanadi. To'liq ishlatilgan
buyurtma `completed` bo'ladi.

## 6. Merchant

### 6.1 Onboarding va ko'rib chiqish
- Ro'yxatdan o'tish: STIR (9 raqam), tashkilot shakli (MChJ/YaTT), soliq rejimi, MFO, hisob raqami,
  mas'ul shaxs, shartnoma versiyasi va qabul qilish. Yuborilgandan keyin tahrirlab bo'lmaydi.
- Hujjatlar: sertifikat va bank xati majburiy; narx isboti (`price_proof`) ixtiyoriy. PDF/JPG/PNG, 10 MB.
  Hujjatlar faqat draft yoki rad etilgan holatda o'zgaradi. Yuklash `PUT`, ko'rish faqat ko'rib chiquvchilarga
  (`Content-Disposition: attachment`, `nosniff`). Saqlash — alohida xususiy papka (`DOCUMENTS_LOCAL_DIR`).
- Admin ko'rib chiqish: boshlash, tasdiqlash, rad etish (sabab bilan). Har bir qadam audit.
- Merchant deal yarata olmaydi, agar business tasdiqlanmagan bo'lsa (`merchant_not_approved`).

### 6.2 Xodimlar (faqat egasi)
- Telefon orqali taklif (xodim avval bir marta kirgan bo'lishi kerak), rol: menejer yoki kassir.
- Rol o'zgartirish, o'chirish. Egasi rolini o'zgartirib yoki o'chirib bo'lmaydi.
- Har bir o'zgarish audit qatoriga yoziladi (bir tranzaksiyada).

### 6.3 Panel API (yangi endpoint'lar)
- `GET /merchant/payouts`: to'lov davrlari, savdo, komissiya, sof summa, holat.
- `GET /merchant/customers`: xaridorlar, buyurtmalar soni, vaucherlar soni, jami xarid. Ism "Alisher U." ko'rinishida.
- `GET /merchant/analytics?from&to`: jami savdo, sotilgan va ishlatilgan vaucherlar, ishlatish foizi, kunlik savdo.
- `GET /merchant/reviews`, `PUT /merchant/reviews/{id}/reply`: bitta javob, 24 soat ichida tahrirlanadi.
  Kassir javob yoza olmaydi.
- `GET /merchant/orders`: buyurtmalar, tovarlar nomi bilan.
- `GET|PATCH /merchant/settings`: buyurtma, sharh, to'lov xabarnomalari va asosiy filial.
- `GET|POST /merchant/staff`, `PATCH|DELETE /merchant/staff/{id}`.
- `GET /merchant/documents`, `PUT /merchant/documents/{kind}`.
- Hali yo'q: `GET /merchant/redemptions` (merchant ishlatish tarixi) va `GET /me/reviews` (xaridorning sharhlari).

### 6.4 Merchant panel sahifalari (web)
Navigatsiya: Overview, Business, Deals, Yangi deal, Media, Redeem, Staff, Orders, Payouts, Customers,
Analytics, Reviews, Calculator, Settings. Har bir sahifa uz/ru/en tarjimali, ranglar tokenlardan.

- **Payouts**: jadval va CSV yuklab olish (statement) brauzerda yasaladi.
- **Calculator**: aksiya narxi, asl narx, vaucher soni va komissiyadan merchant oladigan summani hisoblaydi.
  Serverga yubormaydi.

### 6.5 Deal
- Deal yaratish, tahrirlash, yuborish; 3 ta rasm minimum; variantlar (asl narx, aksiya narxi, zaxira,
  bir xaridor limiti). Deal wizard olti qadam bo'lishi kerak edi, lekin qadamlar va jonli preview to'liq
  tekshirilmagan.

## 7. Admin

- Moderatsiya navbati (deal): tasdiqlash, rad etish (sabab bilan). Holat `in_review` bo'lsa.
- Merchant ko'rib chiqish navbati: hujjatlar ro'yxati va yuklab ko'rish, boshlash/tasdiqlash/rad etish.
- Moliya ishchi stoli (`/admin/finance`): payout tasdiqlash/reconcile, refund tasdiqlash/to'landi,
  dispute hal qilish. Ro'yxat faqat support uchun refund va dispute'larni ko'rsatadi.
- Foydalanuvchi huquqlari: moderator moliyani o'qiy olmaydi (test bor).

## 8. Xaridor (buyer)

- Kirish: telefon + bir martalik kod. Dev rejimida kod sahifada ko'rsatiladi.
- Xarid oqimi: deal sahifasi → "Hozir xarid qilish" → login (agar kirmagan bo'lsa) → checkout → mock to'lov
  → muvaffaqiyat sahifasi → vauchrlar.
- Xabarnomalar (`/me/notifications`): har bir ishlatilishda; "Siz emasmisiz?" tugmasi dispute ochadi.
- Sharhlar: yozish, xabar berish; admin yashirish.
- Vaucher sahifasi: faol, ishlatilgan, muddati o'tgan tablar; `/account/coupons` (ommaviy `/coupons` o'chirildi).

## 9. Kategoriyalar va shaharlar (PRD 03)

- Ochiq (pilot): Beauty & Wellness (`beauty-and-spas`), Food & Drink, Things to Do, Health & Fitness.
- Yopiq (feature flag bilan): Automotive, Education & Classes, Events & Entertainment, Home Services,
  Travel & Hotels, Pickup Goods, Medical (regulated).
- Shaharlar: Toshkent (pilot, ochiq), Namangan, Samarqand, Andijon, Farg'ona (yopiq).
- Eslatma: feature flaglar (`searchable`, `purchasable`, `redeemable` va h.k.) kategoriya × shahar kesimida
  hali to'liq emas; hozir faqat `is_active` bayrog'i bor.

## 10. Tozalash

- Parol va Google login olib tashlandi: `users.password_hash`, `users.google_sub`, parol tiklash kodi,
  argon2 yordamchilari (migratsiya 00018). Kirish faqat telefon kodi bilan.
- Valyuta: `currency_code` faqat `UZS`. USD qatori bo'lsa migratsiya to'xtaydi (ma'lumot jimgina o'zgarmasin).
- Eski `payouts` jadvali va `payout_status` tipi olib tashlandi (00015, 00019).
- Eski `homepage_section_type` faqat `deals` va `categories` qoldi (00020).
- Merchant `status` ustuni olib tashlandi; ommaviy holat `review_status`dan hosil qilinadi (00020).
- Mock'dagi eski valyuta kursi va eski kategoriyalar olib tashlandi.

## 11. Migratsiyalar (00009–00020)

| Fayl | Nima qiladi |
|---|---|
| 00009_dealo_removals | Savat, saqlanganlar, promo, kolleksiya, banner, valyuta kursi, sovg'a/tovar/yetkazish ustunlari |
| 00010_merchant_onboarding | Merchant onboarding maydonlari, `review_status` |
| 00011_merchant_review | Ko'rib chiqish jadvali va holatlari |
| 00012_merchant_documents | Biznes hujjatlari (xususiy saqlash) |
| 00013_notifications | Xaridor xabarnomalari |
| 00014_redemption_attempts | Skanerlash urinishlari (cheklov uchun) |
| 00015_drop_legacy_payouts | Eski payout jadvali |
| 00016_merchant_settings | Merchant sozlamalari |
| 00017_price_proof | `price_proof` hujjat turi |
| 00018_drop_password_google_usd | Parol/Google ustunlari, USD |
| 00019_prd_state_machines | PRD holatlari, vaucher flaglari |
| 00020_merchant_single_status | Yagona merchant holati, homepage tipi |

Oldingi migratsiyalar 00001–00008 — boshlang'ich sxema, auth, telefon OTP, moliya yadrosi, moliya davomi.

## 12. API kontrakti

- Manba: `packages/api-spec/openapi.yaml`. TS: `pnpm --filter @grupon/api-client run gen`. Go:
  `go generate ./...` (`services/api`). Eslatma: root `pnpm gen` bu muhitda ishlamadi, paketni alohida
  ishga tushirish kerak.
- Yangi operatsiyalar: `listMerchantPayouts`, `listMerchantCustomers`, `getMerchantAnalytics`,
  `listMerchantReviews`, `replyToMerchantReview`, `listMerchantOrders`, `getMerchantSettings`,
  `updateMerchantSettings`, staff operatsiyalari, `supportReverseVoucher`, `getFinanceQueue`,
  `listMyNotifications`, merchant review va hujjat operatsiyalari.
- Muhim: `updateMerchantSettings` spec'da `PATCH`. BFF allowlist shunga moslangan.

## 13. Web arxitekturasi

- Next.js App Router, next-intl (uz, ru, en), react-hook-form + Zod, TanStack Query.
- BFF: `apps/web/src/app/api/bff/[...path]/route.ts`. Har bir yo'l `ALLOWED` ro'yxatida; binar yuklash
  va yuklab olish alohida.
- Dinamik tarjima kalitlari o'rniga literal xaritalar (typed next-intl uchun).
- Rasmlar: dev uchun `localhost:8080/media` ruxsat etilgan (`next.config.ts`).

## 14. Seed va test akkauntlar

- `services/api/cmd/seed` — katalog ma'lumotlari qayta yaratiladi; haqiqiy foydalanuvchilar saqlanadi
  (`@local.test` va `@test.local` tashqari). Seed merchant'lari tasdiqlangan holatda.
- Test telefonlari: owner `+998900000002`, user `+998900000001`, cashier `+998900000003`, admin `+998900000004`.
  Kod dev rejimida sahifada ko'rinadi.
- Seed `data.json`: 11 kategoriya, 5 shahar, 30 merchant, 100 deal, 400 sharh.

## 15. Testlar (oxirgi holat)

| Qatlam | Natija |
|---|---|
| Go (`go test ./...`, `TEST_DATABASE_URL` bilan) | Hammasi o'tadi; `go vet` toza |
| Web typecheck (`@grupon/web`) | Toza |
| Web unit (vitest) | 6/6 |
| Playwright e2e (`e2e/`) | 16/16, faqat ketma-ket (`workers: 1`) |

Playwright spec'lari: `public`, `purchase` (xaridor), `merchant` (egasi yaratadi, admin tasdiqlaydi, xaridor
sotib oladi, egasi ishlatadi), `admin-merchant-ui` (moderatsiya, merchant rasm yuklash), `merchant-panel`
(8 ta bo'lim ochiladi, kalkulyator, sozlamalar saqlanadi).

Ma'lum muammo: parallel ishga tushirilganda testlar tasodifiy yiqiladi. Sabab: bir nechta test bitta owner
telefonidan kiradi va har yangi kod oldingisini bekor qiladi. Hal: `workers: 1`.

## 16. Tuzatilgan muammolar (muhim)

- Seed akkauntlarni o'chirib yuborardi → haqiqiy foydalanuvchilar saqlanadi.
- Redemption 500 (takroriy `redemptions_voucher_id_sequence_key`) → alohida ketma-ketlik.
- Yangi merchant holat gate'i seed merchant'larini bloklardi → seed'da `approved`.
- Staff ro'yxati bo'sh 200 qaytarardi: `email` bo'sh edi, JSON kodlash xatosi yashirilardi →
  `writeJSON` endi kodlashni oldin qiladi, xato bo'lsa 500; email qo'shildi.
- Listing sahifasi "Something went wrong": `next/image` `localhost:8080` ruxsatsiz edi.
- E2E: eski `kind` maydoni (API'dan olib tashlangan), eski `/coupons`, eski kategoriya nomlari.
- Migratsiya 00018: `CHECK (currency = 'UZS')` cheklovlari enum o'zgarishini buzardi → cheklovlar vaqtincha
  olib qo'yilib qayta qo'shildi (00019 da ham shunday).
- Dev web cache buzilishi (`.next`) → tozalanadi.

## 17. Tahlil va Dealo bilan solishtirish

`docs/ANALYSIS-DEALO-VS-GRUPON.md` — Dealo PRD va dizayn hujjatlari bilan har bir modul, qoida, holat va
endpoint solishtirilgan. Eslatma: bu hujjatning ba'zi bandlari (masalan zaxira kod 8 belgi, bron 10 daqiqa,
holat nomlari, kategoriyalar, merchant paneli 7 bo'limi) bugungi ishlardan oldin yozilgan va hozir eskirgan.
Shu hisobotdagi bo'limlar haqiqiyroq.

Taxminiy bajarilish: Dealo spetsifikatsiyasining funksional qoidalari bo'yicha taxminan 55–60%
(PSP, SMS, Telegram, mobil ilova va dizayn hisobga olinmagan). Bu aniq o'lchov emas.

## 18. Ochiq qolganlar

**Biznes qoidalari:**
- Dispute: `awaiting_evidence`, `under_review`, `closed` o'tishlari yo'q.
- Chargeback oqimi yo'q.
- `fraud_hold` ustuni bor, lekin qoida uni o'rnatmaydi.
- Kech to'lov qaytarilganda `refund` yozuvi yo'q.
- Kategoriya × shahar feature flaglari (`searchable`, `purchasable` va h.k.) yo'q.
- Filial inventar'i, filial kvotasi, kunlik redemption limiti, blackout kunlar.
- Narx isboti yuklash sahifasi (merchant tarafi) hali ulanmagan; hujjat turi backend'da tayyor.
- Shartlar shabloni (kategoriya bo'yicha), MXIK/QQS, paket kodi.
- Bank rekvizitini o'zgartirish himoyasi.
- Deal wizard olti qadami va jonli preview to'liq tekshirilmagan.

**Platforma:**
- PSP (Payme, Click, Uzum): faqat mock.
- SMS provayder: faqat dev log.
- Telegram Mini App va bot.
- Scanner PWA (kassir mobil skanerlash).
- Support tizimi (ticket).
- SEO: sitemap, robots.
- Redis: hozir ishlatilmaydi, skanerlash cheklovi DB jadvali orqali.

**Sifat:**
- Playwright parallel rejimi (testlar bir-biriga aralashadi).
- Kassir uchun alohida login bilan e2e (hozir faqat egasi orqali).
- Test qamrovi (NFR-12, ≥90%) o'lchanmagan.
- Tahlil hujjatidagi `❔` bandlari tekshirilmagan.
- Deal, buyurtma va vaucher holatlari uchun to'liq holat jurnali (`state_transitions`) yo'q.
- Ledger hisob nomlari PRD 8.3 bilan solishtirilmagan.

**Loyiha:**
- `dealo/` papkasi va `dealo.zip`: `.gitignore`ga qo'shish yoki o'chirish qarori kerak.
- Tahlil hujjatini bugungi holatga yangilash.

## 19. Keyingi qadamlar (tavsiya)

1. Dispute oqimini to'liq qilish (awaiting_evidence, under_review, closed) va holat jurnali.
2. Kassir uchun alohida login bilan skanerlash testi; Scanner PWA.
3. Kategoriya feature flaglari.
4. Narx isboti sahifasi (merchant).
5. Playwright ni parallel ishlatish sababini hal qilish (har testga alohida owner).
6. Sharhlar, support va SEO.
7. PSP va SMS integratsiyasi (test'dan keyin, PRD bo'yicha).
