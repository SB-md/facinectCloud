import { getAccessToken } from './auth';

export type StudentRow = {
  enrollment_id: number;
  student_id: number;
  facility_id: number;
  first_name: string;
  last_name: string;
  contact_email?: string;
  contact_phone?: string;
  whatsapp?: string;
  sport_id?: number | null;
  plan_name?: string;
  start_date?: string;
  end_date?: string;
  actual_fee?: number | null;
  status: string;
  created_at?: string;
};

export type AttendanceRow = {
  enrollment_id: number;
  student_id: number;
  facility_id: number;
  first_name: string;
  last_name: string;
  contact_phone?: string;
  plan_name?: string;
  attendance_date: string;
  status?: string;
  attendance_id?: number | null;
  notes?: string;
};

export type EnrollInput = {
  first_name: string;
  last_name?: string;
  contact_email?: string;
  contact_phone?: string;
  whatsapp?: string;
  sport_id?: number;
  plan_name?: string;
  start_date?: string;
  end_date?: string;
  actual_fee?: number;
};

async function studentsFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getAccessToken();
  const res = await fetch(`/v1/students${path}`, {
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

export function listStudents(facilityId: number, opts?: { sport_id?: number; status?: string }) {
  const q = new URLSearchParams();
  if (opts?.sport_id) q.set('sport_id', String(opts.sport_id));
  if (opts?.status) q.set('status', opts.status);
  const qs = q.toString();
  return studentsFetch<{ students: StudentRow[] }>(
    `/facilities/${facilityId}/students${qs ? `?${qs}` : ''}`,
  );
}

export function studentsSummary(facilityId: number) {
  return studentsFetch<{ active_count: number }>(`/facilities/${facilityId}/students/summary`);
}

export function enrollStudent(facilityId: number, body: EnrollInput) {
  return studentsFetch<StudentRow>(`/facilities/${facilityId}/students`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function listAttendance(facilityId: number, date: string) {
  const q = new URLSearchParams({ date });
  return studentsFetch<{ attendance: AttendanceRow[]; date: string }>(
    `/facilities/${facilityId}/attendance?${q}`,
  );
}

export function markAttendance(
  facilityId: number,
  body: { enrollment_id: number; date: string; status: string; notes?: string },
) {
  return studentsFetch<AttendanceRow>(`/facilities/${facilityId}/attendance`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function todayISO(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

export function studentDisplayName(row: { first_name: string; last_name?: string }): string {
  return [row.first_name, row.last_name].filter(Boolean).join(' ').trim() || 'Student';
}
