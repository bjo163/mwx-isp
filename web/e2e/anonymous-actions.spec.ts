import { expect, test, type Page } from '@playwright/test';

/**
 * ENG-056: public portal/payment surfaces must answer an anonymous click with
 * information, not a context-free error, and must not imply customer
 * self-service that does not exist.
 *
 * The matching server half is TestLandingSurfaceEndpointsFailClosedAnonymously
 * in internal/adminapi/public_surface_test.go: every advertised public API
 * still refuses the anonymous caller, so these tests never open an API up.
 * They only assert how the 401 is presented (info + login/WhatsApp path) and
 * that the fallback payment action targets the branded WhatsApp number.
 */

const BRANDING_WHATSAPP = '628111222333';

const anonymousDenial = {
  error: 'UNAUTHORIZED',
  message: 'Unauthorized',
};

/**
 * Mock the branding endpoint and deny every other API call the way the real
 * operator-JWT middleware does. `publicSurface` can be overridden per test to
 * return data instead of a 401.
 */
async function mockAnonymousApi(
  page: Page,
  publicSurface?: (pathname: string) => Record<string, unknown> | null,
) {
  await page.route('**/api/v1/**', async route => {
    const { pathname } = new URL(route.request().url());

    if (pathname.endsWith('/public/branding')) {
      await route.fulfill({ json: { data: { contact_whatsapp: BRANDING_WHATSAPP } } });
      return;
    }

    const allowed = publicSurface?.(pathname);
    if (allowed) {
      await route.fulfill({ json: { data: allowed } });
      return;
    }

    await route.fulfill({ status: 401, json: anonymousDenial });
  });
}

async function captureOpenedWindows(page: Page) {
  await page.addInitScript(() => {
    const opened: string[] = [];
    (window as unknown as { __openedUrls: string[] }).__openedUrls = opened;
    Object.defineProperty(window, 'open', {
      configurable: true,
      value: (url?: string | URL) => {
        opened.push(String(url));
        return null;
      },
    });
  });
}

async function openedUrls(page: Page): Promise<string[]> {
  return page.evaluate(() => (window as unknown as { __openedUrls: string[] }).__openedUrls ?? []);
}

const staffNotice = (page: Page) =>
  page.getByRole('alert').filter({ hasText: 'Perlu login staf operator' });

test.describe('anonymous visitor sees staff-only boundary', () => {
  test('bill lookup answers 401 with an info notice and a login path', async ({ page }) => {
    await captureOpenedWindows(page);
    await mockAnonymousApi(page);
    await page.goto('/#/home');

    await page.getByPlaceholder('Ketik Nomor Pelanggan / No WhatsApp...').fill('CUST-0001');
    await page.getByRole('button', { name: 'Cek Tagihan' }).click();

    const notice = staffNotice(page);
    await expect(notice).toBeVisible();
    await expect(notice).toContainText('memerlukan login staf operator');
    await expect(page.getByText(anonymousDenial.message, { exact: true })).toHaveCount(0);

    await notice.getByRole('button', { name: 'Hubungi WhatsApp' }).click();
    const urls = await openedUrls(page);
    expect(urls).toHaveLength(1);
    expect(urls[0]).toContain(`https://wa.me/${BRANDING_WHATSAPP}?text=`);

    await notice.getByRole('button', { name: 'Login staf' }).click();
    await expect(page).toHaveURL(/#\/login/);
  });

  test('voucher check answers 401 with an info notice instead of a bad-code error', async ({ page }) => {
    await mockAnonymousApi(page);
    await page.goto('/#/home');

    await page.getByRole('tab', { name: 'Cek Sisa Kuota Voucher Hotspot' }).click();
    await page.getByPlaceholder('Contoh: VCR-AB1234 atau KODE...').fill('VCR-AB1234');
    await page.getByRole('button', { name: 'Cek Voucher' }).click();

    const notice = staffNotice(page);
    await expect(notice).toBeVisible();
    await expect(notice).toContainText('memerlukan login staf operator');
    await expect(page.getByText('Kode voucher tidak valid')).toHaveCount(0);
    await expect(page.getByText(anonymousDenial.message, { exact: true })).toHaveCount(0);
  });

  test('registration submit answers 401 with an info notice inside the dialog', async ({ page }) => {
    await mockAnonymousApi(page);
    await page.goto('/#/home');

    await page.getByRole('button', { name: 'Daftar Pasang Baru' }).first().click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    await dialog.getByLabel('Nama Lengkap Pemohon').fill('Budi Santoso');
    await dialog.getByLabel('No. WhatsApp / HP').fill('08123456789');
    await dialog.getByLabel('Alamat Lengkap Lokasi Pemasangan').fill('Jl. Mawar 1, Jakarta');
    await dialog.getByRole('button', { name: 'Kirim Pendaftaran' }).click();

    const notice = staffNotice(page);
    await expect(notice).toBeVisible();
    await expect(notice).toContainText('memerlukan login staf operator');
    await expect(page.getByText('Terjadi kesalahan jaringan saat mengirim pendaftaran.')).toHaveCount(0);
  });
});

test.describe('operator portal lookup', () => {
  test('anonymous lookup shows the staff-only notice, not a raw Unauthorized', async ({ page }) => {
    await mockAnonymousApi(page);
    await page.goto('/#/portal');

    await page.getByPlaceholder('Contoh: MWX-000001 atau 08123456789').fill('MWX-000001');
    await page.getByRole('button', { name: 'Periksa' }).click();

    const notice = staffNotice(page);
    await expect(notice).toBeVisible();
    await expect(page.getByText(anonymousDenial.message, { exact: true })).toHaveCount(0);

    await notice.getByRole('button', { name: 'Login staf' }).click();
    await expect(page).toHaveURL(/#\/login/);
  });

  test('offline payment confirmation addresses the branded WhatsApp number', async ({ page }) => {
    await captureOpenedWindows(page);
    await mockAnonymousApi(page, pathname => {
      if (!pathname.endsWith('/portal/lookup')) return null;
      return {
        customer_no: 'MWX-000001',
        name: 'Siti Aminah',
        status: 'active',
        package_name: 'Home 100 Mbps',
        package_price: 350000,
        subscription_status: 'active',
        outstanding: 350000,
        invoices: [
          {
            id: 'inv-1',
            invoice_no: 'INV-000001',
            due_date: '2026-11-01',
            total: 350000,
            balance: 350000,
            status: 'unpaid',
          },
        ],
      };
    });
    await page.goto('/#/portal');

    await page.getByPlaceholder('Contoh: MWX-000001 atau 08123456789').fill('MWX-000001');
    await page.getByRole('button', { name: 'Periksa' }).click();

    await expect(page.getByRole('heading', { name: 'Siti Aminah' })).toBeVisible();
    await expect(page.getByText('Pembayaran online belum dikonfigurasi.')).toBeVisible();

    await page.getByRole('button', { name: 'Konfirmasi WA' }).click();
    const urls = await openedUrls(page);
    expect(urls).toHaveLength(1);
    expect(urls[0].startsWith(`https://wa.me/${BRANDING_WHATSAPP}?text=`)).toBe(true);
  });
});
