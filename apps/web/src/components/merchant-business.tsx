'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from '@/i18n/navigation';
import { pickText } from '@/lib/localized';
import { useState } from 'react';

type Business = {
  display_name: string;
  phone: string;
  website: string | null;
  instagram: string | null;
  telegram: string | null;
};

type Location = {
  id: string;
  address: string;
  phone: string;
  lat: number;
  lng: number;
  city: { id: string; name: { uz: string; ru: string; en: string } };
};

type City = { id: string; label: string; lat: number; lng: number };

const inputClass = 'rounded-control-ui border border-border bg-surface px-4 py-3 text-ink';

/** Maps API problem codes to translation keys for the panel. */
function problemKey(code: string | undefined) {
  switch (code) {
    case 'forbidden':
      return 'forbidden' as const;
    case 'location_in_use':
      return 'locationInUse' as const;
    default:
      return 'errorGeneric' as const;
  }
}

type SendResult = { ok: boolean; code?: string | undefined };

async function send(path: string, method: string, body?: unknown): Promise<SendResult> {
  const response = await fetch(`/api/bff/${path}`, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: body === undefined ? null : JSON.stringify(body),
  });
  if (response.status === 204) return { ok: true };
  const data = (await response.json().catch(() => ({}))) as { code?: string };
  return response.ok ? { ok: true } : { ok: false, code: data.code };
}

export function BusinessForm({ business, canEdit }: { business: Business; canEdit: boolean }) {
  const t = useTranslations('Panel');
  const router = useRouter();
  const [message, setMessage] = useState<string | null>(null);
  const [values, setValues] = useState({
    display_name: business.display_name,
    phone: business.phone,
    website: business.website ?? '',
    instagram: business.instagram ?? '',
    telegram: business.telegram ?? '',
  });

  async function save(event: React.FormEvent) {
    event.preventDefault();
    setMessage(null);
    const result = await send('merchant/business', 'PATCH', values);
    if (!result.ok) {
      setMessage(t(problemKey(result.code)));
      return;
    }
    setMessage(t('saved'));
    router.refresh();
  }

  return (
    <form className="grid max-w-xl gap-4" onSubmit={save}>
      {(['display_name', 'phone', 'website', 'instagram', 'telegram'] as const).map((field) => (
        <label key={field} className="grid gap-1 text-sm" htmlFor={field}>
          {t(field === 'display_name' ? 'displayName' : field)}
          <input
            id={field}
            disabled={!canEdit}
            className={inputClass}
            value={values[field]}
            onChange={(event) => setValues({ ...values, [field]: event.target.value })}
          />
        </label>
      ))}
      {!canEdit && <p className="text-sm text-ink-muted">{t('readOnly')}</p>}
      {message && <p role="status" className="text-sm text-ink">{message}</p>}
      {canEdit && (
        <button type="submit" className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface">
          {t('save')}
        </button>
      )}
    </form>
  );
}

export function LocationsPanel({
  locations,
  cities,
  canEdit,
  locale,
}: {
  locations: Location[];
  cities: City[];
  canEdit: boolean;
  locale: string;
}) {
  const t = useTranslations('Panel');
  const router = useRouter();
  const [message, setMessage] = useState<string | null>(null);
  const [draft, setDraft] = useState({ city_id: cities[0]?.id ?? '', address: '', phone: '', hours: '09:00-18:00' });

  async function add(event: React.FormEvent) {
    event.preventDefault();
    setMessage(null);
    const city = cities.find((item) => item.id === draft.city_id);
    const result = await send('merchant/locations', 'POST', {
      city_id: draft.city_id,
      address: draft.address,
      phone: draft.phone,
      lat: city?.lat ?? 0,
      lng: city?.lng ?? 0,
      opening_hours: { mon: draft.hours, tue: draft.hours, wed: draft.hours, thu: draft.hours, fri: draft.hours, sat: draft.hours, sun: draft.hours },
    });
    if (!result.ok) {
      setMessage(t(problemKey(result.code)));
      return;
    }
    router.refresh();
  }

  async function remove(id: string) {
    setMessage(null);
    const result = await send(`merchant/locations/${id}`, 'DELETE');
    if (!result.ok) {
      setMessage(t(problemKey(result.code)));
      return;
    }
    router.refresh();
  }

  return (
    <section aria-labelledby="locations-title" className="grid gap-5">
      <h2 id="locations-title" className="font-semibold">{t('locations')}</h2>
      {locations.length === 0 ? (
        <p className="text-ink-muted">{t('noLocations')}</p>
      ) : (
        <ul className="grid gap-3">
          {locations.map((location) => (
            <li key={location.id} className="flex items-center justify-between gap-4 rounded-card border border-border bg-surface p-4">
              <span>
                <span className="block font-semibold">{pickText(location.city.name, locale)}</span>
                <span className="text-sm text-ink-muted">{location.address}</span>
              </span>
              {canEdit && (
                <button type="button" onClick={() => remove(location.id)} className="text-sm font-semibold text-error">
                  {t('remove')}
                </button>
              )}
            </li>
          ))}
        </ul>
      )}
      {canEdit && (
        <form className="grid max-w-xl gap-3" onSubmit={add}>
          <h3 className="font-semibold">{t('addLocation')}</h3>
          <label className="grid gap-1 text-sm">
            {t('city')}
            <select className={inputClass} value={draft.city_id} onChange={(e) => setDraft({ ...draft, city_id: e.target.value })}>
              {cities.map((city) => (
                <option key={city.id} value={city.id}>{city.label}</option>
              ))}
            </select>
          </label>
          <label className="grid gap-1 text-sm">
            {t('address')}
            <input className={inputClass} value={draft.address} onChange={(e) => setDraft({ ...draft, address: e.target.value })} />
          </label>
          <label className="grid gap-1 text-sm">
            {t('phone')}
            <input className={inputClass} value={draft.phone} onChange={(e) => setDraft({ ...draft, phone: e.target.value })} />
          </label>
          <label className="grid gap-1 text-sm">
            {t('hours')}
            <input className={inputClass} value={draft.hours} onChange={(e) => setDraft({ ...draft, hours: e.target.value })} />
          </label>
          <button type="submit" className="w-fit rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface">
            {t('addLocation')}
          </button>
        </form>
      )}
      {message && <p role="alert" className="text-sm text-error">{message}</p>}
    </section>
  );
}
