import { MenuItem, ListItemIcon, ListItemText } from '@mui/material';
import { useLocale, useSetLocale } from 'react-admin';
import LanguageIcon from '@mui/icons-material/Language';
import { locales } from '../i18n';

export const LanguageSwitcher = ({ onSelect }: { onSelect?: () => void } = {}) => {
  const setLocale = useSetLocale();
  const currentLocale = useLocale();
  return (
    <>
      {locales.map(({ locale, name }) => (
        <MenuItem
          key={locale}
          onClick={() => {
            setLocale(locale);
            onSelect?.();
          }}
          selected={locale === currentLocale}
        >
          <ListItemIcon>
            <LanguageIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText>{name}</ListItemText>
        </MenuItem>
      ))}
    </>
  );
};
