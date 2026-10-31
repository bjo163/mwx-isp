import { expect, test, type Page } from '@playwright/test';

/**
 * Browser half of M19.5.1: two tenants drive the Brand Studio, Website Studio,
 * Package Studio and the public site in the same management shell.
 *
 * The server half is TestPostgresTenantBrandingAndWebsiteStayIsolated and
 * TestPostgresTenantPackageCatalogIsolatesPresentation in
 * test/integration/tenant_brand_experience_test.go, which prove the same
 * boundaries against PostgreSQL with real tenant JWTs. These tests only assert
 * what the UI reads and sends, because a hidden field is not an authorization
 * boundary and a mocked route is not server evidence.
 */

type TenantFixture = {
  slug: string;
  brandName: string;
  primaryColor: string;
  hero: string;
  packageName: string;
  displayName: string;
  price: number;
  presentationVisible: boolean;
  versions: number[];
};

const alpha: TenantFixture = {
  slug: 'alpha',
  brandName: 'Alpha Fiber',
  primaryColor: '#16A34A',
  hero: 'Alpha Internet Cepat',
  packageName: 'Alpha Paket 100 Mbps',
  displayName: 'Alpha Gold 100',
  price: 250000,
  presentationVisible: true,
  versions: [1, 2, 3],
};

const beta: TenantFixture = {
  slug: 'beta',
  brandName: 'Beta Net',
  primaryColor: '#2563EB',
  hero: 'Beta Internet Stabil',
  packageName: 'Beta Paket 50 Mbps',
  displayName: 'Beta Hidden 50',
  price: 150000,
  presentationVisible: false,
  versions: [1],
};

type Recorded = { method: string; pathname: string; body: unknown };

function websiteContent(tenant: TenantFixture) {
  return {
    sections: [
      { key: 'hero', title: tenant.hero, subtitle: 'Gagal konek? Hubungi kami.', body: '', image_url: '', cta_label: '', cta_url: '', visible: true, order: 0 },
      { key: 'footer', title: 'Footer', subtitle: '', body: tenant.brandName, image_url: '', cta_label: '', cta_url: '', visible: false, order: 1 },
    ],
  };
}

/**
 * Serves one tenant's API with its own data and records every write, so a test
 * can assert what the UI sent without a live backend.
 */
