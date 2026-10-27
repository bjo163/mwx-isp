import polyglotI18nProvider from 'ra-i18n-polyglot';
import enUS from './en-US';
import idID from './id-ID';

const translations = {
  'en-US': enUS,
  'id-ID': idID,
};

type SupportedLocale = keyof typeof translations;

export const locales: Array<{ locale: SupportedLocale; name: string }> = [
  { locale: 'en-US', name: 'English' },
  { locale: 'id-ID', name: 'Bahasa Indonesia' },
];

const isSupportedLocale = (locale: string | null): locale is SupportedLocale =>
  locale === 'en-US' || locale === 'id-ID';

const getDefaultLocale = (): SupportedLocale => {
  const savedLocale = localStorage.getItem('locale');
  if (!isSupportedLocale(savedLocale)) {
    localStorage.setItem('locale', 'en-US');
    return 'en-US';
  }
  return savedLocale;
};

const baseI18nProvider = polyglotI18nProvider(
  (locale) => translations[locale as SupportedLocale] || translations['en-US'],
  getDefaultLocale(),
  locales.map(({ locale, name }) => ({ locale, name })),
  { allowMissing: true }
);

// Persist the operator's choice so a reload keeps the selected language.
export const i18nProvider = {
  ...baseI18nProvider,
  changeLocale: (locale: string) => {
    localStorage.setItem('locale', isSupportedLocale(locale) ? locale : 'en-US');
    return baseI18nProvider.changeLocale(locale);
  },
};
