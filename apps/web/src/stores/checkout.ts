'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

/** The one option a buyer is taking to checkout. Multi-item carts are not a product feature. */
export type CheckoutItem = {
  optionId: string;
  quantity: number;
  title: string;
  dealSlug: string;
  unitPrice: number;
  originalPrice: number;
};

type CheckoutState = {
  item: CheckoutItem | null;
  select: (item: CheckoutItem) => void;
  clear: () => void;
};

export const useCheckout = create<CheckoutState>()(
  persist(
    (set) => ({
      item: null,
      select: (item) => set({ item }),
      clear: () => set({ item: null }),
    }),
    { name: 'grupon-checkout' },
  ),
);
