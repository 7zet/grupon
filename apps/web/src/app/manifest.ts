import type { MetadataRoute } from 'next';

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: 'Grupon Scanner',
    short_name: 'Grupon Scan',
    description: 'Online voucher redemption scanner for Grupon merchants.',
    start_url: '/uz/merchant/redeem',
    display: 'standalone',
    background_color: '#ffffff',
    theme_color: '#0f766e',
    icons: [{ src: '/icons/scanner.svg', sizes: 'any', type: 'image/svg+xml', purpose: 'any' }],
  };
}
