'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

/** One guest cart line. Display fields are kept so the cart renders without refetching the deal. */
export type CartItem = {
  optionId: string;
  quantity: number;
  title: string;
  dealSlug: string;
  unitPrice: number;
  originalPrice: number;
};

type CartState = {
  items: CartItem[];
  add: (item: CartItem) => void;
  clear: () => void;
};

export const useCart = create<CartState>()(
  persist(
    (set) => ({
      items: [],
      add: (item) =>
        set((state) => ({
          items: [...state.items.filter((current) => current.optionId !== item.optionId), item],
        })),
      clear: () => set({ items: [] }),
    }),
    { name: 'grupon-cart' },
  ),
);
