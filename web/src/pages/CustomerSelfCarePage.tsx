import React, { useCallback, useEffect, useState } from 'react';
import {
  Alert, Box, Button, Card, CardContent, Chip, CircularProgress,
  Container, Divider, Paper, Stack, Table, TableBody, TableCell,
  TableHead, TableRow, TextField, Typography,
} from '@mui/material';
import { apiRequest } from '../utils/apiClient';

type CustomerPortalData = {
  customer: { customer_no: string; name: string; phone: string; email: string; address: string; status: string };
  subscriptions: Array<{ subscription_no: string; status: string; start_date: string; billing_day: number; package_name: string; package_price: number }>;
  invoices: Array<{ id: string; invoice_no: string; invoice_date: string; due_date: string; total: number; balance: number; status: string }>;
  payments: Array<{ payment_no: string; invoice_id: string; amount: number; method: string; paid_at: string; status: string }>;
  tickets: Array<{ ticket_no: string; subject: string; category: string; priority: string; status: string; created_at: string }>;
};

type OTPRequestResponse = { challenge_id: string; message: string; expires_in_seconds: number };
type OTPVerifyResponse = { access_token: string; token_type: string; expires_at: string };

const tokenKey = 'mwx-customer-session';

const money = (amount: number) => new Intl.NumberFormat('id-ID', {
  style: 'currency',
  currency: 'IDR',
  maximumFractionDigits: 0,
}).format(Number(amount) || 0);

const dateTime = (raw: string) => {
  if (!raw) return '—';
  const date = new Date(raw);
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleDateString('id-ID', {
    day: '2-digit', month: 'short', year: 'numeric',
  });
};

