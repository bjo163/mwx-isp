import React, { useState } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  InputLabel,
  LinearProgress,
  MenuItem,
  Select,
  Stack,
  Tab,
  Tabs,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material';
import {
  AccountTree,
  Add,
  History,
  NetworkCheck,
  PlayArrow,
  Search,
} from '@mui/icons-material';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNotify, useTranslate } from 'react-admin';
import { apiRequest, extractData } from '../utils/apiClient';
import { ConsoleBox, KpiStrip, KpiTile, Mono, PageHeader, Panel, QueryFeedback, StatusChip } from '../components/Enterprise';

interface IPAMPool {
  id: string;
  name: string;
  cidr: string;
  ip_version: number;
  pool_type: string;
  gateway: string;
  dns_primary: string;
  dns_secondary: string;
  total_ips: number;
  capacity_label?: string;
  used_ips: number;
  description: string;
  created_at: string;
}

interface AuditRecord {
  type: string;
  username: string;
  acct_session_id: string;
  nas_addr: string;
  framed_ip: string;
  mac_addr: string;
  start_time: string;
  stop_time?: string;
  session_time?: number;
  status: string;
}

interface AuditResponse {
  queried_ip: string;
  queried_time: string;
  total_found: number;
  results: AuditRecord[];
}

interface PingResponse {
  host: string;
  target_addr?: string;
  probe_method?: string;
  port?: string;
  sent: number;
  received: number;
  loss_percent: number;
  min_rtt_ms: number;
  avg_rtt_ms: number;
  max_rtt_ms: number;
  details: Array<{ seq: number; rtt_ms: number; success: boolean; error?: string }>;
}

