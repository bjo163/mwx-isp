import React, { useState } from 'react';
import {
  Alert, Box, Button, Chip, LinearProgress, Stack, Table, TableBody, TableCell,
  TableHead, TableRow, Typography,
} from '@mui/material';
import { useQuery } from '@tanstack/react-query';
import { apiRequest } from '../utils/apiClient';

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
  if (query.isLoading) return <LinearProgress sx={{ my: 1.5 }} aria-label="Loading impact summary" />;
  if (query.isError) {
    return <Alert severity="error">Impact data could not be loaded. Check the server connection and try again.</Alert>;
  }

  const data = query.data;
  if (!data) return null;

  const state = stateMeta[data.state] ?? stateMeta.unknown;
  const confidence = confidenceMeta[data.confidence] ?? confidenceMeta.unknown;
  const totalPages = Math.max(1, Math.ceil(data.pagination.total / data.pagination.page_size));

  return (
    <Stack spacing={1.5} sx={{ mt: 1 }}>
      <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap" alignItems="center">
        <Chip size="small" color={state.color} label={state.label} sx={{ fontWeight: 700 }} />
        <Chip size="small" color={confidence.color} variant="outlined" label={confidence.label} sx={{ fontWeight: 700 }} />
        <Chip size="small" variant="outlined" label={`${data.counts.customers} customers`} />
        <Chip size="small" variant="outlined" label={`${data.counts.subscriptions} subscriptions`} />
        <Chip size="small" variant="outlined" label={`${data.counts.online_sessions} live sessions`} />
        <Chip size="small" variant="outlined" color={data.counts.open_tickets > 0 ? 'warning' : 'default'} label={`${data.counts.open_tickets} open tickets`} />
      </Stack>

      <Typography variant="caption" color="text.secondary">{data.confidence_reason}</Typography>

      <Stack direction="row" spacing={2} useFlexGap flexWrap="wrap">
        <Typography variant="caption" color="text.secondary">
          Scope: {data.scope.type.toUpperCase()} {data.scope.code || data.scope.name}
          {data.scope.nas_id ? ` · NAS ${data.scope.nas_name || data.scope.nas_id}` : ''}
        </Typography>
        {data.monitoring.monitored || data.monitoring.target_id ? (
          <Typography variant="caption" color="text.secondary">
            Probe: {data.monitoring.status.toUpperCase()} · last check {formatTimestamp(data.monitoring.last_checked_at)} · stale after {data.monitoring.stale_after_seconds}s
          </Typography>
        ) : (
          <Typography variant="caption" color="text.secondary">No direct probe is linked to this scope.</Typography>
        )}
        {data.monitoring.open_incident && (
          <Typography variant="caption" color="error.main">
            Open incident since {formatTimestamp(data.monitoring.open_incident.started_at)}: {data.monitoring.open_incident.summary}
          </Typography>
        )}
      </Stack>

      {data.warnings.map((warning) => (
        <Alert key={warning} severity="warning" sx={{ py: 0.25 }}>{warning}</Alert>
      ))}

      {data.impacted.length === 0 ? (
        <Alert severity={data.confidence === 'unknown' ? 'warning' : 'info'}>
          No subscribers are mapped to this scope. An empty list is not proof that no customer is affected.
        </Alert>
      ) : (
        <Box sx={{ overflowX: 'auto' }}>
          <Table size="small" aria-label="Impacted subscribers">
            <TableHead>
              <TableRow>
                <TableCell>Customer</TableCell>
                <TableCell>ODP</TableCell>
                <TableCell>Subscription</TableCell>
                <TableCell>Username</TableCell>
                <TableCell>Session</TableCell>
                <TableCell>Last seen</TableCell>
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
                    <Typography variant="caption">{row.odp_code || '—'}{row.odp_port ? ` · port ${row.odp_port}` : ''}</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="caption">{row.subscription_no}</Typography>
                    <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>{row.subscription_status}</Typography>
                  </TableCell>
                  <TableCell><Typography variant="caption">{row.username || '—'}</Typography></TableCell>
                  <TableCell>
                    <Chip
                      size="small"
                      color={row.online ? 'success' : 'default'}
                      variant={row.online ? 'filled' : 'outlined'}
                      label={row.online ? 'Online' : 'No live session'}
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
          Online reflects active RADIUS sessions. Subscribers without a live session are not marked offline.
        </Typography>
        {data.pagination.total > 0 && (
          <Stack direction="row" spacing={1} alignItems="center">
            <Typography variant="caption" color="text.secondary">
              Page {data.pagination.page} of {totalPages} · {data.pagination.total} affected subscriptions
            </Typography>
            <Button size="small" disabled={page <= 1 || query.isFetching} onClick={() => setPage((current) => Math.max(1, current - 1))}>
              Previous
            </Button>
            <Button size="small" disabled={page >= totalPages || query.isFetching} onClick={() => setPage((current) => current + 1)}>
              Next
            </Button>
          </Stack>
        )}
      </Stack>
    </Stack>
  );
};
