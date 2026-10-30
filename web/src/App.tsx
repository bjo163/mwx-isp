import { Admin, Resource, CustomRoutes } from 'react-admin';
import { Route } from 'react-router-dom';
import { Box, CircularProgress, Typography } from '@mui/material';
import { useEffect, useMemo, useState } from 'react';
import { dataProvider } from './providers/dataProvider';
import { authProvider } from './providers/authProvider';
import { i18nProvider } from './i18n';
import Dashboard from './pages/Dashboard';
import AccountSettings from './pages/AccountSettings';
import { SystemConfigPage } from './pages/SystemConfigPage';
import OperationsPage from './pages/OperationsPage';
import UserGuidePage from './pages/UserGuidePage';
import BrandingPage from './pages/BrandingPage';
import TenantBrandingPage from './pages/TenantBrandingPage';
import PackageStudioPage from './pages/PackageStudioPage';
import { PlatformTenantsPage } from './pages/PlatformTenantsPage';
import { LoginPage } from './pages/LoginPage';
import LandingPage from './pages/LandingPage';
import CustomerPortalPage from './pages/CustomerPortalPage';
import HotspotVouchersPage from './pages/HotspotVouchersPage';
import IPAMPage from './pages/IPAMPage';
import ODPPage from './pages/ODPPage';
import TroubleTicketsPage from './pages/TroubleTicketsPage';
import WebsiteEditorPage from './pages/WebsiteEditorPage';
import { CustomLayout, CustomError } from './components';
import { createAppTheme } from './theme';
import { BrandingContext, defaultProductBranding, useBranding } from './branding/BrandingContext';
import type { ProductBranding, TenantBranding } from './branding/BrandingContext';
import {
  CustomerList, CustomerCreate, CustomerEdit, CustomerShow,
  PackageList, PackageCreate, PackageEdit,
  SubscriptionList, SubscriptionCreate, SubscriptionShow,
  InvoiceList, InvoiceShow, PaymentList, PaymentCreate, PaymentShow,
} from './resources/isp';

const CustomLoading = () => {
  const { branding } = useBranding();
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '100vh',
        bgcolor: 'background.default',
        color: 'text.primary',
        gap: 2,
      }}
    >
      <CircularProgress size={36} color="primary" />
      <Typography variant="body2" color="text.secondary">Loading {branding.product_name}...</Typography>
    </Box>
  );
};

// 导入资源组件
import {
  RadiusUserList,
  RadiusUserEdit,
  RadiusUserCreate,
  RadiusUserShow,
} from './resources/radiusUsers';
import { OnlineSessionList, OnlineSessionShow } from './resources/onlineSessions';
import { AccountingList, AccountingShow } from './resources/accounting';
import {
  RadiusProfileList,
  RadiusProfileEdit,
  RadiusProfileCreate,
  RadiusProfileShow,
} from './resources/radiusProfiles';
import {
  NASList,
  NASEdit,
  NASCreate,
  NASShow,
} from './resources/nas';
import {
  NodeList,
  NodeEdit,
  NodeCreate,
  NodeShow,
} from './resources/nodes';
import {
  OperatorList,
  OperatorEdit,
  OperatorCreate,
  OperatorShow,
} from './resources/operators';
import {
  CertificateList,
  CertificateEdit,
  CertificateCreate,
  CertificateShow,
  CertificateIcon,
} from './resources/certificates';

type AppProps = { initialBranding?: ProductBranding };