export const IPAMPage: React.FC = () => {
  const notify = useNotify();
  const translate = useTranslate();
  const queryClient = useQueryClient();

  const [tab, setTab] = useState<number>(0);
  const [createOpen, setCreateOpen] = useState<boolean>(false);

  // Pool form
  const [poolForm, setPoolForm] = useState({
    name: 'PPPoE Client Subnet 1',
    cidr: '100.64.0.0/22',
    pool_type: 'cgnat',
    gateway: '100.64.0.1',
    dns_primary: '1.1.1.1',
    dns_secondary: '8.8.8.8',
    description: 'CGNAT allocation pool for residential subscribers',
  });

  // Audit state
  const [auditIP, setAuditIP] = useState<string>('100.64.0.15');
  const [auditTime, setAuditTime] = useState<string>(new Date().toISOString().slice(0, 16));

  // Ping state
  const [pingHost, setPingHost] = useState<string>('8.8.8.8');
  const [pingCount, setPingCount] = useState<number>(4);

  const poolsQuery = useQuery({
    queryKey: ['network', 'ipam', 'pools'],
    queryFn: async () => {
      const res = await apiRequest<unknown>('/network/ipam/pools');
      return extractData<IPAMPool[]>(res) ?? [];
    },
  });

  const createPoolMutation = useMutation({
    mutationFn: () =>
      apiRequest('/network/ipam/pools', {
        method: 'POST',
        body: JSON.stringify(poolForm),
      }),
    onSuccess: () => {
      notify(translate('pages.ipam.notify.pool_created'), { type: 'success' });
      setCreateOpen(false);
      void queryClient.invalidateQueries({ queryKey: ['network', 'ipam', 'pools'] });
    },
    onError: (err: any) =>
      notify(err.message || translate('pages.ipam.notify.pool_create_failed'), { type: 'error' }),
  });

  const auditMutation = useMutation({
    mutationFn: async () => {
      const params = new URLSearchParams();
      params.set('ip', auditIP.trim());
      if (auditTime) params.set('at', new Date(auditTime).toISOString());
      const res = await apiRequest<unknown>(`/network/ipam/audit?${params.toString()}`);
      return extractData<AuditResponse>(res);
    },
  });

  const pingMutation = useMutation({
    mutationFn: async () => {
      const res = await apiRequest<unknown>('/network/diagnostics/ping', {
        method: 'POST',
        body: JSON.stringify({ host: pingHost.trim(), count: Number(pingCount) }),
      });
      return extractData<PingResponse>(res);
    },
  });

  const pools = poolsQuery.data ?? [];
  const auditResults = auditMutation.data?.results ?? [];
  const pingResult = pingMutation.data;
  const totalIps = pools.reduce((acc, p) => acc + (p.total_ips || 0), 0);
  const usedIps = pools.reduce((acc, p) => acc + (p.used_ips || 0), 0);
  const overallUtil = totalIps > 0 ? Math.round((usedIps / totalIps) * 100) : 0;
  const criticalPools = pools.filter((p) => {
    const tot = p.total_ips || 1;
    return ((p.used_ips || 0) / tot) >= 0.85;
  }).length;

  return (
    <Box sx={{ p: { xs: 1.5, md: 2.5 }, maxWidth: 1600, mx: 'auto' }}>
      <PageHeader
        section={translate('pages.ipam.section')}
        title={translate('pages.ipam.title')}
        subtitle={translate('pages.ipam.subtitle')}
        actions={
          tab === 0 ? (
            <Button variant="contained" startIcon={<Add />} onClick={() => setCreateOpen(true)}>
              {translate('pages.ipam.add_pool')}
            </Button>
          ) : undefined
        }
      />

      <KpiStrip>
        <KpiTile label={translate('pages.ipam.kpi.pools')} value={pools.length} icon={<AccountTree fontSize="small" />} />
        <KpiTile label={translate('pages.ipam.kpi.total_ips')} value={totalIps.toLocaleString()} tone="info" hint={translate('pages.ipam.kpi.total_ips_hint')} />
        <KpiTile label={translate('pages.ipam.kpi.used')} value={usedIps.toLocaleString()} tone="primary" hint={translate('pages.ipam.kpi.used_hint')} />
        <KpiTile
          label={translate('pages.ipam.kpi.utilization')}
          value={`${overallUtil}%`}
          tone={overallUtil > 85 ? 'error' : overallUtil > 65 ? 'warning' : 'success'}
          hint={translate('pages.ipam.kpi.utilization_hint')}
        />
        <KpiTile
          label={translate('pages.ipam.kpi.near_exhaustion')}
          value={criticalPools}
          tone={criticalPools > 0 ? 'error' : 'success'}
          hint={translate('pages.ipam.kpi.near_exhaustion_hint')}
        />
        <KpiTile label={translate('pages.ipam.kpi.families')} value={`${pools.filter((p) => p.ip_version === 4).length} / ${pools.filter((p) => p.ip_version === 6).length}`} tone="secondary" hint={translate('pages.ipam.kpi.families_hint')} />
      </KpiStrip>

      <Tabs value={tab} onChange={(_, val) => setTab(val)} sx={{ mb: 2 }}>
        <Tab icon={<AccountTree />} iconPosition="start" label={translate('pages.ipam.tabs.pools')} />
        <Tab icon={<History />} iconPosition="start" label={translate('pages.ipam.tabs.audit')} />
        <Tab icon={<NetworkCheck />} iconPosition="start" label={translate('pages.ipam.tabs.ping')} />
      </Tabs>

      {/* Tab 0: Subnet Pools */}
      {tab === 0 && (
        <QueryFeedback
          loading={poolsQuery.isLoading}
          error={poolsQuery.error}
          isEmpty={pools.length === 0}
          onRetry={() => void poolsQuery.refetch()}
          emptyIcon={<AccountTree sx={{ fontSize: 48, opacity: 0.4 }} />}
          emptyTitle={translate('pages.ipam.empty.title')}
          emptyHint={translate('pages.ipam.empty.hint')}
        >
          <Box
            sx={{
              display: 'grid',
              gridTemplateColumns: { xs: '1fr', md: 'repeat(2, 1fr)', lg: 'repeat(3, 1fr)' },
              gap: 2,
            }}
          >
            {pools.map((p) => {
              const used = p.used_ips || 0;
              const hasExactTotal = p.total_ips > 0;
              const percent = hasExactTotal ? Math.min(100, Math.round((used / p.total_ips) * 100)) : undefined;
              return (
                <Card key={p.id} variant="outlined">
                  <CardContent sx={{ p: 2 }}>
                    <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
                      <Typography variant="subtitle1" fontWeight={700}>
                        {p.name}
                      </Typography>
                      <StatusChip value={p.pool_type} />
                    </Stack>

                    <Typography variant="body2" sx={{ color: 'primary.main', mb: 1.5 }}>
                      <Mono>{p.cidr}</Mono>
                    </Typography>

                    <Box sx={{ mb: 1.5 }}>
                      <Stack direction="row" justifyContent="space-between" sx={{ mb: 0.5 }}>
                        <Typography variant="caption" color="text.secondary">
                          {translate('pages.ipam.pool.utilization')}
                        </Typography>
                        <Typography variant="caption" fontWeight={700}>
                          {percent !== undefined
                            ? translate('pages.ipam.pool.usage_exact', {
                                used,
                                total: p.total_ips,
                                percent,
                              })
                            : translate('pages.ipam.pool.usage_approx', {
                                used,
                                capacity:
                                  p.capacity_label ||
                                  translate('pages.ipam.pool.capacity_unknown'),
                              })}
                        </Typography>
                      </Stack>
                      {percent !== undefined && (
                        <LinearProgress
                          variant="determinate"
                          value={percent}
                          color={percent > 85 ? 'error' : percent > 65 ? 'warning' : 'primary'}
                          sx={{ height: 6, borderRadius: 1 }}
                        />
                      )}
                    </Box>

                    <Divider sx={{ my: 1 }} />

                    <Stack spacing={0.5} sx={{ fontSize: '0.75rem', color: 'text.secondary' }}>
                      <div>{translate('pages.ipam.pool.gateway')} <strong>{p.gateway || '—'}</strong></div>
                      <div>{translate('pages.ipam.pool.dns')} <strong>{p.dns_primary || '1.1.1.1'}</strong> {p.dns_secondary && `, ${p.dns_secondary}`}</div>
                      {p.description && <div>{translate('pages.ipam.pool.note')} {p.description}</div>}
                    </Stack>
                  </CardContent>
                </Card>
              );
            })}
          </Box>
        </QueryFeedback>
      )}

      {/* Tab 1: Reverse IP Audit Trail */}
      {tab === 1 && (
        <Stack spacing={2.5}>
          <Card variant="outlined">
            <CardContent sx={{ p: 2 }}>
              <Typography variant="subtitle1" fontWeight={800} sx={{ mb: 0.5 }}>
                {translate('pages.ipam.audit.title')}
              </Typography>
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 2 }}>
                {translate('pages.ipam.audit.subtitle')}
              </Typography>

              <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems="center">
                <TextField
                  size="small"
                  label={translate('pages.ipam.audit.ip_label')}
                  placeholder={translate('pages.ipam.audit.ip_placeholder')}
                  value={auditIP}
                  onChange={(e) => setAuditIP(e.target.value)}
                  sx={{ minWidth: 220 }}
                />
                <TextField
                  size="small"
                  type="datetime-local"
                  label={translate('pages.ipam.audit.time_label')}
                  value={auditTime}
                  onChange={(e) => setAuditTime(e.target.value)}
                  InputLabelProps={{ shrink: true }}
                  sx={{ minWidth: 220 }}
                />
                <Button
                  variant="contained"
                  startIcon={<Search />}
                  disabled={!auditIP || auditMutation.isPending}
                  onClick={() => auditMutation.mutate()}
                >
                  {auditMutation.isPending
                    ? translate('pages.ipam.audit.searching')
                    : translate('pages.ipam.audit.submit')}
                </Button>
              </Stack>
            </CardContent>
          </Card>

          {auditMutation.isSuccess && (
            <Panel
              title={translate('pages.ipam.audit.results_title', { ip: auditIP })}
              subtitle={translate('pages.ipam.audit.results_subtitle', {
                count: auditResults.length,
              })}
              dense
            >
              {auditResults.length === 0 ? (
                <Box sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
                  <Typography>{translate('pages.ipam.audit.no_match')}</Typography>
                </Box>
              ) : (
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>{translate('pages.ipam.audit.col.username')}</TableCell>
                      <TableCell>{translate('pages.ipam.audit.col.mac')}</TableCell>
                      <TableCell>{translate('pages.ipam.audit.col.nas')}</TableCell>
                      <TableCell>{translate('pages.ipam.audit.col.start')}</TableCell>
                      <TableCell>{translate('pages.ipam.audit.col.stop')}</TableCell>
                      <TableCell>{translate('pages.ipam.audit.col.status')}</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {auditResults.map((r, i) => (
                      <TableRow key={i} hover>
                        <TableCell sx={{ color: 'primary.main', fontWeight: 700 }}>
                          <Mono>{r.username}</Mono>
                        </TableCell>
                        <TableCell><Mono>{r.mac_addr || '—'}</Mono></TableCell>
                        <TableCell><Mono>{r.nas_addr}</Mono></TableCell>
                        <TableCell sx={{ fontSize: '0.78rem' }}>{new Date(r.start_time).toLocaleString()}</TableCell>
                        <TableCell sx={{ fontSize: '0.78rem' }}>{r.stop_time ? new Date(r.stop_time).toLocaleString() : translate('pages.ipam.audit.currently_active')}</TableCell>
                        <TableCell>
                          <StatusChip value={r.status} />
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </Panel>
          )}
        </Stack>
      )}

      {/* Tab 2: Live Diagnostic Ping Tool */}
      {tab === 2 && (
        <Stack spacing={2.5}>
          <Card variant="outlined">
            <CardContent sx={{ p: 2 }}>
              <Typography variant="subtitle1" fontWeight={800} sx={{ mb: 0.5 }}>
                {translate('pages.ipam.ping.title')}
              </Typography>
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 2 }}>
                {translate('pages.ipam.ping.subtitle')}
              </Typography>

              <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems="center">
                <TextField
                  size="small"
                  label={translate('pages.ipam.ping.host_label')}
                  placeholder={translate('pages.ipam.ping.host_placeholder')}
                  value={pingHost}
                  onChange={(e) => setPingHost(e.target.value)}
                  sx={{ minWidth: 280 }}
                />
                <TextField
                  size="small"
                  type="number"
                  label={translate('pages.ipam.ping.count_label')}
                  value={pingCount}
                  onChange={(e) => setPingCount(Number(e.target.value))}
                  inputProps={{ min: 1, max: 10 }}
                  sx={{ width: 90 }}
                />
                <Button
                  variant="contained"
                  startIcon={<PlayArrow />}
                  disabled={!pingHost || pingMutation.isPending}
                  onClick={() => pingMutation.mutate()}
                >
                  {pingMutation.isPending
                    ? translate('pages.ipam.ping.probing')
                    : translate('pages.ipam.ping.submit')}
                </Button>
              </Stack>
            </CardContent>
          </Card>

          {pingResult && (
            <Panel
              title={translate('pages.ipam.ping.results_title', { host: pingResult.host })}
              subtitle={translate('pages.ipam.ping.results_subtitle', {
                received: pingResult.received,
                sent: pingResult.sent,
                method: pingResult.probe_method || 'tcp_syn',
                port: pingResult.port || '80',
              })}
              actions={
                <Stack direction="row" spacing={2} alignItems="center">
                  <Typography variant="caption" color="text.secondary">{translate('pages.ipam.ping.loss')} <strong style={{ color: pingResult.loss_percent > 0 ? '#ef4444' : '#22c55e' }}>{pingResult.loss_percent.toFixed(0)}%</strong></Typography>
                  <Typography variant="caption" color="text.secondary">{translate('pages.ipam.ping.avg')} <strong>{pingResult.avg_rtt_ms} ms</strong></Typography>
                  <Typography variant="caption" color="text.secondary">{translate('pages.ipam.ping.minmax')} <strong>{pingResult.min_rtt_ms}/{pingResult.max_rtt_ms} ms</strong></Typography>
                </Stack>
              }
            >
              <ConsoleBox>
                {pingResult.details.map((d) => (
                  <div key={d.seq}>
                    {d.success
                      ? translate('pages.ipam.ping.reply', {
                          target: pingResult.target_addr || pingResult.host,
                          seq: d.seq,
                          rtt: d.rtt_ms,
                        })
                      : translate('pages.ipam.ping.request_failed', {
                          seq: d.seq,
                          error: d.error || translate('pages.ipam.ping.unreachable'),
                        })}
                  </div>
                ))}
              </ConsoleBox>
            </Panel>
          )}
        </Stack>
      )}

      {/* Add Pool Dialog */}
      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle sx={{ fontWeight: 800 }}>{translate('pages.ipam.dialog.title')}</DialogTitle>
        <DialogContent dividers>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <TextField
              size="small"
              label={translate('pages.ipam.dialog.name')}
              value={poolForm.name}
              onChange={(e) => setPoolForm({ ...poolForm, name: e.target.value })}
              fullWidth
            />
            <TextField
              size="small"
              label={translate('pages.ipam.dialog.cidr')}
              placeholder={translate('pages.ipam.dialog.cidr_placeholder')}
              value={poolForm.cidr}
              onChange={(e) => setPoolForm({ ...poolForm, cidr: e.target.value })}
              fullWidth
            />
            <FormControl size="small" fullWidth>
              <InputLabel>{translate('pages.ipam.dialog.pool_type')}</InputLabel>
              <Select
                label={translate('pages.ipam.dialog.pool_type')}
                value={poolForm.pool_type}
                onChange={(e) => setPoolForm({ ...poolForm, pool_type: e.target.value })}
              >
                <MenuItem value="cgnat">{translate('pages.ipam.pool_type.cgnat')}</MenuItem>
                <MenuItem value="public">{translate('pages.ipam.pool_type.public')}</MenuItem>
                <MenuItem value="static">{translate('pages.ipam.pool_type.static')}</MenuItem>
                <MenuItem value="delegated">{translate('pages.ipam.pool_type.delegated')}</MenuItem>
              </Select>
            </FormControl>
            <TextField
              size="small"
              label={translate('pages.ipam.dialog.gateway')}
              value={poolForm.gateway}
              onChange={(e) => setPoolForm({ ...poolForm, gateway: e.target.value })}
              fullWidth
            />
            <Stack direction="row" spacing={1.5}>
              <TextField
                size="small"
                label={translate('pages.ipam.dialog.dns_primary')}
                value={poolForm.dns_primary}
                onChange={(e) => setPoolForm({ ...poolForm, dns_primary: e.target.value })}
                sx={{ flex: 1 }}
              />
              <TextField
                size="small"
                label={translate('pages.ipam.dialog.dns_secondary')}
                value={poolForm.dns_secondary}
                onChange={(e) => setPoolForm({ ...poolForm, dns_secondary: e.target.value })}
                sx={{ flex: 1 }}
              />
            </Stack>
            <TextField
              size="small"
              label={translate('pages.ipam.dialog.description')}
              multiline
              rows={2}
              value={poolForm.description}
              onChange={(e) => setPoolForm({ ...poolForm, description: e.target.value })}
              fullWidth
            />
          </Stack>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button onClick={() => setCreateOpen(false)}>{translate('pages.ipam.dialog.cancel')}</Button>
          <Button
            variant="contained"
            disabled={createPoolMutation.isPending || !poolForm.cidr}
            onClick={() => createPoolMutation.mutate()}
          >
            {createPoolMutation.isPending
              ? translate('pages.ipam.dialog.saving')
              : translate('pages.ipam.dialog.submit')}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default IPAMPage;
