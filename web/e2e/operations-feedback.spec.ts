import { expect, test, type Page } from '@playwright/test';

/**
 * ENG-055 / ENG-058 browser half: the Operations page separates a failed load
 * from an empty fleet and offers a retry, marks targets whose last sample is
 * older than the polling window, and the NOC impact panel exposes its
 * confidence and staleness labels with the same retry affordance.
 *
 * The backend freshness window is covered by internal/adminapi/noc_impact_test.go;
 * the mocked routes here prove only what the UI renders and requests.
 */

const staleCheckedAt = new Date(Date.now() - 10 * 60 * 1000).toISOString();

const target = {
  id: 't1', name: 'Core Router', kind: 'router', address: '10.0.0.1',
  probe_type: 'icmp', port: 0, interval_seconds: 60, timeout_milliseconds: 1000,
  failure_threshold: 3, enabled: true, last_status: 'up',
  last_latency_milliseconds: 12, last_packet_loss_percent: 0,
  last_checked_at: staleCheckedAt, uptime_percent: 99.9, heartbeats: [],
};

const impact = {
  scope: { type: 'target', id: 't1', name: 'Core Router' },
  state: 'last_known_down_stale',
  confidence: 'inferred',
  confidence_reason: 'nas_activity_inferred',
  monitoring: {
    monitored: true, enabled: true, status: 'up', stale: true,
    stale_after_seconds: 300, last_checked_at: staleCheckedAt,
    interval_seconds: 60, target_id: 't1', target_name: 'Core Router',
  },
  counts: { customers: 2, subscriptions: 2, online_sessions: 1, open_tickets: 0 },
  impacted: [], pagination: { page: 1, page_size: 10, total: 0 },
  warnings: [], generated_at: staleCheckedAt,
};

async function signIn(page: Page) {
  const token = [
    Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64url'),
    Buffer.from(JSON.stringify({
      sub: '1', username: 'e2e-operations', role: 'admin',
      exp: Math.floor(Date.now() / 1000) + 3600,
    })).toString('base64url'),
    'e2e-signature',
  ].join('.');

  await page.addInitScript(value => {
    localStorage.setItem('token', value);
    localStorage.setItem('username', 'e2e-operations');
    localStorage.setItem('user', JSON.stringify({
      id: '1', username: 'e2e-operations', realname: 'E2E Operations', level: 'admin', status: 'enabled',
    }));
    localStorage.setItem('permissions', JSON.stringify([]));
    localStorage.setItem('locale', 'en-US');
  }, token);
}

type Mocks = { failTargets: boolean; failImpact: boolean };

async function mockOperationsApi(page: Page, mocks: Mocks) {
  await page.route('**/api/v1/**', async route => {
    const { pathname } = new URL(route.request().url());
    // 404 disables the query client's automatic retry, so the manual Retry is
    // the only thing that can recover the view inside the test window.
    if (pathname.endsWith('/network/monitor-targets')) {
      if (mocks.failTargets) {
        await route.fulfill({ status: 404, json: { error: { code: 'not_found', message: 'unavailable' } } });
        return;
      }
      await route.fulfill({ json: { data: [target], total: 1 } });
      return;
    }
    if (pathname.endsWith('/network/monitor-incidents')) {
      await route.fulfill({ json: { data: [], total: 0 } });
      return;
    }
    if (pathname.endsWith('/network/impact')) {
      if (mocks.failImpact) {
        await route.fulfill({ status: 404, json: { error: { code: 'not_found', message: 'unavailable' } } });
        return;
      }
      await route.fulfill({ json: { data: impact } });
      return;
    }
    await route.fulfill({ json: { data: [], total: 0 } });
  });
}

// The query client retries a failed load with backoff before surfacing the
// error state, so the error assertions allow more than the 5s default.
const ERROR_SETTLE_TIMEOUT = 20_000;

test('operations marks targets whose last sample is older than the polling window', async ({ page }) => {
  await signIn(page);
  await mockOperationsApi(page, { failTargets: false, failImpact: false });
  await page.goto('/#/operations');

  await expect(page.getByText(/1 monitored target\(s\) have not reported within their expected interval/)).toBeVisible();
  await expect(page.getByText('Stale', { exact: true })).toBeVisible();
});

test('operations separates a failed load from an empty fleet and retries', async ({ page }) => {
  await signIn(page);
  const mocks = { failTargets: true, failImpact: false };
  await mockOperationsApi(page, mocks);
  await page.goto('/#/operations');

  await expect(page.getByText(/Network monitoring data could not be loaded/)).toBeVisible({ timeout: ERROR_SETTLE_TIMEOUT });
  mocks.failTargets = false;
  await page.getByRole('button', { name: 'Retry' }).click();

  await expect(page.getByText('Core Router')).toBeVisible();
  await expect(page.getByText(/Network monitoring data could not be loaded/)).toHaveCount(0);
});

test('noc impact shows confidence and staleness labels and retries a failed mapping', async ({ page }) => {
  await signIn(page);
  const mocks = { failTargets: false, failImpact: true };
  await mockOperationsApi(page, mocks);
  await page.goto('/#/operations');

  await page.getByRole('button', { name: 'Impact' }).click();
  await expect(page.getByText(/Impact data could not be loaded/)).toBeVisible({ timeout: ERROR_SETTLE_TIMEOUT });

  mocks.failImpact = false;
  await page.getByRole('button', { name: 'Retry' }).click();

  await expect(page.getByTestId('noc-impact-confidence')).toHaveText('Inferred mapping');
  await expect(page.getByTestId('noc-impact-stale')).toHaveText('Monitoring stale');
});
