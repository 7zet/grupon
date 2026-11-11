import type { MetadataRoute } from 'next';

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: 'Dealo Scanner',
    short_name: 'Dealo Scan',
    description: 'Online voucher redemption scanner for Dealo merchants.',
    start_url: '/uz/merchant/redeem',
    display: 'standalone',
    background_color: '#ffffff',
    theme_color: '#047857',
    icons: [{ src: '/icons/scanner.svg', sizes: 'any', type: 'image/svg+xml', purpose: 'any' }],
  };
}
