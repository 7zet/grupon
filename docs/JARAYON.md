# Grupon: jarayon, holat va qolgan ishlar

Bu fayl loyihaning qayerda ekanini, nima qilinganini, nima qilinmaganini va keyin nima qilish kerakligini yozadi. Kod haqiqat manbai. Bu yerdagi raqamlar va fayl nomlari `docs/TASKS.md`, `docs/DECISIONS.md` va haqiqiy kod bilan tekshirilgan.

## 1. Qisqacha

- Loyiha: O'zbekiston uchun mahalliy chegirmali takliflar (deals) bozori. Xaridor voucher sotib oladi, merchant aksiya yaratadi, kassir voucherni ishlatadi, admin tasdiqlaydi.
- Bajarilish darajasi: taxminan 60%. Belgilangan (tasdiqlangan) vazifalar 9 ta, qisman bajarilgan 4 ta, bajarilmagan 3 ta (B3 ning to'liq sinovi, D4 ning qolgan qismi, E1 ning Docker qismi).
- Jonli holat: API `:8080`, web `:3000` ishlayapti. Sayt: `http://localhost:3000/uz`.
- Kod git ga commit qilinmagan: `git status` da 150 dan ortiq o'zgarish turibdi.

## 2. Ish tartibi (qoidalar)

- Manba: `CLAUDE.md`. `AGENTS.md` faqat ma'lumot uchun.
- Hech qachon subagent ishlatilmadi (foydalanuvchi talabi, token tejash uchun).
- Bir vazifa bir navbatda, lekin foydalanuvchi ruxsati bilan ba'zi bosqichlar ketma-ket bajarildi.
- Vazifa tugasa `docs/TASKS.md` da belgilanadi, faqat tekshirilgan ish uchun.
- Noaniqlik `docs/DECISIONS.md` ga bir qator bilan yoziladi.
- Shartnoma birinchi: `packages/api-spec/openapi.yaml` o'zgartiriladi, keyin `go generate` va `pnpm gen`.

## 3. Nima bajarildi (tasdiqlangan)

| Vazifa      | Nima qilindi                                                                                                                                   | Qanday tekshirildi                                      |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------- |
| A1–A6       | Auth (email/parol, refresh rotation), akkaunt sahifalari, savat va checkout, mock to'lov, voucher QR, worker ishlari, sotuv parallelligi testi | Go testlari, Playwright purchase testi                  |
| B1          | Merchant ro'yxatdan o'tishi, rol himoyasi                                                                                                      | Go va web testlari                                      |
| B2          | Merchant overview, biznes profili, filiallar                                                                                                   | Go testlari, jonli API                                  |
| B5          | Voucher tekshirish va ishlatish (kod orqali). Ishlatish bitta shartli UPDATE va audit bilan                                                    | Go testlari, Playwright (390px)                         |
| B7          | Kassir deal yarata olmaydi (403). Owner → admin tasdiqlaydi → xaridor sotib oladi → owner ishlatadi                                            | Playwright `merchant.spec.mjs`                          |
| C1          | Sharh yozish (ishlatilgan voucher uchun, bir marta), tahrir, o'chirish, shikoyat                                                               | Go testlari, jonli API (201/409/202/422/204)            |
| D1          | Moderatsiya navbati: tasdiqlash, rad etish sababi bilan                                                                                        | Go testlari, Playwright (brauzer)                       |
| D5          | Har admin yozuvi audit qatori yaratadi; moderator audit o'qiy olmaydi; finance/support moderatsiya qila olmaydi                                | Store testlari (`admin_audit_test.go`)                  |
| E1 (qisman) | ARCHITECTURE.md, ENVIRONMENT.md, Dockerfile'lar, `cmd/migrate`                                                                                 | Faqat hujjat va `go vet`. Docker yo'q, image qurilmagan |

## 4. Qisman bajarilgan

- **B3 (media + deal wizard):** backend to'liq (draft, tahrir, submit, ro'yxat, media). Web: rasm yuklash va wizard ochiladi (brauzerda sinalgan). To'liq wizard yuborish brauzerda sinalmagan.
  - Ochiq: `MediaAsset` da width/height yo'q, shuning uchun wizard 1x1 yuboradi (`docs/DECISIONS.md`).
  - Preview oddiy karta, haqiqiy DealCard emas.
