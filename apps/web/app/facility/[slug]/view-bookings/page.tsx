'use client';

import { Suspense, useCallback, useEffect, useState } from 'react';
import { useParams, useSearchParams } from 'next/navigation';
import FacilityShell from '../FacilityShell';
import { FacilityMembership, fetchSession, findFacilityBySlug, getCachedUser } from '../../../../lib/auth';
import {
  BookingRow,
  cancelBooking,
  formatTimeRange,
  listBookings,
  monthKeyNow,
  todayISO,
} from '../../../../lib/booking';
import styles from '../portal.module.css';

export default function ViewBookingsPage() {
  return (
    <FacilityShell
      pageKey="view_bookings"
      title="View Bookings"
      description="Confirmed and cancelled booking lists for this facility."
    >
      <Suspense fallback={<p>Loading bookings…</p>}>
        <ViewBookingsBody />
      </Suspense>
    </FacilityShell>
  );
}

function ViewBookingsBody() {
  const params = useParams<{ slug: string }>();
  const search = useSearchParams();
  const initialMonth = search.get('month');
  const initialStatus = search.get('status');

  const [facility, setFacility] = useState<FacilityMembership | null>(null);
  const [mode, setMode] = useState<'date' | 'month'>(initialMonth ? 'month' : 'date');
  const [date, setDate] = useState(todayISO());
  const [month, setMonth] = useState(initialMonth || monthKeyNow());
  const [status, setStatus] = useState<'all' | 'confirmed' | 'cancelled'>(() => {
    if (initialStatus === 'cancelled' || initialStatus === 'confirmed' || initialStatus === 'all') {
      return initialStatus;
    }
    return 'confirmed';
  });
  const [rows, setRows] = useState<BookingRow[]>([]);
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState('');
  const [err, setErr] = useState('');
  const [loaded, setLoaded] = useState(false);

  const facilityId = facility?.facilityId;

  const load = useCallback(async (fid: number) => {
    const opts: { date?: string; status?: string; year?: number; month?: number } = {};
    if (status !== 'all') opts.status = status;
    if (mode === 'date') {
      opts.date = date;
    } else {
      const [y, m] = month.split('-').map(Number);
      opts.year = y;
      opts.month = m;
    }
    const res = await listBookings(fid, opts);
    setRows(res.bookings || []);
  }, [date, month, mode, status]);

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
          await load(f.facilityId);
        } catch (e) {
          setErr(e instanceof Error ? e.message : 'load_failed');
        }
      }
      setLoaded(true);
    })();
  }, [params.slug, load]);

  async function onRefresh() {
    if (!facilityId) return;
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      await load(facilityId);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'load_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onCancel(id: number) {
    if (!facilityId) return;
    if (!window.confirm('Cancel this booking and free the slot?')) return;
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      await cancelBooking(id);
      setMsg(`Booking #${id} cancelled.`);
      await load(facilityId);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'cancel_failed');
    } finally {
      setBusy(false);
    }
  }

  if (!loaded) {
    return <p>Loading bookings…</p>;
  }

  if (!facilityId) {
    return <p className={styles.settingsErr}>Facility not found for this account.</p>;
  }

  const confirmed = rows.filter((r) => r.status === 'confirmed').length;
  const cancelled = rows.filter((r) => r.status === 'cancelled').length;

  return (
    <>
      <div className={styles.detailGrid}>
        <div className={styles.detailCard}>
          <span>Rows</span>
          <strong>{rows.length}</strong>
        </div>
        <div className={styles.detailCard}>
          <span>Confirmed</span>
          <strong>{confirmed}</strong>
        </div>
        <div className={styles.detailCard}>
          <span>Cancelled</span>
          <strong>{cancelled}</strong>
        </div>
      </div>

      <section className={styles.panel} style={{ marginBottom: 16 }}>
        <h2>Filters</h2>
        <div className={styles.inlineForm} style={{ marginTop: 12 }}>
          <div className={styles.tabRow} style={{ marginBottom: 0 }}>
            <button
              type="button"
              className={`${styles.tabBtn} ${mode === 'date' ? styles.tabBtnActive : ''}`}
              onClick={() => setMode('date')}
            >
              By date
            </button>
            <button
              type="button"
              className={`${styles.tabBtn} ${mode === 'month' ? styles.tabBtnActive : ''}`}
              onClick={() => setMode('month')}
            >
              By month
            </button>
          </div>
          {mode === 'date' ? (
            <label className={styles.fieldGrow}>
              Date
              <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </label>
          ) : (
            <label className={styles.fieldGrow}>
              Month
              <input type="month" value={month} onChange={(e) => setMonth(e.target.value)} />
            </label>
          )}
          <label>
            Status
            <select value={status} onChange={(e) => setStatus(e.target.value as typeof status)}>
              <option value="all">All</option>
              <option value="confirmed">Confirmed</option>
              <option value="cancelled">Cancelled</option>
            </select>
          </label>
          <button type="button" className={styles.settingsPrimary} disabled={busy} onClick={onRefresh}>
            {busy ? 'Loading…' : 'Apply'}
          </button>
        </div>
      </section>

      <section className={styles.panel}>
        <h2>Ledger</h2>
        {rows.length === 0 ? (
          <p style={{ marginTop: 12 }}>No bookings for these filters.</p>
        ) : (
          <div className={styles.tableWrap}>
            <table className={styles.dataTable}>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>Date</th>
                  <th>Time</th>
                  <th>Court</th>
                  <th>Customer</th>
                  <th>Phone</th>
                  <th>Status</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {rows.map((b) => (
                  <tr key={b.id}>
                    <td>#{b.id}</td>
                    <td>{b.slot_date || '—'}</td>
                    <td>{formatTimeRange(b.start_time, b.end_time)}</td>
                    <td>{b.court_name || '—'}</td>
                    <td>{b.customer_name || '—'}</td>
                    <td>{b.customer_phone || '—'}</td>
                    <td>
                      <span
                        className={`${styles.statusPill} ${
                          b.status === 'confirmed' ? styles.statusOk : styles.statusBad
                        }`}
                      >
                        {b.status}
                      </span>
                    </td>
                    <td>
                      {b.status === 'confirmed' ? (
                        <button
                          type="button"
                          className={styles.settingsDanger}
                          disabled={busy}
                          onClick={() => onCancel(b.id)}
                        >
                          Cancel
                        </button>
                      ) : null}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        {err && <p className={styles.settingsErr}>{err}</p>}
        {msg && <p className={styles.settingsOk}>{msg}</p>}
      </section>
    </>
  );
}
