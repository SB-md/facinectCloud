import { getAccessToken } from './auth';

export type OfferRow = {
  id: number;
  facility_id: number;
  code: string;
  title: string;
  description?: string;
  offer_kind: 'promotion' | 'discount' | string;
  discount_type: 'flat' | 'percent' | string;
  discount_value: number;
  valid_from?: string;
  valid_to?: string;
  usage_limit?: number | null;
  used_count: number;
  sport_id?: number | null;
  status: string;
  is_expired?: boolean;
  created_at?: string;
};

export type CreateOfferInput = {
  code: string;
  title: string;
  description?: string;
  offer_kind?: 'promotion' | 'discount';
  discount_type?: 'flat' | 'percent';
  discount_value: number;
  valid_from?: string;
  valid_to?: string;
  duration_days?: number;
  usage_limit?: number;
  sport_id?: number;
};

async function offersFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getAccessToken();
  const res = await fetch(`/v1/offers${path}`, {
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

export function listOffers(
  facilityId: number,
  opts?: { kind?: string; status?: string },
) {
  const q = new URLSearchParams();
  if (opts?.kind && opts.kind !== 'all') q.set('kind', opts.kind);
  if (opts?.status && opts.status !== 'all') q.set('status', opts.status);
  const qs = q.toString();
  return offersFetch<{ offers: OfferRow[]; count: number }>(
    `/facilities/${facilityId}/offers${qs ? `?${qs}` : ''}`,
  );
}

export function offersSummary(facilityId: number) {
  return offersFetch<{ active_count: number }>(`/facilities/${facilityId}/offers/summary`);
}

export function createOffer(facilityId: number, body: CreateOfferInput) {
  return offersFetch<OfferRow>(`/facilities/${facilityId}/offers`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function updateOfferStatus(facilityId: number, offerId: number, status: string) {
  return offersFetch<OfferRow>(`/facilities/${facilityId}/offers/${offerId}/status`, {
    method: 'POST',
    body: JSON.stringify({ status }),
  });
}

export function validateOffer(facilityId: number, code: string, sportId?: number) {
  const q = new URLSearchParams({ code });
  if (sportId) q.set('sport_id', String(sportId));
  return offersFetch<{
    valid: boolean;
    reason?: string;
    offer?: OfferRow;
    discount_type?: string;
    discount_value?: number;
  }>(`/facilities/${facilityId}/offers/validate?${q.toString()}`);
}

export function todayISO(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}
