'use client';

import { FormEvent, useCallback, useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import FacilityShell from '../FacilityShell';
import { FacilityMembership, fetchSession, findFacilityBySlug, getCachedUser } from '../../../../lib/auth';
import {
  Tournament,
  TournamentInput,
  createTournament,
  listTournaments,
  todayISO,
  updateTournament,
  updateTournamentStatus,
} from '../../../../lib/tournaments';
import styles from '../portal.module.css';

export default function TournamentsPage() {
  return (
    <FacilityShell
      pageKey="tournaments"
      title="Tournaments"
      description="Create and manage tournament events (fixtures come in Phase 2)."
    >
      <TournamentsBody />
    </FacilityShell>
  );
}

function emptyForm(): TournamentInput {
  return {
    name: '',
    start_date: todayISO(),
    end_date: '',
    venue_address: '',
    contact_numbers: '',
    rules: '',
    status: 'draft',
  };
}

function TournamentsBody() {
  const params = useParams<{ slug: string }>();
  const [facility, setFacility] = useState<FacilityMembership | null>(null);
  const [tab, setTab] = useState<'list' | 'create'>('list');
  const [rows, setRows] = useState<Tournament[]>([]);
  const [statusFilter, setStatusFilter] = useState<'all' | 'draft' | 'upcoming' | 'live' | 'completed' | 'cancelled'>(
    'all',
  );
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');
  const [msg, setMsg] = useState('');
  const [loaded, setLoaded] = useState(false);
  const [form, setForm] = useState<TournamentInput>(emptyForm);
  const [editingId, setEditingId] = useState<number | null>(null);

  const facilityId = facility?.facilityId;

  const loadList = useCallback(async (fid: number, status: string) => {
    const res = await listTournaments(fid, { status: status === 'all' ? undefined : status });
    setRows(res.tournaments || []);
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
          await loadList(f.facilityId, statusFilter);
        } catch (e) {
          setErr(e instanceof Error ? e.message : 'load_failed');
        }
      }
      setLoaded(true);
    })();
  }, [params.slug, loadList, statusFilter]);

  async function onRefresh() {
    if (!facilityId) return;
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      await loadList(facilityId, statusFilter);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'load_failed');
    } finally {
      setBusy(false);
    }
  }

  function startEdit(t: Tournament) {
    setEditingId(t.id);
    setForm({
      name: t.name,
      start_date: t.start_date || '',
      end_date: t.end_date || '',
      venue_address: t.venue_address || '',
      location_url: t.location_url || '',
      contact_numbers: t.contact_numbers || '',
      rules: t.rules || '',
      slug: t.slug || '',
      status: t.status,
    });
    setTab('create');
  }

  function startCreate() {
    setEditingId(null);
    setForm(emptyForm());
    setTab('create');
  }

  async function onSave(e: FormEvent) {
    e.preventDefault();
    if (!facilityId) return;
    if (!form.name.trim()) {
      setErr('name_required');
      return;
    }
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      const body: TournamentInput = {
        ...form,
        name: form.name.trim(),
        venue_address: form.venue_address?.trim(),
        location_url: form.location_url?.trim(),
        contact_numbers: form.contact_numbers?.trim(),
        rules: form.rules?.trim(),
        slug: form.slug?.trim(),
      };
      if (editingId) {
        await updateTournament(editingId, body);
        setMsg(`Tournament #${editingId} updated.`);
      } else {
        await createTournament(facilityId, body);
        setMsg('Tournament created.');
      }
      setEditingId(null);
      setForm(emptyForm());
      await loadList(facilityId, statusFilter);
      setTab('list');
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : 'save_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onStatus(t: Tournament, status: string) {
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      await updateTournamentStatus(t.id, status);
      setMsg(`#${t.id} → ${status}`);
      if (facilityId) await loadList(facilityId, statusFilter);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'status_failed');
    } finally {
      setBusy(false);
    }
  }

  if (!loaded) {
    return <p>Loading tournaments…</p>;
  }

  if (!facilityId) {
    return <p className={styles.settingsErr}>Facility not found for this account.</p>;
  }

  return (
    <>
      <div className={styles.detailGrid}>
        <div className={styles.detailCard}>
          <span>Shown</span>
          <strong>{rows.length}</strong>
        </div>
        <div className={styles.detailCard}>
          <span>Filter</span>
          <strong>{statusFilter}</strong>
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
          className={`${styles.tabBtn} ${tab === 'create' ? styles.tabBtnActive : ''}`}
          onClick={startCreate}
        >
          {editingId ? 'Edit' : 'Create'}
        </button>
      </div>

      {err && <p className={styles.settingsErr}>{err}</p>}
      {msg && <p className={styles.settingsOk}>{msg}</p>}

      {tab === 'list' && (
        <section className={styles.panel}>
          <div className={styles.inlineForm} style={{ marginBottom: 12 }}>
            <label className={styles.fieldGrow}>
              Status
              <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value as typeof statusFilter)}
              >
                <option value="all">All</option>
                <option value="draft">Draft</option>
                <option value="upcoming">Upcoming</option>
                <option value="live">Live</option>
                <option value="completed">Completed</option>
                <option value="cancelled">Cancelled</option>
              </select>
            </label>
            <button type="button" className={styles.settingsPrimary} disabled={busy} onClick={onRefresh}>
              {busy ? 'Loading…' : 'Refresh'}
            </button>
          </div>
          {rows.length === 0 ? (
            <p>No tournaments yet. Use Create to add one.</p>
          ) : (
            <div className={styles.tableWrap}>
              <table className={styles.dataTable}>
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Dates</th>
                    <th>Venue</th>
                    <th>Status</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {rows.map((t) => (
                    <tr key={t.id}>
                      <td>
                        <strong>{t.name}</strong>
                        {t.slug ? <div style={{ fontSize: 11, color: '#5a6f68' }}>{t.slug}</div> : null}
                      </td>
                      <td>
                        {t.start_date || '—'}
                        {t.end_date ? ` → ${t.end_date}` : ''}
                      </td>
                      <td>{t.venue_address || '—'}</td>
                      <td>
                        <span
                          className={`${styles.statusPill} ${
                            t.status === 'live' || t.status === 'upcoming'
                              ? styles.statusOk
                              : t.status === 'cancelled'
                                ? styles.statusBad
                                : styles.statusWarn
                          }`}
                        >
                          {t.status}
                        </span>
                      </td>
                      <td>
                        <div className={styles.actionRow}>
                          <button type="button" className={styles.tabBtn} disabled={busy} onClick={() => startEdit(t)}>
                            Edit
                          </button>
                          {t.status === 'draft' ? (
                            <button
                              type="button"
                              className={styles.settingsPrimary}
                              disabled={busy}
                              onClick={() => onStatus(t, 'upcoming')}
                            >
                              Publish
                            </button>
                          ) : null}
                          {t.status === 'upcoming' ? (
                            <button
                              type="button"
                              className={styles.settingsPrimary}
                              disabled={busy}
                              onClick={() => onStatus(t, 'live')}
                            >
                              Go live
                            </button>
                          ) : null}
                          {t.status !== 'cancelled' && t.status !== 'completed' ? (
                            <button
                              type="button"
                              className={styles.settingsDanger}
                              disabled={busy}
                              onClick={() => onStatus(t, 'cancelled')}
                            >
                              Cancel
                            </button>
                          ) : null}
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

      {tab === 'create' && (
        <section className={styles.panel}>
          <h2>{editingId ? `Edit #${editingId}` : 'Create tournament'}</h2>
          <form className={styles.settingsForm} onSubmit={onSave} style={{ marginTop: 12 }}>
            <label>
              Name
              <input
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                required
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
            <label>
              Venue
              <input
                value={form.venue_address || ''}
                onChange={(e) => setForm((f) => ({ ...f, venue_address: e.target.value }))}
              />
            </label>
            <label>
              Location URL
              <input
                value={form.location_url || ''}
                onChange={(e) => setForm((f) => ({ ...f, location_url: e.target.value }))}
              />
            </label>
            <label>
              Contact numbers
              <input
                value={form.contact_numbers || ''}
                onChange={(e) => setForm((f) => ({ ...f, contact_numbers: e.target.value }))}
              />
            </label>
            <label>
              Slug
              <input
                value={form.slug || ''}
                onChange={(e) => setForm((f) => ({ ...f, slug: e.target.value }))}
                placeholder="auto from name if empty"
              />
            </label>
            <label>
              Status
              <select
                value={form.status || 'draft'}
                onChange={(e) => setForm((f) => ({ ...f, status: e.target.value }))}
              >
                <option value="draft">Draft</option>
                <option value="upcoming">Upcoming</option>
                <option value="live">Live</option>
                <option value="completed">Completed</option>
                <option value="cancelled">Cancelled</option>
              </select>
            </label>
            <label>
              Rules
              <textarea
                value={form.rules || ''}
                onChange={(e) => setForm((f) => ({ ...f, rules: e.target.value }))}
                rows={4}
                style={{
                  border: '1px solid var(--portal-line)',
                  borderRadius: 10,
                  padding: '10px 12px',
                  font: 'inherit',
                  fontSize: 14,
                }}
              />
            </label>
            <div className={styles.actionRow}>
              <button type="submit" className={styles.settingsPrimary} disabled={busy}>
                {busy ? 'Saving…' : editingId ? 'Update' : 'Create'}
              </button>
              {editingId ? (
                <button
                  type="button"
                  className={styles.tabBtn}
                  disabled={busy}
                  onClick={() => {
                    setEditingId(null);
                    setForm(emptyForm());
                  }}
                >
                  Clear edit
                </button>
              ) : null}
            </div>
          </form>
        </section>
      )}
    </>
  );
}
