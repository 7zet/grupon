import { currentUser } from '@/lib/account-api';
import { type AdminSection, type BusinessSection, canOpenAdmin, canOpenBusiness } from '@/lib/roles';

import { NoAccess } from './no-access';

/**
 * Section guard for a merchant page. Returns the "No access" page when the signed-in role may not open
 * the section, otherwise null. The layout already redirected anonymous visitors to sign-in.
 */
export async function businessGate(section: BusinessSection) {
  const user = await currentUser();
  if (!user || canOpenBusiness(user.roles, section)) return null;
  return <NoAccess roles={user.roles} />;
}

/** Section guard for an admin page; see businessGate. */
export async function adminGate(section: AdminSection) {
  const user = await currentUser();
  if (!user || canOpenAdmin(user.roles, section)) return null;
  return <NoAccess roles={user.roles} />;
}
