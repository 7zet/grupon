import { defineConfig, globalIgnores } from 'eslint/config';
import nextCoreWebVitals from 'eslint-config-next/core-web-vitals';
import nextTypeScript from 'eslint-config-next/typescript';
import eslintConfigPrettier from 'eslint-config-prettier';

export const nextConfig = defineConfig([
  ...nextCoreWebVitals,
  ...nextTypeScript,
  eslintConfigPrettier,
  globalIgnores(['.next/**', 'coverage/**', 'next-env.d.ts']),
  {
    linterOptions: {
      reportUnusedDisableDirectives: 'error',
    },
  },
]);

export default nextConfig;
