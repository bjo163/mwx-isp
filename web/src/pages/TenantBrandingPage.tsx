import { useEffect, useState } from 'react';
import { useNotify, useTranslate } from 'react-admin';
import {
  Alert, Box, Button, Card, CardContent, CircularProgress, Divider,
  MenuItem, Stack, TextField, Typography,
} from '@mui/material';
import Grid from '@mui/material/GridLegacy';
import { ColorLens as ColorLensIcon, SaveOutlined as SaveIcon } from '@mui/icons-material';
import { apiRequest } from '../utils/apiClient';
import { useBranding } from '../branding/BrandingContext';
import type { TenantBranding } from '../branding/BrandingContext';
import { PageHeader } from '../components/Enterprise';

const defaults: TenantBranding = {
  brand_name: '', short_name: '', company_name: '', tagline: '',
  logo_url: '', compact_logo_url: '', light_logo_url: '', dark_logo_url: '',
  favicon_url: '', social_image_url: '',
  primary_color: '#1976d2', secondary_color: '#dc004e', accent_color: '#16A34A',
  background_color: '', surface_color: '', text_color: '', link_color: '',
  theme: 'light', typography: 'system', website_url: '',
  support_email: '', support_phone: '', support_whatsapp: '', contact_address: '',
};

const fields: Array<{ key: keyof TenantBranding; label: string; helper?: string }> = [
  { key: 'brand_name', label: 'Public brand name' },
  { key: 'short_name', label: 'Short name / mark' },
  { key: 'company_name', label: 'Legacy display name', helper: 'Kept for compatibility. Legal issuer identity remains in Tenant settings.' },
  { key: 'tagline', label: 'Tagline' },
  { key: 'logo_url', label: 'Primary logo URL', helper: 'Relative same-origin path or HTTPS URL.' },
  { key: 'compact_logo_url', label: 'Compact logo URL' },
  { key: 'light_logo_url', label: 'Logo for light surfaces' },
  { key: 'dark_logo_url', label: 'Logo for dark surfaces' },
  { key: 'favicon_url', label: 'Favicon URL' },
  { key: 'social_image_url', label: 'Social preview image URL' },
  { key: 'website_url', label: 'Website URL' },
  { key: 'support_email', label: 'Support email' },
  { key: 'support_phone', label: 'Support phone' },
  { key: 'support_whatsapp', label: 'Support WhatsApp' },
  { key: 'contact_address', label: 'Public contact address' },
];

const colorFields: Array<{ key: keyof TenantBranding; label: string }> = [
  { key: 'primary_color', label: 'Primary' },
  { key: 'secondary_color', label: 'Secondary' },
  { key: 'accent_color', label: 'Accent' },
  { key: 'background_color', label: 'Background' },
  { key: 'surface_color', label: 'Surface' },
  { key: 'text_color', label: 'Text' },
  { key: 'link_color', label: 'Link' },
];

