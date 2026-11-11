import { brandConfig } from '@dealo/config';
import {
  BarChart3,
  Building2,
  Calculator,
  ClipboardList,
  Images,
  LayoutDashboard,
  MessageSquare,
  PlusCircle,
  QrCode,
  Settings,
  Tag,
  UserCog,
  Users,
  Wallet,
  type LucideIcon,
} from 'lucide-react';
import { getTranslations } from 'next-intl/server';
import type { ReactNode } from 'react';

import { Link } from '@/i18n/navigation';
import { type BusinessSection, businessSectionsFor, workspacesFor } from '@/lib/roles';

import { type NavItem, NavLinks } from './nav-links';
import { type WorkspaceLink, WorkspaceMenu } from './workspace-menu';

const HREF: Record<BusinessSection, string> = {
  dashboard: '/merchant/dashboard',
  deals: '/merchant/deals',
  createDeal: '/merchant/deals/new',
  media: '/merchant/media',
  orders: '/merchant/orders',
  customers: '/merchant/customers',
  analytics: '/merchant/analytics',
  reviews: '/merchant/reviews',
  calculator: '/merchant/calculator',
  redeem: '/merchant/redeem',
  business: '/merchant/business',
  staff: '/merchant/staff',
  payouts: '/merchant/payouts',
  settings: '/merchant/settings',
};

const ICON: Record<BusinessSection, LucideIcon> = {
  dashboard: LayoutDashboard,
  deals: Tag,
  createDeal: PlusCircle,
  media: Images,
  orders: ClipboardList,
  customers: Users,
  analytics: BarChart3,
  reviews: MessageSquare,
  calculator: Calculator,
  redeem: QrCode,
  business: Building2,
  staff: UserCog,
  payouts: Wallet,
  settings: Settings,
};

/** Sidebar groups. Redeem is not listed: it is the large button at the top of the sidebar. */
const GROUPS: { key: 'groupSales' | 'groupDeals' | 'groupBusiness'; sections: BusinessSection[] }[] = [
  { key: 'groupSales', sections: ['dashboard', 'orders', 'customers', 'analytics', 'reviews'] },
  { key: 'groupDeals', sections: ['deals', 'createDeal', 'media', 'calculator'] },
  { key: 'groupBusiness', sections: ['business', 'staff', 'payouts', 'settings'] },
];

/** Items on the phone tab bar. Everything else sits under "More". */
const TABS: BusinessSection[] = ['dashboard', 'deals', 'redeem', 'orders'];

