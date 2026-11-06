# Dealo talablari — tekshiruv ro'yxati

> Manba: Dealo PRD v1.1 (`docs/prd/00…18`), ADR-0001…0008, `docs/analysis/prd-gaps.md`.
> Maqsad: loyihani shu ro'yxat bo'yicha tekshirish. Har bir band bitta ID ga ega; tekshiruv natijasi `docs/AUDIT.md` da xuddi shu ID bilan yoziladi.
> Belgi **➖** — bu band demo loyihada **ataylab** qilinmaydi (pastdagi "Ataylab farqlar" bo'limiga qarang). Uni xato deb hisoblamang, faqat `docs/HANDOVER.md` da yozilganini tekshiring.

---

## 0. Loyiha doirasi va ataylab farqlar

Loyiha — Dealo spetsifikatsiyasi asosidagi **demo**: real foydalanuvchi, real pul va yuridik ish yo'q. Quyidagilar Dealo'dan ataylab farq qiladi:

- **DEV-01** Stek: Dealo'da Django + DRF + aiogram + Django Admin; bizda Go API/worker/bot + Next.js (sayt, merchant va admin panellar).
- **DEV-02** To'lov: real PSP (Payme/Click/Uzum), fiskal chek, MXIK, paket kodi, QQS, EDO — yo'q. Mock to'lov + qo'lda tasdiqlash rejimi. PSP interfeysi tayyor bo'lishi kerak.
- **DEV-03** Hosting: Vercel + Render + Neon (O'zbekistondan tashqarida). Faqat demo ma'lumot; real PII saqlanmaydi.
- **DEV-04** Yuridik gate'lar G1–G9 qo'llanilmaydi.
- **DEV-05** Support: ticket tizimi va SLA yo'q; faqat kontaktlar (telefon, Telegram, email) va Telegram bot.
- **DEV-06** Telegram Mini App: to'liq initData login yo'q; faqat botdagi "Saytni ochish" tugmasi.
- **DEV-07** Kategoriya flag'lari faqat kategoriya darajasida (Dealo'da kategoriya × shahar).
- **DEV-08** Tillar: uz, ru, en (Dealo: uz-Latn, ru, uz-Cyrl avtomatik). uz-Cyrl yo'q.
- **DEV-09** DEMO_MODE: kirish kodi sahifada ko'rinadi, test to'lov tugmalari, "Demo" belgisi. DEMO_MODE=false bo'lsa, bularning hech biri ko'rinmasligi shart.
- **DEV-10** Sharhlar (Dealo'da R1b/R2) — demo'da bor.
- **DEV-11** Booking (R2), CRM/referral, reklama, loyalty, sovg'a (R2–R3) — yo'q.
- **DEV-12** Docker fayllari bor, lekin "ixtiyoriy, tekshirilmagan".

---

## 1. Doimiy tamoyillar

- **P-01** Logistika yo'q: `Shipment`, `ShippingAddress`, `delivery_fee`, kuryer, ombor obyektlari modelda umuman yo'q (BR-14).
- **P-02** Pul faqat UZS, tiyinda butun son (int64/bigint); float hech qayerda ishlatilmaydi.
- **P-03** ID'lar UUIDv7; vaqt `timestamptz` UTC'da; kun chegaralari (muddat, kunlik limit) `Asia/Tashkent` bo'yicha.
- **P-04** Ledger ikki yoqlama va faqat qo'shiladigan (append-only); tuzatish faqat teskari yozuv (storno).
- **P-05** Bitta Order = bitta merchant = bitta to'lov; savatcha yo'q, faqat "Hozir sotib olish".
- **P-06** Karta ma'lumotlari platformada saqlanmaydi (BR-07).
- **P-07** Sayt Telegram'siz va real SMS'siz ham to'liq ishlaydi.
- **P-08** Shaxsiy ma'lumotlar O'zbekistonda saqlanadi. ➖ (DEV-03)

## 2. Rollar va ruxsatlar (RBAC)

- **R-01** Mehmon: qidiruv, deal va shartlarni ko'radi; sotib olish uchun telefon tasdig'i kerak.
- **R-02** Xaridor: faqat o'z buyurtma, vaucher, refund va sharhlarini ko'radi; "bu men emasman" shikoyati.
- **R-03** Merchant owner: filiallar, xodimlar, deallar, hisobotlar, bank rekviziti.
- **R-04** Merchant manager: deal draft, kvota, statistika; bank rekviziti va xodim rollari yo'q.
- **R-05** Kassir: faqat QR skanerlash, zaxira kod, o'z redeem tarixi; narx, refund, hisobot, payout yo'q.
- **R-06** Moderator: merchant KYC, deal moderatsiyasi, kontent shikoyatlari; pul amallari yo'q.
- **R-07** Support: refund taklifi, vaucherni vaqtincha bloklash; refund/payoutni yakka tasdiqlay olmaydi.
- **R-08** Finance operator: reconciliation, payout tayyorlash, refund ijrosi; o'zi tayyorlaganini tasdiqlay olmaydi.
- **R-09** Finance approver: payout va refund tasdig'i (2-imzo); ledger yozuvini o'zgartira olmaydi.
- **R-10** Trust & fraud analyst: fraud navbati, payout hold, anomaliyalar. (Alohida rol bo'lmasa, qaysi rol bajarishini yozing.)
- **R-11** Superadmin: rollar, flag'lar; har bir harakati audit qilinadi.
- **R-12** Barcha ruxsatlar serverda tekshiriladi (UI'da yashirish yetarli emas).
- **R-13** Moliyaviy amallar 4-eyes: tayyorlovchi ≠ tasdiqlovchi.

## 3. Kategoriyalar

- **C-01** Ochiq pilot kategoriyalar: Beauty & Wellness, Food & Drink, Things to Do, Health & Fitness.
- **C-02** Qolganlari (Automotive, Education, Events, Home Services, Travel, Pickup Goods, Medical) modelda bor, lekin yopiq.
- **C-03** Flag'lar: `searchable`, `deal_creation_enabled`, `publish_enabled`, `purchasable`, `redeemable` (+ `bookable`, `ads_enabled`) va ular serverda qo'llanadi.
- **C-04** Kategoriya yopilsa, mavjud vaucherlar `redeemable` bo'lib qoladi.
- **C-05** Yopiq kategoriyada deal yuborish aniq sabab bilan rad etiladi (BR-12, AT-10).
- **C-06** Taqiqlangan takliflar (alkogol, tamaki, qimor, retsept dorilari, litsenziyasiz tibbiyot, kredit, kattalar kontenti, qurol) moderatsiya qoidalarida aks etgan.
- **C-07** Kategoriya × shahar flag'lari. ➖ (DEV-07)

## 4. FR-01 Merchant onboarding

- **FR-01.1** Ariza: STIR, yuridik nom, tashkiliy shakl (MChJ/YaTT), soliq rejimi, mas'ul shaxs, telefon, filiallar, bank rekviziti (hisob, MFO).
- **FR-01.2** Hujjatlar yuklanadi va versiyalanadi: guvohnoma, kerak bo'lsa litsenziya, filial fotosi.
- **FR-01.3** Holatlar: `draft → submitted → in_review → approved | rejected`; `approved → suspended → approved | terminated`.
- **FR-01.4** Oferta elektron qabul: vaqt, IP, versiya va qabul qilgan shaxs saqlanadi.
- **FR-01.5** Bank rekviziti o'zgarishi: owner 2FA → platforma tasdig'i → 72 soat payout hold → eski raqamga xabar (BR-11). ➖ (DEV-02, keyingi bosqich)
- **FR-01.6** Owner xodimlarni telefon orqali taklif qiladi va filialga biriktiradi.
- **FR-01.7** Rad etish sababi merchantga ko'rinadi.

## 5. FR-02 Deal Studio

- **FR-02.1** Maydonlar: kategoriya, sarlavha, tavsif (uz majburiy), kamida 3 rasm, variantlar, filiallar, shartlar, sotuv va ishlatish muddati.
- **FR-02.2** Variant: `original_price`, `deal_price`, `voucher_type` (single/multi), `redemptions_allowed`. (MXIK, paket kodi, QQS ➖ DEV-02)
- **FR-02.3** Narx dalili (menyu/prays/chek) majburiy; chegirma 70% dan yuqori bo'lsa qo'shimcha tekshiruv.
- **FR-02.4** Limitlar: global kvota, filial kvotasi, bitta xaridor limiti, kunlik redemption limiti, blackout kunlar/soatlar.
- **FR-02.5** Kategoriya bo'yicha shartlar shabloni (bron kerakmi, bir tashrifda nechta vaucher, boshqa aksiyalar bilan birlashmaslik, kechikish/kelmaslik).
- **FR-02.6** Kampaniya kalkulyatori: sof tushum; manfiy bo'lsa ogohlantirish.
- **FR-02.7** Holatlar: `draft → in_review → approved → scheduled → active ⇄ paused → ended → archived`; `in_review → changes_requested → draft`.
- **FR-02.8** Faol dealda narx/shart/variant o'zgarishi → qayta moderatsiya; faqat stok o'zgarishi → yo'q.

## 6. FR-03 Checkout

- **FR-03.1** Faqat "Hozir sotib olish": bitta variant, 1–N dona (N ≤ bitta xaridor limiti).
- **FR-03.2** To'lovdan oldin ko'rsatiladi: yakuniy narx, asl narx, ishlatish muddati, filiallar, bron talabi, refund qoidasi; xaridor shartlarni belgilab tasdiqlaydi.
- **FR-03.3** Server narxni qayta hisoblaydi; Order'da narx va shartlarning o'zgarmas snapshot'i (BR-06).
- **FR-03.4** Inventory hold 15 daqiqa; tugasa Order `expired`, inventory qaytadi.
- **FR-03.5** Kech to'lov: inventory bo'lsa vaucher chiqariladi, bo'lmasa avtomatik to'liq refund va xabar (AT-13).
- **FR-03.6** `POST /orders` Idempotency-Key bilan; takror so'rov ikkinchi buyurtma yaratmaydi.
- **FR-03.7** Login faqat checkout'da so'raladi; kirgandan keyin xaridor o'sha deal va variantga qaytadi.

## 7. FR-04 Vaucher

- **FR-04.1** Har bir sotilgan dona uchun bitta vaucher (BR-01: `UNIQUE(order_item_id, seq)`).
- **FR-04.2** Token 128-bit tasodifiy, DB'da faqat HMAC-hash. (Boshqacha yechim bo'lsa — qanday va xavfsizligini yozing.)
- **FR-04.3** QR ichida faqat opaque token, PII yo'q; zaxira kod 10 belgi (Crockford base32).
- **FR-04.4** QR har 30 soniyada yangilanadi (web'da ham).
- **FR-04.5** `multi_use`: har skanerda `redemptions_used + 1`; oxirgisida `redeemed`.
- **FR-04.6** Muddat tugashidan 7 va 1 kun oldin eslatma.
- **FR-04.7** Muddat: `min(valid_until, xarid + N kun)`, Asia/Tashkent 23:59:59 da tugaydi (GAP-17).
- **FR-04.8** To'lov server tomonda tasdiqlanmaguncha vaucher `active` bo'lmaydi (BR-02).

## 8. FR-05 Redemption

- **FR-05.1** Kassir QR'ni kamera bilan skanerlaydi yoki zaxira kodni kiritadi (Scanner PWA).
- **FR-05.2** Bitta tranzaksiyada tekshiruv: token, holat, muddat, merchant va filial ruxsati, to'lov holati, flag'lar (`fraud_hold`, `dispute_open`), kunlik limit.
- **FR-05.3** Kassirga natija: deal, variant, xaridorning maskalangan ismi ("Alisher U."), qolgan tashriflar.
- **FR-05.4** Redeemdan keyin darhol xaridorga xabar "…ishlatildi. Siz emasmisiz?" + bir bosishda shikoyat.
- **FR-05.5** Kassir 10 daqiqa ichida redeemni bekor qila oladi; keyin faqat support. Bekor qilinganda teskari provodka va vaucher holati qaytadi (GAP-05, GAP-08).
- **FR-05.6** Offline redeem yo'q; internet bo'lmasa aniq xabar.
- **FR-05.7** Kassir uchun rate limit 30/daqiqa; 10 xato urinishdan keyin blok.
- **FR-05.8** Parallel ikkinchi redeem → 409; begona merchant → 403; expired/refunded → 422.

## 9. FR-06 Refund

- **FR-06.1** Ishlatilmagan, xariddan 14 kun ichida → avtomatik to'liq refund.
- **FR-06.2** Ishlatilmagan, 14 kundan keyin, muddat ichida → so'rov, support + finance tasdig'i.
- **FR-06.3** Merchant xizmat ko'rsatmadi/yopildi → to'liq refund, merchantdan undiriladi.
- **FR-06.4** Ishlatilgan, sifat shikoyati → dispute oqimi, finance approver.
- **FR-06.5** Muddati o'tgan → breakage siyosati (90 kun qaytarish huquqi).
- **FR-06.6** Multi-use qisman ishlatilgan → qolgan tashriflar ulushi (GAP-07).
- **FR-06.7** Refund faqat asl to'lov usuliga.
- **FR-06.8** Refund suiiste'moli: 30 kunda > 3 refund → qo'lda ko'rib chiqish.

## 10. FR-07 Settlement va payout

- **FR-07.1** Merchant majburiyati har bir redeemda hisoblanadi.
- **FR-07.2** Payout haftada bir marta; redeemdan keyin 3 kunlik dispute oynasi o'tgach.
- **FR-07.3** Payout paketi: tayyorlash → tasdiqlash (boshqa odam) → yuborildi → bank hujjat raqami → reconciliation.
- **FR-07.4** Merchant har payout bo'yicha hisobot oladi: redeemlar, komissiya, ushlab qolishlar, refundlar.
- **FR-07.5** Manfiy balans keyingi payoutdan ushlab qolinadi; 60 kundan oshsa undirish.
- **FR-07.6** Ochiq dispute summasi payoutga kirmaydi (BR-15).

## 11. FR-08 Support

- **FR-08.1** Support kontaktlari (telefon, Telegram, email) sozlamadan o'qiladi va saytda ko'rinadi.
- **FR-08.2** Dispute holatlari: `open → awaiting_evidence → under_review → resolved_customer | resolved_merchant → closed`.
- **FR-08.3** Dispute ochilganda merchant payouti shu summa bo'yicha hold.
- **FR-08.4** Ticket tizimi, dalil fayllari, SLA (2 soat / 48 soat). ➖ (DEV-05)

## 12. FR-09 Discovery

- **FR-09.1** Bosh sahifa: kategoriyalar, "tugab bormoqda", tavsiyalar; (tuman tanlash, "yaqin atrofda").
- **FR-09.2** Qidiruv xatolarga chidamli; lotin va kirilda bir xil natija.
- **FR-09.3** Filtrlar: kategoriya, tuman, narx, chegirma, "bugun ishlatsa bo'ladi".
- **FR-09.4** SEO: server-side render, `schema.org/Offer`, sitemap, robots, kategoriya × tuman landing.
- **FR-09.5** Marketing xabarlari faqat alohida rozilik bilan; obunadan chiqish.
- **FR-09.6** Urgency matnlari ("N ta qoldi", "tugaydi") faqat real ma'lumotdan.

## 13. Holat mashinalari

> Har bir o'tish bitta servis funksiyasi orqali; ruxsatsiz o'tish → **422**; o'tish jurnali (`state_transitions` yoki audit) va outbox hodisasi.

- **S-01** Merchant: `draft → submitted → in_review → approved | rejected`; `approved ⇄ suspended → terminated`.
- **S-02** Deal: `draft → in_review → approved → scheduled → active ⇄ paused → ended → archived`; `in_review → changes_requested → draft`.
- **S-03** Order: `created → awaiting_payment → paid → completed`; `awaiting_payment → expired | failed | cancelled`; `paid → partially_refunded | refunded`.
- **S-04** Payment: `created → pending → succeeded | failed`; `succeeded → chargeback_open → chargeback_won | chargeback_lost`.
- **S-05** Refund: `requested → approved | rejected`; `approved → processing → succeeded | failed`.
- **S-06** Voucher: `issued → active → partially_redeemed → redeemed`; `active → expired | refunded | voided`; flag'lar `fraud_hold`, `dispute_open`.
- **S-07** Redemption: `confirmed → reversed`.
- **S-08** Payout: `draft → pending_approval → approved → sent → reconciled`; `sent → failed → draft`.
- **S-09** Dispute: `open → awaiting_evidence → under_review → resolved_customer | resolved_merchant → closed`.
- **S-10** Ruxsatsiz o'tish 422 qaytaradi va holat o'zgarmaydi.
- **S-11** Har bir o'tish jurnalga yoziladi (kim, qachon, eski → yangi).

## 14. Biznes qoidalari (BR)

- **BR-01** Bitta to'lov vaucherlarni ikki marta chiqarmaydi (unique + idempotency).
- **BR-02** To'lov server tomonda tasdiqlanmaguncha vaucher `active` emas.
- **BR-03** `redemptions_used ≤ redemptions_allowed` (`FOR UPDATE` + CHECK).
- **BR-04** Vaucher faqat dealga biriktirilgan filialda ishlatiladi.
- **BR-05** `sold + held ≤ capacity` (atomik shartli UPDATE).
- **BR-06** Order shartlari sotib olingan versiya bo'yicha o'zgarmaydi.
- **BR-07** Karta ma'lumotlari saqlanmaydi.
- **BR-08** Refunded/voided vaucher ishlatilmaydi.
- **BR-09** Moliyaviy holat faqat ledger orqali o'zgaradi; ledger'da UPDATE/DELETE DB trigger bilan taqiqlangan.
- **BR-10** Har bir journal entry balanslangan (debit = kredit), DB cheklovi bilan.
- **BR-11** Bank rekviziti o'zgarsa 72 soat payout hold. ➖ (FR-01.5)
- **BR-12** Yopiq kategoriyada publish yo'q.
- **BR-13** Bitta Order = bitta merchant.
- **BR-14** Jismoniy yetkazish obyekti yo'q.
- **BR-15** Ochiq dispute summasi payoutga kirmaydi.

## 15. Ledger va pul oqimi

- **L-01** Hisoblar: `psp_receivable`, `bank:operating`, `customer_prepaid:{merchant}`, `merchant_payable:{merchant}`, `merchant_reserve`, `refunds_payable`, `vat_payable`, `commission_revenue`, `breakage_revenue`, `psp_fees`, `merchant_receivable:{merchant}` (nomlar farq qilsa, moslik jadvalini yozing).
- **L-02** To'lov tasdiqlandi: `psp_receivable` → `customer_prepaid`.
- **L-03** Vaucher ishlatildi: `customer_prepaid` → `merchant_payable` + `commission_revenue`.
- **L-04** Payout yuborildi: `merchant_payable` → `bank:operating`.
- **L-05** Refund (ishlatilmagan): `customer_prepaid` → `refunds_payable` → `bank:operating`.
- **L-06** Redeem reversal — teskari yozuv; payout yuborilgan bo'lsa `merchant_receivable` (GAP-08).
- **L-07** Ishlatilgandan keyingi refund (dispute) provodkasi (GAP-09).
- **L-08** Multi-use: har redeem `floor(narx/N)`, qoldiq oxirgisiga; komissiya ham shunday (GAP-07).
- **L-09** Breakage: muddat tugagach chegirma yo'qoladi, to'langan pul 90 kun qaytariladi; daromad sifatida rejalashtirilmaydi (`customer_prepaid` invariant — GAP-02).
- **L-10** Reconciliation: kunlik; payout'dan keyin; oylik invariant "`customer_prepaid` = faol vaucherlar summasi".
- **L-11** Komissiya sozlamada (bps), kategoriya bo'yicha bo'lishi mumkin; qattiq kodlanmagan.
- **L-12** Pul oqimi varianti (A/B/C), refund muddatlari, payout chastotasi kodga qattiq yozilmagan — konfiguratsiya.

## 16. Xavfsizlik va antifraud

- **SEC-01** Admin va moliya rollari uchun 2FA (TOTP) majburiy; sessiya ≤ 8 soat.
- **SEC-02** Xaridor sessiyasi httpOnly cookie; refresh token rotatsiyasi.
- **SEC-03** OTP: 60 s cooldown, telefon va IP limiti; 5 xato urinishdan keyin 15 daqiqa blok.
- **SEC-04** Telefon raqami interfeysda maskalangan holda ko'rsatiladi.
- **SEC-05** Merchant hujjatlari maxfiy saqlanadi, faqat vakolatli rollarga ochiladi.
- **SEC-06** Fraud qoidalari: ish vaqtidan tashqari redeem, 1 daqiqada > 5 redeem, filialdan > 500 m → FraudSignal va payout hold.
- **SEC-07** Yangi akkaunt uchun birinchi kunda xarid limiti.
- **SEC-08** Har bir admin va sezgir merchant amali audit jurnaliga yoziladi.
- **SEC-09** CI'da bog'liqliklar skaneri va sir (secret) qidiruvi.
- **SEC-10** Repo'da maxfiy kalit, parol, token yo'q; prod kalitlari dev kalitlaridan farqli.
- **SEC-11** IP allowlist/VPN admin uchun, tashqi pentest. ➖ (DEV-03/DEV-04)

## 17. API standartlari

- **API-01** OpenAPI 3.1 yagona manba; Go va TS kodi undan generatsiya qilinadi.
- **API-02** Xatolar RFC 9457 `problem+json`, barqaror `code` bilan; 400/401/403/404/409/422/429.
- **API-03** Har bir javobda `X-Request-Id`; URL orqali versiyalash (`/v1`).
- **API-04** Deal ro'yxati pagination bilan (Dealo: cursor).
- **API-05** R1a endpointlari bor: auth OTP, deals, orders, payment, webhook, `me/vouchers`, `me/vouchers/{id}/qr`, `me/refunds`, `merchant/redemptions` (+ reverse), `merchant/deals` (+ submit), `merchant/statements`, `admin/deals/{id}/decision`, `admin/payouts/{id}/approve`.
- **API-06** Webhook: imzo tekshiruvi, `provider_event_id` bo'yicha dedup, takror kelganda 200.
- **API-07** Outbox: hodisalar bir marta yuboriladi; Idempotency-Key 24 soat saqlanadi.

## 18. Bildirishnomalar va Telegram

- **T-01** Kirish kodi SMS va Telegram orqali yuboriladi; SMS provayderi sozlamadan tanlanadi.
- **T-02** Telegram bot: /start (til tanlash), /help, /support; token bo'lmasa bot o'chiq.
- **T-03** Hisobni Telegram'ga ulash (bir martalik havola) va uzish.
- **T-04** Xabarlar: vaucher chiqarildi; vaucher ishlatildi + "Bu men emasman" tugmasi (dispute ochadi, takroriy bosish bitta dispute).
- **T-05** Muddat eslatmalari (7 va 1 kun), takrorlanmaydi.
- **T-06** Botda "Saytni ochish" tugmasi.
- **T-07** Webhook Telegram secret-token sarlavhasi bilan tekshiriladi.
- **T-08** SMS xarajatini himoya qilish: kunlik limitlar (IP va umumiy).
- **T-09** Telegram Mini App initData login. ➖ (DEV-06)

## 19. Admin (M15)

- **ADM-01** Deal moderatsiyasi: tasdiqlash, sabab bilan rad etish, `changes_requested`.
- **ADM-02** Merchant KYC: hujjatlarni ko'rish, tasdiqlash/rad etish.
- **ADM-03** Moliya: payout tayyorlash/tasdiqlash/reconcile, refund tasdiqlash, dispute hal qilish.
- **ADM-04** Sharh shikoyatlari va audit jurnali ko'rinishi.
- **ADM-05** Qo'lda to'lov tasdiqlash (pilot rejim), audit bilan.
- **ADM-06** Foydalanuvchi/merchant bloklash, kategoriya va shaharlarni boshqarish.

## 20. Nofunksional talablar (demo uchun moslashtirilgan)

- **NFR-02** Qidiruv/deal API tez (Dealo: p95 ≤ 800/400 ms).
- **NFR-04** Redeem: 100 parallel urinish → aynan 1 muvaffaqiyat.
- **NFR-05** Mobil LCP ≤ 2.5 s.
- **NFR-08** Tillar to'liq tarjima qilingan (UI'da tarjimasiz kalit yo'q).
- **NFR-09** Asosiy oqimlar WCAG 2.2 AA (klaviatura, kontrast, label'lar).
- **NFR-10** Audit jurnali o'zgarmas.
- **NFR-12** Domen yadrosi (to'lov, vaucher, ledger) testlar bilan qoplangan (Dealo: ≥ 90%).
- **NFR-13** Telefon (390 px) va kompyuter (1440 px) ko'rinishida asosiy sahifalar buzilmaydi.

## 21. Qabul sinovlari (AT)

- **AT-01** Bir xil success webhook 5 marta → 1 vaucher to'plami, 1 journal entry.
- **AT-02** Bitta single-use vaucherga 100 parallel skan → 1 × 200, 99 × 409.
- **AT-03** capacity 20, sold 19, 10 parallel order → ko'pi bilan 1 hold.
- **AT-04** B merchant kassiri A vaucherini skanerlaydi → 403, holat o'zgarmaydi, audit.
- **AT-05** Expired/refunded vaucher skani → 422 aniq sabab bilan, ledger yozuvi yo'q.
- **AT-06** Merchant narxni o'zgartiradi → avvalgi buyurtma va vaucher eski shartlarda.
- **AT-07** To'lov failed, hold tugaydi → inventory qaytadi, vaucher yo'q, order `expired`.
- **AT-08** PSP hisobotida mahalliy yo'q to'lov → alert. ➖ (DEV-02)
- **AT-09** Kassir payout tasdiqlashga urinadi → 403 + audit.
- **AT-10** Yopiq kategoriyada deal yuborish → rad, aniq sabab.
- **AT-11** 14 kun ichidagi ishlatilmagan vaucher refund → `refunded`, teskari provodka, xabar.
- **AT-12** Vaucher 23:59 Toshkentda tugaydi → 23:58 da 200, 00:01 da 422.
- **AT-13** Hold tugagan, inventory tugagan, PSP success → avtomatik refund, vaucher yo'q.
- **AT-14** Merchant 1 daqiqada 8 vaucher skanerlaydi → FraudSignal, summa payoutdan chiqadi.
- **AT-15** 10 tashriflik vaucher, 9 tasi ishlatilgan, 2 parallel skan → 1 × 200 (`redeemed`), 1 × 409.
- **AT-16** "Men emasman" bosilgan → summa payoutga kirmaydi.
- **AT-17** Bank rekviziti yangilandi, 72 soat ichida payout → bloklanadi. ➖ (FR-01.5)
- **AT-18** Payout'ni tayyorlagan odam o'zi tasdiqlaydi → 403 + audit.
