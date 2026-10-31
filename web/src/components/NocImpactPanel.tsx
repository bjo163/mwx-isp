import React, { useState } from 'react';
import { useTranslate } from 'react-admin';
import {
  Alert, Box, Button, Chip, LinearProgress, Stack, Table, TableBody, TableCell,
  TableHead, TableRow, Typography,
} from '@mui/material';
import { useQuery } from '@tanstack/react-query';
import { apiRequest } from '../utils/apiClient';
import { translateNocImpactDiagnostic } from './nocImpactDiagnostics';

export interface NocImpactIncident {
  id: string;
  summary: string;
  started_at: string;
}

export interface NocImpactMonitoring {
  monitored: boolean;
  enabled: boolean;
  status: string;
  stale: boolean;
  stale_after_seconds: number;
  last_checked_at?: string;
  interval_seconds: number;
  target_id?: string;
  target_name?: string;
  open_incident?: NocImpactIncident;
}

export interface NocImpactSubscriber {
  customer_id: string;
  customer_no: string;
  customer_name: string;
  odp_code?: string;
  odp_port?: number;
  subscription_id: string;
  subscription_no: string;
  subscription_status: string;
  username?: string;
  online: boolean;
  last_seen_at?: string;
}

export interface NocImpactResult {
  scope: { type: string; id: string; name: string; code?: string; zone?: string; address?: string; kind?: string; target_id?: string; nas_id?: string; nas_name?: string };
  state: string;
  confidence: string;
  confidence_reason: string;
  monitoring: NocImpactMonitoring;
  counts: { customers: number; subscriptions: number; online_sessions: number; open_tickets: number };
  impacted: NocImpactSubscriber[];
  pagination: { page: number; page_size: number; total: number };
  warnings: string[];
  generated_at: string;
}

type ChipColor = 'success' | 'warning' | 'error' | 'info' | 'default';

const stateMeta: Record<string, { label: string; color: ChipColor }> = {
  confirmed_outage: { label: 'Confirmed outage', color: 'error' },
  last_known_down_stale: { label: 'Last known down (stale)', color: 'warning' },
  stale: { label: 'Monitoring stale', color: 'warning' },
  operational: { label: 'Operational', color: 'success' },
  not_monitored: { label: 'Not monitored', color: 'default' },
  unknown: { label: 'Unknown', color: 'default' },
};

const confidenceMeta: Record<string, { label: string; color: ChipColor }> = {
  confirmed: { label: 'Confirmed mapping', color: 'success' },
  inferred: { label: 'Inferred mapping', color: 'warning' },
  unknown: { label: 'Unknown mapping', color: 'default' },
};

const formatTimestamp = (raw?: string) => (raw ? new Date(raw).toLocaleString() : '—');

interface NocImpactPanelProps {
  targetId?: string | number;
  odpId?: string | number;
  nasId?: string | number;
  pageSize?: number;
}

