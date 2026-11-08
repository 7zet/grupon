import type { NextConfig } from 'next';
import path from 'node:path';

const apiURL = process.env.NEXT_PUBLIC_API_BASE_URL;
const apiMediaPattern = apiURL
  ? (() => {
      const url = new URL(apiURL);
      return {
        protocol: url.protocol.replace(':', '') as 'http' | 'https',
        hostname: url.hostname,
        ...(url.port ? { port: url.port } : {}),
        pathname: '/media/**',
      };
    })()
  : undefined;

const nextConfig: NextConfig = {
  reactStrictMode: true,
  allowedDevOrigins: ['127.0.0.1'],
  transpilePackages: ['@grupon/api-client', '@grupon/config', '@grupon/i18n', '@grupon/ui'],
  images: {
    remotePatterns: [
      { protocol: 'https', hostname: 'images.unsplash.com' },
      // Local API media (MEDIA_PUBLIC_BASE in development).
      { protocol: 'http', hostname: 'localhost', port: '8080', pathname: '/media/**' },
      { protocol: 'http', hostname: '127.0.0.1', port: '8080', pathname: '/media/**' },
      ...(apiMediaPattern ? [apiMediaPattern] : []),
    ],
    // The local demo serves media from localhost through the Next image optimizer. A real production
    // deployment (APP_ENV=production) must serve media from a public host instead.
    dangerouslyAllowLocalIP: process.env.APP_ENV !== 'production',
  },
  experimental: { useTypeScriptCli: true, workerThreads: true },
  typescript: { ignoreBuildErrors: true },
  turbopack: { resolveAlias: { 'next-intl/config': './src/i18n/request.ts' } },
  webpack(config) {
    config.resolve.alias['next-intl/config'] = path.resolve(process.cwd(), 'src/i18n/request.ts');
    return config;
  },
};
export default nextConfig;
