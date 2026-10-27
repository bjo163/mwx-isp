import { useGetIdentity } from 'react-admin';

// Mirrors adminapi.requireAdmin(): only tenant super and admin levels may run
// mutation endpoints. The server enforces this; the hook keeps the UI honest.
const ADMIN_LEVELS = ['super', 'admin'];

export const useIsAdmin = (): boolean => {
  const { data: identity } = useGetIdentity();
  const level = (identity as { level?: string } | undefined)?.level;
  return ADMIN_LEVELS.includes(level ?? '');
};
