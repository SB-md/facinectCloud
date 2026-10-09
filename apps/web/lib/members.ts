import { getAccessToken } from './auth';

export type MemberRow = {
  membership_id: number;
  member_id: number;
  facility_id: number;
  full_name: string;
  contact_email?: string;
  contact_phone?: string;
  whatsapp?: string;
  sport_id?: number | null;
  plan_name?: string;
  team_name?: string;
  start_date?: string;
  end_date?: string;
  subscription_fee?: number | null;
  primary_member?: boolean;
  status: string;
  created_at?: string;
};

export type RegisterMemberInput = {
  full_name: string;
  contact_email?: string;
  contact_phone?: string;
  whatsapp?: string;
  sport_id?: number;
  plan_name?: string;
  team_name?: string;
  start_date?: string;
  end_date?: string;
  subscription_fee?: number;
  primary_member?: boolean;
};

async function membersFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getAccessToken();
  const res = await fetch(`/v1/members${path}`, {
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

export function listMembers(facilityId: number, opts?: { sport_id?: number; status?: string }) {
  const q = new URLSearchParams();
  if (opts?.sport_id) q.set('sport_id', String(opts.sport_id));
  if (opts?.status) q.set('status', opts.status);
  const qs = q.toString();
  return membersFetch<{ members: MemberRow[] }>(
    `/facilities/${facilityId}/members${qs ? `?${qs}` : ''}`,
  );
}

export function membersSummary(facilityId: number) {
  return membersFetch<{ active_count: number }>(`/facilities/${facilityId}/members/summary`);
}

export function registerMember(facilityId: number, body: RegisterMemberInput) {
  return membersFetch<MemberRow>(`/facilities/${facilityId}/members`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function updateMembershipStatus(facilityId: number, membershipId: number, status: string) {
  return membersFetch<MemberRow>(
    `/facilities/${facilityId}/memberships/${membershipId}/status`,
    {
      method: 'POST',
      body: JSON.stringify({ status }),
    },
  );
}

export function todayISO(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}
