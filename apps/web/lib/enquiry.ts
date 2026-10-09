import { getAccessToken } from './auth';

export type EnquiryRow = {
  id: number;
  facility_id: number;
  customer_phone: string;
  customer_name?: string;
  enquiry_details: string;
  status: string;
  ai_data?: Record<string, unknown>;
  created_at?: string;
  updated_at?: string;
};

export type CreateEnquiryInput = {
  customer_phone: string;
  customer_name?: string;
  enquiry_details: string;
  run_ai?: boolean;
};

async function enquiryFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getAccessToken();
  const res = await fetch(`/v1/enquiry${path}`, {
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

export function listEnquiries(
  facilityId: number,
  opts?: { status?: string; search?: string; limit?: number },
) {
  const q = new URLSearchParams();
  if (opts?.status) q.set('status', opts.status);
  if (opts?.search) q.set('search', opts.search);
  if (opts?.limit) q.set('limit', String(opts.limit));
  const qs = q.toString();
  return enquiryFetch<{ enquiries: EnquiryRow[]; total: number }>(
    `/facilities/${facilityId}/enquiries${qs ? `?${qs}` : ''}`,
  );
}

export function createEnquiry(facilityId: number, body: CreateEnquiryInput) {
  return enquiryFetch<EnquiryRow>(`/facilities/${facilityId}/enquiries`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function updateEnquiryStatus(enquiryId: number, status: string) {
  return enquiryFetch<EnquiryRow>(`/enquiries/${enquiryId}/status`, {
    method: 'POST',
    body: JSON.stringify({ status }),
  });
}

export function reanalyzeEnquiry(enquiryId: number) {
  return enquiryFetch<EnquiryRow>(`/enquiries/${enquiryId}/analyze`, { method: 'POST' });
}

export function suggestReply(enquiryId: number, facilityName?: string) {
  return enquiryFetch<{ reply: string }>(`/enquiries/${enquiryId}/suggest-reply`, {
    method: 'POST',
    body: JSON.stringify({ facility_name: facilityName || '' }),
  });
}

export function notifyEnquiry(enquiryId: number, message: string) {
  return enquiryFetch<{ ok: boolean; enquiry?: EnquiryRow }>(`/enquiries/${enquiryId}/notify`, {
    method: 'POST',
    body: JSON.stringify({ message }),
  });
}
