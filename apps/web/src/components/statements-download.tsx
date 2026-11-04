'use client';

import { useTranslations } from 'next-intl';

export type StatementRow = {
  period: string;
  gross: string;
  commission: string;
  net: string;
  status: string;
  paid_on: string;
};

/**
 * Downloads the payout list as CSV. The file is built in the browser from rows the server already
 * rendered, so it carries the same figures the page shows. Values are plain text, quoted per CSV rules.
 */
export function StatementsDownload({ rows, filename }: { rows: StatementRow[]; filename: string }) {
  const t = useTranslations('MerchantPanel');

  function download() {
    const header = ['period', 'gross', 'commission', 'net', 'status', 'paid_on'];
    const lines = [header, ...rows.map((row) => [row.period, row.gross, row.commission, row.net, row.status, row.paid_on])];
    const csv = lines.map((cells) => cells.map((cell) => `"${cell.replace(/"/g, '""')}"`).join(',')).join('\n');
    const url = URL.createObjectURL(new Blob([`﻿${csv}`], { type: 'text/csv;charset=utf-8' }));
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    link.click();
    URL.revokeObjectURL(url);
  }

  return (
    <button
      type="button"
      onClick={download}
      disabled={rows.length === 0}
      className="justify-self-start rounded-control-ui border border-border px-5 py-3 font-semibold disabled:opacity-50"
    >
      {t('downloadCsv')}
    </button>
  );
}
