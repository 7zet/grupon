'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from '@/i18n/navigation';
import { useState } from 'react';

import type { MediaAsset } from '@/components/merchant-media-library';
import { tiyinFromSom } from '@grupon/ui';

type Localized = { uz: string; ru: string; en: string };
type Location = { id: string; label: string };

type Props = {
  categories: { id: string; label: string; subcategories: { id: string; label: string }[] }[];
  locations: Location[];
  media: MediaAsset[];
};

type Draft = {
  slug: string;
  kind: 'voucher' | 'product';
  categoryId: string;
  subcategoryId: string;
  title: Localized;
  shortDescription: Localized;
  description: Localized;
  highlights: Localized[];
  terms: Localized;
  redemptionMethod: 'qr' | 'code' | 'qr_or_code';
  redemptionInstructions: Localized;
  maxPerUser: number;
  options: { title: Localized; originalPrice: number; dealPrice: number; stock: number }[];
  mediaIds: string[];
  locationIds: string[];
  startsAt: string;
  endsAt: string;
  validDays: number;
};

const empty: Localized = { uz: '', ru: '', en: '' };
const initial: Draft = {
  slug: '',
  kind: 'voucher',
  categoryId: '',
  subcategoryId: '',
  title: empty,
  shortDescription: empty,
  description: empty,
  highlights: [empty, empty, empty],
  terms: empty,
  redemptionMethod: 'qr_or_code',
  redemptionInstructions: empty,
  maxPerUser: 1,
  options: [{ title: empty, originalPrice: 0, dealPrice: 0, stock: 0 }],
  mediaIds: [],
  locationIds: [],
  startsAt: '',
  endsAt: '',
  validDays: 30,
};

const STEPS = [
  'stepBasics',
  'stepOptions',
  'stepMedia',
  'stepDetails',
  'stepSchedule',
  'stepReview',
] as const;
const inputClass = 'rounded-control-ui border border-border bg-surface px-4 py-3 text-ink';

function localized(text: Localized) {
  return { uz: text.uz.trim(), ru: text.ru.trim(), en: text.en.trim() };
}

function filled(text: Localized) {
  return [text.uz, text.ru, text.en].every((value) => value.trim() !== '');
}

/** Returns the first problem in the current step as a translation key, or null when the step is complete. */
function stepProblem(step: number, draft: Draft): string | null {
  if (step === 0) {
    if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(draft.slug)) return 'invalidSlug';
    if (!draft.categoryId || !draft.subcategoryId) return 'required';
    if (!filled(draft.title) || !filled(draft.shortDescription) || !filled(draft.description))
      return 'required';
  }
  if (step === 1) {
    if (draft.options.length === 0) return 'optionsRequired';
    if (
      draft.options.some(
        (o) => !filled(o.title) || o.dealPrice < 1 || o.dealPrice > o.originalPrice,
      )
    )
      return 'price';
  }
  if (step === 2 && (draft.mediaIds.length < 3 || draft.mediaIds.length > 8)) return 'mediaCount';
  if (step === 3) {
    if (draft.highlights.length < 3 || !draft.highlights.every(filled)) return 'highlights';
    if (!filled(draft.terms) || !filled(draft.redemptionInstructions)) return 'required';
  }
  if (step === 4) {
    if (draft.locationIds.length === 0) return 'locationRequired';
    if (!draft.startsAt || !draft.endsAt || new Date(draft.endsAt) <= new Date(draft.startsAt))
      return 'schedule';
  }
  return null;
}

function toBody(draft: Draft, media: MediaAsset[]) {
  return {
    slug: draft.slug,
    category_id: draft.categoryId,
    subcategory_id: draft.subcategoryId,
    kind: draft.kind,
    title: localized(draft.title),
    short_description: localized(draft.shortDescription),
    description: localized(draft.description),
    highlights: draft.highlights.map(localized),
    whats_included: [],
    terms: localized(draft.terms),
    faq: [],
    redemption_method: draft.redemptionMethod,
    redemption_instructions: localized(draft.redemptionInstructions),
    booking_required: false,
    is_giftable: false,
    starts_at: new Date(draft.startsAt).toISOString(),
    ends_at: new Date(draft.endsAt).toISOString(),
    voucher_valid_days: draft.validDays,
    voucher_valid_until: null,
    max_per_user: draft.maxPerUser,
    location_ids: draft.locationIds,
    options: draft.options.map((o, index) => ({
      title: localized(o.title),
      original_price: tiyinFromSom(o.originalPrice),
      deal_price: tiyinFromSom(o.dealPrice),
      stock_total: o.stock,
      sort_order: index,
      is_active: true,
    })),
    media: draft.mediaIds.map((id, index) => ({
      url: media.find((asset) => asset.id === id)?.url ?? '',
      alt: localized(draft.title),
      sort_order: index,
      width: media.find((asset) => asset.id === id)?.width ?? 1,
      height: media.find((asset) => asset.id === id)?.height ?? 1,
      blurhash: '',
    })),
  };
}