- **D4 (admin):** sharh shikoyatlari ro'yxati, rad etish, sharhni yashirish, audit ko'rish, API jonli tekshirilgan. Web sahifalar brauzerda sinalmagan.
  - Ochiq: support/disputes (jadval ham yo'q), rollarni boshqarish, system settings.
- **E1 (hardening):** hujjat va Dockerfile'lar tayyor, lekin build qilinmagan.

## 5. Qisqartirilgan (foydalanuvchi qarori)

Bu vazifalar `docs/TASKS.md` da "olib tashlandi" deb belgilangan, o'chirilmagan:

- B4: My Deals, Orders & Sales, Customers
- B6: Analytics, Payouts, Settings, Staff invites
- C2: Sovg'a va yetkazib berish
- C3: PWA
- C4: SEO (sitemap, JSON-LD, hreflang)
- D2: Users, merchants, deals, orders (admin)
- D3: Catalog admin, homepage builder
- E2: k6 yuklama testi, OWASP

## 6. Hali qilinmagan (ochiq)

**Funksional:**

- D4: support/disputes, rollarni boshqarish UI va endpoint, system settings.
- B5: kamera orqali QR skanerlash qurilmada sinalmagan (@zxing/browser 0.1.5). Kod orqali ishlatish sinalgan.
- B5: `GET /merchant/redemptions` (oxirgi ishlatishlar) hali 501 qaytaradi.
- Admin sahifalari (`/admin/*`) `apps/web` ichida. `apps/admin` yaratilmagan.

**Sifat va sinov:**

- B3: to'liq wizard brauzer sinovi va jonli preview.
- E1: `docker build` (api: api/worker/migrate targetlari, web). Docker bu mashinada yo'q.
- `purchase.spec.mjs` ba'zan yiqiladi (checkout'da qolib ketadi). Sabab taxmin: dev bazadagi stok tugashi yoki webhook vaqti. Tuzatilmagan.
- Admin/merchant sahifalarida 404 to'g'ri javob bermaydi (200 qaytadi, ichida "topilmadi" ko'rinadi). SEO uchun C4 bekor qilingan, shuning uchun ochiq.
- Ba'zi kod yo'llari faqat API darajasida sinalgan (B2 dashboard, D1 hide natijasi).

**Ma'lumot va kod:**

- Media o'lchamlari (B3, yuqorida).
- Options: boshqa deal ID'li variant jimgina o'tkazib yuboriladi (ownership tekshiruvi yo'q).
- Merchant deal ko'rinishida faqat faol variantlar chiqadi (nofaol yashiriladi).
- Media URL'lari merchant'ning tayyor media'siga tekshirilmaydi.
- Redis keshi seeddan keyin tozalanmaydi (qo'lda tozalandi).
- Google OAuth, SMTP (Mailpit) jonli sinalmagan.

**Git:** hech narsa commit qilinmagan. Foydalanuvchi buyrug'isiz commit qilinmaydi.

## 7. Muhim qarorlar (qisqa)

To'liq ro'yxat `docs/DECISIONS.md` da. Eng muhimlari:

- Lokal disk media adapteri (MinIO/S3 keyinroq). Migratsiya seed/goose orqali, API startup'da emas.
- Voucher ishlatish bitta shartli UPDATE, audit bilan bir tranzaksiyada.
- Sharh: bitta voucherga bitta sharh (unique voucher_id). Takroriy shikoyat idempotent.
- Moderatsiya navbati va admin sahifalari hozircha `apps/web` da, `apps/admin` yaratilgach ko'chiriladi.
- Review fingerprint: narx, shartlar, variantlar o'zgarsa qayta ko'rib chiqish. Faqat stok o'zgarsa — yo'q.
- BFF: 204 javobda body yo'q (bug tuzatilgan).

## 8. Qanday ishga tushirish

```bash
# 1-terminal: API (Go)
cd services/api
export DATABASE_URL="postgres://app:app_dev_password@localhost:5433/app?sslmode=disable"
export JWT_SECRET="e2e-jwt-secret-that-is-long-enough-123456"
export VOUCHER_QR_SECRET="e2e-qr-secret-that-is-long-enough-1234567"
export PAYMENT_MOCK_ENABLED=true
export PAYMENT_MOCK_WEBHOOK_SECRET="e2e-webhook-secret-that-is-32-bytes-long"
export INTERNAL_TOKEN="e2e-internal-token-123456"
go run ./cmd/api

# 2-terminal: sayt
cd ..
export PATH="$PWD/.tools:$PATH"
pnpm --filter @grupon/web build
pnpm --filter @grupon/web start
```

Akkauntlar (parol `Password123!`): `user@local.test`, `owner@local.test`, `cashier@local.test`, `admin@local.test` (superadmin).

Testlar:

- Go: `TEST_DATABASE_URL=... go test ./...` (testlar bazani qayta seed qiladi, faqat dev baza!)
- Browser: `npx playwright test -c e2e/playwright.config.mjs`

## 9. Fayllar xaritasi

- Shartnoma: `packages/api-spec/openapi.yaml`
- Go: `services/api/internal/{domain,store,http}`, migratsiyalar `services/api/migrations`, SQL `services/api/queries`
- Web: `apps/web/src/app` (sahifalar), `apps/web/src/components`, BFF `apps/web/src/app/api/bff`
- Testlar: `services/api/internal/**/*_test.go`, `e2e/*.spec.mjs`
- Hujjatlar: `docs/TASKS.md` (holat), `docs/DECISIONS.md` (qarorlar), `docs/ARCHITECTURE.md`, `docs/ENVIRONMENT.md`, `docs/PROGRESS.md`

## 10. Keyingi qadamlar (tavsiya tartibida)

1. Brauzerda to'liq wizard sinovi (B3) va merchant sahifalarini qo'lda ko'rib chiqish.
2. Kodni commit qilish (siz buyursangiz), shunda o'zgarishlar tarixda saqlanadi.
3. D4 qolgan qismi: rollarni boshqarish, support, settings.
4. Docker bilan image build (E1) — Docker o'rnatilgan mashinada.
5. Admin sahifalarini `apps/admin` ga ko'chirish.
6. Ochiq xatolar: purchase flake, media o'lchamlari, 404 holati.
