export async function register() {
  if (process.env.NEXT_RUNTIME === 'nodejs' && process.env.USE_MSW === 'true') {
    const { mockServer } = await import('./mocks/server');
    mockServer.listen({ onUnhandledFrame: 'bypass' });
  }
}