const CustomerSelfCarePage: React.FC = () => {
  const [tenantSlug, setTenantSlug] = useState(new URLSearchParams(window.location.search).get('tenant') ?? '');
  const [customerNo, setCustomerNo] = useState('');
  const [phone, setPhone] = useState('');
  const [challengeId, setChallengeId] = useState('');
  const [otp, setOtp] = useState('');
  const [token, setToken] = useState(() => sessionStorage.getItem(tokenKey) ?? '');
  const [data, setData] = useState<CustomerPortalData | null>(null);
  const [notice, setNotice] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [paymentLoading, setPaymentLoading] = useState<string | null>(null);

  const loadAccount = useCallback(async (accessToken: string) => {
    const result = await apiRequest<CustomerPortalData>('/customer-portal/me', {
      headers: { Authorization: 'Bearer ' + accessToken },
    });
    setData(result);
  }, []);

  useEffect(() => {
    if (!token) return;
    let active = true;
    setBusy(true);
    void apiRequest<CustomerPortalData>('/customer-portal/me', {
      headers: { Authorization: 'Bearer ' + token },
    }).then(result => {
      if (active) setData(result);
    }).catch(() => {
      if (!active) return;
      sessionStorage.removeItem(tokenKey);
      setToken('');
      setData(null);
    }).finally(() => {
      if (active) setBusy(false);
    });
    return () => { active = false; };
  }, [token]);

  const requestOTP = async (event: React.FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError('');
    setNotice('');
    try {
      const result = await apiRequest<OTPRequestResponse>('/customer-auth/request-otp', {
        method: 'POST',
        body: JSON.stringify({
          tenant_slug: tenantSlug.trim(),
          customer_no: customerNo.trim(),
          phone: phone.trim(),
        }),
      });
      setChallengeId(result.challenge_id);
      setNotice(result.message);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Tidak dapat memulai verifikasi.');
    } finally {
      setBusy(false);
    }
  };

  const verifyOTP = async (event: React.FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError('');
    try {
      const result = await apiRequest<OTPVerifyResponse>('/customer-auth/verify-otp', {
        method: 'POST',
        body: JSON.stringify({
          tenant_slug: tenantSlug.trim(),
          challenge_id: challengeId,
          code: otp.trim(),
        }),
      });
      sessionStorage.setItem(tokenKey, result.access_token);
      setToken(result.access_token);
      setNotice('');
      await loadAccount(result.access_token);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Kode verifikasi tidak valid atau kedaluwarsa.');
    } finally {
      setBusy(false);
    }
  };

  const payInvoice = async (invoiceId: string) => {
    if (!token) return;
    setPaymentLoading(invoiceId);
    setError('');
    try {
      const checkout = await apiRequest<{ redirect_url: string }>(
        '/customer-portal/invoices/' + encodeURIComponent(invoiceId) + '/payment-channel',
        {
          method: 'POST',
          headers: {
            Authorization: 'Bearer ' + token,
            'Idempotency-Key': window.crypto.randomUUID(),
          },
        },
      );
      if (!checkout?.redirect_url) throw new Error('Payment provider tidak mengembalikan tautan pembayaran.');
      window.location.assign(checkout.redirect_url);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Tidak dapat membuat pembayaran online.');
      setPaymentLoading(null);
    }
  };

  const logout = async () => {
    setBusy(true);
    try {
      if (token) {
        await apiRequest('/customer-portal/logout', {
          method: 'POST',
          headers: { Authorization: 'Bearer ' + token },
        });
      }
    } catch {
      // Local session is cleared even when the server cannot be reached.
    } finally {
      sessionStorage.removeItem(tokenKey);
      setToken('');
      setData(null);
      setChallengeId('');
      setOtp('');
      setBusy(false);
    }
  };

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: '#f3efe4', color: '#101010', py: { xs: 2, md: 5 } }}>
      <Container maxWidth="md">
        <Stack spacing={3}>
          <Box>
            <Typography variant="overline" fontWeight={900} letterSpacing=".18em">MWX-ISP / CUSTOMER ZONE</Typography>
            <Typography variant="h3" fontWeight={950} sx={{ letterSpacing: '-.04em' }}>Self-care portal</Typography>
            <Typography color="text.secondary">Kelola layanan internet, tagihan, pembayaran, dan laporan gangguan.</Typography>
          </Box>

          {error && <Alert severity="error" onClose={() => setError('')}>{error}</Alert>}
          {notice && <Alert severity="info">{notice}</Alert>}

          {!data ? (
            <Card variant="outlined" sx={{ border: '2px solid #111', borderRadius: 3, boxShadow: '5px 5px 0 #111' }}>
              <CardContent sx={{ p: { xs: 2, md: 3 } }}>
                {!challengeId ? (
                  <Stack component="form" spacing={2} onSubmit={requestOTP}>
                    <Typography variant="h5" fontWeight={900}>Masuk ke akun pelanggan</Typography>
                    <TextField label="Kode tenant / ISP" value={tenantSlug} onChange={e => setTenantSlug(e.target.value)} required fullWidth helperText="Gunakan slug tenant yang diberikan ISP Anda." />
                    <TextField label="Nomor pelanggan" value={customerNo} onChange={e => setCustomerNo(e.target.value)} required fullWidth />
                    <TextField label="Nomor WhatsApp terdaftar" value={phone} onChange={e => setPhone(e.target.value)} required fullWidth />
                    <Button type="submit" variant="contained" size="large" disabled={busy}>
                      {busy ? <CircularProgress size={22} color="inherit" /> : 'Kirim kode verifikasi'}
                    </Button>
                    <Typography variant="caption" color="text.secondary">Kode hanya dikirim ke nomor WhatsApp yang tersimpan di akun pelanggan. Respons selalu sama demi melindungi privasi akun.</Typography>
                  </Stack>
                ) : (
                  <Stack component="form" spacing={2} onSubmit={verifyOTP}>
                    <Typography variant="h5" fontWeight={900}>Verifikasi WhatsApp</Typography>
                    <Typography color="text.secondary">Masukkan kode 6 digit yang dikirim ke WhatsApp terdaftar. Kode berlaku selama 5 menit.</Typography>
                    <TextField label="Kode OTP" value={otp} onChange={e => setOtp(e.target.value.replace(/\D/g, '').slice(0, 6))} required inputProps={{ inputMode: 'numeric', maxLength: 6 }} fullWidth />
                    <Button type="submit" variant="contained" size="large" disabled={busy || otp.length !== 6}>
                      {busy ? <CircularProgress size={22} color="inherit" /> : 'Verifikasi dan masuk'}
                    </Button>
                    <Button variant="text" onClick={() => { setChallengeId(''); setOtp(''); setNotice(''); }} disabled={busy}>Gunakan nomor lain</Button>
                  </Stack>
                )}
              </CardContent>
            </Card>
          ) : (
            <Stack spacing={2.5}>
              <Card variant="outlined" sx={{ border: '2px solid #111', borderRadius: 3, boxShadow: '5px 5px 0 #111' }}>
                <CardContent>
                  <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" spacing={2}>
                    <Box>
                      <Typography variant="overline">CUSTOMER ACCOUNT</Typography>
                      <Typography variant="h4" fontWeight={950}>{data.customer.name}</Typography>
                      <Typography color="text.secondary">{data.customer.customer_no} · {data.customer.phone}</Typography>
                    </Box>
                    <Stack alignItems={{ xs: 'flex-start', sm: 'flex-end' }} spacing={1}>
                      <Chip label={data.customer.status.toUpperCase()} color={data.customer.status === 'active' ? 'success' : 'warning'} />
                      <Button variant="outlined" color="inherit" onClick={logout} disabled={busy}>Keluar</Button>
                    </Stack>
                  </Stack>
                  <Divider sx={{ my: 2 }} />
                  <Typography>{data.customer.address}</Typography>
                </CardContent>
              </Card>

              <Card variant="outlined">
                <CardContent>
                  <Typography variant="h6" fontWeight={900} gutterBottom>Layanan aktif dan paket</Typography>
                  {data.subscriptions.length === 0 ? <Typography color="text.secondary">Belum ada layanan terdaftar.</Typography> : data.subscriptions.map(sub => (
                    <Paper key={sub.subscription_no} variant="outlined" sx={{ p: 2, mb: 1.5 }}>
                      <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" spacing={1}>
                        <Box>
                          <Typography fontWeight={800}>{sub.package_name || 'Paket internet'}</Typography>
                          <Typography variant="caption" color="text.secondary">{sub.subscription_no} · Mulai {dateTime(sub.start_date)}</Typography>
                        </Box>
                        <Box textAlign={{ sm: 'right' }}>
                          <Chip size="small" label={sub.status.toUpperCase()} />
                          <Typography fontWeight={800}>{money(sub.package_price)}</Typography>
                        </Box>
                      </Stack>
                    </Paper>
                  ))}
                </CardContent>
              </Card>

              <Card variant="outlined">
                <CardContent>
                  <Typography variant="h6" fontWeight={900} gutterBottom>Tagihan</Typography>
                  {data.invoices.length === 0 ? <Typography color="text.secondary">Belum ada tagihan.</Typography> : (
                    <Table size="small">
                      <TableHead><TableRow><TableCell>No. Invoice</TableCell><TableCell>Jatuh tempo</TableCell><TableCell align="right">Total</TableCell><TableCell align="right">Sisa</TableCell><TableCell>Status</TableCell></TableRow></TableHead>
                      <TableBody>{data.invoices.map(invoice => (
                        <TableRow key={invoice.id}>
                          <TableCell>{invoice.invoice_no}</TableCell><TableCell>{dateTime(invoice.due_date)}</TableCell>
                          <TableCell align="right">{money(invoice.total)}</TableCell><TableCell align="right">{money(invoice.balance)}</TableCell>
                          <TableCell>
                            <Stack spacing={0.5} alignItems="flex-start">
                              <Chip size="small" label={invoice.status.toUpperCase()} color={invoice.balance > 0 ? 'warning' : 'success'} />
                              {invoice.balance > 0 && (
                                <Button size="small" variant="contained"
                                  disabled={paymentLoading === String(invoice.id)}
                                  onClick={() => void payInvoice(String(invoice.id))}
                                  sx={{ textTransform: 'none', whiteSpace: 'nowrap' }}>
                                  {paymentLoading === String(invoice.id) ? 'Menyiapkan...' : 'Bayar Online'}
                                </Button>
                              )}
                            </Stack>
                          </TableCell>
                        </TableRow>
                      ))}</TableBody>
                    </Table>
                  )}
                </CardContent>
              </Card>

              <Card variant="outlined">
                <CardContent>
                  <Typography variant="h6" fontWeight={900} gutterBottom>Riwayat pembayaran</Typography>
                  {data.payments.length === 0 ? <Typography color="text.secondary">Belum ada pembayaran.</Typography> : data.payments.map(payment => (
                    <Stack key={payment.payment_no} direction="row" justifyContent="space-between" sx={{ py: 1, borderBottom: '1px solid', borderColor: 'divider' }}>
                      <Box><Typography fontWeight={700}>{payment.payment_no}</Typography><Typography variant="caption" color="text.secondary">{dateTime(payment.paid_at)} · {payment.method}</Typography></Box>
                      <Typography fontWeight={800}>{money(payment.amount)}</Typography>
                    </Stack>
                  ))}
                </CardContent>
              </Card>

              <Card variant="outlined">
                <CardContent>
                  <Typography variant="h6" fontWeight={900} gutterBottom>Laporan gangguan</Typography>
                  {data.tickets.length === 0 ? <Typography color="text.secondary">Belum ada tiket gangguan.</Typography> : data.tickets.map(ticket => (
                    <Stack key={ticket.ticket_no} direction="row" justifyContent="space-between" spacing={2} sx={{ py: 1, borderBottom: '1px solid', borderColor: 'divider' }}>
                      <Box><Typography fontWeight={700}>{ticket.subject}</Typography><Typography variant="caption" color="text.secondary">{ticket.ticket_no} · {dateTime(ticket.created_at)}</Typography></Box>
                      <Chip size="small" label={ticket.status.toUpperCase()} />
                    </Stack>
                  ))}
                </CardContent>
              </Card>
            </Stack>
          )}
        </Stack>
      </Container>
    </Box>
  );
};

export default CustomerSelfCarePage;
