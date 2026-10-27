import { expect, test, type Page } from '@playwright/test';

/**
 * Browser half of the role matrix in docs/access-control-matrix.md.
 *
 * These tests assert that a lower role never sees a control the server would
 * reject. The server remains the enforcement point: every mutation listed in
 * the matrix is rejected independently by TestMutationRouteRoleMatrix in
 * internal/adminapi/access_control_test.go, which drives the real router with
 * signed JWTs. Hiding a button is never accepted as a substitute for that gate.
 */

type Role = 'operator' | 'admin' | 'super' | 'platform';

interface Session {
  level: string;
  permissions: string[];
}

const sessions: Record<Role, Session> = {
  operator: { level: 'operator', permissions: [] },
  admin: { level: 'admin', permissions: [] },
  super: { level: 'super', permissions: [] },
  platform: { level: 'super', permissions: ['platform_admin'] },
};

const roles = Object.keys(sessions) as Role[];

const adminRoles: Role[] = ['admin', 'super', 'platform'];
const platformRoles: Role[] = ['platform'];

const menuMatrix: Array<{ href: string; visibleFor: Role[]; serverRule: string }> = [
  { href: '#/network/ipam', visibleFor: adminRoles, serverRule: 'Network writes are admin-only' },
  { href: '#/network/odp', visibleFor: adminRoles, serverRule: 'Network writes are admin-only' },
  { href: '#/operations', visibleFor: adminRoles, serverRule: 'Network monitoring is admin-only' },
  { href: '#/system/operators', visibleFor: adminRoles, serverRule: 'Operator management requires admin or super' },
  { href: '#/system/website', visibleFor: platformRoles, serverRule: 'Branding writes require platform_admin' },
  { href: '#/system/branding', visibleFor: platformRoles, serverRule: 'Branding writes require platform_admin' },
  { href: '#/system/config', visibleFor: platformRoles, serverRule: 'System config requires platform_admin' },
  { href: '#/system/certificate', visibleFor: platformRoles, serverRule: 'Certificates require platform_admin' },
  { href: '#/platform/tenants', visibleFor: platformRoles, serverRule: 'Platform routes require platform_admin' },
];

const fixtureTicket = {
  id: 'ticket-1',
  ticket_no: 'TKT-0001',
  customer_id: 'customer-1',
  subscription_id: 'subscription-1',
  subject: 'E2E fixture ticket',
  category: 'los_red',
  priority: 'normal',
  status: 'open',
  assigned_technician: 'Field Tech',
  technician_phone: '081200000000',
  description: 'Fixture row used to expose role-gated row actions.',
  created_at: new Date('2026-01-01T00:00:00Z').toISOString(),
};

function fakeJWT(level: string): string {
  const header = Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64url');
  const payload = Buffer.from(
    JSON.stringify({
      sub: '1',
      username: 'e2e-user',
      role: level,
      exp: Math.floor(Date.now() / 1000) + 3600,
    }),
  ).toString('base64url');
  return `${header}.${payload}.e2e-signature`;
}

/**
 * Serves the read-only fixtures the management shell needs. Mutations are not
 * part of this suite; the Go matrix test covers them against the real router.
 */
async function mockReadOnlyApi(page: Page) {
  await page.route('**/api/v1/**', async route => {
    const request = route.request();
    const { pathname } = new URL(request.url());

    if (pathname.endsWith('/public/branding')) {
      await route.fulfill({ json: { data: {} } });
      return;
    }
    if (request.method() !== 'GET') {
      await route.fulfill({ status: 403, json: { message: 'UI fixture is read-only' } });
      return;
    }
    if (pathname.endsWith('/network/diagnostics/flapping')) {
      await route.fulfill({ json: { data: { flapping_count: 0, subscribers: [] } } });
      return;
    }
    if (pathname.endsWith('/isp/tickets')) {
      await route.fulfill({ json: { data: [fixtureTicket], total: 1 } });
      return;
    }
    if (pathname.endsWith('/isp/vouchers/batches')) {
      await route.fulfill({ json: { data: [] } });
      return;
    }
    await route.fulfill({ json: { data: [], total: 0 } });
  });
}

