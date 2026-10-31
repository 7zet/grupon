/**
 * Accepts only same-site relative paths for post-login redirects. Protocol-relative and absolute
 * URLs fall back to the homepage so the login form cannot be used as an open redirect.
 */
export function safeNextPath(value: string | undefined): string {
  if (!value || !value.startsWith('/') || value.startsWith('//') || value.includes('\\')) {
    return '/';
  }
  return value;
}
