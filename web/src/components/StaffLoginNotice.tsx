import { Alert, AlertTitle, Button, Stack } from '@mui/material';
import type { SxProps, Theme } from '@mui/material/styles';
import LoginIcon from '@mui/icons-material/Login';
import WhatsAppIcon from '@mui/icons-material/WhatsApp';
import { useNavigate } from 'react-router-dom';

/**
 * The portal lookup, voucher check and registration APIs sit behind the
 * operator JWT middleware. An anonymous visitor therefore gets 401, not a data
 * error; these surfaces must say so and offer a way forward instead of showing
 * the raw backend message. Customer self-service login is a separate contract
 * (TR-N002) and is deliberately not created here.
 */
export const STAFF_LOGIN_REQUIRED_MESSAGE =
  'Layanan ini memerlukan login staf operator. Silakan login, atau hubungi WhatsApp resmi kami untuk dibantu.';

export const isStaffLoginRequired = (err: unknown): boolean =>
  (err as { status?: number } | null)?.status === 401;

export interface SurfaceFailure {
  message: string;
  needsStaffLogin: boolean;
}

export const describeSurfaceFailure = (err: unknown, fallback: string): SurfaceFailure => {
  const needsStaffLogin = isStaffLoginRequired(err);
  return {
    message: needsStaffLogin ? STAFF_LOGIN_REQUIRED_MESSAGE : (err as Error | null)?.message || fallback,
    needsStaffLogin,
  };
};

interface StaffLoginNoticeProps {
  whatsappNumber: string;
  whatsappMessage?: string;
  sx?: SxProps<Theme>;
}

export const StaffLoginNotice = ({ whatsappNumber, whatsappMessage, sx }: StaffLoginNoticeProps) => {
  const navigate = useNavigate();
  const digits = whatsappNumber.replace(/[^0-9]/g, '');
  const openWhatsApp = () => {
    const target = digits ? `https://wa.me/${digits}` : 'https://wa.me/';
    const text = whatsappMessage ?? 'Halo, saya perlu bantuan untuk mengakses layanan MWX-ISP.';
    window.open(`${target}?text=${encodeURIComponent(text)}`, '_blank');
  };

  return (
    <Alert severity="info" sx={sx}>
      <AlertTitle>Perlu login staf operator</AlertTitle>
      {STAFF_LOGIN_REQUIRED_MESSAGE}
      <Stack direction="row" spacing={1} sx={{ mt: 1.5, flexWrap: 'wrap', gap: 1 }}>
        <Button size="small" variant="contained" startIcon={<LoginIcon />} onClick={() => navigate('/login')}>
          Login staf
        </Button>
        <Button size="small" variant="outlined" color="success" startIcon={<WhatsAppIcon />} onClick={openWhatsApp}>
          Hubungi WhatsApp
        </Button>
      </Stack>
    </Alert>
  );
};

export default StaffLoginNotice;
