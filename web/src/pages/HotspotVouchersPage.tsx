import React, { useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  FormControlLabel,
  InputLabel,
  LinearProgress,
  MenuItem,
  Paper,
  Select,
  Stack,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TablePagination,
  TableRow,
  TextField,
  Typography,
} from '@mui/material';
import {
  Add,
  Block,
  ConfirmationNumber,
  DeleteOutline,
  Print,
  Refresh,
} from '@mui/icons-material';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNotify, useTranslate } from 'react-admin';
import { useIsAdmin } from '../hooks/useIsAdmin';
import { apiRequest, apiRequestPage, extractData } from '../utils/apiClient';
import { useBranding } from '../branding/BrandingContext';
import { KpiStrip, KpiTile, Mono, PageHeader, Panel, StatusChip } from '../components/Enterprise';

interface Voucher {
  id: string;
  batch_id: string;
  code: string;
  password: string;
  price: number;
  validity_seconds: number;
  quota_bytes: number;
  status: string;
  first_login_at?: string;
  expires_at?: string;
  created_at: string;
}

interface Batch {
  id: string;
  batch_no: string;
  name: string;
  quantity: number;
  price: number;
  prefix: string;
  created_at: string;
}

interface InternetPackage {
  id: string;
  code: string;
  name: string;
  status: string;
  radius_profile_id: string;
}

const VOUCHER_PAGE_SIZE = 100;
const VOUCHER_PRINT_LIMIT = 500;

