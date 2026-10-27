import React, { useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { useRecordContext, useTranslate } from 'react-admin';
import {
  Alert, Box, Button, Chip, Divider, LinearProgress, Stack, TextField, Typography,
} from '@mui/material';
import {
  DescriptionOutlined as InvoiceIcon,
  PaymentsOutlined as PaymentIcon,
  ReceiptLongOutlined as BillingIcon,
  SupportAgentOutlined as TicketIcon,
  WifiOutlined as SessionIcon,
} from '@mui/icons-material';
import { useQuery } from '@tanstack/react-query';
import { apiRequest } from '../utils/apiClient';

export interface CustomerTimelineEvent {
  event_id: string;
  source: string;
  type: string;
  occurred_at: string;
  tenant_id?: string;
  customer_id: string;
  subscription_id?: string;
  summary: string;
  source_path: string;
  severity: string;
}

export interface CustomerTimelineResult {
  customer_id: string;
  from: string;
  to: string;
  sources: string[];
  events: CustomerTimelineEvent[];
  pagination: { page: number; page_size: number; total: number };
}

type ChipColor = 'success' | 'warning' | 'error' | 'info' | 'default';

const SOURCE_KEYS = ['billing', 'invoice', 'payment', 'session', 'ticket'] as const;
type SourceKey = typeof SOURCE_KEYS[number];

const sourceMeta: Record<SourceKey, { label: string; icon: React.ReactElement }> = {
  billing: { label: 'Billing', icon: <BillingIcon fontSize="small" /> },
  invoice: { label: 'Invoice', icon: <InvoiceIcon fontSize="small" /> },
  payment: { label: 'Payment', icon: <PaymentIcon fontSize="small" /> },
  session: { label: 'Session', icon: <SessionIcon fontSize="small" /> },
  ticket: { label: 'Ticket', icon: <TicketIcon fontSize="small" /> },
};

const severityMeta: Record<string, { label: string; color: ChipColor }> = {
  success: { label: 'Success', color: 'success' },
  warning: { label: 'Attention', color: 'warning' },
  error: { label: 'Critical', color: 'error' },
  info: { label: 'Info', color: 'default' },
};

const relativeFormatter = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });
const relativeUnits: Array<[Intl.RelativeTimeFormatUnit, number]> = [
  ['year', 31536000], ['month', 2592000], ['week', 604800],
  ['day', 86400], ['hour', 3600], ['minute', 60],
];

const formatRelative = (raw: string) => {
  const timestamp = new Date(raw).getTime();
  if (Number.isNaN(timestamp)) return '';
  const diffSeconds = Math.round((timestamp - Date.now()) / 1000);
  for (const [unit, seconds] of relativeUnits) {
    if (Math.abs(diffSeconds) >= seconds) {
      return relativeFormatter.format(Math.round(diffSeconds / seconds), unit);
    }
  }
  return relativeFormatter.format(diffSeconds, 'second');
};

const formatAbsolute = (raw: string) => {
  const timestamp = new Date(raw);
  return Number.isNaN(timestamp.getTime()) ? raw : timestamp.toLocaleString();
};

interface CustomerTimelineProps {
  customerId?: string | number;
  pageSize?: number;
}

