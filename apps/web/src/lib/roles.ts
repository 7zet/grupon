/**
 * Which workspace each role works in, where a signed-in user lands, and which sections each role may open.
 * The UI uses these rules to show the right menu. The API enforces the same rules on every operation.
 */

export const BUYER = 'user';
export const OWNER = 'merchant_owner';
export const MANAGER = 'merchant_manager';
export const CASHIER = 'merchant_cashier';

export type WorkspaceKey = 'shop' | 'business' | 'kassa' | 'admin';

/** Where each role lands after sign-in and when it opens "/". Earlier entries win for multi-role users. */
const HOME_BY_ROLE: [string, string][] = [
  ['superadmin', '/admin'],
  ['admin', '/admin'],
  ['moderator', '/admin/moderation'],
  ['support', '/admin/finance'],
  ['finance', '/admin/finance'],
  [OWNER, '/merchant/dashboard'],
  [MANAGER, '/merchant/dashboard'],
  [CASHIER, '/merchant/redeem'],
];

/** Path inside the site (without the locale prefix). Buyers and anonymous visitors get the marketplace. */
export function roleHome(roles: readonly string[]): string {
  for (const [role, home] of HOME_BY_ROLE) {
    if (roles.includes(role)) return home;
  }
  return '/';
}

export function isStaff(roles: readonly string[]): boolean {
  return roles.some((role) => ['admin', 'superadmin', 'moderator', 'support', 'finance'].includes(role));
}

/** Every workspace the user may open. The buyer marketplace is always open. */
export function workspacesFor(roles: readonly string[]): WorkspaceKey[] {
  const out: WorkspaceKey[] = ['shop'];
  if (roles.includes(OWNER) || roles.includes(MANAGER)) out.push('business');
  else if (roles.includes(CASHIER)) out.push('kassa');
  if (isStaff(roles)) out.push('admin');
  return out;
}

/** Business sections and the merchant roles allowed to open each one. */
export const BUSINESS_SECTIONS = {
  dashboard: [OWNER, MANAGER],
  deals: [OWNER, MANAGER],
  createDeal: [OWNER, MANAGER],
  media: [OWNER, MANAGER],
  orders: [OWNER, MANAGER],
  customers: [OWNER, MANAGER],
  analytics: [OWNER, MANAGER],
  reviews: [OWNER, MANAGER],
  calculator: [OWNER, MANAGER],
  redeem: [OWNER, MANAGER, CASHIER],
  business: [OWNER],
  staff: [OWNER],
  payouts: [OWNER],
  settings: [OWNER],
} as const satisfies Record<string, readonly string[]>;

export type BusinessSection = keyof typeof BUSINESS_SECTIONS;

/** Admin sections and the staff roles allowed to open each one. */
export const ADMIN_SECTIONS = {
  overview: ['superadmin', 'admin'],
  moderation: ['moderator', 'superadmin', 'admin'],
  merchants: ['moderator', 'superadmin', 'admin'],
  fraud: ['moderator', 'superadmin', 'admin'],
  reports: ['moderator', 'superadmin', 'admin'],
  // Support works the disputes and refunds queue on the finance desk; it never approves payouts.
  finance: ['finance', 'support', 'superadmin', 'admin'],
  audit: ['superadmin', 'admin'],
} as const satisfies Record<string, readonly string[]>;

export type AdminSection = keyof typeof ADMIN_SECTIONS;

export function canOpenBusiness(roles: readonly string[], section: BusinessSection): boolean {
  return roles.some((role) => (BUSINESS_SECTIONS[section] as readonly string[]).includes(role));
}

export function canOpenAdmin(roles: readonly string[], section: AdminSection): boolean {
  return roles.some((role) => (ADMIN_SECTIONS[section] as readonly string[]).includes(role));
}

/** Admin sections a staff user may see in the console menu. */
export function adminSectionsFor(roles: readonly string[]): AdminSection[] {
  return (Object.keys(ADMIN_SECTIONS) as AdminSection[]).filter((section) => canOpenAdmin(roles, section));
}

/** Business sections for the desktop sidebar and the mobile tab bar. */
export function businessSectionsFor(roles: readonly string[]): BusinessSection[] {
  return (Object.keys(BUSINESS_SECTIONS) as BusinessSection[]).filter((section) => canOpenBusiness(roles, section));
}
