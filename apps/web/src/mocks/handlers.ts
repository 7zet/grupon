import { HttpResponse, http } from 'msw';
import { categories, cities, deals, merchants, reviews } from './data';

const base = '*/api/v1';
const page = <T>(items: T[], pageNumber = 1, pageSize = 20) => ({
  items: items.slice((pageNumber - 1) * pageSize, pageNumber * pageSize),
  page: pageNumber,
  page_size: pageSize,
  total: items.length,
  total_pages: Math.max(1, Math.ceil(items.length / pageSize)),
});

export const handlers = [
  http.get(`${base}/config`, () =>
    HttpResponse.json({
      locales: ['uz', 'ru', 'en'],
      currencies: ['UZS'],
      cities,
      categories,
      exchange_rate: {
        base: 'UZS',
        quote: 'USD',
        rate: '12650',
        effective_at: '2026-10-01T00:00:00Z',
      },
    }),
  ),
  http.get(`${base}/homepage`, () =>
    HttpResponse.json({
      sections: [
        {
          id: '019a0000-0000-7000-8000-000000000101',
          key: 'featured',
          title: { uz: 'Tanlangan takliflar', ru: 'Избранные предложения', en: 'Featured deals' },
          type: 'deals',
          sort_order: 1,
          deals: deals.slice(0, 8),
        },
        {
          id: '019a0000-0000-7000-8000-000000000102',
          key: 'popular',
          title: { uz: 'Yaqin atrofda mashhur', ru: 'Популярно рядом', en: 'Popular near you' },
          type: 'deals',
          sort_order: 2,
          deals: deals.slice(8, 16),
        },
      ],
    }),
  ),
  http.get(`${base}/deals`, ({ request }) => {
    const query = new URL(request.url).searchParams;
    let result = [...deals];
    const category = query.get('category');
    const city = query.get('city');
    const search = query.get('q')?.toLowerCase();
    if (category) result = result.filter((deal) => deal.category.slug === category);
    if (city)
      result = result.filter((deal) =>
        deal.locations.some((location) => location.city.slug === city),
      );
    if (search)
      result = result.filter(
        (deal) =>
          Object.values(deal.title).some((title) => title.toLowerCase().includes(search)) ||
          deal.merchant.display_name.toLowerCase().includes(search),
      );
    const minRating = Number(query.get('min_rating') ?? 0);
    if (minRating) result = result.filter((deal) => deal.rating_avg >= minRating);
    const sort = query.get('sort');
    if (sort === 'price_asc')
      result.sort((a, b) => a.options[0]!.deal_price.amount - b.options[0]!.deal_price.amount);
    if (sort === 'price_desc')
      result.sort((a, b) => b.options[0]!.deal_price.amount - a.options[0]!.deal_price.amount);
    if (sort === 'popular') result.sort((a, b) => b.sold_count - a.sold_count);
    return HttpResponse.json(page(result, Number(query.get('page') ?? 1)));
  }),
  http.get(`${base}/deals/:slug`, ({ params }) => {
    const deal = deals.find((item) => item.slug === params.slug);
    return deal ? HttpResponse.json(deal) : new HttpResponse(null, { status: 404 });
  }),
  http.get(`${base}/deals/:slug/reviews`, ({ params }) => {
    const deal = deals.find((item) => item.slug === params.slug);
    const dealReviews = reviews.filter((review) => review.deal_id === deal?.id);
    return HttpResponse.json({
      items: dealReviews.length > 0 ? dealReviews : reviews.slice(0, 6),
    });
  }),
  http.get(`${base}/deals/:slug/related`, ({ params }) => {
    const deal = deals.find((item) => item.slug === params.slug);
    return HttpResponse.json(
      page(
        deals.filter(
          (item) => item.category.slug === deal?.category.slug && item.slug !== params.slug,
        ),
        1,
        8,
      ),
    );
  }),
  http.get(`${base}/businesses/:slug`, ({ params }) => {
    const merchant = merchants.find((item) => item.slug === params.slug);
    return merchant
      ? HttpResponse.json({
          ...merchant,
          deals: deals.filter((deal) => deal.merchant.id === merchant.id),
          reviews: reviews.filter((review) => review.merchant_id === merchant.id),
        })
      : new HttpResponse(null, { status: 404 });
  }),
  http.get(`${base}/search/suggestions`, ({ request }) => {
    const q = new URL(request.url).searchParams.get('q')?.toLowerCase() ?? '';
    return HttpResponse.json({
      items: deals
        .filter((deal) =>
          Object.values(deal.title).some((value) => value.toLowerCase().includes(q)),
        )
        .slice(0, 6)
        .map((deal) => ({ type: 'deal', label: deal.title.uz, href: `/deal/${deal.slug}` })),
    });
  }),
];
