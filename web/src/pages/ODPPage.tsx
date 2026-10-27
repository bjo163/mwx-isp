import React, { useState } from 'react';
import {
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Drawer,
  FormControl,
  IconButton,
  InputLabel,
  LinearProgress,
  Paper,
  MenuItem,
  Select,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import {
  Add,
  DeleteOutline,
  EditOutlined,
  FiberManualRecord,
  Hub,
  LocationOn,
  MyLocation,
  OpenInNew,
  PeopleAltOutlined,
  Refresh,
  Search,
  WarningAmber,
} from '@mui/icons-material';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNotify, useTranslate } from 'react-admin';
import { apiRequest, extractData } from '../utils/apiClient';
import { NocImpactPanel } from '../components/NocImpactPanel';
import { KpiStrip, KpiTile, Mono, PageHeader, Panel, QueryFeedback, StatusChip } from '../components/Enterprise';

interface CustomerInfo {
  id: number;
  customer_no: string;
  name: string;
  phone: string;
  address: string;
  odp_port: number;
  status: string;
}

interface ODPRecord {
  id: number;
  code: string;
  name: string;
  zone: string;
  olt_name: string;
  pon_port: string;
  total_ports: number;
  used_ports: number;
  optical_loss: number;
  status: string;
  latitude: number;
  longitude: number;
  address: string;
  notes: string;
  created_at: string;
  updated_at: string;
  customers?: CustomerInfo[];
}

interface ODPDetailResponse {
  odp: ODPRecord;
  connected_count: number;
  customers: CustomerInfo[];
}

