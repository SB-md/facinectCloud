import { getAccessToken } from './auth';

export type CourtSpec = {
  name: string;
  price_per_hour?: number;
  open_time?: string;
  close_time?: string;
};

export type SportSpec = {
  name: string;
  courts: CourtSpec[];
};

export type FacilityRequest = {
  id: number;
  requester_user_id: number;
  requester_email: string;
  facility_name: string;
  location: string;
  contact_name?: string;
  whatsapp?: string;
  alternate_number?: string;
  google_map_url?: string;
  notes?: string;
  sports?: SportSpec[];
  status: string;
  facility_id?: number | null;
  facility_slug?: string;
  reviewed_at?: string;
  review_note?: string;
  created_at?: string;
};

export type SubmitFacilityInput = {
  facility_name: string;
  location: string;
  contact_name?: string;
  whatsapp?: string;
  alternate_number?: string;
  google_map_url?: string;
  notes?: string;
  sports?: SportSpec[];
  requester_user_id?: number;
  requester_email?: string;
};

async function afFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getAccessToken();
  const res = await fetch(`/v1/add-facility${path}`, {
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

export function submitFacilityRequest(body: SubmitFacilityInput) {
  return afFetch<FacilityRequest>('/requests', {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function listMyFacilityRequests(status?: string) {
  const q = status && status !== 'all' ? `?status=${encodeURIComponent(status)}` : '';
  return afFetch<{ requests: FacilityRequest[]; count: number }>(`/requests/mine${q}`);
}

export function listPendingFacilityRequests() {
  return afFetch<{ requests: FacilityRequest[]; count: number }>('/requests/pending');
}

export function approveFacilityRequest(id: number, note?: string) {
  return afFetch<FacilityRequest>(`/requests/${id}/approve`, {
    method: 'POST',
    body: JSON.stringify({ note: note || '' }),
  });
}

export function rejectFacilityRequest(id: number, note?: string) {
  return afFetch<FacilityRequest>(`/requests/${id}/reject`, {
    method: 'POST',
    body: JSON.stringify({ note: note || '' }),
  });
}

export function facilityRequestSummary() {
  return afFetch<{ pending: number; approved: number; rejected: number }>('/summary');
}
