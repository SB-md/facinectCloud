'use client';

import { FormEvent, useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import {
  FacilityRequest,
  approveFacilityRequest,
  listMyFacilityRequests,
  listPendingFacilityRequests,
  rejectFacilityRequest,
  submitFacilityRequest,
} from '../../lib/addFacility';
import { SessionUser, fetchSession, getCachedUser, postLoginPath } from '../../lib/auth';
import styles from '../facility/[slug]/portal.module.css';
import loginStyles from '../login/login.module.css';

export default function AddFacilityPage() {
  const router = useRouter();
  const [user, setUser] = useState<SessionUser | null>(null);
  const [tab, setTab] = useState<'submit' | 'mine' | 'review'>('submit');
  const [mine, setMine] = useState<FacilityRequest[]>([]);
  const [pending, setPending] = useState<FacilityRequest[]>([]);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');
  const [msg, setMsg] = useState('');
  const [loaded, setLoaded] = useState(false);

  const [form, setForm] = useState({
    facility_name: '',
    location: '',
    contact_name: '',
    whatsapp: '',
    google_map_url: '',
    notes: '',
    sport_name: 'Badminton',
    court_name: 'Court 1',
    court_price: '500',
  });

  const reload = useCallback(async () => {
    const [m, p] = await Promise.all([
      listMyFacilityRequests('all'),
      listPendingFacilityRequests().catch(() => ({ requests: [] as FacilityRequest[] })),
    ]);
    setMine(m.requests || []);
    setPending(p.requests || []);
  }, []);

  useEffect(() => {
    (async () => {
      let u = getCachedUser();
      if (!u) {
        const session = await fetchSession();
        u = session?.user || null;
      }
      if (!u) {
        router.replace('/login');
        return;
      }
      setUser(u);
      try {
        await reload();
      } catch (e) {
        setErr(e instanceof Error ? e.message : 'load_failed');
      }
      setLoaded(true);
    })();
  }, [router, reload]);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      const price = form.court_price.trim() ? Number(form.court_price) : undefined;
      await submitFacilityRequest({
        facility_name: form.facility_name.trim(),
        location: form.location.trim(),
        contact_name: form.contact_name.trim() || undefined,
        whatsapp: form.whatsapp.trim() || undefined,
        google_map_url: form.google_map_url.trim() || undefined,
        notes: form.notes.trim() || undefined,
        sports: [
          {
            name: form.sport_name.trim() || 'General',
            courts: [
              {
                name: form.court_name.trim() || 'Court 1',
                price_per_hour: Number.isFinite(price as number) ? price : undefined,
                open_time: '06:00',
                close_time: '22:00',
              },
            ],
          },
        ],
      });
      setMsg('Request submitted — pending approval.');
      setForm((f) => ({ ...f, facility_name: '', location: '', notes: '' }));
      await reload();
      setTab('mine');
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : 'submit_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onApprove(id: number) {
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      const row = await approveFacilityRequest(id);
      setMsg(
        row.facility_slug
          ? `Approved. Facility ready: /facility/${row.facility_slug}`
          : `Approved request #${id}.`,
      );
      await reload();
      const session = await fetchSession();
      if (session?.user?.facilities?.length) {
        // refresh memberships after approve
      }
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : 'approve_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onReject(id: number) {
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      await rejectFacilityRequest(id, 'Rejected from portal');
      setMsg(`Rejected request #${id}.`);
      await reload();
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : 'reject_failed');
    } finally {
      setBusy(false);
    }
  }

  if (!loaded) {
    return (
      <main className={loginStyles.wrap}>
        <div className={loginStyles.box}>
          <p className={loginStyles.msg}>Loading…</p>
        </div>
      </main>
    );
  }

  const homeHref =
    user?.facilities?.[0]
      ? `/facility/${encodeURIComponent(user.facilities[0].slug || '')}`
      : '/home';

  return (
    <main style={{ maxWidth: 960, margin: '0 auto', padding: '24px 16px 48px' }}>
      <header style={{ marginBottom: 20 }}>
        <p style={{ margin: 0 }}>
          <Link href={homeHref}>← Back</Link>
        </p>
        <h1 style={{ margin: '8px 0 4px' }}>Add facility</h1>
        <p style={{ margin: 0, opacity: 0.8 }}>
          Request a new facility (first onboarding or additional location). Approval creates the
          facility and admin access.
        </p>
      </header>

      <div className={styles.tabRow}>
        <button
          type="button"
          className={`${styles.tabBtn} ${tab === 'submit' ? styles.tabBtnActive : ''}`}
          onClick={() => setTab('submit')}
        >
          Submit
        </button>
        <button
          type="button"
          className={`${styles.tabBtn} ${tab === 'mine' ? styles.tabBtnActive : ''}`}
          onClick={() => setTab('mine')}
        >
          My requests
        </button>
        <button
          type="button"
          className={`${styles.tabBtn} ${tab === 'review' ? styles.tabBtnActive : ''}`}
          onClick={() => setTab('review')}
        >
          Review pending
        </button>
      </div>

      {err && <p className={styles.settingsErr}>{err}</p>}
      {msg && <p className={styles.settingsOk}>{msg}</p>}

      {tab === 'submit' && (
        <section className={styles.panel}>
          <form className={styles.settingsForm} onSubmit={onSubmit}>
            <label>
              Facility name
              <input
                value={form.facility_name}
                onChange={(e) => setForm({ ...form, facility_name: e.target.value })}
                required
              />
            </label>
            <label>
              Location
              <input
                value={form.location}
                onChange={(e) => setForm({ ...form, location: e.target.value })}
                required
              />
            </label>
            <label>
              Contact name
              <input
                value={form.contact_name}
                onChange={(e) => setForm({ ...form, contact_name: e.target.value })}
              />
            </label>
            <label>
              WhatsApp
              <input
                value={form.whatsapp}
                onChange={(e) => setForm({ ...form, whatsapp: e.target.value })}
              />
            </label>
            <label>
              Google Maps URL
              <input
                value={form.google_map_url}
                onChange={(e) => setForm({ ...form, google_map_url: e.target.value })}
                placeholder="https://maps.app.goo.gl/…"
              />
            </label>
            <label>
              Sport
              <input
                value={form.sport_name}
                onChange={(e) => setForm({ ...form, sport_name: e.target.value })}
              />
            </label>
            <label>
              Court name
              <input
                value={form.court_name}
                onChange={(e) => setForm({ ...form, court_name: e.target.value })}
              />
            </label>
            <label>
              Court price / hr
              <input
                type="number"
                min="0"
                value={form.court_price}
                onChange={(e) => setForm({ ...form, court_price: e.target.value })}
              />
            </label>
            <label>
              Notes
              <input
                value={form.notes}
                onChange={(e) => setForm({ ...form, notes: e.target.value })}
              />
            </label>
            <button type="submit" className={styles.settingsPrimary} disabled={busy}>
              {busy ? 'Submitting…' : 'Submit request'}
            </button>
          </form>
        </section>
      )}

      {tab === 'mine' && (
        <section className={styles.panel}>
          {mine.length === 0 ? (
            <p>No requests yet.</p>
          ) : (
            <div className={styles.tableWrap}>
              <table className={styles.dataTable}>
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Location</th>
                    <th>Status</th>
                    <th>Facility</th>
                  </tr>
                </thead>
                <tbody>
                  {mine.map((r) => (
                    <tr key={r.id}>
                      <td>{r.facility_name}</td>
                      <td>{r.location}</td>
                      <td>
                        <span
                          className={`${styles.statusPill} ${
                            r.status === 'approved'
                              ? styles.statusOk
                              : r.status === 'rejected'
                                ? styles.statusBad
                                : ''
                          }`}
                        >
                          {r.status}
                        </span>
                      </td>
                      <td>
                        {r.facility_slug ? (
                          <Link href={`/facility/${encodeURIComponent(r.facility_slug)}`}>
                            Open
                          </Link>
                        ) : (
                          '—'
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      )}

      {tab === 'review' && (
        <section className={styles.panel}>
          <p style={{ opacity: 0.8, marginTop: 0 }}>
            Local/dev: any signed-in user can approve. Production: service key or platform role.
          </p>
          {pending.length === 0 ? (
            <p>No pending requests.</p>
          ) : (
            <div className={styles.tableWrap}>
              <table className={styles.dataTable}>
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Requester</th>
                    <th>Location</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {pending.map((r) => (
                    <tr key={r.id}>
                      <td>{r.facility_name}</td>
                      <td>{r.requester_email}</td>
                      <td>{r.location}</td>
                      <td style={{ display: 'flex', gap: 8 }}>
                        <button
                          type="button"
                          className={styles.settingsPrimary}
                          disabled={busy}
                          onClick={() => onApprove(r.id)}
                        >
                          Approve
                        </button>
                        <button
                          type="button"
                          className={styles.settingsDanger}
                          disabled={busy}
                          onClick={() => onReject(r.id)}
                        >
                          Reject
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {msg.includes('Facility ready') && (
            <p style={{ marginTop: 12 }}>
              <button
                type="button"
                className={styles.settingsPrimary}
                onClick={async () => {
                  const session = await fetchSession();
                  if (session) router.push(postLoginPath(session));
                }}
              >
                Go to my facilities
              </button>
            </p>
          )}
        </section>
      )}
    </main>
  );
}
