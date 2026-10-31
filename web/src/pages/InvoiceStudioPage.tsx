import { useEffect, useState } from 'react';
import {
  Alert, Box, Button, Card, CardContent, CircularProgress, Divider, FormControlLabel,
  MenuItem, Stack, Switch, TextField, Typography,
} from '@mui/material';
import Grid from '@mui/material/GridLegacy';
import {
  RestartAltOutlined as ResetIcon,
  SaveOutlined as SaveIcon,
} from '@mui/icons-material';
import { useNotify } from 'react-admin';
import { apiRequest } from '../utils/apiClient';
import { PageHeader } from '../components/Enterprise';

type DocumentTemplate = {
  version: number;
  issuer_name: string;
  logo_url: string;
  header_text: string;
  footer_text: string;
  accent_color: string;
  paper_size: string;
  show_tax_id: boolean;
  bank_name: string;
  bank_account_name: string;
  bank_account_number: string;
  payment_note: string;
  qr_image_url: string;
  updated_at?: string;
};

const emptyTemplate: DocumentTemplate = {
  version: 1,
  issuer_name: '',
  logo_url: '',
  header_text: '',
  footer_text: '',
  accent_color: '#16A34A',
  paper_size: 'A4',
  show_tax_id: true,
  bank_name: '',
  bank_account_name: '',
  bank_account_number: '',
  payment_note: '',
  qr_image_url: '',
};

/**
 * M20.2/M20.4: presentation-only editor for invoice and receipt documents.
 * Saving here never changes totals, numbering, status or the ledger; invoices
 * already issued keep the snapshot captured when they were created.
 */
const InvoiceStudioPage = () => {
  const notify = useNotify();
  const [template, setTemplate] = useState<DocumentTemplate>(emptyTemplate);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [resetting, setResetting] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    let active = true;
    apiRequest<DocumentTemplate>('/tenant-document-template')
      .then(result => { if (active) setTemplate({ ...emptyTemplate, ...(result || {}) }); })
      .catch((err: Error) => { if (active) setError(err.message || 'Unable to load the document template'); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, []);

  const update = (patch: Partial<DocumentTemplate>) => setTemplate(current => ({ ...current, ...patch }));

  const save = async () => {
    setSaving(true); setError('');
    try {
      const result = await apiRequest<DocumentTemplate>('/tenant-document-template', {
        method: 'PUT', body: JSON.stringify(template),
      });
      setTemplate({ ...emptyTemplate, ...(result || template) });
      notify('Document template saved. Invoices already issued keep their original document.', { type: 'success' });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Save failed';
      setError(message); notify(message, { type: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const reset = async () => {
    setResetting(true); setError('');
    try {
      const result = await apiRequest<DocumentTemplate>('/tenant-document-template/reset', { method: 'POST' });
      setTemplate({ ...emptyTemplate, ...(result || emptyTemplate) });
      notify('Document template reset to the tenant defaults.', { type: 'success' });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Reset failed';
      setError(message); notify(message, { type: 'error' });
    } finally {
      setResetting(false);
    }
  };

  if (loading) {
    return <Box display="flex" justifyContent="center" minHeight="60vh" alignItems="center"><CircularProgress /></Box>;
  }

  const bankIncomplete = [template.bank_name, template.bank_account_name, template.bank_account_number]
    .filter(value => value.trim() !== '').length;

  return (
    <Box sx={{ p: { xs: 2, md: 3 }, maxWidth: 1450, mx: 'auto' }}>
      <PageHeader
        section="TENANT EXPERIENCE"
        title="INVOICE STUDIO"
        subtitle="Choose how this tenant's invoices and receipts look, and which verified payment instructions they carry. Financial values always come from the billing records."
      />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      <Alert severity="info" sx={{ mb: 2 }}>
        Template version {template.version}. Saving creates a new version for future documents; issuing an invoice freezes
        the issuer identity and payment instructions it was issued with.
      </Alert>

      <Grid container spacing={2}>
        <Grid item xs={12} lg={7}>
          <Card variant="outlined">
            <CardContent>
              <Typography variant="h6" fontWeight={800} sx={{ mb: 2 }}>Issuer and layout</Typography>
              <Stack spacing={2}>
                <TextField label="Issuer name" value={template.issuer_name} onChange={event => update({ issuer_name: event.target.value })} inputProps={{ maxLength: 150 }} helperText="Printed as the legal issuer. Leave empty to use the tenant brand." />
                <TextField label="Logo asset URL" value={template.logo_url} onChange={event => update({ logo_url: event.target.value })} inputProps={{ maxLength: 500 }} helperText="Only an image uploaded for this tenant." />
                <TextField label="Header text" value={template.header_text} onChange={event => update({ header_text: event.target.value })} inputProps={{ maxLength: 300 }} />
                <TextField label="Footer text" value={template.footer_text} onChange={event => update({ footer_text: event.target.value })} inputProps={{ maxLength: 300 }} />
                <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
                  <TextField label="Accent color" type="color" value={template.accent_color || '#16A34A'} onChange={event => update({ accent_color: event.target.value })} sx={{ width: 180 }} />
                  <TextField select label="Paper size" value={template.paper_size || 'A4'} onChange={event => update({ paper_size: event.target.value })} sx={{ minWidth: 180 }}>
                    <MenuItem value="A4">A4</MenuItem>
                    <MenuItem value="Letter">Letter</MenuItem>
                  </TextField>
                  <FormControlLabel control={<Switch checked={template.show_tax_id} onChange={event => update({ show_tax_id: event.target.checked })} />} label="Show issuer tax ID" />
                </Stack>
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} lg={5}>
          <Card variant="outlined">
            <CardContent>
              <Typography variant="h6" fontWeight={800} sx={{ mb: 2 }}>Payment instructions</Typography>
              <Stack spacing={2}>
                <TextField label="Bank name" value={template.bank_name} onChange={event => update({ bank_name: event.target.value })} inputProps={{ maxLength: 80 }} />
                <TextField label="Account name" value={template.bank_account_name} onChange={event => update({ bank_account_name: event.target.value })} inputProps={{ maxLength: 120 }} />
                <TextField label="Account number" value={template.bank_account_number} onChange={event => update({ bank_account_number: event.target.value })} inputProps={{ maxLength: 40 }} />
                <TextField label="Payment note" value={template.payment_note} onChange={event => update({ payment_note: event.target.value })} inputProps={{ maxLength: 300 }} multiline minRows={2} />
                <TextField label="QR asset URL" value={template.qr_image_url} onChange={event => update({ qr_image_url: event.target.value })} inputProps={{ maxLength: 500 }} helperText="Payment QR codes are printed only from a tenant-uploaded asset." />
                {bankIncomplete > 0 && bankIncomplete < 3 && (
                  <Alert severity="warning">Bank name, account name and account number must be filled together.</Alert>
                )}
              </Stack>
              <Divider sx={{ my: 2 }} />
              <Stack direction="row" spacing={1} justifyContent="flex-end">
                <Button variant="outlined" color="inherit" startIcon={<ResetIcon />} disabled={resetting || saving} onClick={reset}>
                  Reset template
                </Button>
                <Button variant="contained" startIcon={<SaveIcon />} disabled={saving || resetting} onClick={save}>
                  Save Invoice Template
                </Button>
              </Stack>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};

export default InvoiceStudioPage;