const App = ({ initialBranding = defaultProductBranding }: AppProps) => {
  const [branding, setBranding] = useState(initialBranding);
  const [tenantBranding, setTenantBranding] = useState<TenantBranding | null>(null);
  const effectiveBranding = useMemo<ProductBranding>(() => ({
    ...branding,
    product_name: tenantBranding?.brand_name || tenantBranding?.company_name || branding.product_name,
    short_name: tenantBranding?.short_name || branding.short_name,
    tagline: tenantBranding?.tagline || branding.tagline,
    accent_color: tenantBranding?.accent_color || tenantBranding?.primary_color || branding.accent_color,
    logo_url: tenantBranding?.dark_logo_url || tenantBranding?.logo_url || branding.logo_url,
  }), [branding, tenantBranding]);
  const darkTheme = useMemo(() => createAppTheme('dark', effectiveBranding.accent_color), [effectiveBranding.accent_color]);
  const lightTheme = useMemo(() => createAppTheme('light', effectiveBranding.accent_color), [effectiveBranding.accent_color]);

  useEffect(() => {
    const host = window.location.hostname;
    const parts = host.split('.');
    if (parts.length >= 3 && host.endsWith('e-isp.id')) {
      const subdomain = parts[0];
      if (subdomain && subdomain !== 'www' && subdomain !== 'e-isp') {
        fetch(`/api/v1/tenant-branding/${subdomain}`, {
          headers: { Accept: 'application/json' }
        })
        .then(async response => {
          if (response.ok) {
            const result = await response.json();
            const tb = result?.data;
            if (tb) {
              setTenantBranding(tb);
            }
          }
        })
        .catch(() => {});
      }
    }
  }, []);

  useEffect(() => {
    document.title = `${effectiveBranding.product_name} - ISP Management`;
    let icon = document.querySelector<HTMLLinkElement>('link[rel="icon"]');
    if (!icon) {
      icon = document.createElement('link');
      icon.rel = 'icon';
      document.head.appendChild(icon);
    }
    icon.href = effectiveBranding.logo_url || '/admin/mwx-isp.svg';
    icon.type = effectiveBranding.logo_url ? 'image/png' : 'image/svg+xml';
    if (tenantBranding?.favicon_url) {
      let favicon = document.querySelector<HTMLLinkElement>('link[rel="shortcut icon"]');
      if (!favicon) {
        favicon = document.createElement('link');
        favicon.rel = 'shortcut icon';
        document.head.appendChild(favicon);
      }
      favicon.href = tenantBranding.favicon_url;
    }
  }, [effectiveBranding.product_name, effectiveBranding.logo_url, tenantBranding?.favicon_url]);

  return (
  <BrandingContext.Provider value={{ branding: effectiveBranding, tenantBranding, updateBranding: setBranding, updateTenantBranding: setTenantBranding }}>
  <Admin
    dataProvider={dataProvider}
    authProvider={authProvider}
    i18nProvider={i18nProvider}
    dashboard={Dashboard}
    loginPage={LoginPage}
    title={effectiveBranding.product_name}
    lightTheme={lightTheme}
    darkTheme={darkTheme}
    defaultTheme="dark"
    layout={CustomLayout}
    loading={CustomLoading}
    error={CustomError}
    requireAuth
  >
    <Resource name="isp/customers" list={CustomerList} create={CustomerCreate} edit={CustomerEdit} show={CustomerShow} />
    <Resource name="isp/packages" list={PackageList} create={PackageCreate} edit={PackageEdit} />
    <Resource name="isp/subscriptions" list={SubscriptionList} create={SubscriptionCreate} show={SubscriptionShow} />
    <Resource name="isp/invoices" list={InvoiceList} show={InvoiceShow} />
    <Resource name="isp/payments" list={PaymentList} create={PaymentCreate} show={PaymentShow} />
    {/* RADIUS User management */}
    <Resource
      name="radius/users"
      list={RadiusUserList}
      edit={RadiusUserEdit}
      create={RadiusUserCreate}
      show={RadiusUserShow}
    />

    {/* OnlineSession */}
    <Resource
      name="radius/online"
      list={OnlineSessionList}
      show={OnlineSessionShow}
    />

    {/* Accounting records */}
    <Resource
      name="radius/accounting"
      list={AccountingList}
      show={AccountingShow}
    />

    {/* RADIUS Settings */}
    <Resource
      name="radius/profiles"
      list={RadiusProfileList}
      edit={RadiusProfileEdit}
      create={RadiusProfileCreate}
      show={RadiusProfileShow}
    />

    {/* NAS device management */}
    <Resource
      name="network/nas"
      list={NASList}
      edit={NASEdit}
      create={NASCreate}
      show={NASShow}
    />

    {/* Network node */}
    <Resource
      name="network/nodes"
      list={NodeList}
      edit={NodeEdit}
      create={NodeCreate}
      show={NodeShow}
    />

    {/* Operator management */}
    <Resource
      name="system/operators"
      list={OperatorList}
      edit={OperatorEdit}
      create={OperatorCreate}
      show={OperatorShow}
    />

    {/* Certificate management */}
    <Resource
      name="system/certificate"
      list={CertificateList}
      edit={CertificateEdit}
      create={CertificateCreate}
      show={CertificateShow}
      icon={CertificateIcon}
    />

    {/* Custom routes */}
    <CustomRoutes noLayout>
      <Route path="/home" element={<LandingPage />} />
      <Route path="/portal" element={<CustomerPortalPage />} />
    </CustomRoutes>
    <CustomRoutes>
      <Route path="/account/settings" element={<AccountSettings />} />
      <Route path="/system/config" element={<SystemConfigPage />} />
      <Route path="/system/website" element={<WebsiteEditorPage />} />
      <Route path="/system/branding" element={<BrandingPage />} />
      <Route path="/tenant/branding" element={<TenantBrandingPage />} />
      <Route path="/isp/package-studio" element={<PackageStudioPage />} />
      <Route path="/operations" element={<OperationsPage />} />
      <Route path="/isp/vouchers" element={<HotspotVouchersPage />} />
      <Route path="/network/ipam" element={<IPAMPage />} />
      <Route path="/network/odp" element={<ODPPage />} />
      <Route path="/isp/tickets" element={<TroubleTicketsPage />} />
      <Route path="/guide" element={<UserGuidePage />} />
      <Route path="/platform/tenants" element={<PlatformTenantsPage />} />
    </CustomRoutes>
    </Admin>
  </BrandingContext.Provider>
  );
};

export default App;
