export type FacilityMembership = {
  facilityId: number;
  facilityName: string;
  slug: string;
  role: string;
  page_access: string[];
};

export type SessionUser = {
  id?: number;
  email?: string | null;
  phone?: string | null;
  full_name?: string;
  avatar_url?: string | null;
  status?: string;
  role?: string;
  page_access?: string[];
  current_facility_id?: number | null;
  facilities?: FacilityMembership[];
  facility_ids?: number[];
  redirect?: string;
  has_password?: boolean;
};

export type TokenResponse = {
  access_token?: string;
  refresh_token?: string;
  user?: SessionUser;
  redirect?: string;
  error?: string;
};

export type MeResponse = {
  user?: SessionUser;
  redirect?: string;
  claims?: Record<string, unknown>;
  error?: string;
};

const ACCESS_KEY = 'facinect_access_token';
const REFRESH_KEY = 'facinect_refresh_token';
const USER_KEY = 'facinect_user';

export function storeSession(data: TokenResponse) {
  sessionStorage.setItem(ACCESS_KEY, data.access_token || '');
  sessionStorage.setItem(REFRESH_KEY, data.refresh_token || '');
  sessionStorage.setItem(USER_KEY, JSON.stringify(data.user || {}));
}

export function clearSession() {
  sessionStorage.removeItem(ACCESS_KEY);
  sessionStorage.removeItem(REFRESH_KEY);
  sessionStorage.removeItem(USER_KEY);
}

export function getAccessToken(): string | null {
  if (typeof window === 'undefined') return null;
  return sessionStorage.getItem(ACCESS_KEY);
}

export function getCachedUser(): SessionUser | null {
  if (typeof window === 'undefined') return null;
  const raw = sessionStorage.getItem(USER_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as SessionUser;
  } catch {
    return null;
  }
}

export function postLoginPath(data: TokenResponse | MeResponse): string {
  const fromApi = data.redirect || data.user?.redirect;
  if (fromApi) return fromApi;
  const facilities = data.user?.facilities || [];
  if (facilities.length > 0) {
    const f = facilities[0];
    return facilityPath(f.slug, f.role);
  }
  return '/home';
}

export function facilityPath(slug: string, role?: string): string {
  const base = `/facility/${encodeURIComponent(slug)}`;
  switch ((role || '').toLowerCase()) {
    case 'sub_admin':
      return `${base}/view-bookings`;
    case 'headcoach':
    case 'coach':
      return `${base}/students`;
    case 'tour_admin':
      return `${base}/tournaments`;
    default:
      return base;
  }
}

export function canAccessPage(user: SessionUser | null, pageKey?: string): boolean {
  if (!user) return false;
  if (!pageKey) return true;
  const pages = user.page_access || [];
  if (pages.length === 0) return (user.role || '').toLowerCase() === 'admin';
  return pages.includes(pageKey);
}

export function hasFacility(user: SessionUser | null, facilityId: number): boolean {
  if (!user) return false;
  const ids = user.facility_ids || (user.facilities || []).map((f) => f.facilityId);
  return ids.map(Number).includes(Number(facilityId));
}

export function findFacilityBySlug(user: SessionUser | null, slug: string): FacilityMembership | null {
  if (!user) return null;
  const key = decodeURIComponent(slug).toLowerCase();
  return (user.facilities || []).find((f) => f.slug.toLowerCase() === key) || null;
}

export async function fetchSession(): Promise<MeResponse | null> {
  const token = getAccessToken();
  if (!token) return null;

  let res = await fetch('/v1/auth/me', {
    headers: { Authorization: `Bearer ${token}`, Accept: 'application/json' },
  });

  if (res.status === 401) {
    const refreshed = await tryRefresh();
    if (!refreshed) return null;
    res = await fetch('/v1/auth/me', {
      headers: { Authorization: `Bearer ${getAccessToken()}`, Accept: 'application/json' },
    });
  }

  const data = (await res.json()) as MeResponse;
  if (!res.ok) {
    clearSession();
    return { error: data.error || 'invalid_token' };
  }
  sessionStorage.setItem(USER_KEY, JSON.stringify(data.user || {}));
  return data;
}

async function tryRefresh(): Promise<boolean> {
  const refresh = sessionStorage.getItem(REFRESH_KEY);
  if (!refresh) return false;
  try {
    const res = await fetch('/v1/auth/token/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({ refresh_token: refresh }),
    });
    const data = (await res.json()) as TokenResponse;
    if (!res.ok || !data.access_token) {
      clearSession();
      return false;
    }
    storeSession(data);
    return true;
  } catch {
    clearSession();
    return false;
  }
}

export async function logout(): Promise<void> {
  const refresh = sessionStorage.getItem(REFRESH_KEY) || '';
  try {
    await fetch('/v1/auth/logout', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refresh }),
    });
  } catch {
    /* ignore */
  }
  clearSession();
}
