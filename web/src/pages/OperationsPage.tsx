import React, { useState } from 'react';
import {
  Alert, Box, Button, Card, CardContent, Chip, Divider, FormControlLabel,
  IconButton, LinearProgress, MenuItem, Stack, Switch, Tab, Tabs, Table, TableBody, TableCell,
  TableHead, TableRow, TextField, Tooltip, Typography,
} from '@mui/material';
import {
  Add, CheckCircle, DeleteOutline, Dns, ErrorOutline, NetworkCheck,
  NotificationsActive, Pause, PlayArrow, QrCode2, Refresh, Send, Sensors,
  ShowChart, Speed,
} from '@mui/icons-material';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNotify, usePermissions, useTranslate } from 'react-admin';
import { apiRequest, extractData } from '../utils/apiClient';
import { MrtgTrafficGraph } from '../components/MrtgTrafficGraph';
import { NocImpactPanel } from '../components/NocImpactPanel';
import { KpiStrip, KpiTile, Mono, PageHeader, Panel } from '../components/Enterprise';

type Heartbeat = { id: string; checked_at: string; reachable: boolean; latency_milliseconds: number; packet_loss_percent: number };
type Target = {
  id: string; name: string; kind: string; address: string; probe_type: string; port: number;
  interval_seconds: number; timeout_milliseconds: number; failure_threshold: number; enabled: boolean;
  last_status: string; last_latency_milliseconds: number; last_packet_loss_percent: number;
  last_checked_at?: string; last_error?: string; snmp_secret_configured?: boolean;
  uptime_percent?: number; heartbeats?: Heartbeat[];
};

type Sample = { id: string; checked_at: string; reachable: boolean; latency_milliseconds: number; packet_loss_percent: number; interface_metrics_json?: string };
type InterfaceMetric = { index: number; name: string; operational: string; in_octets?: string; out_octets?: string };
type OutboxItem = { id: string; event_type: string; recipient: string; status: string; attempts: number; created_at: string; sent_at?: string; last_error?: string };
type WhatsApp = { connected: boolean; logged_in: boolean; state: string; account?: string; qr_image?: string; qr_expires?: string };
type NotifySettings = { whatsapp_enabled: boolean; risk_acknowledged_at?: string; recipients: string[]; events: string[]; whatsapp: WhatsApp };
const eventOptions = [
  ['network.down', 'event.network_down'], ['network.recovered', 'event.network_recovered'],
  ['billing.suspended', 'event.billing_suspended'], ['billing.reactivated', 'event.billing_reactivated'], ['scheduler.failure', 'event.scheduler_failure'],
];

const formatBytes = (raw?: string) => {
  const bytes = Number(raw ?? 0);
  if (!Number.isFinite(bytes) || bytes < 0) return '—';
  if (bytes < 1024) return `${bytes} B`;
  const units = ['KB', 'MB', 'GB', 'TB'];
  let value = bytes;
  let unit = -1;
  do { value /= 1024; unit += 1; } while (value >= 1024 && unit < units.length - 1);
  return `${value.toFixed(value >= 100 ? 0 : 1)} ${units[unit]}`;
};

const parseInterfaces = (raw?: string): InterfaceMetric[] => {
  try {
    const parsed: unknown = JSON.parse(raw || '[]');
    return Array.isArray(parsed) ? parsed as InterfaceMetric[] : [];
  } catch {
    return [];
  }
};

