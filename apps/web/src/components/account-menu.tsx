'use client';

import { LogOut, UserRound } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';

import { Link } from '@/i18n/navigation';
import { fetchMe, resetMe } from '@/lib/me';
import { maskPhone } from '@/lib/phone';

type User = { name: string; phone?: string | null; roles: string[] };
type Business = { display_name: string };
type Labels = {
  account: string; coupons: string; logout: string; buyer: string; owner: string; manager: string;
  cashier: string; admin: string; moderator: string; support: string; finance: string; superadmin: string;
};


function identity(user: User, business: Business | null, labels: Labels) {
  const role = user.roles.find((item) => item !== 'user') ?? 'user';
  const key = role.replace(/^merchant_/, '');
  const title = labels[key as keyof Labels] ?? labels.buyer;
  if (role.startsWith('merchant_')) return `${title} · ${business?.display_name ?? ''}`.replace(/ · $/, '');
  return role === 'user' ? labels.buyer : `${labels.admin} · ${title}`;
}

export function AccountMenu({ labels }: { labels: Labels }) {
  const [user, setUser] = useState<User | null>(null);
  const [business, setBusiness] = useState<Business | null>(null);
  const router = useRouter();

  useEffect(() => {
    fetchMe().then((value) => {
      setUser(value);
      if (value?.roles.some((role: string) => role.startsWith('merchant_'))) {
        fetch('/api/bff/merchant/business', { cache: 'no-store' }).then((response) => response.ok ? response.json() : null).then(setBusiness).catch(() => undefined);
      }
    }).catch(() => undefined);
  }, []);

  if (!user) return <Link className="hidden rounded-control-ui p-2 text-ink-muted hover:bg-page hover:text-brand sm:block" href="/login" prefetch={false} aria-label={labels.account}><UserRound /></Link>;

  const merchant = user.roles.some((role) => role.startsWith('merchant_'));
  const staff = user.roles.some((role) => ['admin', 'superadmin', 'moderator', 'finance', 'support'].includes(role));
  async function logout() {
    await fetch('/api/bff/auth/logout', { method: 'POST' });
    resetMe();
    router.replace('/login');
    router.refresh();
  }

  return <details className="relative hidden sm:block"><summary className="flex list-none items-center gap-1 rounded-control-ui p-2 text-ink-muted hover:bg-page hover:text-brand" aria-label={labels.account}><UserRound /><span data-testid="role-badge" className="max-w-40 truncate rounded-full bg-brand-tint px-2 py-1 text-label font-semibold text-brand">{identity(user, business, labels)}</span></summary><div className="absolute right-0 mt-2 w-64 rounded-card-ui border bg-surface p-3 shadow-floating-ui"><p className="font-semibold">{user.name}</p><p className="text-small text-ink-muted">{maskPhone(user.phone)}</p><p className="mt-2 rounded-full bg-brand-tint px-2 py-1 text-label font-semibold text-brand">{identity(user, business, labels)}</p><Link className="mt-3 block text-small" href="/account/coupons">{labels.coupons}</Link>{merchant && <Link className="mt-2 block text-small" href={user.roles.includes('merchant_cashier') ? '/merchant/redeem' : '/merchant/dashboard'}>{labels.owner}</Link>}{staff && <Link className="mt-2 block text-small" href="/admin/moderation">{labels.admin}</Link>}<button className="mt-3 flex w-full items-center gap-2 border-t pt-3 text-small text-ink-muted" onClick={logout}><LogOut size={16} />{labels.logout}</button></div></details>;
}
