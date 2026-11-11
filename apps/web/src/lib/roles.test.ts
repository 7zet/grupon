import { describe, expect, it } from 'vitest';

import {
  adminSectionsFor,
  businessSectionsFor,
  canOpenAdmin,
  canOpenBusiness,
  roleHome,
  workspacesFor,
} from './roles';

describe('roleHome', () => {
  it('sends each role to its own home', () => {
    expect(roleHome(['user'])).toBe('/');
    expect(roleHome([])).toBe('/');
    expect(roleHome(['merchant_owner'])).toBe('/merchant/dashboard');
    expect(roleHome(['merchant_manager'])).toBe('/merchant/dashboard');
    expect(roleHome(['merchant_cashier'])).toBe('/merchant/redeem');
    expect(roleHome(['moderator'])).toBe('/admin/moderation');
    expect(roleHome(['support'])).toBe('/admin/finance');
    expect(roleHome(['finance'])).toBe('/admin/finance');
    expect(roleHome(['superadmin'])).toBe('/admin');
  });

  it('prefers the staff role for a user with several roles', () => {
    expect(roleHome(['merchant_owner', 'superadmin'])).toBe('/admin');
  });
});

describe('workspacesFor', () => {
  it('gives buyers only the marketplace', () => {
    expect(workspacesFor(['user'])).toEqual(['shop']);
  });
  it('gives the cashier the kassa, not the business panel', () => {
    expect(workspacesFor(['merchant_cashier'])).toEqual(['shop', 'kassa']);
  });
  it('lets a staff user with a business role switch between workspaces', () => {
    expect(workspacesFor(['merchant_owner', 'moderator'])).toEqual(['shop', 'business', 'admin']);
  });
});

describe('business section access', () => {
  it('keeps payouts, staff and settings for the owner only', () => {
    expect(canOpenBusiness(['merchant_manager'], 'payouts')).toBe(false);
    expect(canOpenBusiness(['merchant_manager'], 'staff')).toBe(false);
    expect(canOpenBusiness(['merchant_manager'], 'settings')).toBe(false);
    expect(canOpenBusiness(['merchant_owner'], 'payouts')).toBe(true);
  });
  it('lets the cashier only redeem', () => {
    expect(businessSectionsFor(['merchant_cashier'])).toEqual(['redeem']);
    expect(canOpenBusiness(['merchant_cashier'], 'dashboard')).toBe(false);
  });
  it('gives a buyer no business section', () => {
    expect(businessSectionsFor(['user'])).toEqual([]);
  });
});

describe('admin section access', () => {
  it('limits each staff role to its own sections', () => {
    expect(adminSectionsFor(['moderator'])).toEqual(['moderation', 'merchants', 'fraud', 'reports']);
    expect(adminSectionsFor(['support'])).toEqual(['finance']);
    expect(adminSectionsFor(['finance'])).toEqual(['finance']);
    expect(canOpenAdmin(['finance'], 'moderation')).toBe(false);
  });
  it('gives the superadmin everything', () => {
    expect(adminSectionsFor(['superadmin'])).toHaveLength(7);
  });
});
