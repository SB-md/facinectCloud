import { getAccessToken } from './auth';

export type BookingCourt = {
  id: number;
  facility_id: number;
  name: string;
  sport_id?: number | null;
  status: string;
};

export type BookingSlot = {
  id: number;
  facility_id: number;
  court_id: number;
  court_name?: string;
  slot_date: string;
  start_time: string;
  end_time: string;
  status: 'available' | 'booked' | 'blocked' | string;
  notes?: string;
};

export type BookingRow = {
  id: number;
  facility_id: number;
  slot_id: number;
  user_id?: number | null;
  customer_name?: string;
  customer_phone?: string;
  customer_email?: string;
  status: 'confirmed' | 'cancelled' | string;
  notes?: string;
  slot_date?: string;
  start_time?: string;
  end_time?: string;
  court_name?: string;
  created_at?: string;
};

async function bookingFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getAccessToken();
  const res = await fetch(`/v1/booking${path}`, {
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

export function listCourts(facilityId: number) {
  return bookingFetch<{ courts: BookingCourt[] }>(`/facilities/${facilityId}/courts`);
}

export function createCourt(facilityId: number, name: string, sportId?: number) {
  return bookingFetch<BookingCourt>(`/facilities/${facilityId}/courts`, {
    method: 'POST',
    body: JSON.stringify({ name, sport_id: sportId }),
  });
}

export function listSlots(facilityId: number, date: string, courtId?: number) {
  const q = new URLSearchParams({ date });
  if (courtId) q.set('court_id', String(courtId));
  return bookingFetch<{ slots: BookingSlot[] }>(`/facilities/${facilityId}/slots?${q}`);
}

export function generateSlots(
  facilityId: number,
  body: {
    date: string;
    court_id?: number;
    open_hour?: number;
    close_hour?: number;
    slot_minutes?: number;
  },
) {
  return bookingFetch<{ created: number }>(`/facilities/${facilityId}/slots/generate`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function blockSlot(slotId: number, notes?: string) {
  return bookingFetch<BookingSlot>(`/slots/${slotId}/block`, {
    method: 'POST',
    body: JSON.stringify({ notes: notes || '' }),
  });
}

export function unblockSlot(slotId: number) {
  return bookingFetch<BookingSlot>(`/slots/${slotId}/unblock`, { method: 'POST' });
}

export function createBooking(
  facilityId: number,
  body: {
    slot_id: number;
    customer_name?: string;
    customer_phone?: string;
    customer_email?: string;
    notes?: string;
    user_id?: number;
  },
) {
  return bookingFetch<BookingRow>(`/facilities/${facilityId}/bookings`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function listBookings(
  facilityId: number,
  opts?: { date?: string; status?: string; year?: number; month?: number },
) {
  const q = new URLSearchParams();
  if (opts?.date) q.set('date', opts.date);
  if (opts?.status) q.set('status', opts.status);
  if (opts?.year) q.set('year', String(opts.year));
  if (opts?.month) q.set('month', String(opts.month));
  const qs = q.toString();
  return bookingFetch<{ bookings: BookingRow[] }>(
    `/facilities/${facilityId}/bookings${qs ? `?${qs}` : ''}`,
  );
}

export function cancelBooking(bookingId: number) {
  return bookingFetch<BookingRow>(`/bookings/${bookingId}/cancel`, { method: 'POST' });
}

export function groupSlotsByCourt(slots: BookingSlot[]): { courtName: string; slots: BookingSlot[] }[] {
  const map = new Map<string, BookingSlot[]>();
  for (const s of slots) {
    const key = s.court_name || `Court ${s.court_id}`;
    const list = map.get(key) || [];
    list.push(s);
    map.set(key, list);
  }
  return Array.from(map.entries()).map(([courtName, list]) => ({
    courtName,
    slots: list.sort((a, b) => a.start_time.localeCompare(b.start_time)),
  }));
}

export function todayISO(): string {
  const d = new Date();
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

export function formatTimeRange(start?: string, end?: string): string {
  const s = (start || '').slice(0, 5);
  const e = (end || '').slice(0, 5);
  if (!s) return '—';
  return e ? `${s} – ${e}` : s;
}

/** YYYY-MM for the current calendar month (local). */
export function monthKeyNow(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
}

/** Shift a YYYY-MM key by ±months. */
export function shiftMonth(monthKey: string, delta: number): string {
  const [y, m] = monthKey.split('-').map(Number);
  const d = new Date(y, m - 1 + delta, 1);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
}

/**
 * Facinect-style month summary: total count + 4 weekly buckets (days 1–7, 8–14, 15–21, 22–end).
 * Buckets by slot_date when present, else created_at.
 */
export function summarizeMonthBookings(rows: BookingRow[]): { count: number; weekly: number[] } {
  const weekly = [0, 0, 0, 0];
  for (const b of rows) {
    const raw = (b.slot_date || b.created_at || '').slice(0, 10);
    if (!raw) continue;
    const day = Number(raw.slice(8, 10));
    if (!Number.isFinite(day) || day < 1) continue;
    const idx = Math.min(3, Math.floor((day - 1) / 7));
    weekly[idx] += 1;
  }
  return { count: rows.length, weekly };
}
