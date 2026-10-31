import { expect, test, type Page } from '@playwright/test';

/**
 * Browser half of M20: the Invoice Studio edits presentation only, the invoice
 * detail surfaces the issuer snapshot captured at issue time, and both the
 * invoice and the payment receipt download as PDFs through authenticated calls.
 *
 * The server half is TestInvoiceAndReceiptPDFUseIssuedSnapshot and
 * TestInvoiceDocumentSnapshotFreezesIssuerIdentity in
 * internal/adminapi/invoice_document_test.go, plus the PostgreSQL acceptance
 * tests; mocked routes here prove only what the UI reads, sends and requests.
 */

type Recorded = { method: string; pathname: string; body: unknown };

const template = {
  id: '1', version: 4,
  issuer_name: 'PT Alpha Network', logo_url: '', header_text: 'Invoice resmi',
  footer_text: 'Terima kasih', accent_color: '#16A34A', paper_size: 'A4',
  show_tax_id: true, bank_name: 'Bank Mandiri', bank_account_name: 'PT Alpha Network',
  bank_account_number: '1230009876543', payment_note: 'Cantumkan nomor invoice',
  qr_image_url: '',
};

async function signIn(page: Page) {
  const token = [
    Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64url'),
    Buffer.from(JSON.stringify({
      sub: '1', username: 'e2e-invoice', role: 'admin',
      exp: Math.floor(Date.now() / 1000) + 3600,
    })).toString('base64url'),
    'e2e-signature',
  ].join('.');

  await page.addInitScript(token => {
    localStorage.setItem('token', token);
    localStorage.setItem('username', 'e2e-invoice');
    localStorage.setItem('user', JSON.stringify({
      id: '1', username: 'e2e-invoice', realname: 'E2E Invoice', level: 'admin', status: 'enabled',
    }));
    localStorage.setItem('permissions', JSON.stringify([]));
    localStorage.setItem('locale', 'en-US');
  }, token);
}

/**
 * Serves the document template, one invoice and one payment, and records every
 * write so the test can assert the exact body the studio submits.
 */
async function mockBillingApi(page: Page, recorded: Recorded[]) {
  let current = { ...template };

  await page.route('**/api/v1/**', async route => {
    const request = route.request();
    const { pathname } = new URL(request.url());
    const method = request.method();
    const body = request.postDataJSON?.() ?? null;
    if (method !== 'GET') {
      recorded.push({ method, pathname, body });
    }

    if (pathname.endsWith('/tenant-document-template/reset')) {
      current = { ...template, version: 1 };
      await route.fulfill({ json: { data: current } });
      return;
    }
    if (pathname.endsWith('/tenant-document-template')) {
      if (method === 'PUT') {
        current = { ...current, ...(body as Record<string, unknown>), version: current.version + 1 };
      }
      await route.fulfill({ json: { data: current } });
      return;
    }

    if (pathname.endsWith('/isp/invoices/1/document.pdf')) {
      await route.fulfill({
        status: 200,
        headers: {
          'content-type': 'application/pdf',
          'content-disposition': 'attachment; filename="invoice-INV-2026-001.pdf"',
        },
        body: Buffer.from('%PDF-1.4 mocked invoice'),
      });
      return;
    }
    if (pathname.endsWith('/isp/invoices/1/document')) {
      await route.fulfill({
        json: {
          data: {
            kind: 'invoice', number: 'INV-2026-001', status: 'issued', currency: 'IDR',
            snapshot: true, template_version: 3, captured_at: '2026-10-01T08:00:00Z',
            issuer: { name: 'PT Alpha Network', tax_id: '01.234.567.8-901.000', show_tax_id: true },
            payment_instructions: {
              bank_name: 'Bank Mandiri', bank_account_name: 'PT Alpha Network',
              bank_account_number: '1230009876543', note: 'Cantumkan nomor invoice',
            },
            total: 250000, paid_amount: 0, balance: 250000, items: [], payments: [],
          },
        },
      });
      return;
    }
    if (pathname.endsWith('/isp/invoices/1')) {
      await route.fulfill({
        json: {
          data: {
            id: '1', invoice_no: 'INV-2026-001', customer_id: '7', customer_no: 'CUST-7',
            customer_name: 'Siti Aminah', subscription_id: '3', subscription_no: 'SUB-3',
            invoice_date: '2026-10-01T00:00:00Z', due_date: '2026-10-20T00:00:00Z',
            period_start: '2026-10-01T00:00:00Z', period_end: '2026-10-31T23:59:59Z',
            status: 'issued', total: 250000, paid_amount: 0, balance: 250000,
            items: [], payments: [],
          },
        },
      });
      return;
    }
    if (pathname.endsWith('/isp/payments/5/receipt.pdf')) {
      await route.fulfill({
        status: 200,
        headers: {
          'content-type': 'application/pdf',
          'content-disposition': 'attachment; filename="receipt-PAY-2026-009.pdf"',
        },
        body: Buffer.from('%PDF-1.4 mocked receipt'),
      });
      return;
    }
    if (pathname.endsWith('/isp/payments/5')) {
      await route.fulfill({
        json: {
          data: {
            id: '5', payment_no: 'PAY-2026-009', customer_id: '7', invoice_id: '1',
            amount: 250000, method: 'bank_transfer', reference: 'TRF-9911',
            paid_at: '2026-10-05T09:30:00Z', status: 'received', notes: 'Transfer BCA',
          },
        },
      });
      return;
    }
    if (pathname.endsWith('/isp/customers/7')) {
      await route.fulfill({
        json: { data: { id: '7', customer_no: 'CUST-7', name: 'Siti Aminah', phone: '628111222333' } },
      });
      return;
    }

    await route.fulfill({ json: { data: [], total: 0 } });
  });
}

