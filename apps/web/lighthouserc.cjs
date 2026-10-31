module.exports = {
  ci: {
    collect: {
      startServerCommand: 'pnpm --filter @grupon/web start',
      startServerReadyPattern: 'Ready',
      url: [
        'http://localhost:3000/uz',
        'http://localhost:3000/uz/tashkent/beauty-and-spas',
        'http://localhost:3000/uz/deal/beauty-and-spas-special-2',
      ],
      numberOfRuns: 3,
      settings: { preset: 'desktop' },
    },
    assert: {
      assertions: {
        'categories:performance': ['error', { minScore: 0.9 }],
        'categories:accessibility': ['error', { minScore: 0.95 }],
        'categories:best-practices': ['error', { minScore: 0.95 }],
        'categories:seo': ['error', { minScore: 0.95 }],
        'largest-contentful-paint': ['error', { maxNumericValue: 2500 }],
        'cumulative-layout-shift': ['error', { maxNumericValue: 0.1 }],
        'total-blocking-time': ['error', { maxNumericValue: 200 }],
      },
    },
  },
};