async function mockTenantApi(page: Page, tenant: TenantFixture, recorded: Recorded[]) {
  const state = {
    brandName: tenant.brandName,
    draft: websiteContent(tenant),
    versions: [...tenant.versions],
  };

  await page.route('**/api/v1/**', async route => {
    const request = route.request();
    const { pathname } = new URL(request.url());
    const method = request.method();
    const body = request.postDataJSON?.() ?? null;
    if (method !== 'GET') {
      recorded.push({ method, pathname, body });
    }

    if (pathname.endsWith('/public/branding')) {
      await route.fulfill({ json: { data: { contact_whatsapp: '628111222333' } } });
      return;
    }

    if (pathname.endsWith('/branding')) {
      if (method === 'PUT') {
        state.brandName = (body as { brand_name?: string })?.brand_name ?? state.brandName;
      }
      await route.fulfill({
        json: {
          data: {
            brand_name: state.brandName, short_name: tenant.slug.toUpperCase(),
            primary_color: tenant.primaryColor, secondary_color: '#0F172A',
            tagline: `${tenant.brandName} tagline`, theme: 'light', typography: 'system',
          },
        },
      });
      return;
    }

    if (pathname.endsWith('/tenant-website/draft')) {
      if (method === 'PUT') {
        state.draft = body as typeof state.draft;
      }
      await route.fulfill({ json: { data: { draft: state.draft, published_version: state.versions.length } } });
      return;
    }
    if (pathname.endsWith('/tenant-website/publish')) {
      state.versions = [...state.versions, state.versions.length + 1];
      await route.fulfill({ json: { data: { draft: state.draft, published_version: state.versions.length } } });
      return;
    }
    if (pathname.endsWith('/tenant-website/versions')) {
      await route.fulfill({
        json: { data: state.versions.map(version => ({ version, created_at: '2026-10-31T00:00:00Z' })) },
      });
      return;
    }
    if (pathname.includes('/tenant-website/public/')) {
      // Public pages resolve the tenant from the identifier, not the session,
      // so both fixtures stay reachable while admin routes remain tenant-scoped.
      const identifier = decodeURIComponent(pathname.split('/').pop() || '');
      const publicTenant = [alpha, beta].find(item => item.slug === identifier);
      if (!publicTenant) {
        await route.fulfill({ status: 404, json: { error: 'TENANT_NOT_FOUND', message: 'Active tenant not found' } });
        return;
      }
      await route.fulfill({ json: { data: publicTenant === tenant ? state.draft : websiteContent(publicTenant) } });
      return;
    }

    if (pathname.endsWith('/isp/packages') && method === 'GET') {
      await route.fulfill({
        json: {
          data: [
            {
              id: '11', code: 'PKG-11', name: tenant.packageName, price: tenant.price,
              description: 'Operational package fixture', billing_cycle: 'monthly', status: 'active',
            },
          ],
          total: 1,
        },
      });
      return;
    }
    if (pathname.endsWith('/isp/package-presentations')) {
      await route.fulfill({
        json: {
          data: [
            {
              package_id: '11', display_name: tenant.displayName, category: 'Home Broadband',
              short_description: 'Fixture presentation', badge: 'Best Seller',
              highlight_color: tenant.primaryColor, highlights: 'Tanpa FUP',
              display_order: 5, featured: true, public_visible: tenant.presentationVisible,
            },
          ],
        },
      });
      return;
    }
    if (pathname.endsWith('/presentation')) {
      const saved = method === 'PUT' ? body : {};
      await route.fulfill({
        json: {
          data: {
            display_name: tenant.displayName, category: 'Home Broadband',
            highlight_color: tenant.primaryColor, highlights: 'Tanpa FUP',
            display_order: 5, featured: true, public_visible: tenant.presentationVisible,
            ...(saved as Record<string, unknown>),
          },
        },
      });
      return;
    }

    await route.fulfill({ json: { data: [], total: 0 } });
  });
}

async function signIn(page: Page, tenant: TenantFixture, recorded: Recorded[]) {
  const token = [
    Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64url'),
    Buffer.from(JSON.stringify({
      sub: '1', username: `e2e-${tenant.slug}`, role: 'admin',
      exp: Math.floor(Date.now() / 1000) + 3600,
    })).toString('base64url'),
    'e2e-signature',
  ].join('.');

  await page.addInitScript(
    ({ token, tenant }) => {
      localStorage.setItem('token', token);
      localStorage.setItem('username', `e2e-${tenant.slug}`);
      localStorage.setItem('user', JSON.stringify({
        id: '1', username: `e2e-${tenant.slug}`, realname: `E2E ${tenant.slug}`,
        level: 'admin', status: 'enabled',
      }));
      localStorage.setItem('permissions', JSON.stringify([]));
      localStorage.setItem('locale', 'en-US');
    },
    { token, tenant: { slug: tenant.slug } },
  );
  await mockTenantApi(page, tenant, recorded);
}