export const HotspotVouchersPage: React.FC = () => {
  const { branding } = useBranding();
  const notify = useNotify();
  const translate = useTranslate();
  const queryClient = useQueryClient();
  const isAdmin = useIsAdmin();

  const [genOpen, setGenOpen] = useState(false);
  const [printOpen, setPrintOpen] = useState(false);
  const [printMode, setPrintMode] = useState<'thermal' | 'a4'>('thermal');
  const [selectedBatch, setSelectedBatch] = useState<string>('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [search, setSearch] = useState<string>('');
  const [printVouchers, setPrintVouchers] = useState<Voucher[]>([]);
  const [isLoadingPrint, setIsLoadingPrint] = useState<boolean>(false);
  const [page, setPage] = useState(1);

  // Form state for generating vouchers
  const [genForm, setGenForm] = useState({
    name: 'Standard Hotspot',
    package_id: '',
    quantity: 20,
    price: 5000,
    validity_hours: 24,
    prefix: 'HOT',
    code_length: 6,
    same_user_pass: true,
  });

  const handleOpenPrint = async () => {
    setIsLoadingPrint(true);
    try {
      const params = new URLSearchParams();
      if (selectedBatch) params.set('batch_id', selectedBatch);
      if (statusFilter) params.set('status', statusFilter);
      if (search.trim()) params.set('q', search.trim());
      params.set('perPage', String(VOUCHER_PRINT_LIMIT));
      params.set('page', '1');

      const first = await apiRequestPage<Voucher[]>(`/isp/vouchers?${params.toString()}`);
      const total = first.total;
      if (total === 0) {
        throw new Error(translate('pages.vouchers.print_errors.no_matches'));
      }
      if (total > VOUCHER_PRINT_LIMIT) {
        throw new Error(translate('pages.vouchers.print_errors.capped', { total, limit: VOUCHER_PRINT_LIMIT }));
      }
      const all = [...(first.data ?? [])];
      let nextPage = 2;
      while (all.length < total) {
        params.set('page', String(nextPage));
        const next = await apiRequestPage<Voucher[]>(`/isp/vouchers?${params.toString()}`);
        const rows = next.data ?? [];
        if (rows.length === 0) break;
        all.push(...rows);
        nextPage += 1;
      }
      if (all.length !== total) {
        throw new Error(translate('pages.vouchers.print_errors.partial', { loaded: all.length, total }));
      }
      setPrintVouchers(all);
      setPrintOpen(true);
    } catch (err: any) {
      setPrintVouchers([]);
      notify(err?.message || translate('pages.vouchers.notify.print_load_failed'), { type: 'error' });
    } finally {
      setIsLoadingPrint(false);
    }
  };

  const batchesQuery = useQuery({
    queryKey: ['isp', 'vouchers', 'batches'],
    queryFn: async () => {
      const res = await apiRequest<unknown>('/isp/vouchers/batches?perPage=50');
      return extractData<Batch[]>(res) ?? [];
    },
  });

  const packagesQuery = useQuery({
    queryKey: ['isp', 'vouchers', 'packages'],
    queryFn: async () => {
      const res = await apiRequest<unknown>('/isp/packages?status=active&perPage=100');
      return (extractData<InternetPackage[]>(res) ?? []).filter((pkg) => Number(pkg.radius_profile_id) > 0);
    },
  });

  const vouchersQuery = useQuery({
    queryKey: ['isp', 'vouchers', selectedBatch, statusFilter, search, page],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (selectedBatch) params.set('batch_id', selectedBatch);
      if (statusFilter) params.set('status', statusFilter);
      if (search.trim()) params.set('q', search.trim());
      params.set('page', String(page));
      params.set('perPage', String(VOUCHER_PAGE_SIZE));
      return apiRequestPage<Voucher[]>(`/isp/vouchers?${params.toString()}`);
    },
  });

  useEffect(() => {
    setPage(1);
  }, [selectedBatch, statusFilter, search]);

  const generateMutation = useMutation({
    mutationFn: () =>
      apiRequest('/isp/vouchers/generate', {
        method: 'POST',
        body: JSON.stringify({
          name: genForm.name,
          package_id: genForm.package_id,
          quantity: Number(genForm.quantity),
          price: Number(genForm.price),
          validity_seconds: Number(genForm.validity_hours) * 3600,
          quota_mb: 0,
          prefix: genForm.prefix.trim(),
          code_length: Number(genForm.code_length),
          same_user_pass: genForm.same_user_pass,
        }),
      }),
    onSuccess: () => {
      notify(translate('pages.vouchers.notify.generated'), { type: 'success' });
      setGenOpen(false);
      void queryClient.invalidateQueries({ queryKey: ['isp', 'vouchers'] });
    },
    onError: (err: any) => notify(err.message || translate('pages.vouchers.notify.generation_failed'), { type: 'error' }),
  });

  const deleteBatchMutation = useMutation({
    mutationFn: (id: string) =>
      apiRequest(`/isp/vouchers/batches/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      notify(translate('pages.vouchers.notify.batch_removed'), { type: 'success' });
      setSelectedBatch('');
      void queryClient.invalidateQueries({ queryKey: ['isp', 'vouchers'] });
    },
  });

  const revokeVoucherMutation = useMutation({
    mutationFn: (id: string) =>
      apiRequest(`/isp/vouchers/${id}/revoke`, { method: 'POST' }),
    onSuccess: () => {
      notify(translate('pages.vouchers.notify.revoked'), { type: 'success' });
      void queryClient.invalidateQueries({ queryKey: ['isp', 'vouchers'] });
    },
    onError: (err: any) => notify(err.message || translate('pages.vouchers.notify.revoke_failed'), { type: 'error' }),
  });

  const batches = batchesQuery.data ?? [];
  const vouchers = vouchersQuery.data?.data ?? [];
  const vouchersTotal = vouchersQuery.data?.total ?? 0;

  const formatValidity = (seconds: number) => {
    if (seconds >= 86400) return translate('pages.vouchers.units.days', { smart_count: Math.round(seconds / 86400) });
    if (seconds >= 3600) return translate('pages.vouchers.units.hours', { smart_count: Math.round(seconds / 3600) });
    return translate('pages.vouchers.units.minutes', { smart_count: Math.round(seconds / 60) });
  };

  const formatQuota = (bytes: number) => {
    if (!bytes || bytes <= 0) return translate('pages.vouchers.quota.unlimited');
    const mb = bytes / (1024 * 1024);
    const configured = mb >= 1024 ? `${(mb / 1024).toFixed(1)} GB` : `${mb.toFixed(0)} MB`;
    return translate('pages.vouchers.quota.not_enforced', { configured });
  };

  const totalValue = vouchers.reduce((acc, v) => acc + (v.price || 0), 0);
  const countStatus = (s: string) => vouchers.filter((v) => v.status === s).length;

  return (
    <Box sx={{ p: { xs: 1.5, md: 2.5 }, maxWidth: 1600, mx: 'auto' }}>
      <PageHeader
        section={translate('pages.vouchers.section')}
        title={translate('pages.vouchers.title')}
        subtitle={translate('pages.vouchers.subtitle')}
        actions={<>
          <Button
            variant="outlined"
            startIcon={<Print />}
            disabled={vouchersTotal === 0 || isLoadingPrint}
            onClick={handleOpenPrint}
          >
            {isLoadingPrint ? translate('pages.vouchers.actions.loading_print') : translate('pages.vouchers.actions.print', { count: vouchersTotal })}
          </Button>
          {isAdmin && (
            <Button
              variant="contained"
              color="primary"
              startIcon={<Add />}
              onClick={() => setGenOpen(true)}
            >
              {translate('pages.vouchers.actions.generate_batch')}
            </Button>
          )}
        </>}
      />

      <KpiStrip>
        <KpiTile label={translate('pages.vouchers.kpi.total_vouchers')} value={vouchersTotal} icon={<ConfirmationNumber fontSize="small" />} hint={translate('pages.vouchers.kpi.total_vouchers_hint')} />
        <KpiTile label={translate('pages.vouchers.kpi.active_unused')} value={countStatus('active')} tone="success" hint={translate('pages.vouchers.kpi.on_this_page')} />
        <KpiTile label={translate('pages.vouchers.status.used')} value={countStatus('used')} tone="info" hint={translate('pages.vouchers.kpi.on_this_page')} />
        <KpiTile label={translate('pages.vouchers.status.expired')} value={countStatus('expired')} tone="error" hint={translate('pages.vouchers.kpi.on_this_page')} />
        <KpiTile label={translate('pages.vouchers.status.revoked')} value={countStatus('revoked')} tone="error" hint={translate('pages.vouchers.kpi.on_this_page')} />
        <KpiTile label={translate('pages.vouchers.kpi.total_batches')} value={batches.length} tone="warning" hint={translate('pages.vouchers.kpi.total_batches_hint')} />
        <KpiTile
          label={translate('pages.vouchers.kpi.total_face_value')}
          value={`Rp ${(totalValue / 1000).toFixed(0)}k`}
          tone="secondary"
          hint={translate('pages.vouchers.kpi.total_face_value_hint')}
        />
      </KpiStrip>

      <Panel
        title={translate('pages.vouchers.inventory.title')}
        subtitle={translate('pages.vouchers.inventory.subtitle', { loaded: vouchers.length, total: vouchersTotal })}
        dense
        actions={
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.5} alignItems="center">
            <FormControl size="small" sx={{ minWidth: 180 }}>
              <InputLabel>{translate('pages.vouchers.filters.filter_batch')}</InputLabel>
              <Select
                label={translate('pages.vouchers.filters.filter_batch')}
                value={selectedBatch}
                onChange={(e) => setSelectedBatch(e.target.value)}
              >
                <MenuItem value="">{translate('pages.vouchers.filters.all_batches')}</MenuItem>
                {batches.map((b) => (
                  <MenuItem key={b.id} value={b.id}>
                    {translate('pages.vouchers.filters.batch_option', { batch_no: b.batch_no, name: b.name, quantity: b.quantity })}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>

            <FormControl size="small" sx={{ minWidth: 120 }}>
              <InputLabel>{translate('pages.vouchers.filters.status')}</InputLabel>
              <Select
                label={translate('pages.vouchers.filters.status')}
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
              >
                <MenuItem value="">{translate('pages.vouchers.filters.all_statuses')}</MenuItem>
                <MenuItem value="active">{translate('pages.vouchers.status.active')}</MenuItem>
                <MenuItem value="used">{translate('pages.vouchers.status.used')}</MenuItem>
                <MenuItem value="expired">{translate('pages.vouchers.status.expired')}</MenuItem>
                <MenuItem value="revoked">{translate('pages.vouchers.status.revoked')}</MenuItem>
              </Select>
            </FormControl>

            <TextField
              size="small"
              label={translate('pages.vouchers.filters.search_code')}
              placeholder={translate('pages.vouchers.filters.search_placeholder')}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              sx={{ width: 140 }}
            />

            {isAdmin && selectedBatch && (
              <Button
                size="small"
                color="error"
                variant="outlined"
                startIcon={<DeleteOutline />}
                onClick={() => {
                  if (window.confirm(translate('pages.vouchers.confirm.delete_batch'))) {
                    deleteBatchMutation.mutate(selectedBatch);
                  }
                }}
              >
                {translate('pages.vouchers.actions.delete_batch')}
              </Button>
            )}

            <Button
              size="small"
              variant="outlined"
              startIcon={<Refresh />}
              onClick={() => void vouchersQuery.refetch()}
              disabled={vouchersQuery.isFetching}
            >
              {translate('pages.vouchers.actions.refresh')}
            </Button>
          </Stack>
        }
      >
          {vouchersQuery.isLoading ? (
            <LinearProgress />
          ) : vouchersQuery.isError ? (
            <Box sx={{ p: 4, textAlign: 'center' }}>
              <Typography variant="h6" color="error.main">{translate('pages.vouchers.state.load_error')}</Typography>
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 2 }}>
                {(vouchersQuery.error as Error)?.message || translate('pages.vouchers.state.invalid_response')}
              </Typography>
              <Button variant="outlined" onClick={() => void vouchersQuery.refetch()}>{translate('pages.vouchers.actions.retry')}</Button>
            </Box>
          ) : vouchers.length === 0 ? (
            <Box sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
              <ConfirmationNumber sx={{ fontSize: 48, opacity: 0.4, mb: 1 }} />
              <Typography variant="h6">{translate('pages.vouchers.state.empty_title')}</Typography>
              <Typography variant="caption">
                {translate('pages.vouchers.state.empty_hint')}
              </Typography>
            </Box>
          ) : (
            <>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>{translate('pages.vouchers.fields.code_user')}</TableCell>
                  <TableCell>{translate('pages.vouchers.fields.password')}</TableCell>
                  <TableCell>{translate('pages.vouchers.fields.price')}</TableCell>
                  <TableCell>{translate('pages.vouchers.fields.validity')}</TableCell>
                  <TableCell>{translate('pages.vouchers.fields.quota')}</TableCell>
                  <TableCell>{translate('pages.vouchers.fields.status')}</TableCell>
                  <TableCell>{translate('pages.vouchers.fields.created')}</TableCell>
                  <TableCell align="right">{translate('pages.vouchers.fields.actions')}</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {vouchers.map((v) => (
                  <TableRow key={v.id} hover>
                    <TableCell sx={{ color: 'primary.main', fontWeight: 700 }}>
                      <Mono>{v.code}</Mono>
                    </TableCell>
                    <TableCell><Mono>{v.password}</Mono></TableCell>
                    <TableCell sx={{ fontWeight: 600 }}>
                      Rp {new Intl.NumberFormat('id-ID').format(v.price)}
                    </TableCell>
                    <TableCell>{formatValidity(v.validity_seconds)}</TableCell>
                    <TableCell>{formatQuota(v.quota_bytes)}</TableCell>
                    <TableCell>
                      <StatusChip value={v.status} label={translate('pages.vouchers.status.' + v.status, { _: v.status })} />
                    </TableCell>
                    <TableCell sx={{ color: 'text.secondary', fontSize: '0.78rem' }}>
                      {new Date(v.created_at).toLocaleDateString()} {new Date(v.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                    </TableCell>
                    <TableCell align="right">
                      {isAdmin && (v.status === 'active' || v.status === 'used') && (
                        <Button
                          size="small"
                          color="error"
                          startIcon={<Block fontSize="small" />}
                          disabled={revokeVoucherMutation.isPending}
                          onClick={() => {
                            if (window.confirm(translate('pages.vouchers.confirm.revoke', { code: v.code }))) {
                              revokeVoucherMutation.mutate(v.id);
                            }
                          }}
                        >
                          {translate('pages.vouchers.actions.revoke')}
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <TablePagination
              component="div"
              count={vouchersTotal}
              page={Math.max(0, page - 1)}
              onPageChange={(_, next) => setPage(next + 1)}
              rowsPerPage={VOUCHER_PAGE_SIZE}
              rowsPerPageOptions={[VOUCHER_PAGE_SIZE]}
              labelRowsPerPage={translate('pages.vouchers.pagination.rows_per_page')}
              labelDisplayedRows={({ from, to, count }) => translate('pages.vouchers.pagination.displayed_rows', { from, to, count })}
            />
            </>
          )}
      </Panel>

      {/* Generator Dialog Modal */}
      <Dialog open={genOpen} onClose={() => setGenOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle sx={{ fontWeight: 800 }}>{translate('pages.vouchers.generate.title')}</DialogTitle>
        <DialogContent dividers>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <FormControl size="small" fullWidth required>
              <InputLabel id="voucher-package-label">{translate('pages.vouchers.generate.package')}</InputLabel>
              <Select
                labelId="voucher-package-label"
                label={translate('pages.vouchers.generate.package')}
                value={genForm.package_id}
                onChange={(e) => setGenForm({ ...genForm, package_id: e.target.value })}
                disabled={packagesQuery.isLoading}
              >
                {packagesQuery.data?.map((pkg) => (
                  <MenuItem key={pkg.id} value={pkg.id}>{pkg.code} — {pkg.name}</MenuItem>
                ))}
              </Select>
            </FormControl>
            {packagesQuery.isError && <Alert severity="error">{translate('pages.vouchers.generate.packages_error')}</Alert>}
            {!packagesQuery.isLoading && !packagesQuery.isError && packagesQuery.data?.length === 0 && (
              <Alert severity="warning">{translate('pages.vouchers.generate.packages_empty')}</Alert>
            )}
            <TextField
              size="small"
              label={translate('pages.vouchers.generate.name')}
              value={genForm.name}
              onChange={(e) => setGenForm({ ...genForm, name: e.target.value })}
              fullWidth
            />
            <Stack direction="row" spacing={1.5}>
              <TextField
                size="small"
                type="number"
                label={translate('pages.vouchers.generate.quantity')}
                value={genForm.quantity}
                onChange={(e) => setGenForm({ ...genForm, quantity: Number(e.target.value) })}
                sx={{ flex: 1 }}
              />
              <TextField
                size="small"
                label={translate('pages.vouchers.generate.prefix')}
                placeholder={translate('pages.vouchers.generate.prefix_placeholder')}
                value={genForm.prefix}
                onChange={(e) => setGenForm({ ...genForm, prefix: e.target.value })}
                sx={{ width: 110 }}
              />
            </Stack>
            <Stack direction="row" spacing={1.5}>
              <TextField
                size="small"
                type="number"
                label={translate('pages.vouchers.generate.price')}
                value={genForm.price}
                onChange={(e) => setGenForm({ ...genForm, price: Number(e.target.value) })}
                sx={{ flex: 1 }}
              />
              <TextField
                size="small"
                type="number"
                label={translate('pages.vouchers.generate.validity_hours')}
                value={genForm.validity_hours}
                onChange={(e) => setGenForm({ ...genForm, validity_hours: Number(e.target.value) })}
                sx={{ flex: 1 }}
              />
            </Stack>
            <Alert severity="info">
              {translate('pages.vouchers.generate.validity_hint')}
            </Alert>
            <Stack direction="row" spacing={1.5}>
              <TextField
                size="small"
                type="number"
                label={translate('pages.vouchers.generate.code_length')}
                value={genForm.code_length}
                onChange={(e) => setGenForm({ ...genForm, code_length: Number(e.target.value) })}
                sx={{ width: 110 }}
              />
            </Stack>
            <FormControlLabel
              control={
                <Switch
                  checked={genForm.same_user_pass}
                  onChange={(e) => setGenForm({ ...genForm, same_user_pass: e.target.checked })}
                />
              }
              label={translate('pages.vouchers.generate.same_user_pass')}
            />
          </Stack>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button onClick={() => setGenOpen(false)}>{translate('pages.vouchers.actions.cancel')}</Button>
          <Button
            variant="contained"
            disabled={generateMutation.isPending || !genForm.quantity || !genForm.package_id || packagesQuery.isLoading || (packagesQuery.data?.length ?? 0) === 0}
            onClick={() => generateMutation.mutate()}
          >
            {generateMutation.isPending ? translate('pages.vouchers.generate.generating') : translate('pages.vouchers.generate.create', { count: genForm.quantity })}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Print Preview Dialog */}
      <Dialog open={printOpen} onClose={() => setPrintOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle sx={{ fontWeight: 800, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span>{translate('pages.vouchers.print.title', { count: printVouchers.length })}</span>
          <Stack direction="row" spacing={1}>
            <Button
              size="small"
              variant={printMode === 'thermal' ? 'contained' : 'outlined'}
              onClick={() => setPrintMode('thermal')}
            >
              {translate('pages.vouchers.print.thermal')}
            </Button>
            <Button
              size="small"
              variant={printMode === 'a4' ? 'contained' : 'outlined'}
              onClick={() => setPrintMode('a4')}
            >
              {translate('pages.vouchers.print.a4')}
            </Button>
            <Button
              size="small"
              variant="contained"
              color="primary"
              startIcon={<Print />}
              onClick={() => window.print()}
            >
              {translate('pages.vouchers.print.print_now')}
            </Button>
          </Stack>
        </DialogTitle>
        <DialogContent dividers>
          <Alert severity="info" sx={{ mb: 2 }} className="no-print">
            {translate('pages.vouchers.print.instructions')}
            <Box component="span" sx={{ ml: 1, fontWeight: 700 }}>
              {translate('pages.vouchers.print.loaded_all', { count: printVouchers.length })}
            </Box>
          </Alert>

          {/* Printable Container */}
          <Box id="printable-vouchers">
            {printMode === 'thermal' ? (
              <Box sx={{ width: 280, mx: 'auto', p: 1, border: '1px solid #ddd', borderRadius: 1 }}>
                {printVouchers.map((v) => (
                  <Box
                    key={v.id}
                    sx={{
                      p: 1.5,
                      borderBottom: '1px dashed #000',
                      textAlign: 'center',
                      pageBreakInside: 'avoid',
                    }}
                  >
                    <Typography variant="subtitle1" fontWeight={900}>
                      {branding.product_name}
                    </Typography>
                    <Typography variant="caption" sx={{ display: 'block', mb: 1 }}>
                      {translate('pages.vouchers.print.wifi_access')}
                    </Typography>
                    <Box sx={{ bgcolor: '#000', color: '#fff', py: 0.5, px: 1, borderRadius: 0.5, mb: 1 }}>
                      <Typography variant="body2" sx={{ fontFamily: 'monospace', fontWeight: 900, letterSpacing: 2 }}>
                        {v.code}
                      </Typography>
                    </Box>
                    {v.password !== v.code && (
                      <Typography variant="caption" sx={{ display: 'block' }}>
                        {translate('pages.vouchers.print.pin')} <strong>{v.password}</strong>
                      </Typography>
                    )}
                    <Typography variant="caption" sx={{ display: 'block', fontWeight: 700, mt: 0.5 }}>
                      Rp {new Intl.NumberFormat('id-ID').format(v.price)} · {formatValidity(v.validity_seconds)}
                    </Typography>
                    <Typography variant="caption" sx={{ display: 'block', color: 'text.secondary', fontSize: '0.65rem' }}>
                      {translate('pages.vouchers.print.data_allowance')} {formatQuota(v.quota_bytes)}
                    </Typography>
                  </Box>
                ))}
              </Box>
            ) : (
              <Box
                sx={{
                  display: 'grid',
                  gridTemplateColumns: 'repeat(3, 1fr)',
                  gap: 1.5,
                  p: 1,
                }}
              >
                {printVouchers.map((v) => (
                  <Paper
                    key={v.id}
                    variant="outlined"
                    sx={{
                      p: 1.5,
                      border: '1px dashed #555',
                      borderRadius: 1,
                      pageBreakInside: 'avoid',
                      position: 'relative',
                    }}
                  >
                    <Stack direction="row" justifyContent="space-between" alignItems="center">
                      <Typography variant="caption" fontWeight={800} color="primary.main">
                        {branding.product_name}
                      </Typography>
                      <Chip
                        size="small"
                        label={`Rp ${new Intl.NumberFormat('id-ID').format(v.price)}`}
                        sx={{ height: 18, fontSize: '0.65rem', fontWeight: 800 }}
                      />
                    </Stack>
                    <Box sx={{ my: 1, textAlign: 'center', bgcolor: 'action.hover', p: 0.75, borderRadius: 0.5 }}>
                      <Typography variant="caption" color="text.secondary" display="block">
                        {translate('pages.vouchers.print.voucher_code')}
                      </Typography>
                      <Typography variant="body1" sx={{ fontFamily: 'monospace', fontWeight: 900, letterSpacing: 1.5 }}>
                        {v.code}
                      </Typography>
                    </Box>
                    <Stack direction="row" justifyContent="space-between" sx={{ fontSize: '0.72rem' }}>
                      <span>{translate('pages.vouchers.print.duration')} <strong>{formatValidity(v.validity_seconds)}</strong></span>
                      <span>{translate('pages.vouchers.print.data_allowance')} <strong>{formatQuota(v.quota_bytes)}</strong></span>
                    </Stack>
                  </Paper>
                ))}
              </Box>
            )}
          </Box>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button onClick={() => setPrintOpen(false)}>{translate('pages.vouchers.actions.close')}</Button>
        </DialogActions>
      </Dialog>

      {/* Print CSS */}
      <style>{`
        @media print {
          body {
            background: #fff !important;
            color: #000 !important;
          }
          header, nav, .MuiAppBar-root, .MuiDrawer-root, .no-print, button, .RaTopToolbar-root {
            display: none !important;
          }
          #printable-vouchers {
            width: 100% !important;
            margin: 0 !important;
            padding: 0 !important;
          }
          .MuiPaper-root {
            box-shadow: none !important;
          }
        }
      `}</style>
    </Box>
  );
};

export default HotspotVouchersPage;
