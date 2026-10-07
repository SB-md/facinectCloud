import { getAccessToken } from './auth';

export type Tournament = {
  id: number;
  facility_id: number;
  sport_id?: number | null;
  name: string;
  start_date?: string;
  end_date?: string;
  venue_address?: string;
  location_url?: string;
  contact_numbers?: string;
  rules?: string;
  slug?: string;
  status: string;
  created_at?: string;
  updated_at?: string;
};

export type TournamentInput = {
  name: string;
  sport_id?: number;
  start_date?: string;
  end_date?: string;
  venue_address?: string;
  location_url?: string;
  contact_numbers?: string;
  rules?: string;
  slug?: string;
  status?: string;
};

async function tournamentsFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getAccessToken();
  const res = await fetch(`/v1/tournaments${path}`, {
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

export function listTournaments(facilityId: number, opts?: { status?: string }) {
  const q = new URLSearchParams();
  if (opts?.status) q.set('status', opts.status);
  const qs = q.toString();
  return tournamentsFetch<{ tournaments: Tournament[] }>(
    `/facilities/${facilityId}/tournaments${qs ? `?${qs}` : ''}`,
  );
}

export function tournamentsSummary(facilityId: number) {
  return tournamentsFetch<{ active_count: number }>(
    `/facilities/${facilityId}/tournaments/summary`,
  );
}

export function createTournament(facilityId: number, body: TournamentInput) {
  return tournamentsFetch<Tournament>(`/facilities/${facilityId}/tournaments`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function getTournament(tournamentId: number) {
  return tournamentsFetch<Tournament>(`/tournaments/${tournamentId}`);
}

export function updateTournament(tournamentId: number, body: TournamentInput) {
  return tournamentsFetch<Tournament>(`/tournaments/${tournamentId}`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function updateTournamentStatus(tournamentId: number, status: string) {
  return tournamentsFetch<Tournament>(`/tournaments/${tournamentId}/status`, {
    method: 'POST',
    body: JSON.stringify({ status }),
  });
}

export function todayISO(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}