async function signIn(page: Page, role: Role) {
  const session = sessions[role];
  const user = {
    id: '1',
    username: `e2e-${role}`,
    realname: `E2E ${role}`,
    level: session.level,
    status: 'enabled',
  };

  await page.addInitScript(
    ({ token, user, permissions }) => {
      localStorage.setItem('token', token);
      localStorage.setItem('username', user.username);
      localStorage.setItem('user', JSON.stringify(user));
      localStorage.setItem('permissions', JSON.stringify(permissions));
      localStorage.setItem('locale', 'en-US');
    },
    { token: fakeJWT(session.level), user, permissions: session.permissions },
  );
  await mockReadOnlyApi(page);
}

async function openListPage(page: Page, hash: string) {
  await page.goto(`/${hash}`);
  // The mocked fixtures are empty lists, so a datagrid header may never render.
  // Wait for the shell to settle instead.
  await page.waitForLoadState('networkidle');
}

for (const role of roles) {
  test(`menu gating for ${role}`, async ({ page }) => {
    await signIn(page, role);
    await openListPage(page, '#/isp/customers');

    await expect(page.locator('a[href="#/isp/customers"]').first()).toBeVisible();

    for (const item of menuMatrix) {
      const expected = item.visibleFor.includes(role) ? 1 : 0;
      await expect(
        page.locator(`a[href="${item.href}"]`),
        `${role} menu entry ${item.href} (${item.serverRule})`,
      ).toHaveCount(expected);
    }
  });

  test(`customer create control for ${role}`, async ({ page }) => {
    await signIn(page, role);
    await openListPage(page, '#/isp/customers');

    const createLink = page.locator('a[href="#/isp/customers/create"]');
    const expected = adminRoles.includes(role) ? 1 : 0;
    await expect(createLink, `${role} customer create`).toHaveCount(expected);
  });

  test(`radius user controls for ${role}`, async ({ page }) => {
    await signIn(page, role);
    await openListPage(page, '#/radius/users');

    const createLink = page.locator('a[href="#/radius/users/create"]');
    const importButton = page.getByRole('button', { name: 'Batch Import' });
    const expected = adminRoles.includes(role) ? 1 : 0;
    await expect(createLink, `${role} radius user create`).toHaveCount(expected);
    await expect(importButton, `${role} radius user import`).toHaveCount(expected);
    // Read-only export stays available to every role.
    await expect(page.getByRole('button', { name: /Export/i }).first()).toBeVisible();
  });
}

test('operator keeps the voucher print control but loses batch generation', async ({ page }) => {
  await signIn(page, 'operator');
  await page.goto('/#/isp/vouchers');

  const print = page.getByRole('button', { name: /Print/i }).first();
  await expect(print).toBeVisible();
  await expect(page.getByRole('button', { name: 'Generate Batch' })).toHaveCount(0);
});

test('admin keeps the voucher batch generation control', async ({ page }) => {
  await signIn(page, 'admin');
  await page.goto('/#/isp/vouchers');

  await expect(page.getByRole('button', { name: /Print/i }).first()).toBeVisible();
  await expect(page.getByRole('button', { name: 'Generate Batch' })).toHaveCount(1);
});

test('operator keeps ticket creation but loses WhatsApp dispatch', async ({ page }) => {
  await signIn(page, 'operator');
  await page.goto('/#/isp/tickets');

  await expect(page.getByRole('button', { name: 'New Ticket' })).toBeVisible();
  await expect(page.getByText('E2E fixture ticket')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Dispatch WA' })).toHaveCount(0);
});

test('admin sees WhatsApp dispatch on a dispatchable ticket', async ({ page }) => {
  await signIn(page, 'admin');
  await page.goto('/#/isp/tickets');

  await expect(page.getByRole('button', { name: 'New Ticket' })).toBeVisible();
  await expect(page.getByText('E2E fixture ticket')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Dispatch WA' })).toHaveCount(1);
});