/** "Dealo Business": the owner and manager workspace, with its own header, sidebar and phone tab bar. */
export async function BusinessShell({
  children,
  roles,
  name,
  businessName,
}: {
  children: ReactNode;
  roles: string[];
  name: string;
  businessName: string;
}) {
  const t = await getTranslations('Panel');
  const m = await getTranslations('MerchantPanel');
  const w = await getTranslations('Workspace');
  const identity = await getTranslations('Identity');
  const navigation = await getTranslations('Navigation');
  const sections = businessSectionsFor(roles);
  const labels: Record<BusinessSection, string> = {
    dashboard: t('navOverview'),
    deals: t('navDeals'),
    createDeal: t('navCreateDeal'),
    media: t('navMedia'),
    orders: m('navOrders'),
    customers: m('navCustomers'),
    analytics: m('navAnalytics'),
    reviews: m('navReviews'),
    calculator: m('navCalculator'),
    redeem: t('navRedeem'),
    business: t('navBusiness'),
    staff: m('navStaff'),
    payouts: m('navPayouts'),
    settings: m('navSettings'),
  };
  const item = (section: BusinessSection): NavItem => {
    const Icon = ICON[section];
    return { href: HREF[section], label: labels[section], icon: <Icon className="size-[18px] shrink-0" aria-hidden="true" /> };
  };
  const groups = GROUPS.map((group) => ({ key: group.key, items: group.sections.filter((s) => sections.includes(s)).map(item) })).filter(
    (group) => group.items.length > 0,
  );
  const canRedeem = sections.includes('redeem');
  const tabs = TABS.filter((section) => sections.includes(section)).map(item);
  const more = sections.filter((section) => !TABS.includes(section)).map(item);
  const role = roles.includes('merchant_owner') ? identity('owner') : identity('manager');
  const switchTo: WorkspaceLink[] = workspacesFor(roles)
    .filter((key) => key !== 'business')
    .map((key) => ({
      href: key === 'shop' ? '/tashkent' : key === 'admin' ? '/admin' : '/merchant/redeem',
      label:
        key === 'shop'
          ? w('viewMarketplace')
          : key === 'admin'
            ? w('admin', { brand: brandConfig.name })
            : w('kassa', { brand: brandConfig.name }),
    }));

  return (
    <div className="min-h-dvh bg-page">
      <header className="sticky top-0 z-40 border-b bg-surface">
        <div className="mx-auto flex h-16 max-w-6xl items-center gap-3 px-4 sm:px-6">
          <Link href="/merchant/dashboard" className="shrink-0 text-subtitle font-bold text-brand">
            {w('business', { brand: brandConfig.name })}
          </Link>
          <span className="hidden min-w-0 truncate text-small text-ink-muted md:inline">{businessName}</span>
          <div className="ml-auto">
            <WorkspaceMenu
              name={name}
              roleLabel={`${role} · ${businessName}`}
              switchTo={switchTo}
              labels={{
                account: navigation('account'),
                signOut: navigation('logout'),
                switchWorkspace: w('switchWorkspace'),
                viewMarketplace: w('viewMarketplace'),
              }}
            />
          </div>
        </div>
      </header>
      <div className="mx-auto grid max-w-6xl gap-6 px-4 py-6 sm:px-6 lg:grid-cols-[14rem_minmax(0,1fr)]">
        <aside className="hidden lg:block">
          <div className="sticky top-24 rounded-card-ui bg-brand-dark p-3 shadow-card">
            <p className="truncate px-3 pb-3 pt-1 text-label font-semibold uppercase tracking-widest text-surface/70">{businessName}</p>
            {canRedeem && (
              <Link
                href="/merchant/redeem"
                prefetch={false}
                className="mb-3 flex min-h-12 items-center justify-center gap-2 rounded-control-ui bg-surface px-3 font-semibold text-brand-dark transition-colors hover:bg-brand-50"
              >
                <QrCode className="size-5" aria-hidden="true" />
                {labels.redeem}
              </Link>
            )}
            <div className="grid gap-4">
              {groups.map((group) => (
                <div key={group.key}>
                  <p className="px-3 pb-1 text-label font-semibold uppercase tracking-wider text-surface/70">{w(group.key)}</p>
                  <NavLinks items={group.items} label={w(group.key)} variant="sidebar" tone="dark" />
                </div>
              ))}
            </div>
          </div>
        </aside>
        <main className="min-w-0 pb-28 lg:pb-0">{children}</main>
      </div>
      <nav aria-label={w('workspace')} className="fixed inset-x-0 bottom-0 z-40 border-t bg-surface pb-[env(safe-area-inset-bottom)] lg:hidden">
        <div className="flex items-stretch">
          <div className="grid flex-1">
            <NavLinks items={tabs} label={w('workspace')} variant="tabs" />
          </div>
          {more.length > 0 && (
            <details className="relative flex items-center px-2">
              <summary className="flex min-h-14 min-w-14 cursor-pointer list-none flex-col items-center justify-center gap-1 text-[12px] text-ink-muted">
                <span aria-hidden="true" className="text-lg leading-none">&#8943;</span>
                <span>{w('more')}</span>
              </summary>
              <div className="absolute bottom-full right-0 mb-2 grid w-60 gap-1 rounded-card-ui border bg-surface p-2 shadow-floating-ui">
                <NavLinks items={more} label={w('more')} variant="sidebar" />
              </div>
            </details>
          )}
        </div>
      </nav>
    </div>
  );
}
