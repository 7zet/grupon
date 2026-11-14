# Dealo — umumiy test rejasi (barcha rollar)

> Har bir band: **qadam → kutilgan natija**. Natija boshqacha bo'lsa — `❌` qo'ying va pastdagi "Topilgan xatolar" jadvaliga yozing (skrinshot bilan).
> Telefonlar va parollar: `docs/DEMO.md`. Kod demo rejimda login sahifasida ko'rinadi.

## 0. Tayyorgarlik

- [ ] `powershell -ExecutionPolicy Bypass -File scripts\demo.ps1 -Reset` → xatosiz ishga tushadi, oxirida URL va demo akkauntlar chiqadi
- [ ] `http://localhost:3000/uz` ochiladi, stillar joyida (header, kartochkalar, chetlarda bo'sh joy)
- [ ] Har rol uchun **alohida oyna**: oddiy, inkognito, boshqa brauzer (rollar aralashmasin)
- [ ] Admin rollari uchun telefonda authenticator ilovasi tayyor
- [ ] Telefon testi uchun: `ngrok http 3000` → telefonda https havola (`docs/SHARE.md`)

---

## 1. Mehmon (login qilmagan)

- [ ] Bosh sahifa uz / ru / en tillarida → hamma matn tarjima qilingan, inglizcha so'z qolmagan
- [ ] Kategoriya menyusi → subkategoriyalar tarjima qilingan (salons/spa emas)
- [ ] Kategoriya sahifasi: filtr (subkategoriya, reyting, "tugab bormoqda"), saralash → natija o'zgaradi
- [ ] Qidiruv: "massaj", "masaj", "массаж", "spa", "kofe" → tegishli deallar chiqadi
- [ ] Mavjud bo'lmagan sahifa `/uz/abc` → "Topilmadi" sahifasi (yuklanishda qotmaydi)
- [ ] Deal sahifasi: galereya faqat shu deal rasmlari; variant tanlansa narx va "Tejaysiz" o'zgaradi; miqdor ±
- [ ] Narx, chegirma va "Sotib olish" birinchi ekranda ko'rinadi (kompyuter va telefon)
- [ ] Sharh 0 ta bo'lsa reyting ko'rinmaydi; "Amal qilish: 30 kun"; ish vaqti "Du–Ju 09:00–20:00"
- [ ] Biznes profili sahifasi: ma'lumot, filiallar, xarita, deallar
- [ ] `/uz/merchant` (biznes uchun sahifa) va `/uz/help` (support kontaktlari ko'rinadi)
- [ ] Sahifalar orasida o'tish tez (1 soniyadan kam)

## 2. Xaridor (Buyer 1)

- [ ] Deal → variant + miqdor 2 → "Sotib olish" → login → **o'sha deal, variant va miqdor** checkout'da
- [ ] Login'dan keyin **profil belgisi bosilsa menyu ochiladi** (login sahifasiga qaytmaydi)
- [ ] Yuqorida "Xaridor" belgisi
- [ ] Checkout: yakuniy narx, eski narx, tejash, muddat, filiallar, shartlarga rozilik → test to'lov "muvaffaqiyatli"
- [ ] Muvaffaqiyat sahifasi → "Vaucherlarim" → 2 ta vaucher
- [ ] Vaucher sahifasi: QR + zaxira kod; **30 soniyadan keyin QR o'zi yangilanadi**
- [ ] Test to'lov "muvaffaqiyatsiz" → "Qayta urinish" ishlaydi, vaucher chiqmaydi
- [ ] Profil: telefon raqami yashirilgan (masalan +998 90 *** ** 01), til almashadi
- [ ] Xabarnomalar sahifasi ochiladi
- [ ] Ishlatilmagan vaucher → refund so'rovi → (14 kun ichida) avtomatik qaytariladi, holat "qaytarilgan"
- [ ] Xaridor `/uz/merchant/dashboard` yoki `/uz/admin` ga kirmoqchi bo'lsa → "Ruxsat yo'q" sahifasi

## 3. Kassir (Lotus Spa) — **telefonda ham** tekshiring

- [ ] Login → **darhol skaner ekrani** ("Dealo Kassa"), do'kon menyusi yo'q
- [ ] Yuqorida: biznes · filial · kassir ismi
- [ ] Katta "Skanerlash" tugmasi → telefonda kamera ochiladi (ngrok https orqali)
- [ ] Xaridorning QR kodini skanerlash → natija kartasi: deal, variant, maskalangan ism ("Aziz U."), qolgan tashriflar → "Tasdiqlash" → yashil belgi
- [ ] Zaxira kodni qo'lda kiritish → xuddi shunday ishlaydi
- [ ] O'sha vaucherni qayta kiritish → "Allaqachon ishlatilgan" (qizil)
- [ ] Noto'g'ri kod → aniq xato matni
- [ ] "Bugungi ishlatishlar" ro'yxatida paydo bo'ladi → 10 daqiqa ichida "Bekor qilish" ishlaydi
- [ ] Kassir manzilga `/uz/merchant/payouts` yoki `/uz/merchant/dashboard` yozsa → "Ruxsat yo'q"
- [ ] Narx, to'lovlar, hisobot hech qayerda ko'rinmaydi

## 4. Boshqa biznes kassiri (Urban Grill)

- [ ] Lotus Spa vaucherini skanerlash → "Bu biznes vaucheri emas" (rad etiladi)

## 5. Xaridor — "Siz emasmisiz?" (kassir ishlatgandan keyin)

- [ ] Kassir vaucherni ishlatgach, xaridor xabarnomalarida "…ishlatildi. Siz emasmisiz?" chiqadi
- [ ] "Bu men emasman" bosilsa → shikoyat ochiladi; ikkinchi marta bosilsa yangi shikoyat yaratilmaydi

## 6. Merchant egasi (Lotus Spa)

- [ ] Login → **"Dealo Business" paneli** (do'kon emas), "Egasi · Lotus Spa" belgisi
- [ ] Overview: 4 ta raqam + kunlik sotuv grafigi (ustunlar har xil balandlikda)
- [ ] Yangi deal: barcha qadamlar → variantlar (60/90 daqiqa) → 3+ rasm → yuborish → holat "ko'rib chiqilmoqda"
- [ ] Chegirma 70% dan katta deal → yuborilganda moderatorda belgi chiqadi
- [ ] Faol deal'ni "To'xtatish" → saytdan yo'qoladi; "Davom ettirish" → qaytadi; "Arxivlash"
- [ ] Faol deal narxini o'zgartirish → yana moderatsiyaga ketadi; faqat zaxirani o'zgartirish → ketmaydi
- [ ] Buyurtmalar, mijozlar (ismlar maskalangan), analitika, sharhga javob yozish
- [ ] To'lovlar (payout) sahifasi va CSV yuklab olish
- [ ] Kalkulyator: sof daromad manfiy bo'lsa ogohlantirish
- [ ] Xodimlar: kassir qo'shish/rolini o'zgartirish
- [ ] Rad etilgan deal → rad etish sababi ko'rinadi
- [ ] Telefonda: pastki tab panel, barcha sahifalar ochiladi, yonga siljimaydi

## 7. Menejer (Lotus Spa)

- [ ] Login → Business panel, "Menejer" belgisi
- [ ] Deal yaratish va tahrirlash ishlaydi
- [ ] Menyuda **To'lovlar, Xodimlar, bank sozlamalari yo'q**; manzil orqali kirsa → "Ruxsat yo'q"

## 8. Moderator

- [ ] Login + 2FA → Admin konsol, faqat moderatsiya bo'limlari
- [ ] Merchant yuborgan deal → ko'rish → **tasdiqlash** → saytda paydo bo'ladi
- [ ] Boshqa deal → **sabab bilan rad etish** → merchant sababni ko'radi
- [ ] Merchant arizasi: hujjatlarni ochish, tasdiqlash/rad etish
- [ ] Sharh shikoyatlari → sharhni yashirish
- [ ] Moliya sahifalariga kirmoqchi bo'lsa → "Ruxsat yo'q"

## 9. Support

- [ ] Login + 2FA → o'z navbati (refund so'rovlari, shikoyatlar)
- [ ] Shikoyat: "dalil kutilmoqda" → "ko'rib chiqilmoqda" holatlariga o'tkazish
- [ ] Refundni **taklif qiladi**, lekin o'zi tasdiqlay olmaydi

## 10. Moliya №1 va №2 (ikki kishi tasdig'i)

- [ ] Moliya №1: merchant uchun payout **tayyorlaydi** (3 kunlik oynadan o'tgan redeemlar kiradi)
- [ ] Moliya №1 o'zi tasdiqlashga urinsa → rad etiladi
- [ ] Moliya №2: o'sha payout'ni **tasdiqlaydi** → "yuborildi" → reconcile
- [ ] Ochiq shikoyatli summa payout'ga **kirmaydi**
- [ ] Support taklif qilgan refund → moliya tasdiqlaydi
- [ ] Shikoyatni hal qilish (xaridor yoki merchant foydasiga)

## 11. Superadmin

- [ ] Login + 2FA → Overview: foydalanuvchilar, bizneslar, faol deallar, bugungi buyurtmalar
- [ ] Fraud signallari sahifasi (kassir 1 daqiqada ko'p skanerlasa signal chiqadi)
- [ ] Audit jurnali: yuqoridagi admin amallari yozilgan
- [ ] Biznesni to'xtatish (suspend) → uning deallari ko'rinmaydi; tiklash → qaytadi
- [ ] Qo'lda to'lov tasdiqlash (agar yoqilgan bo'lsa)

## 12. Umumiy tekshiruvlar

- [ ] Har bir rolda **profil belgisi → menyu** (hech qachon login sahifasi emas); "Chiqish" ishlaydi
- [ ] Har bir rolda yuqorida to'g'ri rol belgisi
- [ ] Barcha rol interfeyslari **390 px telefonda**: yonga siljimaydi, tugmalar bosiladigan o'lchamda
- [ ] uz / ru / en almashganda sahifa buzilmaydi, narx formati: "137 500 so'm" / "137 500 сум" / "137,500 UZS"
- [ ] Brauzer konsolida qizil xato yo'q (F12 → Console)
- [ ] Hech qayerda "Grupon", "Special local offer", "Seed ko'chasi" kabi qoldiq matn yo'q

---

## Topilgan xatolar

| # | Rol | Sahifa | Nima qildim | Nima bo'ldi | Nima bo'lishi kerak | Jiddiylik |
|---|---|---|---|---|---|---|
| 1 | | | | | | Bloklovchi / O'rta / Kosmetik |