function LocalizedFields({
  label,
  value,
  onChange,
}: {
  label: string;
  value: Localized;
  onChange: (next: Localized) => void;
}) {
  const t = useTranslations('Deals');
  return (
    <fieldset className="grid gap-2 rounded-card border border-border p-4">
      <legend className="px-1 text-sm font-semibold">{label}</legend>
      {(['uz', 'ru', 'en'] as const).map((locale) => (
        <label key={locale} className="grid gap-1 text-sm">
          {t(`lang_${locale}`)}
          <input
            className={inputClass}
            value={value[locale]}
            onChange={(e) => onChange({ ...value, [locale]: e.target.value })}
          />
        </label>
      ))}
    </fieldset>
  );
}

export function DealWizard({ categories, locations, media }: Props) {
  const t = useTranslations('Deals');
  const router = useRouter();
  const [step, setStep] = useState(0);
  const [draft, setDraft] = useState<Draft>(initial);
  const [problem, setProblem] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const subcategories = categories.find((c) => c.id === draft.categoryId)?.subcategories ?? [];

  function update(patch: Partial<Draft>) {
    setDraft((current) => ({ ...current, ...patch }));
  }

  function next() {
    const issue = stepProblem(step, draft);
    setProblem(issue);
    if (!issue) setStep((current) => Math.min(current + 1, STEPS.length - 1));
  }

  async function save(submit: boolean) {
    const issue = [0, 1, 2, 3, 4].map((s) => stepProblem(s, draft)).find(Boolean) ?? null;
    setProblem(issue);
    if (issue) return;
    setBusy(true);
    const created = await fetch('/api/bff/merchant/deals', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(toBody(draft, media)),
    });
    if (!created.ok) {
      setBusy(false);
      return setProblem('errorGeneric');
    }
    const deal = (await created.json()) as { id: string };
    if (submit) {
      const response = await fetch(`/api/bff/merchant/deals/${deal.id}/submit`, { method: 'POST' });
      if (!response.ok) {
        setBusy(false);
        return setProblem('errorGeneric');
      }
    }
    setBusy(false);
    router.refresh();
    setProblem(submit ? 'submitted' : 'saved');
  }

  return (
    <div className="grid gap-8 lg:grid-cols-[1fr_320px]">
      <div className="grid gap-6">
        <ol className="flex flex-wrap gap-2 text-sm">
          {STEPS.map((key, index) => (
            <li
              key={key}
              aria-current={index === step ? 'step' : undefined}
              className={index === step ? 'font-bold text-brand' : 'text-ink-muted'}
            >
              {index + 1}. {t(key)}
            </li>
          ))}
        </ol>

        {step === 0 && (
          <div className="grid max-w-xl gap-4">
            <label className="grid gap-1 text-sm">
              {t('slug')}
              <input
                className={inputClass}
                value={draft.slug}
                onChange={(e) => update({ slug: e.target.value })}
              />
            </label>
            <label className="grid gap-1 text-sm">
              {t('kind')}
              <select
                className={inputClass}
                value={draft.kind}
                onChange={(e) => update({ kind: e.target.value as Draft['kind'] })}
              >
                <option value="voucher">{t('kindVoucher')}</option>
                <option value="product">{t('kindProduct')}</option>
              </select>
            </label>
            <label className="grid gap-1 text-sm">
              {t('category')}
              <select
                className={inputClass}
                value={draft.categoryId}
                onChange={(e) => update({ categoryId: e.target.value, subcategoryId: '' })}
              >
                <option value="">{t('choose')}</option>
                {categories.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.label}
                  </option>
                ))}
              </select>
            </label>
            <label className="grid gap-1 text-sm">
              {t('subcategory')}
              <select
                className={inputClass}
                value={draft.subcategoryId}
                onChange={(e) => update({ subcategoryId: e.target.value })}
              >
                <option value="">{t('choose')}</option>
                {subcategories.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.label}
                  </option>
                ))}
              </select>
            </label>
            <LocalizedFields
              label={t('titleLabel')}
              value={draft.title}
              onChange={(title) => update({ title })}
            />
            <LocalizedFields
              label={t('shortDescription')}
              value={draft.shortDescription}
              onChange={(shortDescription) => update({ shortDescription })}
            />
            <LocalizedFields
              label={t('description')}
              value={draft.description}
              onChange={(description) => update({ description })}
            />
          </div>
        )}

        {step === 1 && (
          <div className="grid max-w-xl gap-4">
            {draft.options.map((option, index) => (
              <div key={index} className="grid gap-3 rounded-card border border-border p-4">
                <LocalizedFields
                  label={t('optionTitle')}
                  value={option.title}
                  onChange={(title) =>
                    update({
                      options: draft.options.map((o, i) => (i === index ? { ...o, title } : o)),
                    })
                  }
                />
                <label className="grid gap-1 text-sm">
                  {t('originalPrice')}
                  <input
                    type="number"
                    min={1}
                    className={inputClass}
                    value={option.originalPrice}
                    onChange={(e) =>
                      update({
                        options: draft.options.map((o, i) =>
                          i === index ? { ...o, originalPrice: Number(e.target.value) } : o,
                        ),
                      })
                    }
                  />
                </label>
                <label className="grid gap-1 text-sm">
                  {t('dealPrice')}
                  <input
                    type="number"
                    min={1}
                    className={inputClass}
                    value={option.dealPrice}
                    onChange={(e) =>
                      update({
                        options: draft.options.map((o, i) =>
                          i === index ? { ...o, dealPrice: Number(e.target.value) } : o,
                        ),
                      })
                    }
                  />
                </label>
                <label className="grid gap-1 text-sm">
                  {t('stock')}
                  <input
                    type="number"
                    min={0}
                    className={inputClass}
                    value={option.stock}
                    onChange={(e) =>
                      update({
                        options: draft.options.map((o, i) =>
                          i === index ? { ...o, stock: Number(e.target.value) } : o,
                        ),
                      })
                    }
                  />
                </label>
              </div>
            ))}
            <button
              type="button"
              className="w-fit font-semibold text-brand"
              onClick={() =>
                update({
                  options: [
                    ...draft.options,
                    { title: empty, originalPrice: 0, dealPrice: 0, stock: 0 },
                  ],
                })
              }
            >
              {t('addOption')}
            </button>
          </div>
        )}

        {step === 2 && (
          <div className="grid gap-4">
            <p className="text-sm text-ink-muted">{t('mediaPick')}</p>
            {media.length === 0 ? (
              <p className="text-ink-muted">{t('mediaEmpty')}</p>
            ) : (
              <ul className="grid grid-cols-2 gap-3 md:grid-cols-4">
                {media.map((asset) => {
                  const selected = draft.mediaIds.includes(asset.id);
                  return (
                    <li key={asset.id}>
                      <label className="grid cursor-pointer gap-2">
                        <input
                          type="checkbox"
                          className="sr-only"
                          checked={selected}
                          onChange={() =>
                            update({
                              mediaIds: selected
                                ? draft.mediaIds.filter((id) => id !== asset.id)
                                : [...draft.mediaIds, asset.id].slice(0, 8),
                            })
                          }
                        />
                        {/* eslint-disable-next-line @next/next/no-img-element -- merchant uploads are served by the API */}
                        <img
                          src={asset.url}
                          alt=""
                          className={`aspect-square w-full rounded-control-ui object-cover ${selected ? 'ring-2 ring-brand' : ''}`}
                        />
                      </label>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>
        )}

        {step === 3 && (
          <div className="grid max-w-xl gap-4">
            {draft.highlights.map((highlight, index) => (
              <LocalizedFields
                key={index}
                label={`${t('highlights')} ${index + 1}`}
                value={highlight}
                onChange={(next) =>
                  update({ highlights: draft.highlights.map((h, i) => (i === index ? next : h)) })
                }
              />
            ))}
            <LocalizedFields
              label={t('terms')}
              value={draft.terms}
              onChange={(terms) => update({ terms })}
            />
            <label className="grid gap-1 text-sm">
              {t('redemptionMethod')}
              <select
                className={inputClass}
                value={draft.redemptionMethod}
                onChange={(e) =>
                  update({ redemptionMethod: e.target.value as Draft['redemptionMethod'] })
                }
              >
                <option value="qr_or_code">{t('redemptionBoth')}</option>
                <option value="qr">{t('redemptionQr')}</option>
                <option value="code">{t('redemptionCode')}</option>
              </select>
            </label>
            <LocalizedFields
              label={t('redemptionInstructions')}
              value={draft.redemptionInstructions}
              onChange={(redemptionInstructions) => update({ redemptionInstructions })}
            />
            <label className="grid gap-1 text-sm">
              {t('maxPerUser')}
              <input
                type="number"
                min={1}
                className={inputClass}
                value={draft.maxPerUser}
                onChange={(e) => update({ maxPerUser: Number(e.target.value) })}
              />
            </label>
          </div>
        )}

        {step === 4 && (
          <div className="grid max-w-xl gap-4">
            <fieldset className="grid gap-2">
              <legend className="text-sm font-semibold">{t('locations')}</legend>
              {locations.map((location) => (
                <label key={location.id} className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={draft.locationIds.includes(location.id)}
                    onChange={() =>
                      update({
                        locationIds: draft.locationIds.includes(location.id)
                          ? draft.locationIds.filter((id) => id !== location.id)
                          : [...draft.locationIds, location.id],
                      })
                    }
                  />
                  {location.label}
                </label>
              ))}
            </fieldset>
            <label className="grid gap-1 text-sm">
              {t('startsAt')}
              <input
                type="datetime-local"
                className={inputClass}
                value={draft.startsAt}
                onChange={(e) => update({ startsAt: e.target.value })}
              />
            </label>
            <label className="grid gap-1 text-sm">
              {t('endsAt')}
              <input
                type="datetime-local"
                className={inputClass}
                value={draft.endsAt}
                onChange={(e) => update({ endsAt: e.target.value })}
              />
            </label>
            <label className="grid gap-1 text-sm">
              {t('validDays')}
              <input
                type="number"
                min={1}
                className={inputClass}
                value={draft.validDays}
                onChange={(e) => update({ validDays: Number(e.target.value) })}
              />
            </label>
          </div>
        )}

        {step === 5 && (
          <div className="grid gap-2 text-sm">
            <p>{t('reviewHint')}</p>
            <p>{draft.title.en || draft.title.uz}</p>
            <p>
              {draft.options.length} · {draft.mediaIds.length} · {draft.locationIds.length}
            </p>
          </div>
        )}

        {problem && (
          <p
            role="alert"
            className="flex items-start gap-2 rounded-control-ui bg-error-tint p-3 text-error"
          >
            <span aria-hidden="true">!</span>
            <span>{t(problem as Parameters<typeof t>[0])}</span>
          </p>
        )}

        <div className="flex flex-wrap gap-3">
          {step > 0 && (
            <button
              type="button"
              onClick={() => setStep(step - 1)}
              className="rounded-control-ui border border-border px-5 py-3"
            >
              {t('back')}
            </button>
          )}
          {step < STEPS.length - 1 && (
            <button
              type="button"
              onClick={next}
              className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface"
            >
              {t('next')}
            </button>
          )}
          {step === STEPS.length - 1 && (
            <>
              <button
                type="button"
                disabled={busy}
                onClick={() => save(false)}
                className="rounded-control-ui border border-border px-5 py-3"
              >
                {t('saveDraft')}
              </button>
              <button
                type="button"
                disabled={busy}
                onClick={() => save(true)}
                className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface"
              >
                {t('submitReview')}
              </button>
            </>
          )}
        </div>
      </div>

      <aside
        aria-label={t('previewTitle')}
        className="grid h-fit gap-3 rounded-card border border-border bg-surface p-4"
      >
        <h2 className="font-semibold">{t('previewTitle')}</h2>
        <p className="font-semibold">{draft.title.en || draft.title.uz || '—'}</p>
        <p className="text-sm text-ink-muted">
          {draft.shortDescription.en || draft.shortDescription.uz}
        </p>
        {draft.options[0] && draft.options[0].dealPrice > 0 && (
          <p className="text-sm">
            <span className="text-ink-muted line-through">{draft.options[0].originalPrice}</span>{' '}
            <span className="font-bold text-ink">{draft.options[0].dealPrice}</span>
          </p>
        )}
      </aside>
    </div>
  );
}