export const CustomerTimeline: React.FC<CustomerTimelineProps> = ({ customerId: providedCustomerId, pageSize = 15 }) => {
  const translate = useTranslate();
  const record = useRecordContext();
  const customerId = providedCustomerId ?? record?.id;
  const [selectedSources, setSelectedSources] = useState<SourceKey[]>([...SOURCE_KEYS]);
  const [fromDate, setFromDate] = useState('');
  const [toDate, setToDate] = useState('');
  const [page, setPage] = useState(1);

  const toggleSource = (source: SourceKey) => {
    setPage(1);
    setSelectedSources((current) => {
      if (current.includes(source)) {
        return current.length === 1 ? current : current.filter((item) => item !== source);
      }
      return SOURCE_KEYS.filter((item) => item === source || current.includes(item));
    });
  };

  const query = useQuery({
    queryKey: ['isp', 'customer-timeline', customerId, selectedSources.join(','), fromDate, toDate, page, pageSize],
    queryFn: () => {
      const params = new URLSearchParams({ page: String(page), pageSize: String(pageSize) });
      if (selectedSources.length < SOURCE_KEYS.length) params.set('sources', selectedSources.join(','));
      if (fromDate) params.set('from', fromDate);
      if (toDate) params.set('to', toDate);
      return apiRequest<CustomerTimelineResult>(`/isp/customers/${encodeURIComponent(String(customerId))}/timeline?${params.toString()}`);
    },
    enabled: Boolean(customerId),
  });

  if (!customerId) return null;
  if (query.isLoading) return <LinearProgress sx={{ my: 1.5 }} aria-label={translate('pages.noc.timeline.loading')} />;
  if (query.isError) {
    return (
      <Alert
        severity="error"
        action={<Button size="small" onClick={() => void query.refetch()}>{translate('pages.noc.timeline.retry')}</Button>}
      >
        {translate('pages.noc.timeline.error')}
      </Alert>
    );
  }

  const data = query.data;
  if (!data) return null;
  const totalPages = Math.max(1, Math.ceil(data.pagination.total / data.pagination.page_size));

  return (
    <Stack spacing={1.5}>
      <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.5} alignItems={{ md: 'center' }} justifyContent="space-between">
        <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap" role="group" aria-label={translate('pages.noc.timeline.filter_aria')}>
          {SOURCE_KEYS.map((source) => (
            <Chip
              key={source}
              size="small"
              icon={sourceMeta[source].icon}
              label={translate(`pages.noc.timeline.source.${source}`, { _: sourceMeta[source].label })}
              onClick={() => toggleSource(source)}
              color={selectedSources.includes(source) ? 'primary' : 'default'}
              variant={selectedSources.includes(source) ? 'filled' : 'outlined'}
              aria-pressed={selectedSources.includes(source)}
              sx={{ fontWeight: 700 }}
            />
          ))}
        </Stack>
        <Stack direction="row" spacing={1} alignItems="center">
          <TextField
            size="small"
            type="date"
            label={translate('pages.noc.timeline.from')}
            value={fromDate}
            onChange={(event) => { setPage(1); setFromDate(event.target.value); }}
            InputLabelProps={{ shrink: true }}
            inputProps={{ 'aria-label': translate('pages.noc.timeline.start_date_aria') }}
          />
          <TextField
            size="small"
            type="date"
            label={translate('pages.noc.timeline.to')}
            value={toDate}
            onChange={(event) => { setPage(1); setToDate(event.target.value); }}
            InputLabelProps={{ shrink: true }}
            inputProps={{ 'aria-label': translate('pages.noc.timeline.end_date_aria') }}
          />
        </Stack>
      </Stack>

      <Typography variant="caption" color="text.secondary">
        {`${translate('pages.noc.timeline.event_count', { count: data.pagination.total })} · ${translate('pages.noc.timeline.events_note')}`}
      </Typography>

      {data.events.length === 0 ? (
        <Alert severity="info">{translate('pages.noc.timeline.empty')}</Alert>
      ) : (
        <Box role="list" aria-label={translate('pages.noc.timeline.list_aria')}>
          {data.events.map((event, index) => {
            const source = sourceMeta[event.source as SourceKey] ?? { label: event.source, icon: <BillingIcon fontSize="small" /> };
            const severity = severityMeta[event.severity] ?? severityMeta.info;
            const severityKey = severityMeta[event.severity] ? event.severity : 'info';
            return (
              <Box key={event.event_id} role="listitem">
                {index > 0 && <Divider component="div" />}
                <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} alignItems={{ sm: 'center' }} sx={{ py: 1.25 }}>
                  <Stack direction="row" spacing={1} alignItems="center" sx={{ minWidth: { sm: 280 } }}>
                    <Box sx={{ display: 'flex', color: 'text.secondary' }} aria-hidden="true">{source.icon}</Box>
                    <Box>
                      <Typography variant="body2" fontWeight={700}>{event.summary}</Typography>
                      <Stack direction="row" spacing={0.75} useFlexGap flexWrap="wrap" sx={{ mt: 0.5 }}>
                        <Chip size="small" variant="outlined" label={translate(`pages.noc.timeline.source.${event.source}`, { _: source.label })} sx={{ height: 20, fontSize: '0.68rem' }} />
                        <Chip size="small" color={severity.color} variant="outlined" label={translate(`pages.noc.timeline.severity.${severityKey}`, { _: severity.label })} sx={{ height: 20, fontSize: '0.68rem' }} />
                        <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>{translate(`pages.noc.timeline.event_type.${event.type}`, { _: event.type })}</Typography>
                      </Stack>
                    </Box>
                  </Stack>
                  <Box sx={{ flex: 1 }} />
                  <Stack direction="row" spacing={1} alignItems="center">
                    <Typography variant="caption" color="text.secondary" component="time" dateTime={event.occurred_at} title={new Date(event.occurred_at).toISOString()}>
                      {formatRelative(event.occurred_at)} · {formatAbsolute(event.occurred_at)}
                    </Typography>
                    <Button component={RouterLink} to={event.source_path} size="small" variant="text" sx={{ fontWeight: 700, whiteSpace: 'nowrap' }}>
                      {translate('pages.noc.timeline.open')}
                    </Button>
                  </Stack>
                </Stack>
              </Box>
            );
          })}
        </Box>
      )}

      <Stack direction="row" justifyContent="space-between" alignItems="center" flexWrap="wrap" gap={1}>
        <Typography variant="caption" color="text.secondary">
          {translate('pages.noc.timeline.footer_note')}
        </Typography>
        {data.pagination.total > 0 && (
          <Stack direction="row" spacing={1} alignItems="center">
            <Typography variant="caption" color="text.secondary">
              {`${translate('pages.noc.timeline.page_of', { page: data.pagination.page, total: totalPages })} · ${translate('pages.noc.timeline.event_count', { count: data.pagination.total })}`}
            </Typography>
            <Button size="small" disabled={page <= 1 || query.isFetching} onClick={() => setPage((current) => Math.max(1, current - 1))}>
              {translate('pages.noc.timeline.previous')}
            </Button>
            <Button size="small" disabled={page >= totalPages || query.isFetching} onClick={() => setPage((current) => current + 1)}>
              {translate('pages.noc.timeline.next')}
            </Button>
          </Stack>
        )}
      </Stack>
    </Stack>
  );
};
