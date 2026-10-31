import { describe, expect, it } from 'vitest';

import type { ApiProblem } from './index';

describe('generated API contract', () => {
  it('exposes RFC 9457 problems with a stable code', () => {
    const problem: ApiProblem = {
      type: 'https://api.local.test/problems/not-found',
      title: 'Not found',
      status: 404,
      code: 'not_found',
    };

    expect(problem.code).toBe('not_found');
  });
});