const TenantBrandingPage = () => {
  const [branding, setBranding] = useState<TenantBranding>(defaults);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const notify = useNotify();
  const translate = useTranslate();
  const { updateTenantBranding } = useBranding();

  useEffect(() => {
    let active = true;
    apiRequest<TenantBranding>('/branding')
      .then((response) => {
        if (!active) return;
        const next = { ...defaults, ...response };
        setBranding(next);
        updateTenantBranding(next);
      })
      .catch(() => notify(translate('ra.notification.http_error'), { type: 'error' }))
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [notify, translate, updateTenantBranding]);

  const updateField = (key: keyof TenantBranding, value: string) => {
    setBranding(current => ({ ...current, [key]: value }));
  };
  const uploadAsset = async (key: keyof TenantBranding, file: File) => {
    if (file.type !== 'image/png') {
      notify('Only PNG images are accepted for tenant branding assets.', { type: 'warning' });
      return;
    }
    const body = new FormData();
    body.set('asset', file, file.name);
    try {
      const uploaded = await apiRequest<{ url: string }>('/branding/assets', { method: 'POST', body });
      const next = { ...branding, [key]: uploaded.url };
      setBranding(next);
      updateTenantBranding(next);
      notify('Tenant asset uploaded and linked to this brand field.', { type: 'success' });
    } catch (err) {
      notify(err instanceof Error ? err.message : 'Tenant asset upload failed.', { type: 'error' });
    }
  };


  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setSaving(true);
    try {
      const saved = await apiRequest<TenantBranding>('/branding', {
        method: 'PUT',
        body: JSON.stringify(branding),
      });
      const next = { ...defaults, ...saved };
      setBranding(next);
      updateTenantBranding(next);
      notify(translate('resources.branding.notifications.updated'), { type: 'success' });
    } catch {
      notify(translate('ra.notification.http_error'), { type: 'error' });
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <Box display="flex" justifyContent="center" alignItems="center" minHeight="60vh"><CircularProgress /></Box>;

  return (
    <Box sx={{ p: { xs: 2, md: 3 }, maxWidth: 1440, mx: 'auto' }}>
      <PageHeader section="TENANT EXPERIENCE" title="BRAND STUDIO" subtitle="Configure this ISP's public brand, visual system, contact details and previews. Legal billing identity remains separate." />
      <Alert severity="info" sx={{ mb: 2 }}>
        Brand fields control presentation only. They do not change legal issuer identity, package price, RADIUS profile, speed, billing or FUP.
      </Alert>
      <form onSubmit={handleSubmit}>
        <Grid container spacing={2}>
          <Grid item xs={12} lg={7}>
            <Card variant="outlined">
              <CardContent>
                <Typography variant="h6" fontWeight={800}>Brand identity & assets</Typography>
                <Divider sx={{ my: 2 }} />
                <Grid container spacing={2}>
                  {fields.map(field => (
                    <Grid item xs={12} sm={6} key={field.key}>
                      <Stack spacing={1}>
                        <TextField fullWidth size="small" label={field.label}
                          value={String(branding[field.key] ?? '')}
                          helperText={field.helper}
                          onChange={event => updateField(field.key, event.target.value)} />
                        {(['logo_url', 'compact_logo_url', 'light_logo_url', 'dark_logo_url', 'favicon_url', 'social_image_url'] as Array<keyof TenantBranding>).includes(field.key) && (
                          <Button component="label" size="small" variant="outlined" sx={{ alignSelf: 'flex-start' }}>
                            Upload PNG
                            <input hidden type="file" accept="image/png" onChange={event => {
                              const file = event.target.files?.[0];
                              if (file) void uploadAsset(field.key, file);
                              event.target.value = '';
                            }} />
                          </Button>
                        )}
                      </Stack>
                    </Grid>
                  ))}
                </Grid>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} lg={5}>
            <Card variant="outlined">
              <CardContent>
                <Typography variant="h6" fontWeight={800}>Design system</Typography>
                <Divider sx={{ my: 2 }} />
                <Grid container spacing={2}>
                  <Grid item xs={12} sm={6}>
                    <TextField select fullWidth size="small" label="Theme" value={branding.theme || 'light'}
                      onChange={event => updateField('theme', event.target.value)}>
                      <MenuItem value="light">Light</MenuItem><MenuItem value="dark">Dark</MenuItem><MenuItem value="system">System</MenuItem>
                    </TextField>
                  </Grid>
                  <Grid item xs={12} sm={6}>
                    <TextField select fullWidth size="small" label="Typography" value={branding.typography || 'system'}
                      onChange={event => updateField('typography', event.target.value)}>
                      <MenuItem value="system">System</MenuItem><MenuItem value="inter">Inter</MenuItem>
                      <MenuItem value="roboto">Roboto</MenuItem><MenuItem value="plus-jakarta-sans">Plus Jakarta Sans</MenuItem>
                    </TextField>
                  </Grid>
                  {colorFields.map(field => (
                    <Grid item xs={12} sm={6} key={field.key}>
                      <TextField fullWidth size="small" type="color" label={field.label}
                        value={String(branding[field.key] || '#000000')}
                        onChange={event => updateField(field.key, event.target.value)}
                        InputLabelProps={{ shrink: true }} />
                    </Grid>
                  ))}
                </Grid>
              </CardContent>
            </Card>
            <Card variant="outlined" sx={{ mt: 2 }}>
              <CardContent>
                <Stack direction="row" alignItems="center" spacing={1}>
                  <ColorLensIcon color="primary" /><Typography variant="h6" fontWeight={800}>Live preview</Typography>
                </Stack>
                <Divider sx={{ my: 2 }} />
                <Box sx={{
                  p: 3, borderRadius: 2,
                  bgcolor: branding.theme === 'dark' ? (branding.background_color || '#151923') : (branding.background_color || '#fff'),
                  color: branding.text_color || (branding.theme === 'dark' ? '#fff' : '#172033'),
                  border: '1px solid', borderColor: branding.primary_color || '#1976d2',
                  fontFamily: branding.typography === 'plus-jakarta-sans' ? '"Plus Jakarta Sans", sans-serif' : branding.typography === 'roboto' ? 'Roboto, sans-serif' : 'Inter, sans-serif',
                }}>
                  {(branding.dark_logo_url || branding.light_logo_url || branding.logo_url) && (
                    <Box component="img" src={branding.theme === 'dark' ? (branding.dark_logo_url || branding.logo_url) : (branding.light_logo_url || branding.logo_url)}
                      alt="Tenant logo preview" sx={{ display: 'block', maxWidth: 220, maxHeight: 84, objectFit: 'contain', mb: 2 }} />
                  )}
                  <Typography variant="h5" fontWeight={900} sx={{ color: branding.primary_color || 'primary.main' }}>
                    {branding.brand_name || branding.company_name || 'Your ISP Brand'}
                  </Typography>
                  {branding.tagline && <Typography sx={{ color: branding.secondary_color, mt: 1 }}>{branding.tagline}</Typography>}
                  <Stack direction="row" flexWrap="wrap" gap={1} sx={{ mt: 2 }}>
                    {[branding.support_email, branding.support_phone, branding.support_whatsapp, branding.website_url].filter(Boolean).map((value, index) => (
                      <Typography key={index} variant="caption" sx={{ color: branding.link_color || 'inherit' }}>{value}</Typography>
                    ))}
                  </Stack>
                </Box>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12}>
            <Stack direction="row" justifyContent="flex-end">
              <Button type="submit" variant="contained" size="large" disabled={saving} startIcon={saving ? <CircularProgress size={18} /> : <SaveIcon />}>
                {saving ? 'Saving…' : 'Save Brand Settings'}
              </Button>
            </Stack>
          </Grid>
        </Grid>
      </form>
    </Box>
  );
};

export default TenantBrandingPage;
