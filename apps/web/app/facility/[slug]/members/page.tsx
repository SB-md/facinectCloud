'use client';

import { FormEvent, useCallback, useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import FacilityShell from '../FacilityShell';
import { FacilityMembership, fetchSession, findFacilityBySlug, getCachedUser } from '../../../../lib/auth';
import {
  MemberRow,
  RegisterMemberInput,
  listMembers,
  registerMember,
  todayISO,
  updateMembershipStatus,
} from '../../../../lib/members';
import styles from '../portal.module.css';

export default function MembersPage() {
  return (
    <FacilityShell
      pageKey="members"
      title="Members"
      description="Membership directory and registrations."
    >
      <MembersBody />
    </FacilityShell>
  );
}

function MembersBody() {
  const params = useParams<{ slug: string }>();
  const [facility, setFacility] = useState<FacilityMembership | null>(null);
  const [tab, setTab] = useState<'list' | 'register'>('list');
  const [members, setMembers] = useState<MemberRow[]>([]);
  const [statusFilter, setStatusFilter] = useState<'active' | 'all' | 'inactive'>('active');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');
  const [msg, setMsg] = useState('');
  const [loaded, setLoaded] = useState(false);

  const [form, setForm] = useState<RegisterMemberInput>({
    full_name: '',
    contact_phone: '',
    contact_email: '',
    whatsapp: '',
    plan_name: '',
    team_name: '',
    start_date: todayISO(),
    primary_member: false,
  });
  const [feeText, setFeeText] = useState('');

  const facilityId = facility?.facilityId;

  const loadList = useCallback(
    async (fid: number, status: string) => {
      const res = await listMembers(fid, { status: status === 'all' ? undefined : status });
      setMembers(res.members || []);
    },
    [],
  );

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

  async function onToggleStatus(row: MemberRow) {
    if (!facilityId) return;
    const next = row.status === 'active' ? 'inactive' : 'active';
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      await updateMembershipStatus(facilityId, row.membership_id, next);
      setMsg(`Membership #${row.membership_id} set to ${next}.`);
      await loadList(facilityId, statusFilter);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'status_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onRegister(e: FormEvent) {
    e.preventDefault();
    if (!facilityId) return;
    if (!form.full_name.trim()) {
      setErr('full_name_required');
      return;
    }
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      const fee = feeText.trim() ? Number(feeText) : undefined;
      await registerMember(facilityId, {
        ...form,
        full_name: form.full_name.trim(),
        contact_phone: form.contact_phone?.trim(),
        contact_email: form.contact_email?.trim(),
        whatsapp: form.whatsapp?.trim(),
        plan_name: form.plan_name?.trim(),
        team_name: form.team_name?.trim(),
        subscription_fee: Number.isFinite(fee as number) ? fee : undefined,
      });
      setMsg('Member registered.');
      setFeeText('');
      setForm({
        full_name: '',
        contact_phone: '',
        contact_email: '',
        whatsapp: '',
        plan_name: '',
        team_name: '',
        start_date: todayISO(),
        primary_member: false,
      });
      await loadList(facilityId, statusFilter);
      setTab('list');
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : 'register_failed');
    } finally {
      setBusy(false);
    }
  }

  if (!loaded) {
    return <p>Loading members…</p>;
  }

  if (!facilityId) {
    return <p className={styles.settingsErr}>Facility not found for this account.</p>;
  }

  return (
    <>
      <div className={styles.detailGrid}>
        <div className={styles.detailCard}>
          <span>Shown</span>
          <strong>{members.length}</strong>
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
          Directory
        </button>
        <button
          type="button"
          className={`${styles.tabBtn} ${tab === 'register' ? styles.tabBtnActive : ''}`}
          onClick={() => setTab('register')}
        >
          Register
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
                <option value="active">Active</option>
                <option value="inactive">Inactive</option>
                <option value="all">All</option>
              </select>
            </label>
            <button type="button" className={styles.settingsPrimary} disabled={busy} onClick={onRefresh}>
              {busy ? 'Loading…' : 'Refresh'}
            </button>
          </div>
          {members.length === 0 ? (
            <p>No members for this filter. Use Register to add one.</p>
          ) : (
            <div className={styles.tableWrap}>
              <table className={styles.dataTable}>
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Phone</th>
                    <th>Plan</th>
                    <th>Team</th>
                    <th>Fee</th>
                    <th>Primary</th>
                    <th>Status</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {members.map((m) => (
                    <tr key={m.membership_id}>
                      <td>{m.full_name}</td>
                      <td>{m.whatsapp || m.contact_phone || '—'}</td>
                      <td>{m.plan_name || '—'}</td>
                      <td>{m.team_name || '—'}</td>
                      <td>
                        {m.subscription_fee != null ? `₹${Number(m.subscription_fee).toLocaleString()}` : '—'}
                      </td>
                      <td>{m.primary_member ? 'Yes' : 'No'}</td>
                      <td>
                        <span
                          className={`${styles.statusPill} ${
                            m.status === 'active' ? styles.statusOk : styles.statusBad
                          }`}
                        >
                          {m.status}
                        </span>
                      </td>
                      <td>
                        <button
                          type="button"
                          className={m.status === 'active' ? styles.settingsDanger : styles.settingsPrimary}
                          disabled={busy}
                          onClick={() => onToggleStatus(m)}
                        >
                          {m.status === 'active' ? 'Deactivate' : 'Activate'}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      )}

      {tab === 'register' && (
        <section className={styles.panel}>
          <h2>Register member</h2>
          <form className={styles.settingsForm} onSubmit={onRegister} style={{ marginTop: 12 }}>
            <label>
              Full name
              <input
                value={form.full_name}
                onChange={(e) => setForm((f) => ({ ...f, full_name: e.target.value }))}
                required
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
              WhatsApp
              <input
                value={form.whatsapp || ''}
                onChange={(e) => setForm((f) => ({ ...f, whatsapp: e.target.value }))}
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
                placeholder="e.g. Monthly court access"
              />
            </label>
            <label>
              Team
              <input
                value={form.team_name || ''}
                onChange={(e) => setForm((f) => ({ ...f, team_name: e.target.value }))}
              />
            </label>
            <label>
              Subscription fee
              <input
                type="number"
                min="0"
                step="1"
                placeholder="0"
                value={feeText}
                onChange={(e) => setFeeText(e.target.value)}
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
            <label style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
              <input
                type="checkbox"
                checked={!!form.primary_member}
                onChange={(e) => setForm((f) => ({ ...f, primary_member: e.target.checked }))}
              />
              Primary member
            </label>
            <button type="submit" className={styles.settingsPrimary} disabled={busy}>
              {busy ? 'Saving…' : 'Register'}
            </button>
          </form>
        </section>
      )}
    </>
  );
}
