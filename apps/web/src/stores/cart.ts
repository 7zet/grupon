'use client';
import { create } from 'zustand';
import { persist } from 'zustand/middleware';
type CartItem = { optionId: string; quantity: number; isGift: boolean };
type CartState = { items: CartItem[]; add: (item: CartItem) => void };
export const useCart = create<CartState>()(
  persist(
    (set) => ({
      items: [],
      add: (item) =>
        set((state) => ({
          items: [...state.items.filter((current) => current.optionId !== item.optionId), item],
        })),
    }),
    { name: 'grupon-cart' },
  ),
);