test('invoice studio edits presentation and resets to tenant defaults', async ({ page }) => {
  const recorded: Recorded[] = [];
  await signIn(page);
  await mockBillingApi(page, recorded);
  await page.goto('/#/tenant/invoice');

  await expect(page.getByText(/Template version 4/)).toBeVisible();
  const issuerField = page.getByLabel('Issuer name');
  await expect(issuerField).toHaveValue('PT Alpha Network');
  await expect(page.getByLabel('Account number')).toHaveValue('1230009876543');

  await issuerField.fill('PT Alpha Fiber');
  await page.getByRole('button', { name: 'Save Invoice Template' }).click();

  const saved = recorded.find(entry => entry.method === 'PUT' && entry.pathname.endsWith('/tenant-document-template'));
  expect(saved, 'saving the template must call PUT /tenant-document-template').toBeTruthy();
  const savedBody = saved?.body as Record<string, unknown>;
  expect(savedBody.issuer_name).toBe('PT Alpha Fiber');
  expect(savedBody, 'the studio must not send financial values').not.toHaveProperty('total');
  await expect(page.getByText(/Template version 5/)).toBeVisible();

  await page.getByRole('button', { name: 'Reset template' }).click();
  await expect.poll(() => recorded.some(entry => entry.method === 'POST' && entry.pathname.endsWith('/tenant-document-template/reset')), {
    message: 'resetting must call POST /tenant-document-template/reset',
  }).toBe(true);
  await expect(page.getByText(/Template version 1/)).toBeVisible();
});

test('invoice detail shows the frozen issuer snapshot and downloads the branded PDF', async ({ page }) => {
  const recorded: Recorded[] = [];
  await signIn(page);
  await mockBillingApi(page, recorded);
  await page.goto('/#/isp/invoices/1/show');

  const issued = page.getByTestId('issued-document');
  await expect(issued).toContainText('INV-2026-001');
  await expect(issued).toContainText('PT Alpha Network');
  await expect(issued).toContainText('template v3');
  await expect(issued).toContainText('frozen at issue');

  const download = page.waitForEvent('download');
  await page.getByRole('button', { name: 'Branded PDF' }).click();
  expect((await download).suggestedFilename()).toBe('invoice-INV-2026-001.pdf');
});

test('payment detail downloads the branded receipt PDF', async ({ page }) => {
  const recorded: Recorded[] = [];
  await signIn(page);
  await mockBillingApi(page, recorded);
  await page.goto('/#/isp/payments/5/show');

  const download = page.waitForEvent('download');
  await page.getByRole('button', { name: 'Receipt PDF' }).click();
  expect((await download).suggestedFilename()).toBe('receipt-PAY-2026-009.pdf');
});