export const NocImpactPanel: React.FC<NocImpactPanelProps> = ({ targetId, odpId, nasId, pageSize = 10 }) => {
  const translate = useTranslate();
  const [page, setPage] = useState(1);
  const selector = targetId !== undefined
    ? `target_id=${encodeURIComponent(String(targetId))}`
    : odpId !== undefined
      ? `odp_id=${encodeURIComponent(String(odpId))}`
      : nasId !== undefined
        ? `nas_id=${encodeURIComponent(String(nasId))}`
        : '';

  const query = useQuery({
    queryKey: ['network', 'impact', selector, page, pageSize],
    queryFn: () => apiRequest<NocImpactResult>(`/network/impact?${selector}&page=${page}&pageSize=${pageSize}`),
    enabled: selector !== '',
  });

  if (!selector) return null;
  if (query.isLoading) return <LinearProgress sx={{ my: 1.5 }} aria-label={translate('pages.noc.impact.loading')} />;
  if (query.isError) {
    return (
      <Alert
        severity="error"
        action={
          <Button size="small" color="inherit" onClick={() => void query.refetch()}>
            {translate('pages.noc.impact.retry')}
          </Button>
        }
      >
        {translate('pages.noc.impact.error')}
      </Alert>
    );
  }

  const data = query.data;
  if (!data) return null;

  const stateEntry = stateMeta[data.state];
  const state = stateEntry ?? stateMeta.unknown;
  const stateKey = stateEntry ? data.state : 'unknown';
  const confidenceEntry = confidenceMeta[data.confidence];
  const confidence = confidenceEntry ?? confidenceMeta.unknown;
  const confidenceKey = confidenceEntry ? data.confidence : 'unknown';
  const totalPages = Math.max(1, Math.ceil(data.pagination.total / data.pagination.page_size));

  return (
    <Stack spacing={1.5} sx={{ mt: 1 }} data-testid="noc-impact-panel">
      <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap" alignItems="center">
        <Chip size="small" color={state.color} label={translate(`pages.noc.impact.state.${stateKey}`, { _: state.label })} sx={{ fontWeight: 700 }} />
        <Chip size="small" color={confidence.color} variant="outlined" label={translate(`pages.noc.impact.confidence.${confidenceKey}`, { _: confidence.label })} sx={{ fontWeight: 700 }} data-testid="noc-impact-confidence" />
        {data.monitoring.stale && (
          <Chip size="small" color="warning" variant="outlined" label={translate('pages.noc.impact.state.stale')} data-testid="noc-impact-stale" />
        )}
        <Chip size="small" variant="outlined" label={translate('pages.noc.impact.customers_count', { count: data.counts.customers })} />
        <Chip size="small" variant="outlined" label={translate('pages.noc.impact.subscriptions_count', { count: data.counts.subscriptions })} />
        <Chip size="small" variant="outlined" label={translate('pages.noc.impact.sessions_count', { count: data.counts.online_sessions })} />
        <Chip size="small" variant="outlined" color={data.counts.open_tickets > 0 ? 'warning' : 'default'} label={translate('pages.noc.impact.tickets_count', { count: data.counts.open_tickets })} />
      </Stack>

      <Typography variant="caption" color="text.secondary">{translateNocImpactDiagnostic(translate, data.confidence_reason)}</Typography>

      <Stack direction="row" spacing={2} useFlexGap flexWrap="wrap">
        <Typography variant="caption" color="text.secondary">
          {translate('pages.noc.impact.scope', { type: data.scope.type.toUpperCase(), name: data.scope.code || data.scope.name })}
          {data.scope.nas_id ? ` · ${translate('pages.noc.impact.scope_nas', { name: data.scope.nas_name || data.scope.nas_id })}` : ''}
        </Typography>
        {data.monitoring.monitored || data.monitoring.target_id ? (
          <Typography variant="caption" color="text.secondary">
            {translate('pages.noc.impact.probe', {
              status: data.monitoring.status.toUpperCase(),
              last_check: formatTimestamp(data.monitoring.last_checked_at),
              seconds: data.monitoring.stale_after_seconds,
            })}
          </Typography>
        ) : (
          <Typography variant="caption" color="text.secondary">{translate('pages.noc.impact.no_probe')}</Typography>
        )}
        {data.monitoring.open_incident && (
          <Typography variant="caption" color="error.main">
            {translate('pages.noc.impact.open_incident', {
              since: formatTimestamp(data.monitoring.open_incident.started_at),
              summary: data.monitoring.open_incident.summary,
            })}
          </Typography>
        )}
      </Stack>

      {data.warnings.map((warning) => (
        <Alert key={warning} severity="warning" sx={{ py: 0.25 }}>{translateNocImpactDiagnostic(translate, warning)}</Alert>
      ))}

      {data.impacted.length === 0 ? (
        <Alert severity={data.confidence === 'unknown' ? 'warning' : 'info'}>
          {translate('pages.noc.impact.empty')}
        </Alert>
      ) : (
        <Box sx={{ overflowX: 'auto' }}>
          <Table size="small" aria-label={translate('pages.noc.impact.table_aria')}>
            <TableHead>
              <TableRow>
                <TableCell>{translate('pages.noc.impact.column.customer')}</TableCell>
                <TableCell>{translate('pages.noc.impact.column.odp')}</TableCell>
                <TableCell>{translate('pages.noc.impact.column.subscription')}</TableCell>
                <TableCell>{translate('pages.noc.impact.column.username')}</TableCell>
                <TableCell>{translate('pages.noc.impact.column.session')}</TableCell>
                <TableCell>{translate('pages.noc.impact.column.last_seen')}</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {data.impacted.map((row) => (
                <TableRow key={row.subscription_id} hover>
                  <TableCell>
                    <Typography variant="body2" fontWeight={600}>{row.customer_name}</Typography>
                    <Typography variant="caption" color="text.secondary">{row.customer_no}</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="caption">{row.odp_code || '—'}{row.odp_port ? ` · ${translate('pages.noc.impact.port', { port: row.odp_port })}` : ''}</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="caption">{row.subscription_no}</Typography>
                    <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>{translate(`pages.noc.impact.status.${row.subscription_status}`, { _: row.subscription_status })}</Typography>
                  </TableCell>
                  <TableCell><Typography variant="caption">{row.username || '—'}</Typography></TableCell>
                  <TableCell>
                    <Chip
                      size="small"
                      color={row.online ? 'success' : 'default'}
                      variant={row.online ? 'filled' : 'outlined'}
                      label={row.online ? translate('pages.noc.impact.online') : translate('pages.noc.impact.no_live_session')}
                      sx={{ fontSize: '0.68rem', height: 20 }}
                    />
                  </TableCell>
                  <TableCell><Typography variant="caption">{formatTimestamp(row.last_seen_at)}</Typography></TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Box>
      )}

      <Stack direction="row" justifyContent="space-between" alignItems="center" flexWrap="wrap" gap={1}>
        <Typography variant="caption" color="text.secondary">
          {translate('pages.noc.impact.footer_note')}
        </Typography>
        {data.pagination.total > 0 && (
          <Stack direction="row" spacing={1} alignItems="center">
            <Typography variant="caption" color="text.secondary">
              {`${translate('pages.noc.impact.page_of', { page: data.pagination.page, total: totalPages })} · ${translate('pages.noc.impact.affected_subscriptions', { count: data.pagination.total })}`}
            </Typography>
            <Button size="small" disabled={page <= 1 || query.isFetching} onClick={() => setPage((current) => Math.max(1, current - 1))}>
              {translate('pages.noc.impact.previous')}
            </Button>
            <Button size="small" disabled={page >= totalPages || query.isFetching} onClick={() => setPage((current) => current + 1)}>
              {translate('pages.noc.impact.next')}
            </Button>
          </Stack>
        )}
      </Stack>
    </Stack>
  );
};