export const ODPPage: React.FC = () => {
  const notify = useNotify();
  const translate = useTranslate();
  const queryClient = useQueryClient();

  const [search, setSearch] = useState('');
  const [zoneFilter, setZoneFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');

  // Dialog State
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingODP, setEditingODP] = useState<ODPRecord | null>(null);

  // Customer Drawer State
  const [selectedODPId, setSelectedODPId] = useState<number | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);

  // Form State
  const [formData, setFormData] = useState({
    code: '',
    name: '',
    zone: '',
    olt_name: '',
    pon_port: '',
    total_ports: 16,
    optical_loss: -18.5,
    status: 'active',
    latitude: 0,
    longitude: 0,
    address: '',
    notes: '',
  });

  // Query ODP List
  const { data: odpData, isLoading, error: odpError, refetch } = useQuery<{ data: ODPRecord[]; total: number }>({
    queryKey: ['network-odps', search, zoneFilter, statusFilter],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (search) params.append('q', search);
      if (zoneFilter) params.append('zone', zoneFilter);
      if (statusFilter) params.append('status', statusFilter);
      const res = await apiRequest(`/network/odp?${params.toString()}`);
      return extractData(res);
    },
  });

  // Query ODP Detail for Drawer
  const { data: odpDetail, isLoading: isDetailLoading } = useQuery<ODPDetailResponse>({
    queryKey: ['network-odp-detail', selectedODPId],
    queryFn: async () => {
      if (!selectedODPId) return null as any;
      const res = await apiRequest(`/network/odp/${selectedODPId}`);
      return extractData(res);
    },
    enabled: !!selectedODPId,
  });

  // Create or Update ODP Mutation
  const saveMutation = useMutation({
    mutationFn: async (payload: typeof formData & { id?: number }) => {
      if (payload.id) {
        return apiRequest(`/network/odp/${payload.id}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        });
      }
      return apiRequest('/network/odp', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
    },
    onSuccess: () => {
      notify(
        editingODP
          ? translate('pages.odp.notify.updated')
          : translate('pages.odp.notify.created'),
        { type: 'success' }
      );
      setDialogOpen(false);
      setEditingODP(null);
      queryClient.invalidateQueries({ queryKey: ['network-odps'] });
    },
    onError: (err: any) => {
      notify(err?.message || translate('pages.odp.notify.save_failed'), { type: 'error' });
    },
  });

  // Delete ODP Mutation
  const deleteMutation = useMutation({
    mutationFn: async (id: number) => {
      return apiRequest(`/network/odp/${id}`, { method: 'DELETE' });
    },
    onSuccess: () => {
      notify(translate('pages.odp.notify.deleted'), { type: 'success' });
      queryClient.invalidateQueries({ queryKey: ['network-odps'] });
    },
    onError: (err: any) => {
      notify(err?.message || translate('pages.odp.notify.delete_failed'), { type: 'error' });
    },
  });

  const handleOpenCreate = () => {
    setEditingODP(null);
    setFormData({
      code: '',
      name: '',
      zone: 'Central',
      olt_name: 'OLT-ZTE-CORE-01',
      pon_port: 'gpon-olt_1/1/1',
      total_ports: 16,
      optical_loss: -19.2,
      status: 'active',
      latitude: -6.2088,
      longitude: 106.8456,
      address: '',
      notes: '',
    });
    setDialogOpen(true);
  };

  const handleOpenEdit = (odp: ODPRecord) => {
    setEditingODP(odp);
    setFormData({
      code: odp.code,
      name: odp.name,
      zone: odp.zone,
      olt_name: odp.olt_name,
      pon_port: odp.pon_port,
      total_ports: odp.total_ports,
      optical_loss: odp.optical_loss,
      status: odp.status,
      latitude: odp.latitude,
      longitude: odp.longitude,
      address: odp.address,
      notes: odp.notes,
    });
    setDialogOpen(true);
  };

  const handleOpenSubscribers = (id: number) => {
    setSelectedODPId(id);
    setDrawerOpen(true);
  };

  const handleSave = () => {
    if (!formData.code) {
      notify(translate('pages.odp.notify.code_required'), { type: 'warning' });
      return;
    }
    saveMutation.mutate({
      ...formData,
      id: editingODP?.id,
    });
  };

  const getCurrentLocation = () => {
    if (navigator.geolocation) {
      navigator.geolocation.getCurrentPosition(
        (pos) => {
          setFormData((prev) => ({
            ...prev,
            latitude: Number(pos.coords.latitude.toFixed(6)),
            longitude: Number(pos.coords.longitude.toFixed(6)),
          }));
          notify(translate('pages.odp.notify.gps_updated'), { type: 'info' });
        },
        () => {
          notify(translate('pages.odp.notify.gps_failed'), { type: 'warning' });
        }
      );
    }
  };

  const odps = odpData?.data || [];
  const totalODPs = odps.length;
  const totalCapacity = odps.reduce((acc, o) => acc + (o.total_ports || 0), 0);
  const totalConnected = odps.reduce((acc, o) => acc + (o.used_ports || 0), 0);
  const availablePorts = Math.max(0, totalCapacity - totalConnected);
  const highLossCount = odps.filter((o) => o.optical_loss < -24).length;

  return (
    <Box sx={{ p: { xs: 2, sm: 3 } }}>
      <PageHeader
        title={translate('pages.odp.title')}
        subtitle={translate('pages.odp.subtitle')}
        actions={
          <Stack direction="row" spacing={1.5}>
            <Button
              variant="outlined"
              startIcon={<Refresh />}
              onClick={() => refetch()}
              sx={{ fontWeight: 700 }}
            >
              {translate('pages.odp.refresh')}
            </Button>
            <Button
              variant="contained"
              startIcon={<Add />}
              onClick={handleOpenCreate}
              sx={{ fontWeight: 700 }}
            >
              {translate('pages.odp.add_enclosure')}
            </Button>
          </Stack>
        }
      />

      {/* KPI METRIC STRIP */}
      <KpiStrip>
        <KpiTile
          label={translate('pages.odp.kpi.total_odps')}
          value={totalODPs}
          hint={translate('pages.odp.kpi.total_odps_hint')}
        />
        <KpiTile
          label={translate('pages.odp.kpi.total_capacity')}
          value={totalCapacity}
          hint={translate('pages.odp.kpi.capacity_hint')}
        />
        <KpiTile
          label={translate('pages.odp.kpi.connected_subscribers')}
          value={totalConnected}
          hint={translate('pages.odp.kpi.free_ports_hint', { count: availablePorts })}
        />
        <KpiTile
          label={translate('pages.odp.kpi.high_loss')}
          value={highLossCount}
          hint={translate('pages.odp.kpi.high_loss_hint')}
          tone={highLossCount > 0 ? "error" : "success"}
        />
      </KpiStrip>

      {/* SEARCH AND FILTERS */}
      <Panel
        title={translate('pages.odp.registry_title')}
        actions={
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} alignItems="center">
            <TextField
              size="small"
              placeholder={translate('pages.odp.search_placeholder')}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              InputProps={{
                startAdornment: <Search sx={{ mr: 1, color: 'text.secondary', fontSize: 20 }} />,
              }}
              sx={{ minWidth: 260 }}
            />
            <FormControl size="small" sx={{ minWidth: 140 }}>
              <InputLabel>{translate('pages.odp.filters.zone')}</InputLabel>
              <Select
                value={zoneFilter}
                label={translate('pages.odp.filters.zone')}
                onChange={(e) => setZoneFilter(e.target.value)}
              >
                <MenuItem value="">{translate('pages.odp.filters.all_zones')}</MenuItem>
                <MenuItem value="Kuningan">Kuningan</MenuItem>
                <MenuItem value="Sudirman">Sudirman</MenuItem>
                <MenuItem value="BSD">BSD</MenuItem>
                <MenuItem value="Central">Central</MenuItem>
                <MenuItem value="Utara">Utara</MenuItem>
                <MenuItem value="Barat">Barat</MenuItem>
              </Select>
            </FormControl>
            <FormControl size="small" sx={{ minWidth: 140 }}>
              <InputLabel>{translate('pages.odp.filters.status')}</InputLabel>
              <Select
                value={statusFilter}
                label={translate('pages.odp.filters.status')}
                onChange={(e) => setStatusFilter(e.target.value)}
              >
                <MenuItem value="">{translate('pages.odp.filters.all_statuses')}</MenuItem>
                <MenuItem value="active">{translate('pages.odp.status.active')}</MenuItem>
                <MenuItem value="full">{translate('pages.odp.filters.full_no_ports')}</MenuItem>
                <MenuItem value="maintenance">{translate('pages.odp.status.maintenance')}</MenuItem>
              </Select>
            </FormControl>
          </Stack>
        }
      >
        <QueryFeedback
          loading={isLoading}
          error={odpError}
          isEmpty={odps.length === 0}
          onRetry={() => void refetch()}
          emptyIcon={<Hub sx={{ fontSize: 48 }} />}
          emptyTitle={translate('pages.odp.empty_title')}
          emptyHint={translate('pages.odp.empty_hint')}
          emptyAction={
            <Button variant="contained" startIcon={<Add />} onClick={handleOpenCreate}>
              {translate('pages.odp.create_first')}
            </Button>
          }
        >
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell sx={{ fontWeight: 800 }}>{translate('pages.odp.columns.code')}</TableCell>
                <TableCell sx={{ fontWeight: 800 }}>{translate('pages.odp.columns.name_zone')}</TableCell>
                <TableCell sx={{ fontWeight: 800 }}>{translate('pages.odp.columns.uplink')}</TableCell>
                <TableCell sx={{ fontWeight: 800, minWidth: 170 }}>{translate('pages.odp.columns.port_occupancy')}</TableCell>
                <TableCell sx={{ fontWeight: 800 }}>{translate('pages.odp.columns.optical_loss')}</TableCell>
                <TableCell sx={{ fontWeight: 800 }}>{translate('pages.odp.columns.status')}</TableCell>
                <TableCell sx={{ fontWeight: 800 }}>{translate('pages.odp.columns.coordinates')}</TableCell>
                <TableCell sx={{ fontWeight: 800, textAlign: 'right' }}>{translate('pages.odp.columns.actions')}</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {odps.map((row) => {
                const used = row.used_ports || 0;
                const total = row.total_ports || 16;
                const percent = Math.min(100, Math.round((used / total) * 100));
                const isHighLoss = row.optical_loss < -24;
                const isOptimal = row.optical_loss >= -21;

                return (
                  <TableRow key={row.id} hover>
                    <TableCell>
                      <Stack direction="row" spacing={1} alignItems="center">
                        <Hub sx={{ fontSize: 18, color: 'primary.main' }} />
                        <Mono sx={{ fontWeight: 800, fontSize: '0.85rem' }}>{row.code}</Mono>
                      </Stack>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" fontWeight={700}>
                        {row.name}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {translate('pages.odp.zone_value', { zone: row.zone || '-' })} {row.address ? `• ${row.address}` : ''}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" fontWeight={600}>
                        {row.olt_name || 'OLT-CORE'}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {translate('pages.odp.port_value', { port: row.pon_port || 'gpon-olt_1/1/1' })}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Box sx={{ width: '100%' }}>
                        <Stack direction="row" justifyContent="space-between" sx={{ mb: 0.5 }}>
                          <Typography variant="caption" fontWeight={700}>
                            {translate('pages.odp.ports_usage', { used, total })}
                          </Typography>
                          <Typography variant="caption" color="text.secondary">
                            {percent}%
                          </Typography>
                        </Stack>
                        <LinearProgress
                          variant="determinate"
                          value={percent}
                          color={percent >= 90 ? 'error' : percent >= 70 ? 'warning' : 'primary'}
                          sx={{ height: 6, borderRadius: 1 }}
                        />
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Tooltip
                        title={
                          isOptimal
                            ? translate('pages.odp.loss.optimal')
                            : isHighLoss
                            ? translate('pages.odp.loss.high')
                            : translate('pages.odp.loss.acceptable')
                        }
                      >
                        <Chip
                          size="small"
                          icon={
                            isHighLoss ? (
                              <WarningAmber sx={{ fontSize: '14px !important' }} />
                            ) : (
                              <FiberManualRecord sx={{ fontSize: '10px !important' }} />
                            )
                          }
                          label={`${row.optical_loss.toFixed(1)} dBm`}
                          color={isHighLoss ? 'error' : isOptimal ? 'success' : 'warning'}
                          variant="outlined"
                          sx={{ fontWeight: 700, fontFamily: 'monospace' }}
                        />
                      </Tooltip>
                    </TableCell>
                    <TableCell>
                      <StatusChip
                        status={
                          row.status === 'active'
                            ? 'active'
                            : row.status === 'full'
                            ? 'warning'
                            : 'error'
                        }
                        label={translate('pages.odp.status.' + row.status, { _: row.status }).toUpperCase()}
                      />
                    </TableCell>
                    <TableCell>
                      {row.latitude && row.longitude ? (
                        <Button
                          size="small"
                          variant="text"
                          startIcon={<LocationOn sx={{ fontSize: 16 }} />}
                          endIcon={<OpenInNew sx={{ fontSize: 14 }} />}
                          href={`https://www.google.com/maps?q=${row.latitude},${row.longitude}`}
                          target="_blank"
                          sx={{ textTransform: 'none', py: 0, px: 0.5, fontSize: '0.75rem' }}
                        >
                          {row.latitude.toFixed(4)}, {row.longitude.toFixed(4)}
                        </Button>
                      ) : (
                        <Typography variant="caption" color="text.disabled">
                          {translate('pages.odp.no_gps')}
                        </Typography>
                      )}
                    </TableCell>
                    <TableCell align="right">
                      <Stack direction="row" spacing={0.5} justifyContent="flex-end">
                        <Tooltip title={translate('pages.odp.actions.view_subscribers')}>
                          <IconButton
                            size="small"
                            color="primary"
                            onClick={() => handleOpenSubscribers(row.id)}
                          >
                            <PeopleAltOutlined fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title={translate('pages.odp.actions.edit')}>
                          <IconButton
                            size="small"
                            onClick={() => handleOpenEdit(row)}
                          >
                            <EditOutlined fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title={translate('pages.odp.actions.delete')}>
                          <IconButton
                            size="small"
                            color="error"
                            disabled={used > 0}
                            onClick={() => {
                              if (window.confirm(translate('pages.odp.actions.delete_confirm', { code: row.code }))) {
                                deleteMutation.mutate(row.id);
                              }
                            }}
                          >
                            <DeleteOutline fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </Stack>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </QueryFeedback>
      </Panel>

      {/* CREATE / EDIT DIALOG */}
      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ fontWeight: 800 }}>
          {editingODP
            ? translate('pages.odp.form.edit_title', { code: editingODP.code })
            : translate('pages.odp.form.create_title')}
        </DialogTitle>
        <DialogContent dividers>
          <Stack spacing={2.5} sx={{ pt: 1 }}>
            <Stack direction="row" spacing={2}>
              <TextField
                fullWidth
                label={translate('pages.odp.form.code')}
                placeholder="ODP-KNG-001"
                value={formData.code}
                onChange={(e) => setFormData({ ...formData, code: e.target.value.toUpperCase() })}
                required
                helperText={translate('pages.odp.form.code_helper')}
              />
              <TextField
                fullWidth
                label={translate('pages.odp.form.zone_area')}
                placeholder="Kuningan"
                value={formData.zone}
                onChange={(e) => setFormData({ ...formData, zone: e.target.value })}
              />
            </Stack>

            <TextField
              fullWidth
              label={translate('pages.odp.form.name')}
              placeholder="ODP Kuningan Barat Tiang 14"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            />

            <Stack direction="row" spacing={2}>
              <TextField
                fullWidth
                label={translate('pages.odp.form.olt_name')}
                placeholder="OLT-ZTE-CORE-01"
                value={formData.olt_name}
                onChange={(e) => setFormData({ ...formData, olt_name: e.target.value })}
              />
              <TextField
                fullWidth
                label={translate('pages.odp.form.pon_port')}
                placeholder="gpon-olt_1/1/3"
                value={formData.pon_port}
                onChange={(e) => setFormData({ ...formData, pon_port: e.target.value })}
              />
            </Stack>

            <Stack direction="row" spacing={2}>
              <FormControl fullWidth>
                <InputLabel>{translate('pages.odp.form.splitter_ratio')}</InputLabel>
                <Select
                  value={formData.total_ports}
                  label={translate('pages.odp.form.splitter_ratio')}
                  onChange={(e) => setFormData({ ...formData, total_ports: Number(e.target.value) })}
                >
                  <MenuItem value={8}>{translate('pages.odp.form.splitter_8')}</MenuItem>
                  <MenuItem value={16}>{translate('pages.odp.form.splitter_16')}</MenuItem>
                  <MenuItem value={24}>{translate('pages.odp.form.splitter_24')}</MenuItem>
                  <MenuItem value={32}>{translate('pages.odp.form.splitter_32')}</MenuItem>
                </Select>
              </FormControl>
              <TextField
                fullWidth
                label={translate('pages.odp.form.attenuation')}
                type="number"
                inputProps={{ step: '0.1' }}
                value={formData.optical_loss}
                onChange={(e) => setFormData({ ...formData, optical_loss: Number(e.target.value) })}
                helperText={translate('pages.odp.form.attenuation_hint')}
              />
            </Stack>

            <Stack direction="row" spacing={2} alignItems="center">
              <TextField
                fullWidth
                label={translate('pages.odp.form.latitude')}
                type="number"
                inputProps={{ step: '0.000001' }}
                value={formData.latitude}
                onChange={(e) => setFormData({ ...formData, latitude: Number(e.target.value) })}
              />
              <TextField
                fullWidth
                label={translate('pages.odp.form.longitude')}
                type="number"
                inputProps={{ step: '0.000001' }}
                value={formData.longitude}
                onChange={(e) => setFormData({ ...formData, longitude: Number(e.target.value) })}
              />
              <Button
                variant="outlined"
                startIcon={<MyLocation />}
                onClick={getCurrentLocation}
                sx={{ minWidth: 130, height: 40 }}
              >
                {translate('pages.odp.form.get_gps')}
              </Button>
            </Stack>

            <TextField
              fullWidth
              label={translate('pages.odp.form.address')}
              multiline
              rows={2}
              placeholder="Depan Ruko Niaga Blok B No. 12, Tiang PLN No. 44"
              value={formData.address}
              onChange={(e) => setFormData({ ...formData, address: e.target.value })}
            />

            <TextField
              fullWidth
              label={translate('pages.odp.form.notes')}
              multiline
              rows={2}
              placeholder={translate('pages.odp.form.notes_helper')}
              value={formData.notes}
              onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
            />

            <FormControl fullWidth>
              <InputLabel>{translate('pages.odp.form.status')}</InputLabel>
              <Select
                value={formData.status}
                label={translate('pages.odp.form.status')}
                onChange={(e) => setFormData({ ...formData, status: e.target.value })}
              >
                <MenuItem value="active">{translate('pages.odp.form.status_active')}</MenuItem>
                <MenuItem value="full">{translate('pages.odp.form.status_full')}</MenuItem>
                <MenuItem value="maintenance">{translate('pages.odp.form.status_maintenance')}</MenuItem>
              </Select>
            </FormControl>
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, py: 2 }}>
          <Button onClick={() => setDialogOpen(false)}>{translate('pages.odp.cancel')}</Button>
          <Button
            variant="contained"
            onClick={handleSave}
            loading={saveMutation.isPending}
            sx={{ fontWeight: 700 }}
          >
            {editingODP ? translate('pages.odp.save_changes') : translate('pages.odp.register')}
          </Button>
        </DialogActions>
      </Dialog>

      {/* CONNECTED SUBSCRIBERS DRAWER */}
      <Drawer
        anchor="right"
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        PaperProps={{ sx: { width: { xs: '100%', sm: 540 }, p: 3 } }}
      >
        {isDetailLoading || !odpDetail ? (
          <Box sx={{ py: 6, textAlign: 'center' }}>
            <LinearProgress sx={{ mb: 2 }} />
            <Typography variant="body2" color="text.secondary">
              {translate('pages.odp.drawer.loading')}
            </Typography>
          </Box>
        ) : (
          <Box>
            <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
              <Box>
                <Typography variant="overline" color="text.secondary" fontWeight={700}>
                  {translate('pages.odp.drawer.telemetry')}
                </Typography>
                <Typography variant="h5" fontWeight={800}>
                  {odpDetail.odp.code}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {odpDetail.odp.name} • {odpDetail.odp.zone}
                </Typography>
              </Box>
              <Chip
                label={translate('pages.odp.drawer.ports_chip', {
                  connected: odpDetail.connected_count,
                  total: odpDetail.odp.total_ports,
                })}
                color="primary"
                sx={{ fontWeight: 800, fontFamily: 'monospace' }}
              />
            </Stack>

            <Divider sx={{ my: 2 }} />

            <Stack spacing={1.5} sx={{ mb: 3 }}>
              <Typography variant="subtitle2" fontWeight={800}>
                {translate('pages.odp.drawer.specs')}
              </Typography>
              <Stack direction="row" spacing={3}>
                <Box>
                  <Typography variant="caption" color="text.secondary">{translate('pages.odp.drawer.olt_uplink')}</Typography>
                  <Typography variant="body2" fontWeight={700}>{odpDetail.odp.olt_name}</Typography>
                </Box>
                <Box>
                  <Typography variant="caption" color="text.secondary">{translate('pages.odp.drawer.pon_port')}</Typography>
                  <Typography variant="body2" fontWeight={700}>{odpDetail.odp.pon_port}</Typography>
                </Box>
                <Box>
                  <Typography variant="caption" color="text.secondary">{translate('pages.odp.drawer.attenuation')}</Typography>
                  <Typography variant="body2" fontWeight={700} color={odpDetail.odp.optical_loss < -24 ? 'error.main' : 'success.main'}>
                    {odpDetail.odp.optical_loss} dBm
                  </Typography>
                </Box>
              </Stack>
            </Stack>

            <Typography variant="subtitle2" fontWeight={800} sx={{ mb: 1.5 }}>
              {translate('pages.odp.drawer.subscriber_impact')}
            </Typography>
            <Box sx={{ mb: 3 }}>
              <NocImpactPanel odpId={selectedODPId ?? undefined} />
            </Box>

            <Typography variant="subtitle2" fontWeight={800} sx={{ mb: 1.5 }}>
              {translate('pages.odp.drawer.connected_subscribers', { count: odpDetail.customers.length })}
            </Typography>

            {odpDetail.customers.length === 0 ? (
              <Box sx={{ py: 4, textAlign: 'center', bgcolor: 'action.hover', borderRadius: 2 }}>
                <Typography variant="body2" color="text.secondary">
                  {translate('pages.odp.drawer.no_subscribers')}
                </Typography>
              </Box>
            ) : (
              <Stack spacing={1.5}>
                {odpDetail.customers.map((cust) => (
                  <Paper key={cust.id} variant="outlined" sx={{ p: 2, borderRadius: 2 }}>
                    <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                      <Box>
                        <Stack direction="row" spacing={1} alignItems="center">
                          <Chip
                            size="small"
                            label={translate('pages.odp.drawer.port_chip', { port: cust.odp_port || 1 })}
                            color="info"
                            variant="outlined"
                            sx={{ fontWeight: 800, fontFamily: 'monospace', height: 20 }}
                          />
                          <Typography variant="subtitle2" fontWeight={800}>
                            {cust.name}
                          </Typography>
                        </Stack>
                        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
                          {translate('pages.odp.drawer.customer_meta', { id: cust.customer_no, phone: cust.phone || '-' })}
                        </Typography>
                        <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
                          {translate('pages.odp.drawer.address', { address: cust.address || '-' })}
                        </Typography>
                      </Box>
                      <StatusChip
                        status={cust.status === 'active' ? 'active' : 'warning'}
                        label={translate('pages.odp.status.' + cust.status, { _: cust.status }).toUpperCase()}
                      />
                    </Stack>
                  </Paper>
                ))}
              </Stack>
            )}

            <Button
              fullWidth
              variant="outlined"
              onClick={() => setDrawerOpen(false)}
              sx={{ mt: 3, fontWeight: 700 }}
            >
              {translate('pages.odp.drawer.close')}
            </Button>
          </Box>
        )}
      </Drawer>
    </Box>
  );
};

export default ODPPage;
