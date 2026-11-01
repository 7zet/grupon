import type { components } from '@grupon/api-client';
import seed from '../../../../seed/data.json';

type City = components['schemas']['City'];
type Category = components['schemas']['Category'];
type Merchant = components['schemas']['Merchant'];
type Deal = components['schemas']['Deal'];
type Review = components['schemas']['Review'];
type LocalizedText = components['schemas']['LocalizedText'];

const uid = (kind: number, index: number) =>
  `019a${kind.toString().padStart(4, '0')}-0000-7000-8000-${index.toString().padStart(12, '0')}`;
const future = (days: number) => new Date(Date.UTC(2027, 0, 15 + days, 12)).toISOString();

const subcategoryNames: Record<string, LocalizedText> = {
  massage: { uz: 'Massaj', ru: 'Массаж', en: 'Massage' },
  restaurants: { uz: 'Restoranlar', ru: 'Рестораны', en: 'Restaurants' },
  activities: { uz: 'Faol hordiq', ru: 'Активный отдых', en: 'Activities' },
  hotels: { uz: 'Mehmonxonalar', ru: 'Отели', en: 'Hotels' },
};

export const cities: City[] = seed.cities.map((city, index) => ({
  id: uid(1, index + 1),
  ...city,
}));
export const categories: Category[] = seed.categories.map((category, index) => ({
  id: uid(2, index + 1),
  slug: category.slug,
  name: category.name,
  icon: category.icon,
  subcategories: category.subcategories.map((slug, subIndex) => ({
    id: uid(3 + index, subIndex + 1),
    slug,
    name: subcategoryNames[slug] ?? {
      uz: slug.replaceAll('-', ' '),
      ru: slug.replaceAll('-', ' '),
      en: slug.replaceAll('-', ' '),
    },
  })),
}));

const imageUrls = [
  'https://images.unsplash.com/photo-1540555700478-4be289fbecef',
  'https://images.unsplash.com/photo-1414235077428-338989a2e8c0',
  'https://images.unsplash.com/photo-1500530855697-b586d89ba3ee',
  'https://images.unsplash.com/photo-1520250497591-112f2f40a3f4',
  'https://images.unsplash.com/photo-1524504388940-b1c1722653e1',
  'https://images.unsplash.com/photo-1556740749-887f6717d7e4',
  'https://images.unsplash.com/photo-1544161515-4ab6ce6db874',
  'https://images.unsplash.com/photo-1517248135467-4c7edcad34c4',
  'https://images.unsplash.com/photo-1514525253161-7a46d19cd819',
  'https://images.unsplash.com/photo-1441986300917-64674bd600d8',
];

const titleTemplates: LocalizedText[] = [
  {
    uz: 'Ikki kishilik unutilmas taassurot',
    ru: 'Незабываемое впечатление для двоих',
    en: 'An unforgettable experience for two',
  },
  {
    uz: 'Premium xizmatga maxsus taklif',
    ru: 'Специальное предложение на премиум-услугу',
    en: 'A special premium service offer',
  },
  {
    uz: 'Oila uchun mazmunli dam olish',
    ru: 'Яркий отдых для всей семьи',
    en: 'A memorable family escape',
  },
  {
    uz: 'Sevimli joyda 40% gacha chegirma',
    ru: 'До 40% скидки в любимом месте',
    en: 'Up to 40% off at a local favorite',
  },
];

export const merchants: Merchant[] = seed.merchantNames.map((name, index) => ({
  id: uid(20, index + 1),
  slug: name.toLowerCase().replaceAll(' ', '-'),
  legal_name: `${name} MChJ`,
  display_name: name,
  description: {
    uz: `${name} — ishonchli mahalliy biznes.`,
    ru: `${name} — проверенный местный бизнес.`,
    en: `${name} is a trusted local business.`,
  },
  logo_url: `${imageUrls[index % imageUrls.length]}?auto=format&fit=crop&w=160&q=80`,
  cover_url: `${imageUrls[(index + 3) % imageUrls.length]}?auto=format&fit=crop&w=1600&q=80`,
  phone: '+998 71 200 00 00',
  website: 'https://example.com',
  instagram: `@${name.toLowerCase().replaceAll(' ', '')}`,
  telegram: null,
  status: 'active',
  rating_avg: 4.6 + (index % 4) / 10,
  rating_count: 36 + index * 9,
  commission_bps: 1500,
}));

