import { redirect } from 'next/navigation';

export default async function CartPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  redirect(`/${locale}/checkout`);
}
