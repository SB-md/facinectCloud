'use client';

import { FormEvent, useCallback, useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import FacilityShell from '../FacilityShell';
import { FacilityMembership, fetchSession, findFacilityBySlug, getCachedUser } from '../../../../lib/auth';
import {
  AttendanceRow,
  EnrollInput,
  StudentRow,
  enrollStudent,
  listAttendance,
  listStudents,
  markAttendance,
  studentDisplayName,
  todayISO,
} from '../../../../lib/students';
import styles from '../portal.module.css';

export default function StudentsPage() {
  return (
    <FacilityShell
      pageKey="students"
      title="Students"
      description="Coaching enrollments and daily attendance."
    >
      <StudentsBody />
    </FacilityShell>
  );
}

function StudentsBody() {
  const params = useParams<{ slug: string }>();
  const [facility, setFacility] = useState<FacilityMembership | null>(null);
  const [tab, setTab] = useState<'list' | 'attendance' | 'enroll'>('list');
  const [students, setStudents] = useState<StudentRow[]>([]);
  const [attendance, setAttendance] = useState<AttendanceRow[]>([]);
  const [date, setDate] = useState(todayISO());
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');
  const [msg, setMsg] = useState('');
  const [loaded, setLoaded] = useState(false);

  const [form, setForm] = useState<EnrollInput>({
    first_name: '',
    last_name: '',
    contact_phone: '',
    contact_email: '',
    plan_name: '',
    start_date: todayISO(),
  });

  const facilityId = facility?.facilityId;

  const loadList = useCallback(async (fid: number) => {
    const res = await listStudents(fid, { status: 'active' });
    setStudents(res.students || []);
  }, []);

  const loadAttendance = useCallback(async (fid: number, d: string) => {
    const res = await listAttendance(fid, d);
    setAttendance(res.attendance || []);
  }, []);

  useEffect(() => {
    (async () => {
      setErr('');
      let user = getCachedUser();
      if (!user) {
        const session = await fetchSession();
        user = session?.user || null;
      }
      const f = findFacilityBySlug(user, params.slug);
      setFacility(f);
      if (f?.facilityId) {
        try {
          await loadList(f.facilityId);
        } catch (e) {
          setErr(e instanceof Error ? e.message : 'load_failed');
        }
      }
      setLoaded(true);
    })();
  }, [params.slug, loadList]);

  async function onRefreshList() {
    if (!facilityId) return;
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      await loadList(facilityId);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'load_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onLoadAttendance() {
    if (!facilityId) return;
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      await loadAttendance(facilityId, date);
      setTab('attendance');
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'load_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onMark(enrollmentId: number, status: string) {
    if (!facilityId) return;
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      await markAttendance(facilityId, { enrollment_id: enrollmentId, date, status });
      await loadAttendance(facilityId, date);
      setMsg(`Marked ${status}.`);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'mark_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onEnroll(e: FormEvent) {
    e.preventDefault();
    if (!facilityId) return;
    if (!form.first_name.trim()) {
      setErr('first_name_required');
      return;
    }
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      await enrollStudent(facilityId, {
        ...form,
        first_name: form.first_name.trim(),
        last_name: form.last_name?.trim(),
        contact_phone: form.contact_phone?.trim(),
        contact_email: form.contact_email?.trim(),
        plan_name: form.plan_name?.trim(),
      });
      setMsg('Student enrolled.');
      setForm({
        first_name: '',
        last_name: '',
        contact_phone: '',
        contact_email: '',
        plan_name: '',
        start_date: todayISO(),
      });
      await loadList(facilityId);
      setTab('list');
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : 'enroll_failed');
    } finally {
      setBusy(false);
    }
  }

  if (!loaded) {
    return <p>Loading students…</p>;
  }

  if (!facilityId) {
    return <p className={styles.settingsErr}>Facility not found for this account.</p>;
  }

  return (
    <>
      <div className={styles.detailGrid}>
        <div className={styles.detailCard}>
          <span>Active enrollments</span>
          <strong>{students.length}</strong>
        </div>
        <div className={styles.detailCard}>
          <span>Attendance date</span>
          <strong>{date}</strong>
        </div>
      </div>

      <div className={styles.tabRow}>
        <button
          type="button"
          className={`${styles.tabBtn} ${tab === 'list' ? styles.tabBtnActive : ''}`}
          onClick={() => setTab('list')}
        >
          List
        </button>
        <button
          type="button"
          className={`${styles.tabBtn} ${tab === 'attendance' ? styles.tabBtnActive : ''}`}
          onClick={() => onLoadAttendance()}
        >
          Attendance
        </button>
        <button
          type="button"
          className={`${styles.tabBtn} ${tab === 'enroll' ? styles.tabBtnActive : ''}`}
          onClick={() => setTab('enroll')}
        >
          Enroll
        </button>
      </div>

      {err && <p className={styles.settingsErr}>{err}</p>}
      {msg && <p className={styles.settingsOk}>{msg}</p>}

      {tab === 'list' && (
        <section className={styles.panel}>
          <div className={styles.actionRow} style={{ marginBottom: 12 }}>
            <button type="button" className={styles.settingsPrimary} disabled={busy} onClick={onRefreshList}>
              {busy ? 'Loading…' : 'Refresh'}
            </button>
          </div>
          {students.length === 0 ? (
            <p>No active students yet. Use Enroll to add one.</p>
          ) : (
            <div className={styles.tableWrap}>
              <table className={styles.dataTable}>
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Phone</th>
                    <th>Plan</th>
                    <th>Start</th>
                    <th>End</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {students.map((s) => (
                    <tr key={s.enrollment_id}>
                      <td>{studentDisplayName(s)}</td>
                      <td>{s.contact_phone || '—'}</td>
                      <td>{s.plan_name || '—'}</td>
                      <td>{s.start_date || '—'}</td>
                      <td>{s.end_date || '—'}</td>
                      <td>
                        <span className={`${styles.statusPill} ${styles.statusOk}`}>{s.status}</span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      )}

      {tab === 'attendance' && (
        <section className={styles.panel}>
          <div className={styles.inlineForm} style={{ marginBottom: 14 }}>
            <label className={styles.fieldGrow}>
              Date
              <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </label>
            <button type="button" className={styles.settingsPrimary} disabled={busy} onClick={onLoadAttendance}>
              {busy ? 'Loading…' : 'Load'}
            </button>
          </div>
          {attendance.length === 0 ? (
            <p>No active enrollments for attendance.</p>
          ) : (
            <div className={styles.tableWrap}>
              <table className={styles.dataTable}>
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Plan</th>
                    <th>Status</th>
                    <th>Mark</th>
                  </tr>
                </thead>
                <tbody>
                  {attendance.map((a) => (
                    <tr key={a.enrollment_id}>
                      <td>{studentDisplayName(a)}</td>
                      <td>{a.plan_name || '—'}</td>
                      <td>
                        {a.status ? (
                          <span
                            className={`${styles.statusPill} ${
                              a.status === 'present'
                                ? styles.statusOk
                                : a.status === 'leave'
                                  ? styles.statusWarn
                                  : styles.statusBad
                            }`}
                          >
                            {a.status}
                          </span>
                        ) : (
                          <span className={styles.statusMuted}>unmarked</span>
                        )}
                      </td>
                      <td>
                        <div className={styles.actionRow}>
                          <button
                            type="button"
                            className={styles.settingsPrimary}
                            disabled={busy}
                            onClick={() => onMark(a.enrollment_id, 'present')}
                          >
                            Present
                          </button>
                          <button
                            type="button"
                            className={styles.settingsDanger}
                            disabled={busy}
                            onClick={() => onMark(a.enrollment_id, 'absent')}
                          >
                            Absent
                          </button>
                          <button
                            type="button"
                            className={styles.tabBtn}
                            disabled={busy}
                            onClick={() => onMark(a.enrollment_id, 'leave')}
                          >
                            Leave
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      )}

      {tab === 'enroll' && (
        <section className={styles.panel}>
          <h2>Enroll student</h2>
          <form className={styles.settingsForm} onSubmit={onEnroll} style={{ marginTop: 12 }}>
            <label>
              First name
              <input
                value={form.first_name}
                onChange={(e) => setForm((f) => ({ ...f, first_name: e.target.value }))}
                required
              />
            </label>
            <label>
              Last name
              <input
                value={form.last_name || ''}
                onChange={(e) => setForm((f) => ({ ...f, last_name: e.target.value }))}
              />
            </label>
            <label>
              Phone
              <input
                value={form.contact_phone || ''}
                onChange={(e) => setForm((f) => ({ ...f, contact_phone: e.target.value }))}
              />
            </label>
            <label>
              Email
              <input
                type="email"
                value={form.contact_email || ''}
                onChange={(e) => setForm((f) => ({ ...f, contact_email: e.target.value }))}
              />
            </label>
            <label>
              Plan
              <input
                value={form.plan_name || ''}
                onChange={(e) => setForm((f) => ({ ...f, plan_name: e.target.value }))}
                placeholder="e.g. Monthly coaching"
              />
            </label>
            <label>
              Start date
              <input
                type="date"
                value={form.start_date || ''}
                onChange={(e) => setForm((f) => ({ ...f, start_date: e.target.value }))}
              />
            </label>
            <label>
              End date
              <input
                type="date"
                value={form.end_date || ''}
                onChange={(e) => setForm((f) => ({ ...f, end_date: e.target.value }))}
              />
            </label>
            <button type="submit" className={styles.settingsPrimary} disabled={busy}>
              {busy ? 'Saving…' : 'Enroll'}
            </button>
          </form>
        </section>
      )}
    </>
  );
}
