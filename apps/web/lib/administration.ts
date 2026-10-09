import { getAccessToken } from './auth';

export type FacilityProfile = {
  facility_id: number;
  display_name: string;
  location?: string;
  city?: string;
  address?: string;
  phone?: string;
  whatsapp?: string;
  email?: string;
  logo_url?: string;
  map_url?: string;
  notes?: string;
  open_time?: string;
  close_time?: string;
  slug?: string;
  status?: string;
};

export type SportRow = {
  id: number;
  facility_id: number;
  name: string;
  sort_order: number;
  status: string;
};

export type CourtRow = {
  id: number;
  facility_id: number;
  sport_id?: number | null;
  name: string;
  price_per_hour?: number | null;
  open_time?: string;
  close_time?: string;
  status: string;
};

export type PaymentSetting = {
  id?: number;
  facility_id: number;
  sport_id: number;
  allow_full_payment: boolean;
  allow_advance_payment: boolean;
  allow_spot_payment: boolean;
  advance_amount: number;
};

export type ServiceFlags = {
  facility_id: number;
  staff_enabled: boolean;
  customer_enabled: boolean;
};

export type StaffRow = {
  membership_id: number;
  user_id: number;
  facility_id: number;
  email: string;
  name?: string;
  role: string;
  page_access: string[];
  status: string;
};

async function adminFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getAccessToken();
  const res = await fetch(`/v1/administration${path}`, {
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

export function getProfile(facilityId: number) {
  return adminFetch<FacilityProfile>(`/facilities/${facilityId}/profile`);
}

export function saveProfile(facilityId: number, body: Partial<FacilityProfile>) {
  return adminFetch<FacilityProfile>(`/facilities/${facilityId}/profile`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
}

export function listSports(facilityId: number, status?: string) {
  const q = status && status !== 'all' ? `?status=${encodeURIComponent(status)}` : '';
  return adminFetch<{ sports: SportRow[] }>(`/facilities/${facilityId}/sports${q}`);
}

export function createSport(facilityId: number, body: { name: string; sort_order?: number }) {
  return adminFetch<SportRow>(`/facilities/${facilityId}/sports`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function updateSportStatus(facilityId: number, sportId: number, status: string) {
  return adminFetch<SportRow>(`/facilities/${facilityId}/sports/${sportId}/status`, {
    method: 'POST',
    body: JSON.stringify({ status }),
  });
}

export function listCourts(facilityId: number, status?: string) {
  const q = status && status !== 'all' ? `?status=${encodeURIComponent(status)}` : '';
  return adminFetch<{ courts: CourtRow[] }>(`/facilities/${facilityId}/courts${q}`);
}

export function createCourt(
  facilityId: number,
  body: {
    name: string;
    sport_id?: number;
    price_per_hour?: number;
    open_time?: string;
    close_time?: string;
  },
) {
  return adminFetch<CourtRow>(`/facilities/${facilityId}/courts`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function updateCourtStatus(facilityId: number, courtId: number, status: string) {
  return adminFetch<CourtRow>(`/facilities/${facilityId}/courts/${courtId}/status`, {
    method: 'POST',
    body: JSON.stringify({ status }),
  });
}

export function listPayments(facilityId: number) {
  return adminFetch<{ payment_settings: PaymentSetting[] }>(
    `/facilities/${facilityId}/payment-settings`,
  );
}

export function savePayment(facilityId: number, body: PaymentSetting) {
  return adminFetch<PaymentSetting>(`/facilities/${facilityId}/payment-settings`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
}

export function getServices(facilityId: number) {
  return adminFetch<ServiceFlags>(`/facilities/${facilityId}/services`);
}

export function saveServices(facilityId: number, body: Partial<ServiceFlags>) {
  return adminFetch<ServiceFlags>(`/facilities/${facilityId}/services`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
}

export function listStaff(facilityId: number) {
  return adminFetch<{ staff: StaffRow[] }>(`/facilities/${facilityId}/staff`);
}

export function upsertStaff(
  facilityId: number,
  body: { email: string; name?: string; role: string; page_access?: string[] },
) {
  return adminFetch<StaffRow>(`/facilities/${facilityId}/staff`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function updateStaffStatus(facilityId: number, membershipId: number, status: string) {
  return adminFetch<StaffRow>(`/facilities/${facilityId}/staff/${membershipId}/status`, {
    method: 'POST',
    body: JSON.stringify({ status }),
  });
}

export function adminSummary(facilityId: number) {
  return adminFetch<{
    sports_active: number;
    courts_active: number;
    staff_active: number;
    staff_enabled: boolean;
    customer_enabled: boolean;
  }>(`/facilities/${facilityId}/summary`);
}