for (const tenant of [alpha, beta]) {
  const neighbour = tenant === alpha ? beta : alpha;

  test(`${tenant.slug} brand studio reads and saves only its own brand`, async ({ page }) => {
    const recorded: Recorded[] = [];
    await signIn(page, tenant, recorded);
    await page.goto('/#/tenant/branding');

    const brandField = page.getByLabel('Public brand name');
    await expect(brandField).toHaveValue(tenant.brandName);
    await expect(page.getByText(neighbour.brandName, { exact: false })).toHaveCount(0);

    await brandField.fill(`${tenant.brandName} Plus`);
    await page.getByRole('button', { name: 'Save Brand Settings' }).click();

    const saved = recorded.find(entry => entry.method === 'PUT' && entry.pathname.endsWith('/branding'));
    expect(saved, 'the brand save must call PUT /branding').toBeTruthy();
    expect((saved?.body as { brand_name?: string }).brand_name).toBe(`${tenant.brandName} Plus`);
    await expect(page.getByLabel('Public brand name')).toHaveValue(`${tenant.brandName} Plus`);
  });

  test(`${tenant.slug} website studio edits its own draft, saves and publishes`, async ({ page }) => {
    const recorded: Recorded[] = [];
    await signIn(page, tenant, recorded);
    await page.goto('/#/tenant/website');

    // The selected hero section renders the tenant's own draft, not a neighbour's.
    await expect(page.getByText(tenant.hero, { exact: false }).first()).toBeVisible();
    await expect(page.getByText(neighbour.hero, { exact: false })).toHaveCount(0);

    await page.getByRole('button', { name: 'Save draft' }).click();
    await expect.poll(() => recorded.some(entry => entry.method === 'PUT' && entry.pathname.endsWith('/tenant-website/draft')), {
      message: 'saving a draft must call PUT /tenant-website/draft',
    }).toBe(true);

    await page.getByRole('button', { name: 'Publish website' }).click();
    await expect.poll(() => recorded.some(entry => entry.method === 'POST' && entry.pathname.endsWith('/tenant-website/publish')), {
      message: 'publishing must call POST /tenant-website/publish',
    }).toBe(true);

    // The history is the tenant's own: one row per published version plus the
    // snapshot the publish just added, never the neighbour's longer history.
    await expect(page.getByText(/^Version \d+$/)).toHaveCount(tenant.versions.length + 1);
  });

  test(`${tenant.slug} package studio keeps price operational and saves presentation only`, async ({ page }) => {
    const recorded: Recorded[] = [];
    await signIn(page, tenant, recorded);
    await page.goto('/#/isp/package-studio');

    const money = (value: number) => new Intl.NumberFormat('id-ID', {
      style: 'currency', currency: 'IDR', maximumFractionDigits: 0,
    }).format(value);

    await expect(page.getByText(/Operational source:/)).toContainText(tenant.packageName);
    await expect(page.getByText(/Operational source:/)).toContainText(money(tenant.price));
    await expect(page.getByText(neighbour.packageName, { exact: false })).toHaveCount(0);

    const displayField = page.getByLabel('Public display name');
    await expect(displayField).toHaveValue(tenant.displayName);

    await page.getByRole('button', { name: 'Save Package Presentation' }).click();
    const saved = recorded.find(entry => entry.method === 'PUT' && entry.pathname.endsWith('/presentation'));
    expect(saved, 'saving a presentation must call PUT /isp/packages/:id/presentation').toBeTruthy();
    const savedBody = saved?.body as Record<string, unknown>;
    expect(savedBody.display_name).toBe(tenant.displayName);
    expect(savedBody, 'the studio must not send operational pricing').not.toHaveProperty('price');
  });
}

test('public tenant site renders only the requested tenant', async ({ page }) => {
  const recorded: Recorded[] = [];
  await signIn(page, alpha, recorded);
  await page.goto('/#/site/alpha');
  await expect(page.getByText(alpha.hero, { exact: false }).first()).toBeVisible();
  await expect(page.getByText(beta.hero, { exact: false })).toHaveCount(0);
  // Footer is marked invisible in the fixture and must not be rendered.
  await expect(page.getByText(alpha.brandName, { exact: true })).toHaveCount(0);

  await page.goto('/#/site/beta');
  await expect(page.getByText(beta.hero, { exact: false }).first()).toBeVisible();
  await expect(page.getByText(alpha.hero, { exact: false })).toHaveCount(0);
});

test('unknown tenant site shows a non-leaking notice', async ({ page }) => {
  const recorded: Recorded[] = [];
  await signIn(page, alpha, recorded);
  await page.goto('/#/site/tidak-terdaftar');
  // The page surfaces the server's 404 message instead of any tenant content.
  await expect(page.getByRole('alert')).toBeVisible();
  await expect(page.getByRole('alert')).toContainText(/not found|not published/i);
  await expect(page.getByText(alpha.hero, { exact: false })).toHaveCount(0);
});
