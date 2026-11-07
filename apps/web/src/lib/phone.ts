/** Hides the middle of a phone number for display: +998 90 ••• •• 67. Short values are returned as is. */
export function maskPhone(phone?: string | null): string {
  if (!phone) return '';
  const digits = phone.replace(/\D/g, '');
  if (digits.length < 9) return phone;
  const country = digits.length > 9 ? digits.slice(0, digits.length - 9) : '';
  const national = digits.slice(-9);
  return `${country ? `+${country} ` : ''}${national.slice(0, 2)} ••• •• ${national.slice(-2)}`;
}
