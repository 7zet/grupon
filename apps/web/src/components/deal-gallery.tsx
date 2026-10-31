'use client';
import type { components } from '@grupon/api-client';
import { localized, type Locale } from '@grupon/config';
import useEmblaCarousel from 'embla-carousel-react';
import Image from 'next/image';
import { useCallback } from 'react';
type Media = components['schemas']['DealMedia'];
export function DealGallery({ media, locale }: { media: Media[]; locale: Locale }) {
  const [ref, api] = useEmblaCarousel({ loop: true });
  const scroll = useCallback((index: number) => api?.scrollTo(index), [api]);
  return (
    <div>
      <div ref={ref} className="overflow-hidden rounded-panel-ui bg-brand-tint">
        <div className="flex">
          {media.map((image, index) => (
            <div className="relative h-28 min-w-0 flex-[0_0_100%] sm:h-96" key={image.id}>
              <Image
                src={image.url}
                alt={localized(image.alt, locale)}
                fill
                priority={index === 0}
                className="object-cover"
                sizes="(max-width: 1024px) 100vw, 60vw"
              />
            </div>
          ))}
        </div>
      </div>
      <div className="mt-3 hidden grid-cols-5 gap-2 sm:grid">
        {media.slice(0, 5).map((image, index) => (
          <button
            className="relative aspect-[4/3] overflow-hidden rounded-control-ui border"
            onClick={() => scroll(index)}
            key={image.id}
            type="button"
          >
            <Image
              src={image.url}
              alt={localized(image.alt, locale)}
              fill
              className="object-cover"
              sizes="10vw"
            />
          </button>
        ))}
      </div>
    </div>
  );
}
