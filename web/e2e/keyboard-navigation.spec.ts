import { expect, test, type Locator, type Page } from '@playwright/test';

/**
 * ENG-053: keyboard-only walk through the customer, billing and NAS lists plus
 * the critical payment dialog, against the WCAG 2.2 expectations that apply
 * here — 2.1.1 (keyboard), 2.4.3 (focus order), 2.4.7 (focus visible) and
 * 2.4.11 (focus not obscured).
 *
 * The fixtures are read-only. Mutations are not exercised; the goal is that a
 * mouse is never required to open a record or to leave a dialog.
 */

const ADMIN = { level: 'admin', username: 'e2e-admin' };

const customer = {
  id: 'cust-1',
  customer_no: 'CUST-0001',
  name: 'Acme Fiber',
  phone: '081200000001',
  email: 'acme@example.test',
  city: 'Jakarta',
  province: 'DKI Jakarta',
  package_name: 'Home 100',
  radius_username: 'acme',
  status: 'active',
  outstanding: 0,
};

const invoice = {
  id: 'inv-1',
  invoice_no: 'INV-000001',
  customer_id: 'cust-1',
  customer_name: 'Acme Fiber',
  subscription_no: 'SUB-000001',
  package_name: 'Home 100',
  invoice_date: '2026-10-01T00:00:00Z',
  due_date: '2026-10-20T00:00:00Z',
  total: 350000,
  paid_amount: 0,
  balance: 350000,
  status: 'unpaid',
};

const nas = {
  id: 'nas-1',
  name: 'Core NAS Jakarta',
  identifier: 'core-nas-jkt',
  ipaddr: '10.0.0.2',
  status: 'enabled',
  description: 'E2E fixture NAS',
};

function fakeJWT(): string {
  const header = Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64url');
  const payload = Buffer.from(
    JSON.stringify({
      sub: '1',
      username: ADMIN.username,
      role: ADMIN.level,
      exp: Math.floor(Date.now() / 1000) + 3600,
    }),
  ).toString('base64url');
  return `${header}.${payload}.e2e-signature`;
}

async function signInAsAdmin(page: Page) {
  const token = fakeJWT();
  const user = {
    id: '1',
    username: ADMIN.username,
    realname: 'E2E Admin',
    level: ADMIN.level,
    status: 'enabled',
  };
  await page.addInitScript(
    ({ token, user }) => {
      localStorage.setItem('token', token);
      localStorage.setItem('username', user.username);
      localStorage.setItem('user', JSON.stringify(user));
      localStorage.setItem('permissions', '[]');
      localStorage.setItem('locale', 'en-US');
    },
    { token, user },
  );
}

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
    if (pathname.endsWith('/isp/customers/cust-1')) {
      await route.fulfill({ json: { data: customer } });
      return;
    }
    if (pathname.endsWith('/isp/customers')) {
      await route.fulfill({ json: { data: [customer], total: 1 } });
      return;
    }
    if (pathname.endsWith('/isp/invoices/inv-1')) {
      await route.fulfill({ json: { data: invoice } });
      return;
    }
    if (pathname.endsWith('/isp/invoices')) {
      await route.fulfill({ json: { data: [invoice], total: 1 } });
      return;
    }
    if (pathname.endsWith('/network/nas/nas-1')) {
      await route.fulfill({ json: { data: nas } });
      return;
    }
    if (pathname.endsWith('/network/nas')) {
      await route.fulfill({ json: { data: [nas], total: 1 } });
      return;
    }
    await route.fulfill({ json: { data: [], total: 0 } });
  });
}

/** Presses Tab until `target` holds focus. Fails when it is unreachable. */
async function tabTo(page: Page, target: Locator, maxPresses = 250) {
  await expect(target).toHaveCount(1);
  for (let press = 0; press <= maxPresses; press++) {
    const focused = await target.evaluate(el => el === document.activeElement);
    if (focused) {
      await expect(target).toBeFocused();
      return;
    }
    await page.keyboard.press('Tab');
  }
  throw new Error(`element was not reachable with ${maxPresses} Tab presses`);
}

async function activeElementInside(page: Page, container: Locator): Promise<boolean> {
  return container.evaluate(el => el.contains(document.activeElement));
}

test.beforeEach(async ({ page }) => {
  await signInAsAdmin(page);
  await mockReadOnlyApi(page);
});

test('customer row opens with the keyboard alone', async ({ page }) => {
  await page.goto('/#/isp/customers');

  const rowLink = page.getByRole('link', { name: 'Open CUST-0001' });
  await tabTo(page, rowLink);
  await expect(rowLink).toBeFocused();
  expect(await rowLink.evaluate(el => el.matches(':focus-visible'))).toBe(true);

  await page.keyboard.press('Enter');
  await expect(page).toHaveURL(/#\/isp\/customers\/cust-1\/show/);
});

test('invoice row opens with the keyboard alone', async ({ page }) => {
  await page.goto('/#/isp/invoices');

  const rowLink = page.getByRole('link', { name: 'Open INV-000001' });
  await tabTo(page, rowLink);
  await page.keyboard.press('Enter');
  await expect(page).toHaveURL(/#\/isp\/invoices\/inv-1\/show/);
});

test('NAS row opens with the keyboard alone', async ({ page }) => {
  await page.goto('/#/network/nas');

  const rowLink = page.getByRole('link', { name: 'Open Core NAS Jakarta' });
  await tabTo(page, rowLink);
  await page.keyboard.press('Enter');
  await expect(page).toHaveURL(/#\/network\/nas\/nas-1\/show/);
});

test('payment dialog traps focus and returns it to the trigger on Escape', async ({ page }) => {
  await page.goto('/#/isp/invoices/inv-1/show');

  const trigger = page.getByRole('button', { name: 'Record Payment' });
  await tabTo(page, trigger);
  expect(await trigger.evaluate(el => el.matches(':focus-visible'))).toBe(true);

  await page.keyboard.press('Enter');
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  expect(await activeElementInside(page, dialog)).toBe(true);

  // Shift+Tab and Tab must stay inside the dialog: no keyboard trap escape.
  for (let i = 0; i < 6; i++) {
    await page.keyboard.press('Tab');
    expect(await activeElementInside(page, dialog)).toBe(true);
  }
  await page.keyboard.press('Shift+Tab');
  expect(await activeElementInside(page, dialog)).toBe(true);

  await page.keyboard.press('Escape');
  await expect(dialog).toBeHidden();
  await expect(trigger).toBeFocused();
});
