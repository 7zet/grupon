import type { components } from '@grupon/api-client';
import { getFormatter, getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { ReportActions } from '@/components/report-actions';
import { accountGet, accountGetOrNull, loadForVisitor } from '@/lib/account-api';

type ReportPage = components['schemas']['ReviewReportPage'];
type User = components['schemas']['User'];

const REVIEWERS = ['superadmin', 'admin', 'moderator'];

export default async function ReportsPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const t = await getTranslations('Reports');
  const format = await getFormatter();
  const user = await loadForVisitor(locale, `/${locale}/admin/reports`, () =>
    accountGetOrNull<User>('/me'),
  );
  if (!user?.roles.some((role) => REVIEWERS.includes(role))) notFound();
  const reports = await accountGet<ReportPage>('/admin/review-reports');

  return (
    <div className="grid grid-cols-[minmax(0,1fr)] gap-8">
      <h1 className="text-title font-bold">{t('title')}</h1>
      {reports.items.length === 0 ? (
        <p className="text-ink-muted">{t('empty')}</p>
      ) : (
        <ul className="grid gap-4">
          {reports.items.map((report) => (
            <li
              key={report.id}
              className="grid gap-3 rounded-card border border-border bg-surface p-4"
            >
              <p className="text-sm text-ink-muted">
                {t('reason')}: {report.reason}
              </p>
              <p className="text-sm">
                {format.dateTime(new Date(report.created_at), { dateStyle: 'medium' })}
              </p>
              <blockquote className="border-l-2 border-border pl-3 text-ink">
                <span aria-label={`${report.review.rating}/5`}>
                  {'★'.repeat(report.review.rating)}
                </span>{' '}
                {report.review.text}
              </blockquote>
              <ReportActions reportId={report.id} reviewId={report.review.id} />
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