const KumaHeartbeatBar = ({ heartbeats, enabled }: { heartbeats?: Heartbeat[]; enabled: boolean }) => {
  const translate = useTranslate();
  const count = 30;
  const list = heartbeats ?? [];
  const emptySlotsCount = Math.max(0, count - list.length);
  const slots: Array<{ sample?: Heartbeat; status: 'up' | 'down' | 'empty' | 'paused' }> = [];

  for (let i = 0; i < emptySlotsCount; i++) {
    slots.push({ status: 'empty' });
  }

  for (const sample of list) {
    slots.push({
      sample,
      status: !enabled ? 'paused' : sample.reachable ? 'up' : 'down',
    });
  }

  const oldestTime = list.length > 0 ? new Date(list[0].checked_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : '—';
  const latestTime = list.length > 0 ? new Date(list[list.length - 1].checked_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : translate('pages.operations.now');

  return (
    <Box sx={{ width: '100%', mt: 1.25 }}>
      <Stack direction="row" spacing={0.4} sx={{ width: '100%', height: 26, alignItems: 'center' }}>
        {slots.map((slot, idx) => {
          if (!slot.sample || slot.status === 'empty') {
            return (
              <Box
                key={`empty-${idx}`}
                sx={{
                  flex: 1,
                  height: 20,
                  borderRadius: '3px',
                  bgcolor: 'action.hover',
                  opacity: 0.35,
                }}
              />
            );
          }
          const s = slot.sample;
          const isUp = slot.status === 'up';
          const isPaused = slot.status === 'paused';
          const statusLabel = translate(`pages.operations.status.${isPaused ? 'paused' : isUp ? 'up' : 'down'}`);
          const label = translate('pages.operations.heartbeat_label', {
            time: new Date(s.checked_at).toLocaleTimeString(),
            status: statusLabel,
            latency: s.latency_milliseconds,
            loss: s.packet_loss_percent.toFixed(0),
          });
          const barColor = isPaused ? '#9ca3af' : isUp ? '#22c55e' : '#ef4444';

          return (
            <Tooltip key={s.id || idx} title={label} arrow placement="top">
              <Box
                role="img"
                aria-label={label}
                sx={{
                  flex: 1,
                  height: 20,
                  borderRadius: '3px',
                  bgcolor: barColor,
                  transition: 'all 0.15s ease-in-out',
                  cursor: 'pointer',
                  '&:hover': {
                    transform: 'scaleY(1.3)',
                    opacity: 0.95,
                    boxShadow: `0 0 8px ${barColor}`,
                  },
                }}
              />
            </Tooltip>
          );
        })}
      </Stack>
      <Stack direction="row" justifyContent="space-between" sx={{ mt: 0.5, px: 0.25 }}>
        <Typography variant="caption" color="text.secondary" sx={{ fontSize: '0.72rem' }}>
          {oldestTime} ({translate('pages.operations.check_count', { count: list.length })})
        </Typography>
        <Typography variant="caption" color="text.secondary" sx={{ fontSize: '0.72rem' }}>
          {latestTime}
        </Typography>
      </Stack>
    </Box>
  );
};

const SampleTrend = ({ samples }: { samples: Sample[] }) => {
  const translate = useTranslate();
  const chronological = [...samples].reverse();
  const maxLatency = Math.max(1, ...chronological.map(sample => sample.latency_milliseconds));
  return <Stack direction="row" spacing={0.75} alignItems="flex-end" sx={{ minHeight: 92, pt: 1 }}>
    {chronological.map(sample => {
      const height = Math.max(8, (sample.latency_milliseconds / maxLatency) * 68);
      const label = translate('pages.operations.sample_label', {
        time: new Date(sample.checked_at).toLocaleString(),
        latency: sample.latency_milliseconds,
        loss: sample.packet_loss_percent.toFixed(0),
      });
      return <Tooltip key={sample.id} title={label}>
        <Box role="img" aria-label={label} sx={{ flex: 1, minWidth: 8, maxWidth: 42, height, borderRadius: '4px 4px 1px 1px', bgcolor: sample.reachable ? 'success.main' : 'error.main', opacity: 0.82 }} />
      </Tooltip>;
    })}
  </Stack>;
};

const OperationsPage: React.FC = () => {
  const [tab, setTab] = useState(0);
  const [form, setForm] = useState({ name: '', kind: 'router', address: '', probe_type: 'icmp', port: 80, interval_seconds: 60, timeout_milliseconds: 1500, failure_threshold: 2, enabled: true, snmp_version: 'v3', snmp_username: '', snmp_auth_protocol: 'SHA256', snmp_privacy_protocol: 'AES', snmp_community: '', snmp_auth_password: '', snmp_privacy_password: '' });
  const [selectedTarget, setSelectedTarget] = useState('');
  const [impactTarget, setImpactTarget] = useState('');
  const [mrtgTargetId, setMrtgTargetId] = useState('');
  const [syslogSeverity, setSyslogSeverity] = useState('');
  const [syslogTag, setSyslogTag] = useState('');
  const [syslogQ, setSyslogQ] = useState('');
  const [syslogAutoRefresh, setSyslogAutoRefresh] = useState(true);

  const [recipientsText, setRecipientsText] = useState('');
  const [events, setEvents] = useState<string[]>(['network.down', 'network.recovered']);
  const [enabled, setEnabled] = useState(false);
  const [riskAck, setRiskAck] = useState(false);
  const [selectedRecipient, setSelectedRecipient] = useState('');
  const queryClient = useQueryClient();
  const notify = useNotify();
  const translate = useTranslate();
  const { permissions = [] } = usePermissions();
  const canControlWhatsApp = permissions.includes('platform_admin');
  const targetsQuery = useQuery({ queryKey: ['network', 'monitor-targets'], queryFn: () => apiRequest<Target[]>('/network/monitor-targets?perPage=100') });
  const incidentsQuery = useQuery({ queryKey: ['network', 'monitor-incidents'], queryFn: () => apiRequest<Array<Record<string, unknown>>>('/network/monitor-incidents?perPage=20') });
  const samplesQuery = useQuery({ queryKey: ['network', 'samples', selectedTarget], queryFn: () => apiRequest<Sample[]>(`/network/monitor-targets/${selectedTarget}/samples?perPage=12`), enabled: Boolean(selectedTarget) });
  const waQuery = useQuery({ queryKey: ['notifications', 'whatsapp'], queryFn: () => apiRequest<NotifySettings>('/system/notifications/whatsapp'), enabled: tab === 3, refetchInterval: tab === 3 ? 3000 : false });
  const outboxQuery = useQuery({ queryKey: ['notifications', 'outbox'], queryFn: () => apiRequest<OutboxItem[]>('/system/notifications/outbox?perPage=10'), enabled: tab === 3, refetchInterval: tab === 3 ? 10000 : false });
  const settings = waQuery.data;
  const settingsLoaded = settings !== undefined;
  const whatsappEnabled = settings?.whatsapp_enabled ?? false;
  const riskAcknowledgedAt = settings?.risk_acknowledged_at ?? '';
  const recipientsSignature = (settings?.recipients ?? []).join(',');
  const eventsSignature = (settings?.events ?? []).join(',');
  React.useEffect(() => {
    if (!settingsLoaded) return;
    const configuredRecipients = recipientsSignature ? recipientsSignature.split(',') : [];
    const configuredEvents = eventsSignature ? eventsSignature.split(',') : [];
    setEnabled(whatsappEnabled);
    setRiskAck(Boolean(riskAcknowledgedAt));
    setRecipientsText(configuredRecipients.join(', '));
    setEvents(configuredEvents);
    setSelectedRecipient(current => configuredRecipients.includes(current) ? current : configuredRecipients[0] ?? '');
  }, [settingsLoaded, whatsappEnabled, riskAcknowledgedAt, recipientsSignature, eventsSignature]);
  const targets = targetsQuery.data ?? [];
  const samples = samplesQuery.data ?? [];
  const incidents = incidentsQuery.data ?? [];
  const outboxItems = outboxQuery.data ?? [];
  const refresh = async () => { await queryClient.invalidateQueries({ queryKey: ['network'] }); await queryClient.invalidateQueries({ queryKey: ['notifications'] }); };
  const mutation = useMutation({
    mutationFn: ({ path, method, body }: { path: string; method: string; body?: unknown }) => apiRequest(path, { method, body: body === undefined ? undefined : JSON.stringify(body) }),
    onSuccess: async () => { await refresh(); notify(translate('pages.operations.operation_completed'), { type: 'success' }); },
    onError: (error: Error) => notify(error.message || translate('pages.operations.operation_failed'), { type: 'error' }),
  });
  const saveNotify = () => mutation.mutate({ path: '/system/notifications/whatsapp', method: 'PUT', body: { whatsapp_enabled: enabled, risk_acknowledged: riskAck, recipients: recipientsText.split(/[\s,;]+/).filter(Boolean), events } });
  const addTarget = () => mutation.mutate({ path: '/network/monitor-targets', method: 'POST', body: form });
  const statusColor = (status: string) => status === 'up' ? 'success' : status === 'down' ? 'error' : status === 'paused' ? 'default' : 'info';

  const upTargets = targets.filter(x => x.enabled && x.last_status === 'up');
  const downTargets = targets.filter(x => x.enabled && x.last_status === 'down');
  const pausedTargets = targets.filter(x => !x.enabled || x.last_status === 'paused');
  const anyDown = downTargets.length > 0;
  const activeTargets = targets.filter(t => t.enabled);
  const avgUptime = activeTargets.length > 0
    ? activeTargets.reduce((acc, t) => acc + (t.uptime_percent ?? 100), 0) / activeTargets.length
    : 100;

  return (
    <Box sx={{ p: { xs: 1.5, md: 2.5 }, maxWidth: 1600, mx: 'auto' }}>
      <PageHeader
        section={translate('pages.operations.section')}
        title={translate('pages.operations.title')}
        subtitle={translate('pages.operations.subtitle')}
        actions={
          <Button
            size="small"
            variant="outlined"
            startIcon={<Refresh />}
            onClick={() => {
              void targetsQuery.refetch();
              void incidentsQuery.refetch();
            }}
            disabled={targetsQuery.isFetching}
          >
            {translate('pages.operations.refresh')}
          </Button>
        }
      />

      <KpiStrip>
        <KpiTile label={translate('pages.operations.kpi.monitored_targets')} value={targets.length} icon={<Sensors fontSize="small" />} />
        <KpiTile label={translate('pages.operations.kpi.operational_up')} value={upTargets.length} tone="success" hint={translate('pages.operations.kpi.passing_health_check')} />
        <KpiTile label={translate('pages.operations.kpi.failing_down')} value={downTargets.length} tone={anyDown ? 'error' : 'success'} hint={anyDown ? translate('pages.operations.kpi.incidents_open') : translate('pages.operations.kpi.no_alerts')} />
        <KpiTile label={translate('pages.operations.kpi.average_uptime')} value={`${avgUptime.toFixed(1)}%`} tone={avgUptime < 99 ? 'warning' : 'success'} hint={translate('pages.operations.kpi.active_fleet')} />
        <KpiTile label={translate('pages.operations.kpi.paused')} value={pausedTargets.length} tone="info" hint={translate('pages.operations.kpi.monitoring_suspended')} />
        <KpiTile label={translate('pages.operations.kpi.active_alerts')} value={incidents.filter((i) => i.status === 'firing').length} tone={incidents.some((i) => i.status === 'firing') ? 'error' : 'success'} hint={translate('pages.operations.kpi.firing_incidents')} />
      </KpiStrip>

      <Tabs value={tab} onChange={(_, value) => setTab(value)} sx={{ mb: 2 }}>
        <Tab icon={<NetworkCheck/>} iconPosition="start" label={translate('pages.operations.tabs.network_monitoring')} />
        <Tab icon={<ShowChart/>} iconPosition="start" label={translate('pages.operations.tabs.interface_traffic')} />
        <Tab icon={<Dns/>} iconPosition="start" label={translate('pages.operations.tabs.syslog_stream')} />
        <Tab icon={<NotificationsActive/>} iconPosition="start" label={translate('pages.operations.tabs.whatsapp_alerts')} />
      </Tabs>

    {tab === 0 && <Stack spacing={2.5}>
      {(targetsQuery.isError || incidentsQuery.isError) && <Alert severity="error">{translate('pages.operations.load_error')}</Alert>}

      {/* Kuma-style System Health Status Banner */}
      {targets.length > 0 && (
        <Card
          variant="outlined"
          sx={{
            p: 2.25,
            borderRadius: 1.5,
            borderLeft: 6,
            borderColor: anyDown ? 'error.main' : 'success.main',
            bgcolor: anyDown ? 'rgba(239, 68, 68, 0.08)' : 'rgba(34, 197, 94, 0.08)',
          }}
        >
          <Stack direction="row" alignItems="center" justifyContent="space-between" flexWrap="wrap" gap={2}>
            <Stack direction="row" alignItems="center" spacing={2}>
              {anyDown ? (
                <ErrorOutline color="error" sx={{ fontSize: 36 }} />
              ) : (
                <CheckCircle color="success" sx={{ fontSize: 36 }} />
              )}
              <Box>
                <Typography variant="h6" fontWeight={800}>
                  {anyDown
                    ? translate('pages.operations.incidents_detected', { count: downTargets.length })
                    : translate('pages.operations.all_operational')}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {anyDown
                    ? translate('pages.operations.failing_targets', { targets: downTargets.map(t => t.name).join(', ') })
                    : translate('pages.operations.all_devices_ok', { count: activeTargets.length })}
                </Typography>
              </Box>
            </Stack>
            <Stack direction="row" spacing={3} alignItems="center">
              <Box textAlign="right">
                <Typography variant="caption" color="text.secondary" sx={{ textTransform: 'uppercase', letterSpacing: 0.5, fontWeight: 700 }}>
                  {translate('pages.operations.average_uptime')}
                </Typography>
                <Typography variant="h5" fontWeight={800} color={avgUptime < 99 ? 'warning.main' : 'success.main'}>
                  {avgUptime.toFixed(1)}%
                </Typography>
              </Box>
            </Stack>
          </Stack>
        </Card>
      )}

      <Card variant="outlined">
        <CardContent>
          <Stack direction="row" justifyContent="space-between" alignItems="center" flexWrap="wrap" gap={1}>
            <Box>
              <Typography variant="h6" fontWeight={700}>{translate('pages.operations.monitored_targets')}</Typography>
              <Typography variant="body2" color="text.secondary">
                {translate('pages.operations.monitored_targets_desc')}
              </Typography>
            </Box>
            <Button startIcon={<Refresh/>} variant="outlined" size="small" onClick={() => void refresh()}>
              {translate('pages.operations.refresh')}
            </Button>
          </Stack>

          <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap" sx={{ my: 2 }}>
            <Chip label={translate('pages.operations.target_count', { count: targets.length })} variant="outlined" />
            <Chip label={translate('pages.operations.online_count', { count: upTargets.length })} color="success" />
            <Chip label={translate('pages.operations.down_count', { count: downTargets.length })} color={downTargets.length > 0 ? 'error' : 'default'} />
            <Chip label={translate('pages.operations.paused_count', { count: pausedTargets.length })} color={pausedTargets.length > 0 ? 'warning' : 'default'} />
            <Chip label={translate('pages.operations.incidents_count', { count: incidents.length })} />
          </Stack>

          {targetsQuery.isLoading && <LinearProgress sx={{ my: 2 }} />}
          {!targetsQuery.isLoading && targets.length === 0 && (
            <Alert severity="info" sx={{ mt: 1 }}>
              {translate('pages.operations.no_targets')}
            </Alert>
          )}

          {/* Kuma-style Target Cards List */}
          <Stack spacing={1.75}>
            {targets.map(target => {
              const isPaused = !target.enabled || target.last_status === 'paused';
              const isUp = target.enabled && target.last_status === 'up';
              const isDown = target.enabled && target.last_status === 'down';
              const dotColor = isPaused ? '#9ca3af' : isUp ? '#22c55e' : isDown ? '#ef4444' : '#38bdf8';

              return (
                <Card
                  key={target.id}
                  variant="outlined"
                  sx={{
                    p: 2,
                    borderRadius: 1,
                    transition: 'border-color 0.2s, box-shadow 0.2s',
                    '&:hover': { borderColor: 'primary.main', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' },
                  }}
                >
                  <Stack direction="row" alignItems="center" justifyContent="space-between" flexWrap="wrap" gap={1.5}>
                    {/* Identity & Status */}
                    <Stack direction="row" alignItems="center" spacing={1.5} sx={{ minWidth: 260, flex: '1 1 300px' }}>
                      <Box
                        sx={{
                          width: 12,
                          height: 12,
                          borderRadius: '50%',
                          bgcolor: dotColor,
                          boxShadow: `0 0 6px ${dotColor}`,
                          flexShrink: 0,
                        }}
                      />
                      <Box>
                        <Stack direction="row" alignItems="center" spacing={1}>
                          <Typography fontWeight={700} variant="subtitle1">
                            {target.name}
                          </Typography>
                          <Chip
                            size="small"
                            color={statusColor(target.last_status)}
                            label={translate(`pages.operations.status.${target.last_status}`, { _: target.last_status.toUpperCase() })}
                            sx={{ height: 20, fontSize: '0.68rem', fontWeight: 700 }}
                          />
                        </Stack>
                        <Typography variant="caption" color="text.secondary">
                          {target.address} · {translate(`pages.operations.kind.${target.kind}`, { _: target.kind.toUpperCase() })} · {target.probe_type.toUpperCase()}{target.probe_type !== 'icmp' ? `:${target.port}` : ''}
                        </Typography>
                      </Box>
                    </Stack>

                    {/* Performance Chips */}
                    <Stack direction="row" spacing={1} alignItems="center" flexWrap="wrap">
                      <Chip
                        size="small"
                        color={target.uptime_percent !== undefined && target.uptime_percent < 99 ? 'warning' : 'success'}
                        variant={target.uptime_percent !== undefined && target.uptime_percent < 99 ? 'filled' : 'outlined'}
                        label={translate('pages.operations.uptime_chip', { value: (target.uptime_percent ?? 100).toFixed(1) })}
                        sx={{ fontWeight: 700 }}
                      />
                      <Chip size="small" variant="outlined" label={`${target.last_latency_milliseconds} ms`} />
                      <Chip size="small" variant="outlined" label={translate('pages.operations.loss_chip', { value: target.last_packet_loss_percent.toFixed(0) })} />
                      <Typography variant="caption" color="text.secondary" sx={{ minWidth: 100 }}>
                        {target.last_checked_at ? new Date(target.last_checked_at).toLocaleTimeString() : translate('pages.operations.not_checked')}
                      </Typography>
                    </Stack>

                    {/* Actions */}
                    <Stack direction="row" spacing={1} alignItems="center">
                      <Tooltip title={target.enabled ? translate('pages.operations.pause_monitoring') : translate('pages.operations.resume_monitoring')}>
                        <Button
                          size="small"
                          variant="outlined"
                          color={target.enabled ? 'inherit' : 'primary'}
                          startIcon={target.enabled ? <Pause sx={{ fontSize: 16 }} /> : <PlayArrow sx={{ fontSize: 16 }} />}
                          onClick={() => mutation.mutate({ path: `/network/monitor-targets/${target.id}/toggle`, method: 'POST' })}
                        >
                          {target.enabled ? translate('pages.operations.pause') : translate('pages.operations.resume')}
                        </Button>
                      </Tooltip>
                      <Button
                        size="small"
                        disabled={!target.enabled}
                        onClick={() => mutation.mutate({ path: `/network/monitor-targets/${target.id}/check`, method: 'POST' })}
                      >
                        {translate('pages.operations.check_now')}
                      </Button>
                      <Button
                        size="small"
                        color={selectedTarget === target.id ? 'primary' : 'inherit'}
                        onClick={() => setSelectedTarget(selectedTarget === target.id ? '' : target.id)}
                      >
                        {selectedTarget === target.id ? translate('pages.operations.hide_history') : translate('pages.operations.history')}
                      </Button>
                      <Button
                        size="small"
                        color={impactTarget === target.id ? 'primary' : 'inherit'}
                        onClick={() => setImpactTarget(impactTarget === target.id ? '' : target.id)}
                      >
                        {impactTarget === target.id ? translate('pages.operations.hide_impact') : translate('pages.operations.impact')}
                      </Button>
                      <IconButton
                        size="small"
                        color="error"
                        onClick={() => {
                          if (window.confirm(translate('pages.operations.remove_confirm', { name: target.name }))) {
                            mutation.mutate({ path: `/network/monitor-targets/${target.id}`, method: 'DELETE' });
                          }
                        }}
                      >
                        <DeleteOutline fontSize="small" />
                      </IconButton>
                    </Stack>
                  </Stack>

                  {/* Kuma Horizontal Heartbeat Bar */}
                  <KumaHeartbeatBar heartbeats={target.heartbeats} enabled={target.enabled} />

                  {/* Error banner if last check failed */}
                  {target.last_error && target.enabled && (
                    <Typography variant="caption" color="error" sx={{ display: 'block', mt: 1, fontStyle: 'italic' }}>
                      {translate('pages.operations.last_failure', { error: target.last_error })}
                    </Typography>
                  )}
                </Card>
              );
            })}
          </Stack>

          {/* Expanded Target Details (History & SNMP Counters) */}
          {impactTarget && (
            <Box sx={{ mt: 2.5, p: 2, borderRadius: 1, bgcolor: 'action.hover', border: '1px solid', borderColor: 'divider' }}>
              <Stack direction="row" justifyContent="space-between" alignItems="center" flexWrap="wrap" gap={1}>
                <Box>
                  <Typography variant="subtitle1" fontWeight={700}>
                    {translate('pages.operations.subscriber_impact', { name: targets.find((target) => target.id === impactTarget)?.name ?? impactTarget })}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    {translate('pages.operations.subscriber_impact_desc')}
                  </Typography>
                </Box>
                <Button size="small" onClick={() => setImpactTarget('')}>{translate('pages.operations.close')}</Button>
              </Stack>
              <NocImpactPanel targetId={impactTarget} />
            </Box>
          )}

          {selectedTarget && (
            <Box sx={{ mt: 2.5, p: 2, borderRadius: 1, bgcolor: 'action.hover', border: '1px solid', borderColor: 'divider' }}>
              <Typography variant="subtitle1" fontWeight={700}>{translate('pages.operations.history_title')}</Typography>
              <Typography variant="caption" color="text.secondary">{translate('pages.operations.history_desc')}</Typography>
              {samplesQuery.isLoading ? <LinearProgress sx={{ mt: 1 }} /> : null}
              {samplesQuery.isError && <Alert severity="error" sx={{ mt: 1 }}>{translate('pages.operations.history_load_error')}</Alert>}
              {!samplesQuery.isLoading && samples.length === 0 && (
                <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>{translate('pages.operations.no_checks')}</Typography>
              )}
              {samples.length > 0 && <SampleTrend samples={samples} />}

              {samples.slice(0, 1).map(sample => {
                const ifaces = parseInterfaces(sample.interface_metrics_json).slice(0, 24);
                return ifaces.length ? (
                  <Box sx={{ mt: 2 }} key={sample.id}>
                    <Typography variant="subtitle2" sx={{ mb: 0.5 }}>{translate('pages.operations.interface_counters')}</Typography>
                    <Box sx={{ overflowX: 'auto' }}>
                      <Table size="small" aria-label={translate('pages.operations.interface_counters_table')}>
                        <TableHead>
                          <TableRow>
                            <TableCell>{translate('pages.operations.table.interface')}</TableCell>
                            <TableCell>{translate('pages.operations.table.status')}</TableCell>
                            <TableCell align="right">{translate('pages.operations.table.received')}</TableCell>
                            <TableCell align="right">{translate('pages.operations.table.sent')}</TableCell>
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {ifaces.map(iface => (
                            <TableRow key={iface.index} hover>
                              <TableCell>{iface.name || `if${iface.index}`}</TableCell>
                              <TableCell>
                                <Chip size="small" label={translate(`pages.operations.operational_status.${iface.operational || 'unknown'}`, { _: iface.operational || 'unknown' })} color={iface.operational === 'up' ? 'success' : 'default'} />
                              </TableCell>
                              <TableCell align="right" sx={{ fontVariantNumeric: 'tabular-nums' }}>{formatBytes(iface.in_octets)}</TableCell>
                              <TableCell align="right" sx={{ fontVariantNumeric: 'tabular-nums' }}>{formatBytes(iface.out_octets)}</TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </Box>
                  </Box>
                ) : null;
              })}
            </Box>
          )}
        </CardContent>
      </Card>

      {/* Add Target Form Card */}
      <Card variant="outlined">
        <CardContent>
          <Typography variant="h6" fontWeight={700} sx={{ mb: 1.5 }}>{translate('pages.operations.add_target_title')}</Typography>
          <Stack direction="row" spacing={1.5} useFlexGap flexWrap="wrap" alignItems="center">
            <TextField size="small" label={translate('pages.operations.form.name')} value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} />
            <TextField size="small" label={translate('pages.operations.form.ip_address')} value={form.address} onChange={e => setForm({ ...form, address: e.target.value })} />
            <TextField size="small" select label={translate('pages.operations.form.type')} value={form.kind} onChange={e => setForm({ ...form, kind: e.target.value })}>
              {['router','switch','server','nas','other'].map(x => <MenuItem key={x} value={x}>{translate(`pages.operations.kind.${x}`, { _: x.toUpperCase() })}</MenuItem>)}
            </TextField>
            <TextField size="small" select label={translate('pages.operations.form.probe')} value={form.probe_type} onChange={e => setForm({ ...form, probe_type: e.target.value })}>
              {['icmp','tcp','http','snmp'].map(x => <MenuItem key={x} value={x}>{x.toUpperCase()}</MenuItem>)}
            </TextField>
            {form.probe_type === 'tcp' && (
              <TextField size="small" type="number" label={translate('pages.operations.form.tcp_port')} value={form.port} onChange={e => setForm({ ...form, port: Number(e.target.value) })} />
            )}
            {form.probe_type === 'http' && (
              <TextField size="small" type="number" label={translate('pages.operations.form.http_port')} value={form.port || 80} onChange={e => setForm({ ...form, port: Number(e.target.value) })} helperText={translate('pages.operations.form.http_port_helper')} />
            )}
            {form.probe_type === 'snmp' && <>
              <TextField size="small" select label={translate('pages.operations.form.snmp_version')} value={form.snmp_version} onChange={e => setForm({ ...form, snmp_version: e.target.value })}>
                <MenuItem value="v3">{translate('pages.operations.snmp_v3_authpriv')}</MenuItem>
                <MenuItem value="v2c">{translate('pages.operations.snmp_v2c')}</MenuItem>
              </TextField>
              <TextField size="small" type="number" label={translate('pages.operations.form.udp_port')} value={form.port || 161} onChange={e => setForm({ ...form, port: Number(e.target.value) })} />
              {form.snmp_version === 'v2c' ? (
                <TextField size="small" type="password" label={translate('pages.operations.form.community')} value={form.snmp_community} onChange={e => setForm({ ...form, snmp_community: e.target.value })} />
              ) : <>
                <TextField size="small" label={translate('pages.operations.form.username')} value={form.snmp_username} onChange={e => setForm({ ...form, snmp_username: e.target.value })} />
                <TextField size="small" select label={translate('pages.operations.form.auth')} value={form.snmp_auth_protocol} onChange={e => setForm({ ...form, snmp_auth_protocol: e.target.value })}>
                  {['SHA256','SHA','SHA384','SHA512','MD5'].map(x => <MenuItem key={x} value={x}>{x}</MenuItem>)}
                </TextField>
                <TextField size="small" type="password" label={translate('pages.operations.form.auth_password')} value={form.snmp_auth_password} onChange={e => setForm({ ...form, snmp_auth_password: e.target.value })} />
                <TextField size="small" select label={translate('pages.operations.form.privacy')} value={form.snmp_privacy_protocol} onChange={e => setForm({ ...form, snmp_privacy_protocol: e.target.value })}>
                  {['AES','AES192','AES256','DES'].map(x => <MenuItem key={x} value={x}>{x}</MenuItem>)}
                </TextField>
                <TextField size="small" type="password" label={translate('pages.operations.form.privacy_password')} value={form.snmp_privacy_password} onChange={e => setForm({ ...form, snmp_privacy_password: e.target.value })} />
              </>}
            </>}
            <TextField size="small" type="number" label={translate('pages.operations.form.interval')} value={form.interval_seconds} inputProps={{ min: 30, max: 3600 }} onChange={e => setForm({ ...form, interval_seconds: Number(e.target.value) })} />
            <Button variant="contained" startIcon={<Add/>} onClick={addTarget} disabled={!form.name || !form.address}>
              {translate('pages.operations.form.add')}
            </Button>
          </Stack>
        </CardContent>
      </Card>

      <Alert severity="info">
        {translate('pages.operations.snmp_readonly_notice')}
      </Alert>

      {/* Incidents Card */}
      <Card variant="outlined">
        <CardContent>
          <Typography variant="h6" fontWeight={700} sx={{ mb: 1 }}>{translate('pages.operations.incidents_title')}</Typography>
          {incidentsQuery.isLoading ? (
            <LinearProgress />
          ) : incidents.length === 0 ? (
            <Typography color="text.secondary">{translate('pages.operations.no_incidents')}</Typography>
          ) : (
            incidents.map((incident, i) => (
              <Box key={String(incident.id ?? i)} sx={{ py: 1.25, borderBottom: '1px solid', borderColor: 'divider' }}>
                <Stack direction="row" alignItems="center" spacing={1}>
                  <Chip
                    size="small"
                    color={incident.resolved_at ? 'success' : 'error'}
                    label={incident.resolved_at ? translate('pages.operations.resolved') : translate('pages.operations.ongoing')}
                    sx={{ height: 20, fontSize: '0.68rem', fontWeight: 700 }}
                  />
                  <Typography fontWeight={600}>{String(incident.summary ?? '')}</Typography>
                  {incident.target_id !== undefined && incident.target_id !== null && (
                    <Button size="small" sx={{ ml: 'auto' }} onClick={() => setImpactTarget(String(incident.target_id))}>
                      {translate('pages.operations.view_impact')}
                    </Button>
                  )}
                </Stack>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
                  {translate('pages.operations.incident_started', { time: new Date(String(incident.started_at)).toLocaleString() })} · {incident.resolved_at ? translate('pages.operations.incident_resolved', { time: new Date(String(incident.resolved_at)).toLocaleString() }) : translate('pages.operations.incident_unresolved')}
                </Typography>
              </Box>
            ))
          )}
        </CardContent>
      </Card>
    </Stack>}

    {/* Tab 1: MRTG Interface Telemetry Graph */}
    {tab === 1 && (
      <Stack spacing={2.5}>
        <Card variant="outlined" sx={{ p: 2 }}>
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems="center" justifyContent="space-between">
            <Box>
              <Typography variant="h6" fontWeight={800}>{translate('pages.operations.mrtg.device_title')}</Typography>
              <Typography variant="caption" color="text.secondary">{translate('pages.operations.mrtg.device_desc')}</Typography>
            </Box>
            <TextField
              select
              size="small"
              sx={{ minWidth: 260 }}
              label={translate('pages.operations.mrtg.monitored_device')}
              value={mrtgTargetId || (targets[0]?.id ?? '')}
              onChange={(e) => setMrtgTargetId(e.target.value)}
            >
              {targets.map((t) => (
                <MenuItem key={t.id} value={t.id}>
                  {t.name} ({t.address}) — {t.probe_type.toUpperCase()}
                </MenuItem>
              ))}
            </TextField>
          </Stack>
        </Card>

        {targets.length > 0 ? (
          <MrtgTrafficGraph
            endpoint={`/network/monitor-targets/${mrtgTargetId || targets[0].id}/traffic`}
            showInterfaceSelector
            title={translate('pages.operations.mrtg.title', { name: targets.find(t => t.id === (mrtgTargetId || targets[0].id))?.name || translate('pages.operations.mrtg.device_fallback') })}
          />
        ) : (
          <Alert severity="info">{translate('pages.operations.mrtg.no_targets')}</Alert>
        )}
      </Stack>
    )}

    {/* Tab 2: Syslog Stream */}
    {tab === 2 && (
      <SyslogPanel
        severity={syslogSeverity}
        setSeverity={setSyslogSeverity}
        tag={syslogTag}
        setTag={setSyslogTag}
        q={syslogQ}
        setQ={setSyslogQ}
        autoRefresh={syslogAutoRefresh}
        setAutoRefresh={setSyslogAutoRefresh}
      />
    )}

    {/* Tab 3: WhatsApp Alerts */}
    {tab === 3 && <Stack spacing={2}>
      <Alert severity="warning">{translate('pages.operations.whatsapp.risk_notice')}</Alert>
      <Card variant="outlined"><CardContent><Stack direction="row" alignItems="center" spacing={1}><NotificationsActive color="primary"/><Typography variant="h6">{translate('pages.operations.whatsapp.settings_title')}</Typography></Stack>
        <FormControlLabel sx={{ mt: 1 }} control={<Switch checked={enabled} onChange={e => setEnabled(e.target.checked)}/>} label={translate('pages.operations.whatsapp.enable')}/>
        <FormControlLabel control={<Switch checked={riskAck} onChange={e => setRiskAck(e.target.checked)}/>} label={translate('pages.operations.whatsapp.risk_ack')}/>
        <TextField fullWidth sx={{ mt: 1 }} label={translate('pages.operations.whatsapp.recipients')} helperText={translate('pages.operations.whatsapp.recipients_helper')} value={recipientsText} onChange={e => setRecipientsText(e.target.value)}/>
        <Typography variant="subtitle2" sx={{ mt: 2 }}>{translate('pages.operations.whatsapp.alert_events')}</Typography><Stack direction="row" useFlexGap flexWrap="wrap">{eventOptions.map(([value, labelKey]) => <FormControlLabel key={value} control={<Switch size="small" checked={events.includes(value)} onChange={e => setEvents(prev => e.target.checked ? [...prev, value] : prev.filter(x => x !== value))}/>} label={translate(`pages.operations.${labelKey}`)}/>)}</Stack>
        <Button variant="contained" onClick={saveNotify} disabled={mutation.isPending}>{translate('pages.operations.whatsapp.save')}</Button>
      </CardContent></Card>
      <Card variant="outlined"><CardContent><Stack direction="row" spacing={1} alignItems="center"><QrCode2 color="primary"/><Typography variant="h6">{translate('pages.operations.whatsapp.pairing_title')}</Typography><Chip size="small" label={settings?.whatsapp.state ? translate(`pages.operations.whatsapp.state.${settings.whatsapp.state}`, { _: settings.whatsapp.state }) : translate('pages.operations.whatsapp.loading')} color={settings?.whatsapp.logged_in ? 'success' : 'default'}/></Stack>
        {!canControlWhatsApp && <Alert severity="info" sx={{ mt: 1 }}>{translate('pages.operations.whatsapp.admin_only')}</Alert>}
        {settings?.whatsapp.account && <Typography variant="body2" sx={{ mt: 1 }}>{translate('pages.operations.whatsapp.linked_account', { account: settings.whatsapp.account })}</Typography>}
        {settings?.whatsapp.qr_image && <Box component="img" src={settings.whatsapp.qr_image} alt={translate('pages.operations.whatsapp.qr_alt')} sx={{ display: 'block', width: 260, maxWidth: '100%', my: 2, bgcolor: 'white', p: 1, borderRadius: 1 }}/ >}
        {settings?.whatsapp.qr_expires && <Typography variant="caption">{translate('pages.operations.whatsapp.qr_expires', { time: new Date(settings.whatsapp.qr_expires).toLocaleTimeString() })}</Typography>}
        <Stack direction="row" spacing={1} sx={{ mt: 1 }}><Button variant="outlined" startIcon={<QrCode2/>} disabled={!canControlWhatsApp || !enabled || !riskAck || mutation.isPending} onClick={() => mutation.mutate({ path: '/system/notifications/whatsapp/pair', method: 'POST' })}>{translate('pages.operations.whatsapp.start_pairing')}</Button><Button color="error" disabled={!canControlWhatsApp || mutation.isPending} onClick={() => mutation.mutate({ path: '/system/notifications/whatsapp/disconnect', method: 'POST' })}>{translate('pages.operations.whatsapp.unlink')}</Button><Button startIcon={<Send/>} disabled={!selectedRecipient} onClick={() => mutation.mutate({ path: '/system/notifications/whatsapp/test', method: 'POST', body: { recipient: selectedRecipient } })}>{translate('pages.operations.whatsapp.send_test')}</Button></Stack>
        {settings?.recipients?.length ? <TextField select size="small" sx={{ mt: 2, minWidth: 260 }} label={translate('pages.operations.whatsapp.test_recipient')} value={selectedRecipient} onChange={e => setSelectedRecipient(e.target.value)}>{settings.recipients.map(number => <MenuItem key={number} value={number}>{number}</MenuItem>)}</TextField> : <Alert severity="info" sx={{ mt: 2 }}>{translate('pages.operations.whatsapp.save_recipient_first')}</Alert>}
      </CardContent></Card>
      <Card variant="outlined"><CardContent><Typography variant="h6" sx={{ mb: 1 }}>{translate('pages.operations.whatsapp.delivery_title')}</Typography>{outboxItems.length === 0 ? <Typography color="text.secondary">{translate('pages.operations.whatsapp.no_deliveries')}</Typography> : <Stack spacing={0.75}>{outboxItems.map(item => <Box key={item.id} sx={{ display: 'flex', gap: 1, alignItems: 'center', flexWrap: 'wrap', py: 0.75, borderBottom: '1px solid', borderColor: 'divider' }}><Chip size="small" label={translate(`pages.operations.delivery_status.${item.status}`, { _: item.status.toUpperCase() })} color={item.status === 'sent' ? 'success' : item.status === 'failed' ? 'error' : 'default'}/><Typography variant="body2" sx={{ flex: '1 1 160px' }}>{translate(`pages.operations.event.${item.event_type.replace(/\./g, '_')}`, { _: item.event_type })} → {item.recipient}</Typography><Typography variant="caption" color="text.secondary">{item.sent_at ? translate('pages.operations.whatsapp.sent_at', { time: new Date(item.sent_at).toLocaleString() }) : translate('pages.operations.whatsapp.attempts', { count: item.attempts, time: new Date(item.created_at).toLocaleString() })}</Typography>{item.last_error && <Typography variant="caption" color="error.main">{item.last_error}</Typography>}</Box>)}</Stack>}</CardContent></Card>
      <Divider/><Typography variant="caption" color="text.secondary">{translate('pages.operations.whatsapp.queue_note')}</Typography>
    </Stack>}
  </Box>
  );
};

interface SyslogEventItem {
  id: string;
  nas_ip: string;
  facility: number;
  severity: number;
  severity_name: string;
  tag: string;
  message: string;
  created_at: string;
}

const SyslogPanel = ({
  severity,
  setSeverity,
  tag,
  setTag,
  q,
  setQ,
  autoRefresh,
  setAutoRefresh,
}: {
  severity: string;
  setSeverity: (s: string) => void;
  tag: string;
  setTag: (t: string) => void;
  q: string;
  setQ: (q: string) => void;
  autoRefresh: boolean;
  setAutoRefresh: (b: boolean | ((p: boolean) => boolean)) => void;
}) => {
  const notify = useNotify();
  const translate = useTranslate();
  const query = useQuery({
    queryKey: ['syslog', severity, tag, q],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (severity !== '') params.set('severity', severity);
      if (tag.trim()) params.set('tag', tag.trim());
      if (q.trim()) params.set('q', q.trim());
      params.set('perPage', '50');
      const res = await apiRequest<unknown>(`/network/syslog?${params.toString()}`);
      return extractData<SyslogEventItem[]>(res) ?? [];
    },
    refetchInterval: autoRefresh ? 5000 : false,
  });

  const sendTestLog = useMutation({
    mutationFn: () =>
      apiRequest('/network/syslog/test', {
        method: 'POST',
        body: JSON.stringify({
          nas_ip: '192.168.88.1',
          severity: 6,
          tag: 'pppoe,info',
          message: 'PPPoE session connected: user test01 authenticated via RADIUS',
        }),
      }),
    onSuccess: () => {
      notify(translate('pages.operations.syslog.test_success'), { type: 'success' });
      void query.refetch();
    },
    onError: (err: any) => notify(err.message || translate('pages.operations.syslog.test_failed'), { type: 'error' }),
  });

  const logs = query.data ?? [];

  const getSeverityChip = (sev: number, name: string) => {
    let color: 'error' | 'warning' | 'info' | 'default' = 'default';
    if (sev <= 3) color = 'error';
    else if (sev === 4) color = 'warning';
    else if (sev <= 6) color = 'info';
    return (
      <Chip
        size="small"
        color={color}
        label={translate(`pages.operations.syslog.level.${name}`, { _: name.toUpperCase() })}
        sx={{ fontWeight: 700, fontSize: '0.68rem', height: 20 }}
      />
    );
  };

  return (
    <Stack spacing={2}>
      <Card variant="outlined">
        <CardContent sx={{ p: 2 }}>
          <Stack
            direction={{ xs: 'column', md: 'row' }}
            spacing={1.5}
            alignItems={{ xs: 'stretch', md: 'center' }}
            justifyContent="space-between"
          >
            <Stack direction="row" spacing={1.5} alignItems="center" flexWrap="wrap">
              <TextField
                size="small"
                label={translate('pages.operations.syslog.search_messages')}
                placeholder={translate('pages.operations.syslog.search_placeholder')}
                value={q}
                onChange={(e) => setQ(e.target.value)}
                sx={{ minWidth: 200 }}
              />
              <TextField
                size="small"
                label={translate('pages.operations.syslog.tag_app')}
                placeholder={translate('pages.operations.syslog.tag_placeholder')}
                value={tag}
                onChange={(e) => setTag(e.target.value)}
                sx={{ width: 140 }}
              />
              <TextField
                select
                size="small"
                label={translate('pages.operations.syslog.severity')}
                value={severity}
                onChange={(e) => setSeverity(e.target.value)}
                sx={{ width: 130 }}
              >
                <MenuItem value="">{translate('pages.operations.syslog.all_severities')}</MenuItem>
                <MenuItem value="3">{translate('pages.operations.syslog.error')}</MenuItem>
                <MenuItem value="4">{translate('pages.operations.syslog.warning')}</MenuItem>
                <MenuItem value="5">{translate('pages.operations.syslog.notice')}</MenuItem>
                <MenuItem value="6">{translate('pages.operations.syslog.info')}</MenuItem>
                <MenuItem value="7">{translate('pages.operations.syslog.debug')}</MenuItem>
              </TextField>
            </Stack>

            <Stack direction="row" spacing={1} alignItems="center" justifyContent="flex-end">
              <Tooltip title={autoRefresh ? translate('pages.operations.syslog.live_active') : translate('pages.operations.syslog.paused_tooltip')}>
                <Button
                  size="small"
                  variant={autoRefresh ? 'contained' : 'outlined'}
                  color={autoRefresh ? 'primary' : 'inherit'}
                  startIcon={<Speed />}
                  onClick={() => setAutoRefresh((prev) => !prev)}
                >
                  {autoRefresh ? translate('pages.operations.syslog.live_button') : translate('pages.operations.syslog.paused_button')}
                </Button>
              </Tooltip>

              <Button
                size="small"
                variant="outlined"
                startIcon={<Refresh />}
                onClick={() => void query.refetch()}
                disabled={query.isFetching}
              >
                {translate('pages.operations.refresh')}
              </Button>

              <Button
                size="small"
                variant="outlined"
                color="secondary"
                startIcon={<Send />}
                onClick={() => sendTestLog.mutate()}
                disabled={sendTestLog.isPending}
              >
                {translate('pages.operations.syslog.test')}
              </Button>
            </Stack>
          </Stack>
        </CardContent>
      </Card>

      <Panel
        title={translate('pages.operations.syslog.stream_title')}
        subtitle={translate('pages.operations.syslog.event_count', { count: logs.length })}
        dense
      >
        {query.isLoading ? (
          <LinearProgress />
        ) : logs.length === 0 ? (
          <Box sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
            <Typography variant="body1" fontWeight={600}>{translate('pages.operations.syslog.no_events')}</Typography>
            <Typography variant="caption">
              {translate('pages.operations.syslog.configure_hint')} /system logging add action=remote topics=pppoe,account remote=&lt;server-ip&gt;:1514
            </Typography>
          </Box>
        ) : (
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell sx={{ width: 140 }}>{translate('pages.operations.syslog.timestamp')}</TableCell>
                <TableCell sx={{ width: 130 }}>{translate('pages.operations.syslog.device_ip')}</TableCell>
                <TableCell sx={{ width: 100 }}>{translate('pages.operations.syslog.severity')}</TableCell>
                <TableCell sx={{ width: 120 }}>{translate('pages.operations.syslog.tag')}</TableCell>
                <TableCell>{translate('pages.operations.syslog.message')}</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {logs.map((log) => (
                <TableRow key={log.id} hover>
                  <TableCell sx={{ fontSize: '0.78rem' }}>
                    <Mono>{new Date(log.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}</Mono>
                  </TableCell>
                  <TableCell>
                    <Mono>{log.nas_ip || '127.0.0.1'}</Mono>
                  </TableCell>
                  <TableCell>
                    {getSeverityChip(log.severity, log.severity_name)}
                  </TableCell>
                  <TableCell>
                    <Chip size="small" variant="outlined" label={log.tag} sx={{ fontFamily: 'monospace', fontSize: '0.7rem' }} />
                  </TableCell>
                  <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8rem', wordBreak: 'break-all' }}>
                    {log.message}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Panel>
    </Stack>
  );
};

export default OperationsPage;
