import type { NextConfig } from 'next';
import path from 'node:path';

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
    ],
    // Development serves media from localhost; production media is on a public host.
    dangerouslyAllowLocalIP: process.env.NODE_ENV !== 'production',
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
