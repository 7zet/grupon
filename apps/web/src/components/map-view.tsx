'use client';
import type { components } from '@dealo/api-client';
import { useEffect, useRef } from 'react';
type Location = components['schemas']['MerchantLocation'];
export function MapView({ locations }: { locations: Location[] }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    let map: import('leaflet').Map | undefined;
    let cancelled = false;
    if (!ref.current || !locations[0]) return;
    void Promise.all([import('leaflet'), import('leaflet/dist/leaflet.css')]).then(([L]) => {
      if (cancelled || !ref.current || !locations[0]) return;
      map = L.map(ref.current, { scrollWheelZoom: false }).setView(
        [locations[0].lat, locations[0].lng],
        13,
      );
      L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '&copy; OpenStreetMap',
      }).addTo(map);
      const brand = getComputedStyle(document.documentElement).getPropertyValue('--brand');
      locations.forEach((item) =>
        L.circleMarker([item.lat, item.lng], {
          color: brand,
          fillColor: brand,
          fillOpacity: 1,
          radius: 7,
        })
          .bindPopup(item.address)
          .addTo(map!),
      );
    });
    return () => {
      cancelled = true;
      map?.remove();
    };
  }, [locations]);
  return <div ref={ref} className="h-80 overflow-hidden rounded-panel-ui border" />;
}
