/** One shared, short-lived request for the signed-in user, so the header components do not each fetch it. */
export type Me = { name: string; phone: string; roles: string[] };

let pending: { at: number; promise: Promise<Me | null> } | null = null;

export function fetchMe(): Promise<Me | null> {
  if (pending && Date.now() - pending.at < 5000) return pending.promise;
  const promise = fetch('/api/bff/me', { cache: 'no-store' })
    .then((response) => (response.ok ? (response.json() as Promise<Me>) : null))
    .catch(() => null);
  pending = { at: Date.now(), promise };
  // A signed-out answer must not linger: the visitor may sign in a moment later.
  void promise.then((value) => {
    if (!value) pending = null;
  });
  return promise;
}

/** Forget the cached user, for example after signing out. */
export function resetMe(): void {
  pending = null;
}
