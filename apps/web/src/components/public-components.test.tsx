import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { NextIntlClientProvider } from 'next-intl';
import { describe, expect, it } from 'vitest';
import { NuqsTestingAdapter } from 'nuqs/adapters/testing';
import messages from '@grupon/i18n/messages/en';
import { deals } from '@/mocks/data';
import { DealCard } from './deal-card';
import { ListingControls } from './listing-controls';
import { PurchasePanel } from './purchase-panel';

function Wrapper({ children }: { children: React.ReactNode }) {
  return (
    <NextIntlClientProvider locale="en" messages={messages}>
      <NuqsTestingAdapter>{children}</NuqsTestingAdapter>
    </NextIntlClientProvider>
  );
}

describe('public marketplace components', () => {
  it('renders an accessible deal card with prices and one deal destination', () => {
    render(<DealCard deal={deals[0]!} locale="en" currency="UZS" />, { wrapper: Wrapper });
    expect(screen.getByRole('link', { name: new RegExp(deals[0]!.title.en) })).toHaveAttribute(
      'href',
      `/deal/${deals[0]!.slug}`,
    );
    expect(screen.getAllByText(/UZS/).length).toBeGreaterThanOrEqual(2);
  });
  it('updates purchase quantity without exceeding the selected option', async () => {
    const user = userEvent.setup();
    render(<PurchasePanel deal={deals[0]!} locale="en" currency="UZS" />, { wrapper: Wrapper });
    const plus = screen.getByRole('button', { name: '+' });
    await user.click(plus);
    expect(screen.getByText('2')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Buy now' })).toBeEnabled();
  });
  it('exposes listing filters and every required sort', () => {
    render(<ListingControls subcategories={[{ slug: 'massage', label: 'Massage' }]} />, {
      wrapper: Wrapper,
    });
    expect(screen.getByLabelText('Subcategory')).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'Biggest discount' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'Ending soon' })).toBeInTheDocument();
  });
});
