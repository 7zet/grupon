import nextConfig from '@dealo/eslint-config/next';
import { defineConfig, globalIgnores } from 'eslint/config';

export default defineConfig([nextConfig, globalIgnores(['public/mockServiceWorker.js'])]);
