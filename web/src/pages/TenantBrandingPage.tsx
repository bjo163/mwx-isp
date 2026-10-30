import { useState, useEffect } from 'react';
import { useNotify, useTranslate } from 'react-admin';
import {
  Box,
  Card,
  CardContent,
  TextField,
  Button,
  Typography,
  CircularProgress,
  Divider,
  Alert,
} from '@mui/material';
import Grid from '@mui/material/GridLegacy';
import { ColorLens as ColorLensIcon } from '@mui/icons-material';
import { apiRequest } from '../utils/apiClient';
import { useBranding } from '../branding/BrandingContext';

interface TenantBranding {
  id?: string;
  tenant_id?: string;
  logo_url: string;
  favicon_url: string;
  primary_color: string;
  secondary_color: string;
  theme: string;
  company_name: string;
  tagline: string;
  website_url: string;
  support_email: string;
  support_phone: string;
  created_at?: string;
  updated_at?: string;
}

const TenantBrandingPage = () => {
  const [branding, setBranding] = useState<TenantBranding>({
    logo_url: '',
    favicon_url: '',
    primary_color: '#1976d2',
    secondary_color: '#dc004e',
    theme: 'light',
    company_name: '',
    tagline: '',
    website_url: '',
    support_email: '',
    support_phone: '',
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const notify = useNotify();
  const translate = useTranslate();
  const { updateTenantBranding } = useBranding();

  useEffect(() => {
    loadBranding();
  }, []);

  const loadBranding = async () => {
    try {
      const response = await apiRequest<TenantBranding>('/branding');
      if (response) {
        setBranding((current) => ({ ...current, ...response }));
        updateTenantBranding(response);
      }
    } catch (error) {
      console.error('Failed to load branding:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (field: keyof TenantBranding) => (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
  ) => {
    setBranding(prev => ({ ...prev, [field]: event.target.value }));
  };

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setSaving(true);
    try {
      const saved = await apiRequest<TenantBranding>('/branding', {
        method: 'PUT',
        body: JSON.stringify(branding),
      });
      updateTenantBranding(saved);
      notify(translate('resources.branding.notifications.updated'), { type: 'success' });
    } catch (error) {
      notify(translate('ra.notification.http_error'), { type: 'error' });
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="60vh">
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ padding: 3, maxWidth: 1200, margin: '0 auto' }}>
      <Typography variant="h4" gutterBottom sx={{ mb: 3 }}>
        <ColorLensIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
        Tenant Branding
      </Typography>

      <Alert severity="info" sx={{ mb: 3 }}>
        Customize your organization's branding. Changes will be reflected on your subdomain portal.
      </Alert>

      <form onSubmit={handleSubmit}>
        <Grid container spacing={3}>
          <Grid item xs={12} md={6}>
            <Card>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Visual Identity
                </Typography>
                <Divider sx={{ mb: 2 }} />

                <TextField
                  fullWidth
                  label="Logo URL"
                  value={branding.logo_url}
                  onChange={handleChange('logo_url')}
                  placeholder="https://example.com/logo.png"
                  sx={{ mb: 2 }}
                  helperText="Full URL to your organization's logo"
                />

                <TextField
                  fullWidth
                  label="Favicon URL"
                  value={branding.favicon_url}
                  onChange={handleChange('favicon_url')}
                  placeholder="https://example.com/favicon.ico"
                  sx={{ mb: 2 }}
                  helperText="Small icon for browser tabs"
                />

                <Grid container spacing={2} sx={{ mb: 2 }}>
                  <Grid item xs={6}>
                    <TextField
                      fullWidth
                      label="Primary Color"
                      value={branding.primary_color}
                      onChange={handleChange('primary_color')}
                      type="color"
                      InputLabelProps={{ shrink: true }}
                    />
                  </Grid>
                  <Grid item xs={6}>
                    <TextField
                      fullWidth
                      label="Secondary Color"
                      value={branding.secondary_color}
                      onChange={handleChange('secondary_color')}
                      type="color"
                      InputLabelProps={{ shrink: true }}
                    />
                  </Grid>
                </Grid>

                <TextField
                  fullWidth
                  select
                  label="Theme"
                  value={branding.theme}
                  onChange={handleChange('theme')}
                  SelectProps={{ native: true }}
                  sx={{ mb: 2 }}
                >
                  <option value="light">Light</option>
                  <option value="dark">Dark</option>
                </TextField>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12} md={6}>
            <Card>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Organization Information
                </Typography>
                <Divider sx={{ mb: 2 }} />

                <TextField
                  fullWidth
                  label="Company Name"
                  value={branding.company_name}
                  onChange={handleChange('company_name')}
                  sx={{ mb: 2 }}
                  required
                />

                <TextField
                  fullWidth
                  label="Tagline"
                  value={branding.tagline}
                  onChange={handleChange('tagline')}
                  sx={{ mb: 2 }}
                  placeholder="Your organization's tagline or slogan"
                />

                <TextField
                  fullWidth
                  label="Website URL"
                  value={branding.website_url}
                  onChange={handleChange('website_url')}
                  sx={{ mb: 2 }}
                  placeholder="https://yourcompany.com"
                />

                <TextField
                  fullWidth
                  label="Support Email"
                  value={branding.support_email}
                  onChange={handleChange('support_email')}
                  type="email"
                  sx={{ mb: 2 }}
                  placeholder="support@yourcompany.com"
                />

                <TextField
                  fullWidth
                  label="Support Phone"
                  value={branding.support_phone}
                  onChange={handleChange('support_phone')}
                  sx={{ mb: 2 }}
                  placeholder="+62 21 1234 5678"
                />
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12}>
            <Card>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Preview
                </Typography>
                <Divider sx={{ mb: 2 }} />

                <Box
                  sx={{
                    p: 3,
                    borderRadius: 2,
                    bgcolor: branding.theme === 'dark' ? '#1a1a1a' : '#ffffff',
                    color: branding.theme === 'dark' ? '#ffffff' : '#000000',
                    border: '1px solid',
                    borderColor: branding.primary_color,
                  }}
                >
                  {branding.logo_url && (
                    <Box
                      component="img"
                      src={branding.logo_url}
                      alt="Logo preview"
                      sx={{ maxWidth: 200, maxHeight: 80, mb: 2 }}
                      onError={(e) => {
                        e.currentTarget.style.display = 'none';
                      }}
                    />
                  )}
                  <Typography
                    variant="h5"
                    sx={{ color: branding.primary_color, fontWeight: 'bold', mb: 1 }}
                  >
                    {branding.company_name || 'Your Company Name'}
                  </Typography>
                  {branding.tagline && (
                    <Typography variant="body1" sx={{ mb: 2, color: branding.secondary_color }}>
                      {branding.tagline}
                    </Typography>
                  )}
                  <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap' }}>
                    {branding.website_url && (
                      <Typography variant="body2">
                        🌐 {branding.website_url}
                      </Typography>
                    )}
                    {branding.support_email && (
                      <Typography variant="body2">
                        📧 {branding.support_email}
                      </Typography>
                    )}
                    {branding.support_phone && (
                      <Typography variant="body2">
                        📞 {branding.support_phone}
                      </Typography>
                    )}
                  </Box>
                </Box>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12}>
            <Box display="flex" justifyContent="flex-end" gap={2}>
              <Button
                variant="contained"
                type="submit"
                disabled={saving}
                size="large"
                startIcon={saving ? <CircularProgress size={20} /> : <ColorLensIcon />}
              >
                {saving ? 'Saving...' : 'Save Branding'}
              </Button>
            </Box>
          </Grid>
        </Grid>
      </form>
    </Box>
  );
};

export default TenantBrandingPage;