export const deals: Deal[] = Array.from({ length: 100 }, (_, index) => {
  const category = categories[index % categories.length]!;
  const merchant = merchants[index % merchants.length]!;
  const city = cities[index % cities.length]!;
  const original = 180000 + (index % 8) * 50000;
  const price = Math.round((original * (0.55 + (index % 3) * 0.05)) / 1000) * 1000;
  const title = titleTemplates[index % titleTemplates.length]!;
  const media = Array.from({ length: 4 }, (_, mediaIndex) => ({
    id: uid(40 + index, mediaIndex + 1),
    url: `${imageUrls[(index + mediaIndex) % imageUrls.length]}?auto=format&fit=crop&w=1200&q=82`,
    alt: title,
    sort_order: mediaIndex,
    width: 1200,
    height: 900,
    blurhash: 'L6PZfSi_.AyE_3t7t7R**0o#DgR4',
  }));
  const location = {
    id: uid(200, index + 1),
    city,
    address: `${12 + index} Amir Temur ko'chasi`,
    lat: city.lat + (index % 5) / 100,
    lng: city.lng + (index % 4) / 100,
    opening_hours: {
      mon: '09:00–20:00',
      tue: '09:00–20:00',
      wed: '09:00–20:00',
      thu: '09:00–20:00',
      fri: '09:00–21:00',
      sat: '10:00–21:00',
      sun: '10:00–18:00',
    },
    phone: merchant.phone,
  };
  return {
    id: uid(100, index + 1),
    slug: `${category.slug}-special-${index + 1}`,
    merchant: { ...merchant, locations: [location] },
    category,
    subcategory: category.subcategories[0]!,
    title,
    short_description: {
      uz: 'Sifatli xizmat va qulay narx bir taklifda.',
      ru: 'Качественный сервис и выгодная цена в одном предложении.',
      en: 'Quality service and excellent value in one offer.',
    },
    description: {
      uz: 'Yaqinlaringiz bilan yangi taassurotlarni kashf eting. Tajribali jamoa, qulay joylashuv va oldindan kelishilgan barcha xizmatlar sizni kutmoqda.',
      ru: 'Откройте новые впечатления вместе с близкими. Вас ждут опытная команда, удобное расположение и все заявленные услуги.',
      en: 'Discover something memorable with people you love. An experienced team, convenient location, and every listed service are ready for you.',
    },
    highlights: [
      {
        uz: 'Tasdiqlangan mahalliy hamkor',
        ru: 'Проверенный местный партнёр',
        en: 'Verified local partner',
      },
      { uz: 'Oldindan kelishish oson', ru: 'Простое бронирование', en: 'Easy advance booking' },
    ],
    whats_included: [
      {
        uz: 'Tanlangan paketdagi barcha xizmatlar',
        ru: 'Все услуги выбранного пакета',
        en: 'All services in the selected package',
      },
    ],
    terms: {
      uz: 'Kupon bir marta ishlatiladi. Tashrifdan oldin joyni band qiling.',
      ru: 'Купон используется один раз. Забронируйте визит заранее.',
      en: 'Voucher is valid for one use. Book before visiting.',
    },
    faq: [
      {
        q: {
          uz: 'Oldindan band qilish kerakmi?',
          ru: 'Нужно ли бронировать заранее?',
          en: 'Do I need to book?',
        },
        a: {
          uz: 'Ha, kamida bir kun oldin.',
          ru: 'Да, минимум за один день.',
          en: 'Yes, at least one day ahead.',
        },
      },
    ],
    redemption_method: 'qr_or_code',
    redemption_instructions: {
      uz: 'QR kodni xodimga ko‘rsating.',
      ru: 'Покажите QR-код сотруднику.',
      en: 'Show the QR code to staff.',
    },
    booking_required: true,
    voucher_valid_days: 60,
    voucher_valid_until: null,
    max_per_user: 5,
    status: index === 98 ? 'sold_out' : index === 99 ? 'expired' : 'active',
    options: [
      {
        id: uid(300, index * 2 + 1),
        title: { uz: 'Standart paket', ru: 'Стандартный пакет', en: 'Standard package' },
        original_price: { amount: original, currency: 'UZS' },
        deal_price: { amount: price, currency: 'UZS' },
        stock_total: 100,
        stock_sold: index % 91,
        stock_reserved: index % 3,
        is_active: true,
      },
      {
        id: uid(300, index * 2 + 2),
        title: { uz: 'Premium paket', ru: 'Премиум-пакет', en: 'Premium package' },
        original_price: { amount: original + 100000, currency: 'UZS' },
        deal_price: { amount: price + 70000, currency: 'UZS' },
        stock_total: 60,
        stock_sold: index % 47,
        stock_reserved: 0,
        is_active: true,
      },
    ],
    media,
    sold_count: 32 + index * 17,
    rating_avg: 4.5 + (index % 5) / 10,
    rating_count: 12 + index * 3,
    starts_at: future(-20),
    ends_at: future(index % 9 === 0 ? 1 : 20 + (index % 40)),
    locations: [location],
  } satisfies Deal;
});

export const reviews: Review[] = Array.from({ length: 20 }, (_, index) => ({
  id: uid(500, index + 1),
  deal_id: deals[index % deals.length]!.id,
  merchant_id: merchants[index % merchants.length]!.id,
  user_id: uid(510, index + 1),
  voucher_id: uid(520, index + 1),
  rating: 4 + (index % 2),
  text: seed.reviewTexts[index % 3 === 0 ? 'uz' : index % 3 === 1 ? 'ru' : 'en'],
  locale: index % 3 === 0 ? 'uz' : index % 3 === 1 ? 'ru' : 'en',
  status: 'published',
  created_at: new Date(Date.UTC(2026, 8, 20 - index)).toISOString(),
  ...(index % 4 === 0
    ? {
        reply: {
          id: uid(530, index + 1),
          text: 'Rahmat! / Спасибо! / Thank you!',
          created_at: new Date(Date.UTC(2026, 8, 21 - index)).toISOString(),
        },
      }
    : {}),
}));


