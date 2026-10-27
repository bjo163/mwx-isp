import React, { useState } from 'react';
import {
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  InputLabel,
  LinearProgress,
  MenuItem,
  Select,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material';
import {
  Add,
  Build,
  Edit,
  Refresh,
  WhatsApp,
} from '@mui/icons-material';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNotify, useTranslate } from 'react-admin';
import { useIsAdmin } from '../hooks/useIsAdmin';
import { apiRequest, extractData } from '../utils/apiClient';
import { KpiStrip, KpiTile, PageHeader, Panel, StatusChip } from '../components/Enterprise';

interface FlappingSubscriber {
  username: string;
  customer_no: string;
  customer_name: string;
  phone: string;
  address: string;
  odp_code: string;
  disconnect_count: number;
  last_terminate_cause: string;
  last_seen_time: string;
  severity: string;
  suggested_resolution: string;
}

interface Ticket {
  id: string;
  ticket_no: string;
  customer_id: string;
  subscription_id: string;
  subject: string;
  category: string;
  priority: string;
  status: string;
  assigned_technician: string;
  technician_phone: string;
  description: string;
  resolution_notes?: string;
  created_at: string;
  resolved_at?: string;
}

export const TroubleTicketsPage: React.FC = () => {
  const notify = useNotify();
  const translate = useTranslate();
  const queryClient = useQueryClient();
  const isAdmin = useIsAdmin();

  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [selectedTicket, setSelectedTicket] = useState<Ticket | null>(null);

  const [statusFilter, setStatusFilter] = useState('');
  const [priorityFilter, setPriorityFilter] = useState('');
  const [search, setSearch] = useState('');

  // Ticket create form
  const [form, setForm] = useState({
    subject: '',
    category: 'los_red',
    priority: 'normal',
    assigned_technician: '',
    technician_phone: '',
    description: '',
  });

  const [editForm, setEditForm] = useState({
    status: 'open',
    assigned_technician: '',
    technician_phone: '',
    resolution_notes: '',
  });

  const flappingQuery = useQuery({
    queryKey: ['network-diagnostics-flapping'],
    queryFn: async () => {
      const res = await apiRequest<any>('/network/diagnostics/flapping');
      return extractData<{ flapping_count: number; subscribers: FlappingSubscriber[] }>(res) || { flapping_count: 0, subscribers: [] };
    },
  });

  const autoTicketMutation = useMutation({
    mutationFn: async (flap: FlappingSubscriber) => {
      return apiRequest('/network/diagnostics/flapping/auto-ticket', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          username: flap.username,
          customer_no: flap.customer_no,
          description: `Intelligent Flapping: ${flap.disconnect_count}x putus/jam. ODP: ${flap.odp_code || '-'}, Alamat: ${flap.address || '-'}. ${flap.suggested_resolution}`,
        }),
      });
    },
    onSuccess: () => {
      notify(translate('pages.tickets.notify.auto_ticket_success'), { type: 'success' });
      void queryClient.invalidateQueries({ queryKey: ['isp', 'tickets'] });
      void queryClient.invalidateQueries({ queryKey: ['network-diagnostics-flapping'] });
    },
    onError: (err: any) => notify(err?.message || translate('pages.tickets.notify.auto_ticket_failed'), { type: 'error' }),
  });

  const ticketsQuery = useQuery({
    queryKey: ['isp', 'tickets', statusFilter, priorityFilter, search],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (statusFilter) params.set('status', statusFilter);
      if (priorityFilter) params.set('priority', priorityFilter);
      if (search.trim()) params.set('q', search.trim());
      params.set('perPage', '50');
      const res = await apiRequest<unknown>(`/isp/tickets?${params.toString()}`);
      return extractData<Ticket[]>(res) ?? [];
    },
  });

  const createMutation = useMutation({
    mutationFn: () =>
      apiRequest('/isp/tickets', {
        method: 'POST',
        body: JSON.stringify(form),
      }),
    onSuccess: () => {
      notify(translate('pages.tickets.notify.created'), { type: 'success' });
      setCreateOpen(false);
      setForm({
        subject: '',
        category: 'los_red',
        priority: 'normal',
        assigned_technician: '',
        technician_phone: '',
        description: '',
      });
      void queryClient.invalidateQueries({ queryKey: ['isp', 'tickets'] });
    },
    onError: (err: any) => notify(err.message || translate('pages.tickets.notify.create_failed'), { type: 'error' }),
  });

  const updateMutation = useMutation({
    mutationFn: () =>
      apiRequest(`/isp/tickets/${selectedTicket?.id}`, {
        method: 'PUT',
        body: JSON.stringify(editForm),
      }),
    onSuccess: () => {
      notify(translate('pages.tickets.notify.updated'), { type: 'success' });
      setEditOpen(false);
      void queryClient.invalidateQueries({ queryKey: ['isp', 'tickets'] });
    },
    onError: (err: any) => notify(err.message || translate('pages.tickets.notify.update_failed'), { type: 'error' }),
  });

  const dispatchMutation = useMutation({
    mutationFn: (ticketId: string) =>
      apiRequest(`/isp/tickets/${ticketId}/dispatch`, { method: 'POST' }),
    onSuccess: () => {
      notify(translate('pages.tickets.notify.dispatched'), { type: 'success' });
    },
    onError: (err: any) => notify(err.message || translate('pages.tickets.notify.dispatch_failed'), { type: 'error' }),
  });

  const tickets = ticketsQuery.data ?? [];

  const getPriorityChip = (p: string) => {
    let color: 'error' | 'warning' | 'info' | 'default' = 'default';
    if (p === 'urgent') color = 'error';
    else if (p === 'high') color = 'warning';
    else if (p === 'normal') color = 'info';
    return <Chip size="small" color={color} label={translate('pages.tickets.priority.' + p, { _: p }).toUpperCase()} sx={{ fontWeight: 700, fontSize: '0.68rem', height: 20 }} />;
  };


  const count = (pred: (t: Ticket) => boolean) => tickets.filter(pred).length;

  return (
    <Box sx={{ p: { xs: 1.5, md: 2.5 }, maxWidth: 1600, mx: 'auto' }}>
      <PageHeader
        section={translate('pages.tickets.section')}
        title={translate('pages.tickets.title')}
        subtitle={translate('pages.tickets.subtitle')}
        actions={<>
          <Button variant="outlined" startIcon={<Refresh />} onClick={() => void ticketsQuery.refetch()} disabled={ticketsQuery.isFetching}>{translate('pages.tickets.refresh')}</Button>
          <Button variant="contained" startIcon={<Add />} onClick={() => setCreateOpen(true)}>{translate('pages.tickets.new_ticket')}</Button>
        </>}
      />
      <KpiStrip>
        <KpiTile label={translate('pages.tickets.kpi.total')} value={tickets.length} icon={<Build fontSize="small" />} />
        <KpiTile label={translate('pages.tickets.status.open')} value={count((t) => t.status === 'open')} tone="error" hint={translate('pages.tickets.kpi.open_hint')} />
        <KpiTile label={translate('pages.tickets.status.in_progress')} value={count((t) => t.status === 'in_progress' || t.status === 'scheduled')} tone="warning" hint={translate('pages.tickets.kpi.in_progress_hint')} />
        <KpiTile label={translate('pages.tickets.kpi.urgent_high')} value={count((t) => t.priority === 'urgent' || t.priority === 'high')} tone="secondary" />
        <KpiTile label={translate('pages.tickets.unassigned')} value={count((t) => !t.assigned_technician)} tone="info" hint={translate('pages.tickets.kpi.unassigned_hint')} />
        <KpiTile label={translate('pages.tickets.status.resolved')} value={count((t) => t.status === 'resolved' || t.status === 'closed')} tone="success" />
      </KpiStrip>

      {flappingQuery.data && flappingQuery.data.flapping_count > 0 && (
        <Box sx={{ mb: 3 }}>
          <Panel
            title={translate('pages.tickets.flapping.title', { count: flappingQuery.data.flapping_count })}
            subtitle={translate('pages.tickets.flapping.subtitle')}
          >
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell sx={{ fontWeight: 800 }}>{translate('pages.tickets.flapping.col_subscriber')}</TableCell>
                  <TableCell sx={{ fontWeight: 800 }}>{translate('pages.tickets.flapping.col_customer_odp')}</TableCell>
                  <TableCell sx={{ fontWeight: 800 }}>{translate('pages.tickets.flapping.col_disconnect_freq')}</TableCell>
                  <TableCell sx={{ fontWeight: 800 }}>{translate('pages.tickets.flapping.col_severity')}</TableCell>
                  <TableCell sx={{ fontWeight: 800 }}>{translate('pages.tickets.flapping.col_analysis')}</TableCell>
                  <TableCell sx={{ fontWeight: 800, textAlign: 'right' }}>{translate('pages.tickets.flapping.col_action')}</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {flappingQuery.data.subscribers.map((flap) => (
                  <TableRow key={flap.username} hover>
                    <TableCell sx={{ fontFamily: 'monospace', fontWeight: 800, color: 'primary.main' }}>
                      {flap.username}
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" fontWeight={700}>
                        {flap.customer_name || translate('pages.tickets.flapping.subscriber_fallback')} ({flap.customer_no || '-'})
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {translate('pages.tickets.flapping.odp_line', {
                          odp: flap.odp_code || translate('pages.tickets.unassigned'),
                          address: flap.address || '-',
                        })}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Chip
                        size="small"
                        color="error"
                        label={translate('pages.tickets.flapping.drops_per_hour', { count: flap.disconnect_count })}
                        sx={{ fontWeight: 800, fontFamily: 'monospace' }}
                      />
                    </TableCell>
                    <TableCell>
                      <Chip
                        size="small"
                        color={flap.severity === 'urgent' ? 'error' : 'warning'}
                        label={translate('pages.tickets.severity.' + flap.severity, { _: flap.severity }).toUpperCase()}
                        sx={{ fontWeight: 800 }}
                      />
                    </TableCell>
                    <TableCell>
                      <Typography variant="caption" color="text.secondary">
                        {flap.suggested_resolution}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      {isAdmin && (
                        <Button
                          size="small"
                          variant="contained"
                          color="error"
                          startIcon={<Build sx={{ fontSize: 16 }} />}
                          onClick={() => autoTicketMutation.mutate(flap)}
                          disabled={autoTicketMutation.isPending}
                          sx={{ textTransform: 'none', fontWeight: 700 }}
                        >
                          {translate('pages.tickets.flapping.auto_ticket')}
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Panel>
        </Box>
      )}

      <Panel title={translate('pages.tickets.queue.title')} subtitle={translate('pages.tickets.queue.subtitle', { count: tickets.length })} dense actions={
        <Box>
          <Stack
            direction={{ xs: 'column', md: 'row' }}
            spacing={2}
            alignItems={{ xs: 'stretch', md: 'center' }}
            justifyContent="space-between"
          >
            <Stack direction="row" spacing={1.5} alignItems="center" flexWrap="wrap">
              <FormControl size="small" sx={{ minWidth: 140 }}>
                <InputLabel>{translate('pages.tickets.fields.status')}</InputLabel>
                <Select
                  label={translate('pages.tickets.fields.status')}
                  value={statusFilter}
                  onChange={(e) => setStatusFilter(e.target.value)}
                >
                  <MenuItem value="">{translate('pages.tickets.filters.all_statuses')}</MenuItem>
                  <MenuItem value="open">{translate('pages.tickets.status.open')}</MenuItem>
                  <MenuItem value="scheduled">{translate('pages.tickets.status.scheduled')}</MenuItem>
                  <MenuItem value="in_progress">{translate('pages.tickets.status.in_progress')}</MenuItem>
                  <MenuItem value="resolved">{translate('pages.tickets.status.resolved')}</MenuItem>
                </Select>
              </FormControl>

              <FormControl size="small" sx={{ minWidth: 130 }}>
                <InputLabel>{translate('pages.tickets.fields.priority')}</InputLabel>
                <Select
                  label={translate('pages.tickets.fields.priority')}
                  value={priorityFilter}
                  onChange={(e) => setPriorityFilter(e.target.value)}
                >
                  <MenuItem value="">{translate('pages.tickets.filters.all_priorities')}</MenuItem>
                  <MenuItem value="urgent">{translate('pages.tickets.priority.urgent')}</MenuItem>
                  <MenuItem value="high">{translate('pages.tickets.priority.high')}</MenuItem>
                  <MenuItem value="normal">{translate('pages.tickets.priority.normal')}</MenuItem>
                  <MenuItem value="low">{translate('pages.tickets.priority.low')}</MenuItem>
                </Select>
              </FormControl>

              <TextField
                size="small"
                label={translate('pages.tickets.filters.search')}
                placeholder={translate('pages.tickets.filters.search_placeholder')}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                sx={{ width: 160 }}
              />
            </Stack>
          </Stack>
        </Box>
      }>
          {ticketsQuery.isLoading ? (
            <LinearProgress />
          ) : ticketsQuery.isError ? (
            <Box sx={{ p: 4, textAlign: 'center' }}>
              <Typography variant="h6" color="error.main">{translate('pages.tickets.load_error.title')}</Typography>
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 2 }}>
                {(ticketsQuery.error as Error)?.message || translate('pages.tickets.load_error.fallback')}
              </Typography>
              <Button variant="outlined" onClick={() => void ticketsQuery.refetch()}>{translate('pages.tickets.retry')}</Button>
            </Box>
          ) : tickets.length === 0 ? (
            <Box sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
              <Build sx={{ fontSize: 48, opacity: 0.4, mb: 1 }} />
              <Typography variant="h6">{translate('pages.tickets.empty.title')}</Typography>
              <Typography variant="caption">{translate('pages.tickets.empty.subtitle')}</Typography>
            </Box>
          ) : (
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>{translate('pages.tickets.fields.ticket_no')}</TableCell>
                  <TableCell>{translate('pages.tickets.fields.subject')}</TableCell>
                  <TableCell>{translate('pages.tickets.fields.category')}</TableCell>
                  <TableCell>{translate('pages.tickets.fields.priority')}</TableCell>
                  <TableCell>{translate('pages.tickets.fields.status')}</TableCell>
                  <TableCell>{translate('pages.tickets.fields.technician')}</TableCell>
                  <TableCell>{translate('pages.tickets.fields.created')}</TableCell>
                  <TableCell align="right">{translate('pages.tickets.fields.actions')}</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {tickets.map((t) => (
                  <TableRow key={t.id} hover>
                    <TableCell sx={{ fontFamily: 'monospace', fontWeight: 800, color: 'primary.main' }}>
                      {t.ticket_no}
                    </TableCell>
                    <TableCell sx={{ fontWeight: 600 }}>{t.subject}</TableCell>
                    <TableCell sx={{ textTransform: 'capitalize' }}>
                      {t.category.replace(/_/g, ' ')}
                    </TableCell>
                    <TableCell>{getPriorityChip(t.priority)}</TableCell>
                    <TableCell><StatusChip value={t.status} label={translate('pages.tickets.status.' + t.status, { _: t.status }).toUpperCase()} /></TableCell>
                    <TableCell>
                      {t.assigned_technician ? (
                        <span>{t.assigned_technician}</span>
                      ) : (
                        <Typography variant="caption" color="text.secondary">{translate('pages.tickets.unassigned')}</Typography>
                      )}
                    </TableCell>
                    <TableCell sx={{ fontSize: '0.8rem', color: 'text.secondary' }}>
                      {new Date(t.created_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell align="right">
                      <Stack direction="row" spacing={0.5} justifyContent="flex-end">
                        {isAdmin && t.technician_phone && (
                          <Button
                            size="small"
                            color="success"
                            variant="outlined"
                            startIcon={<WhatsApp />}
                            disabled={dispatchMutation.isPending}
                            onClick={() => dispatchMutation.mutate(t.id)}
                            sx={{ fontSize: '0.72rem', px: 1 }}
                          >
                            {translate('pages.tickets.row_actions.dispatch_wa')}
                          </Button>
                        )}
                        <Button
                          size="small"
                          variant="outlined"
                          startIcon={<Edit />}
                          onClick={() => {
                            setSelectedTicket(t);
                            setEditForm({
                              status: t.status,
                              assigned_technician: t.assigned_technician,
                              technician_phone: t.technician_phone,
                              resolution_notes: t.resolution_notes || '',
                            });
                            setEditOpen(true);
                          }}
                          sx={{ fontSize: '0.72rem', px: 1 }}
                        >
                          {translate('pages.tickets.row_actions.update')}
                        </Button>
                      </Stack>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
      </Panel>

      {/* Create Ticket Modal */}
      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle sx={{ fontWeight: 800 }}>{translate('pages.tickets.create.title')}</DialogTitle>
        <DialogContent dividers>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <TextField
              size="small"
              label={translate('pages.tickets.create.subject')}
              placeholder={translate('pages.tickets.create.subject_placeholder')}
              value={form.subject}
              onChange={(e) => setForm({ ...form, subject: e.target.value })}
              required
              fullWidth
            />
            <FormControl size="small" fullWidth>
              <InputLabel>{translate('pages.tickets.fields.category')}</InputLabel>
              <Select
                label={translate('pages.tickets.fields.category')}
                value={form.category}
                onChange={(e) => setForm({ ...form, category: e.target.value })}
              >
                <MenuItem value="los_red">{translate('pages.tickets.category.los_red')}</MenuItem>
                <MenuItem value="slow_speed">{translate('pages.tickets.category.slow_speed')}</MenuItem>
                <MenuItem value="no_internet">{translate('pages.tickets.category.no_internet')}</MenuItem>
                <MenuItem value="router_damage">{translate('pages.tickets.category.router_damage')}</MenuItem>
                <MenuItem value="relocation">{translate('pages.tickets.category.relocation')}</MenuItem>
                <MenuItem value="billing">{translate('pages.tickets.category.billing')}</MenuItem>
              </Select>
            </FormControl>
            <FormControl size="small" fullWidth>
              <InputLabel>{translate('pages.tickets.fields.priority')}</InputLabel>
              <Select
                label={translate('pages.tickets.fields.priority')}
                value={form.priority}
                onChange={(e) => setForm({ ...form, priority: e.target.value })}
              >
                <MenuItem value="urgent">{translate('pages.tickets.create.priority_option.urgent')}</MenuItem>
                <MenuItem value="high">{translate('pages.tickets.create.priority_option.high')}</MenuItem>
                <MenuItem value="normal">{translate('pages.tickets.create.priority_option.normal')}</MenuItem>
                <MenuItem value="low">{translate('pages.tickets.create.priority_option.low')}</MenuItem>
              </Select>
            </FormControl>
            <Stack direction="row" spacing={1.5}>
              <TextField
                size="small"
                label={translate('pages.tickets.create.assigned_technician')}
                placeholder={translate('pages.tickets.create.assigned_technician_placeholder')}
                value={form.assigned_technician}
                onChange={(e) => setForm({ ...form, assigned_technician: e.target.value })}
                sx={{ flex: 1 }}
              />
              <TextField
                size="small"
                label={translate('pages.tickets.create.technician_phone')}
                placeholder={translate('pages.tickets.create.phone_placeholder')}
                value={form.technician_phone}
                onChange={(e) => setForm({ ...form, technician_phone: e.target.value })}
                sx={{ flex: 1 }}
              />
            </Stack>
            <TextField
              size="small"
              label={translate('pages.tickets.create.description')}
              multiline
              rows={3}
              placeholder={translate('pages.tickets.create.description_placeholder')}
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
              fullWidth
            />
          </Stack>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button onClick={() => setCreateOpen(false)}>{translate('pages.tickets.cancel')}</Button>
          <Button
            variant="contained"
            disabled={!form.subject || createMutation.isPending}
            onClick={() => createMutation.mutate()}
          >
            {createMutation.isPending ? translate('pages.tickets.create.saving') : translate('pages.tickets.create.submit')}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Edit / Resolve Ticket Modal */}
      <Dialog open={editOpen} onClose={() => setEditOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle sx={{ fontWeight: 800 }}>{translate('pages.tickets.edit.title', { ticketNo: selectedTicket?.ticket_no ?? '' })}</DialogTitle>
        <DialogContent dividers>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <FormControl size="small" fullWidth>
              <InputLabel>{translate('pages.tickets.fields.status')}</InputLabel>
              <Select
                label={translate('pages.tickets.fields.status')}
                value={editForm.status}
                onChange={(e) => setEditForm({ ...editForm, status: e.target.value })}
              >
                <MenuItem value="open">{translate('pages.tickets.status.open')}</MenuItem>
                <MenuItem value="scheduled">{translate('pages.tickets.status.scheduled')}</MenuItem>
                <MenuItem value="in_progress">{translate('pages.tickets.status.in_progress')}</MenuItem>
                <MenuItem value="resolved">{translate('pages.tickets.status.resolved')}</MenuItem>
                <MenuItem value="closed">{translate('pages.tickets.status.closed')}</MenuItem>
              </Select>
            </FormControl>
            <Stack direction="row" spacing={1.5}>
              <TextField
                size="small"
                label={translate('pages.tickets.edit.technician')}
                value={editForm.assigned_technician}
                onChange={(e) => setEditForm({ ...editForm, assigned_technician: e.target.value })}
                sx={{ flex: 1 }}
              />
              <TextField
                size="small"
                label={translate('pages.tickets.edit.technician_wa')}
                value={editForm.technician_phone}
                onChange={(e) => setEditForm({ ...editForm, technician_phone: e.target.value })}
                sx={{ flex: 1 }}
              />
            </Stack>
            <TextField
              size="small"
              label={translate('pages.tickets.edit.resolution_notes')}
              multiline
              rows={3}
              placeholder={translate('pages.tickets.edit.resolution_notes_placeholder')}
              value={editForm.resolution_notes}
              onChange={(e) => setEditForm({ ...editForm, resolution_notes: e.target.value })}
              fullWidth
            />
          </Stack>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button onClick={() => setEditOpen(false)}>{translate('pages.tickets.cancel')}</Button>
          <Button
            variant="contained"
            disabled={updateMutation.isPending}
            onClick={() => updateMutation.mutate()}
          >
            {updateMutation.isPending ? translate('pages.tickets.edit.updating') : translate('pages.tickets.edit.save')}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default TroubleTicketsPage;
