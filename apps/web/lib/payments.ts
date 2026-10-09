import { getAccessToken } from './auth';

export type LedgerRow = {
  id: number;
  facility_id: number;
  category: string;
  title: string;
  customer_name?: string;
  customer_phone?: string;
  amount: number;
  paid_amount: number;
  status: string;
  payment_method?: string;
  reference_type?: string;
  reference_id?: string;
  notes?: string;
  paid_at?: string;
  created_at?: string;
};

export type CreatePaymentInput = {
  category?: string;
  title: string;
  customer_name?: string;
  customer_phone?: string;
  amount: number;
  paid_amount?: number;
  status?: string;
  payment_method?: string;
  reference_type?: string;
  reference_id?: string;
  notes?: string;
};

export type PaymentSummary = {
  total_amount: number;
  total_paid: number;
  pending_count: number;
  paid_count: number;
  ledger_count: number;
  pending_amount: number;
};

export type LedgerFilter = {
  category?: string;
  status?: string;
  from?: string;
  to?: string;
};

async function paymentsFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getAccessToken();
  const res = await fetch(`/v1/payments${path}`, {
    ...init,
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(init?.headers || {}),
    },
  });
  const data = (await res.json().catch(() => ({}))) as T & { error?: string };
  if (!res.ok) {
    throw new Error((data as { error?: string }).error || `http_${res.status}`);
  }
  return data;
}

function qs(opts?: LedgerFilter): string {
  const q = new URLSearchParams();
  if (opts?.category && opts.category !== 'ALL') q.set('category', opts.category);
  if (opts?.status && opts.status !== 'all') q.set('status', opts.status);
  if (opts?.from) q.set('from', opts.from);
  if (opts?.to) q.set('to', opts.to);
  const s = q.toString();
  return s ? `?${s}` : '';
}

export function listLedger(facilityId: number, opts?: LedgerFilter) {
  return paymentsFetch<{ ledger: LedgerRow[]; count: number }>(
    `/facilities/${facilityId}/ledger${qs(opts)}`,
  );
}

export function paymentsSummary(facilityId: number, opts?: LedgerFilter) {
  return paymentsFetch<PaymentSummary>(`/facilities/${facilityId}/summary${qs(opts)}`);
}

export function createPayment(facilityId: number, body: CreatePaymentInput) {
  return paymentsFetch<LedgerRow>(`/facilities/${facilityId}/payments`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function markPaymentPaid(
  facilityId: number,
  paymentId: number,
  body?: { paid_amount?: number; payment_method?: string; notes?: string },
) {
  return paymentsFetch<LedgerRow>(`/facilities/${facilityId}/payments/${paymentId}/mark-paid`, {
    method: 'POST',
    body: JSON.stringify(body || {}),
  });
}

export function todayISO(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}
