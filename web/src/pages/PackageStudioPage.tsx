import { useEffect, useMemo, useState } from 'react';
import { Alert, Box, Button, Card, CardContent, CircularProgress, Divider, FormControlLabel, MenuItem, Stack, Switch, TextField, Typography } from '@mui/material';
import Grid from '@mui/material/GridLegacy';
import { useNotify } from 'react-admin';
import { SaveOutlined as SaveIcon, Inventory2Outlined as PackageIcon } from '@mui/icons-material';
import { apiRequest } from '../utils/apiClient';
import { PageHeader } from '../components/Enterprise';

type PackageRecord = { id: string; code: string; name: string; price: number; description?: string; billing_cycle?: string; status?: string };
type PackagePresentation = { display_name: string; short_description: string; category: string; icon: string; image_url: string; badge: string; featured: boolean; highlight_color: string; highlights: string; cta_label: string; terms: string; display_order: number; public_visible: boolean };
const empty: PackagePresentation = { display_name: '', short_description: '', category: 'Home Broadband', icon: 'wifi', image_url: '', badge: '', featured: false, highlight_color: '', highlights: '', cta_label: 'Daftar Pasang Baru', terms: '', display_order: 0, public_visible: true };
const money = (value: number) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value || 0);

const PackageStudioPage = () => {
  const notify = useNotify();
  const [packages, setPackages] = useState<PackageRecord[]>([]);
  const [selectedID, setSelectedID] = useState('');
  const [draft, setDraft] = useState<PackagePresentation>(empty);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const selected = useMemo(() => packages.find(item => item.id === selectedID), [packages, selectedID]);

  useEffect(() => {
    let active = true;
    Promise.all([
      apiRequest<PackageRecord[]>('/isp/packages?page=1&perPage=200'),
      apiRequest<Array<PackagePresentation & { package_id: string }>>('/isp/package-presentations'),
    ]).then(([items, presentations]) => {
      if (!active) return;
      const rows = items || [];
      setPackages(rows);
      if (rows.length) {
        setSelectedID(rows[0].id);
        const current = presentations?.find(row => row.package_id === rows[0].id);
        setDraft({ ...empty, ...(current || {}) });
      }
    }).catch((err: Error) => { if (active) setError(err.message || 'Failed to load packages'); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, []);

  useEffect(() => {
    if (!selectedID) return;
    let active = true;
    apiRequest<PackagePresentation>('/isp/packages/' + selectedID + '/presentation')
      .then(value => { if (active) setDraft({ ...empty, ...(value || {}) }); })
      .catch((err: Error) => { if (active) setError(err.message || 'Failed to load package presentation'); });
    return () => { active = false; };
  }, [selectedID]);

  const set = (key: keyof PackagePresentation) => (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
    setDraft(current => ({ ...current, [key]: event.target.value }));
  const setBool = (key: 'featured' | 'public_visible') => (_event: React.ChangeEvent<HTMLInputElement>, checked: boolean) =>
    setDraft(current => ({ ...current, [key]: checked }));

  const save = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!selectedID) return;
    setSaving(true); setError('');
    try {
      const saved = await apiRequest<PackagePresentation>('/isp/packages/' + selectedID + '/presentation', {
        method: 'PUT', body: JSON.stringify({ ...draft, display_order: Number(draft.display_order) || 0 }),
      });
      setDraft({ ...empty, ...saved });
      notify('Package presentation saved. Operational price, speed and FUP are unchanged.', { type: 'success' });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to save package presentation';
      setError(message); notify(message, { type: 'error' });
    } finally { setSaving(false); }
  };

  if (loading) return <Box display="flex" justifyContent="center" minHeight="60vh" alignItems="center"><CircularProgress /></Box>;

  return (
    <Box sx={{ p: { xs: 2, md: 3 }, maxWidth: 1500, mx: 'auto' }}>
      <PageHeader section="TENANT CATALOG" title="PACKAGE STUDIO" subtitle="Edit the marketing presentation of ISP packages. Price, speed, billing, RADIUS and FUP stay read-only operational data." />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      {!packages.length ? <Alert severity="info">No ISP packages are configured yet. Create an operational package first, then return here to style its public card.</Alert> : (
        <Grid container spacing={2}>
          <Grid item xs={12} lg={7}>
            <Card variant="outlined"><CardContent>
              <Typography variant="h6" fontWeight={800} sx={{ mb: 2 }}>Choose package</Typography>
              <TextField select fullWidth label="Operational package" value={selectedID} onChange={event => setSelectedID(event.target.value)} sx={{ mb: 2 }}>
                {packages.map(item => <MenuItem key={item.id} value={item.id}>{item.name} · {money(item.price)} · {item.status || '—'}</MenuItem>)}
              </TextField>
              {selected && <Alert severity="info" icon={<PackageIcon />}>Operational source: {selected.name} · {money(selected.price)} / {selected.billing_cycle || 'cycle'}. This editor cannot change price, status, billing cycle, RADIUS profile, speed or FUP.</Alert>}
              <Divider sx={{ my: 2 }} />
              <form onSubmit={save}>
                <Grid container spacing={2}>
                  <Grid item xs={12} sm={6}><TextField fullWidth size="small" label="Public display name" value={draft.display_name} onChange={set('display_name')} /></Grid>
                  <Grid item xs={12} sm={6}><TextField select fullWidth size="small" label="Category" value={draft.category} onChange={set('category')}>{['Home Broadband','Business','Corporate / Dedicated','Enterprise / Leased Line','Prepaid Hotspot','Other'].map(value => <MenuItem key={value} value={value}>{value}</MenuItem>)}</TextField></Grid>
                  <Grid item xs={12}><TextField fullWidth size="small" multiline minRows={2} label="Short description" value={draft.short_description} onChange={set('short_description')} /></Grid>
                  <Grid item xs={12} sm={6}><TextField fullWidth size="small" label="Icon key" value={draft.icon} onChange={set('icon')} helperText="Simple identifier such as wifi, home or building." /></Grid>
                  <Grid item xs={12} sm={6}><TextField fullWidth size="small" label="Image URL" value={draft.image_url} onChange={set('image_url')} helperText="Same-origin relative path or HTTPS URL." /></Grid>
                  <Grid item xs={12} sm={6}><TextField fullWidth size="small" label="Badge" value={draft.badge} onChange={set('badge')} placeholder="Best Seller, Recommended…" /></Grid>
                  <Grid item xs={12} sm={6}><TextField fullWidth size="small" type="color" label="Highlight color" value={draft.highlight_color || '#16A34A'} onChange={set('highlight_color')} InputLabelProps={{ shrink: true }} /></Grid>
                  <Grid item xs={12}><TextField fullWidth size="small" multiline minRows={3} label="Highlights (one per line)" value={draft.highlights} onChange={set('highlights')} /></Grid>
                  <Grid item xs={12} sm={6}><TextField fullWidth size="small" label="CTA label" value={draft.cta_label} onChange={set('cta_label')} /></Grid>
                  <Grid item xs={12} sm={6}><TextField fullWidth size="small" type="number" label="Display order" value={draft.display_order} onChange={set('display_order')} inputProps={{ min: 0, max: 100000 }} /></Grid>
                  <Grid item xs={12}><TextField fullWidth size="small" multiline minRows={2} label="Terms & conditions" value={draft.terms} onChange={set('terms')} /></Grid>
                  <Grid item xs={12}><Stack direction="row" spacing={3} flexWrap="wrap"><FormControlLabel control={<Switch checked={draft.featured} onChange={setBool('featured')} />} label="Featured package" /><FormControlLabel control={<Switch checked={draft.public_visible} onChange={setBool('public_visible')} />} label="Visible in public catalog" /></Stack></Grid>
                  <Grid item xs={12}><Stack direction="row" justifyContent="flex-end"><Button type="submit" variant="contained" disabled={saving} startIcon={saving ? <CircularProgress size={18} /> : <SaveIcon />}>{saving ? 'Saving…' : 'Save Package Presentation'}</Button></Stack></Grid>
                </Grid>
              </form>
            </CardContent></Card>
          </Grid>
          <Grid item xs={12} lg={5}>
            <Card variant="outlined"><CardContent>
              <Typography variant="h6" fontWeight={800} sx={{ mb: 2 }}>Card preview</Typography>
              <Box sx={{ p: 3, bgcolor: '#0E1422', border: '2px solid', borderColor: draft.featured ? (draft.highlight_color || '#16A34A') : '#263244', color: '#fff', borderRadius: 1 }}>
                {draft.image_url && <Box component="img" src={draft.image_url} alt={draft.display_name || selected?.name || 'Package'} sx={{ width: '100%', maxHeight: 180, objectFit: 'cover', mb: 2 }} />}
                <Stack direction="row" justifyContent="space-between" alignItems="center"><Typography variant="caption" color="primary">{draft.category || 'Home Broadband'}</Typography>{draft.badge && <Typography variant="caption" sx={{ px: 1, py: .4, bgcolor: draft.highlight_color || '#16A34A', color: '#07110a', fontWeight: 900 }}>{draft.badge}</Typography>}</Stack>
                <Typography variant="h5" fontWeight={900} sx={{ my: 1 }}>{draft.display_name || selected?.name || 'Package name'}</Typography>
                <Typography variant="body2" color="#CBD5E1">{draft.short_description || selected?.description || 'Package description'}</Typography>
                <Typography variant="h5" fontWeight={900} sx={{ color: draft.highlight_color || '#16A34A', mt: 2 }}>{selected ? money(selected.price) : '—'}</Typography>
                <Typography variant="caption" color="#94A3B8">Price is read from operational package data.</Typography>
                {draft.highlights.split('\n').filter(Boolean).map((item, index) => <Typography key={index} variant="body2" sx={{ mt: 1 }}>✓ {item}</Typography>)}
                <Button variant="contained" fullWidth sx={{ mt: 3 }} disabled>{draft.cta_label || 'Choose package'}</Button>
              </Box>
            </CardContent></Card>
          </Grid>
        </Grid>
      )}
    </Box>
  );
};

export default PackageStudioPage;