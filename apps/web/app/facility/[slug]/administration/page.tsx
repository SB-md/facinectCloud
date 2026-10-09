'use client';

import { FormEvent, useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import FacilityShell from '../FacilityShell';
import { FacilityMembership, fetchSession, findFacilityBySlug, getCachedUser } from '../../../../lib/auth';
import {
  CourtRow,
  FacilityProfile,
  PaymentSetting,
  ServiceFlags,
  SportRow,
  StaffRow,
  createCourt,
  createSport,
  getProfile,
  getServices,
  listCourts,
  listPayments,
  listSports,
  listStaff,
  savePayment,
  saveProfile,
  saveServices,
  updateCourtStatus,
  updateSportStatus,
  updateStaffStatus,
  upsertStaff,
} from '../../../../lib/administration';
import styles from '../portal.module.css';

type Tab = 'profile' | 'sports' | 'courts' | 'payments' | 'services' | 'staff';

export default function AdministrationPage() {
  return (
    <FacilityShell
      pageKey="administration"
      title="Administration"
      description="Facility profile, sports, courts, payments, staff, and service toggles."
    >
      <AdminBody />
    </FacilityShell>
  );
}

function AdminBody() {
  const params = useParams<{ slug: string }>();
  const [facility, setFacility] = useState<FacilityMembership | null>(null);
  const [tab, setTab] = useState<Tab>('profile');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');
  const [msg, setMsg] = useState('');
  const [loaded, setLoaded] = useState(false);

  const [profile, setProfile] = useState<FacilityProfile | null>(null);
  const [sports, setSports] = useState<SportRow[]>([]);
  const [courts, setCourts] = useState<CourtRow[]>([]);
  const [payments, setPayments] = useState<PaymentSetting[]>([]);
  const [services, setServices] = useState<ServiceFlags | null>(null);
  const [staff, setStaff] = useState<StaffRow[]>([]);

  const [sportName, setSportName] = useState('');
  const [courtName, setCourtName] = useState('');
  const [courtPrice, setCourtPrice] = useState('');
  const [staffEmail, setStaffEmail] = useState('');
  const [staffName, setStaffName] = useState('');
  const [staffRole, setStaffRole] = useState('coach');

  const facilityId = facility?.facilityId;

  const reload = useCallback(async (fid: number) => {
    const [p, sp, ct, pay, svc, st] = await Promise.all([
      getProfile(fid),
      listSports(fid, 'all'),
      listCourts(fid, 'all'),
      listPayments(fid),
      getServices(fid),
      listStaff(fid),
    ]);
    setProfile(p);
    setSports(sp.sports || []);
    setCourts(ct.courts || []);
    setPayments(pay.payment_settings || []);
    setServices(svc);
    setStaff(st.staff || []);
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
          await reload(f.facilityId);
        } catch (e) {
          setErr(e instanceof Error ? e.message : 'load_failed');
        }
      }
      setLoaded(true);
    })();
  }, [params.slug, reload]);

  async function wrap(fn: () => Promise<void>) {
    if (!facilityId) return;
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      await fn();
      await reload(facilityId);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'request_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onSaveProfile(e: FormEvent) {
    e.preventDefault();
    if (!facilityId || !profile) return;
    await wrap(async () => {
      await saveProfile(facilityId, profile);
      setMsg('Profile saved.');
    });
  }

  async function onAddSport(e: FormEvent) {
    e.preventDefault();
    if (!facilityId || !sportName.trim()) return;
    await wrap(async () => {
      await createSport(facilityId, { name: sportName.trim() });
      setSportName('');
      setMsg('Sport added.');
    });
  }

  async function onAddCourt(e: FormEvent) {
    e.preventDefault();
    if (!facilityId || !courtName.trim()) return;
    await wrap(async () => {
      const price = courtPrice.trim() ? Number(courtPrice) : undefined;
      await createCourt(facilityId, {
        name: courtName.trim(),
        price_per_hour: Number.isFinite(price as number) ? price : undefined,
      });
      setCourtName('');
      setCourtPrice('');
      setMsg('Court added.');
    });
  }

  async function onSavePayment(e: FormEvent) {
    e.preventDefault();
    if (!facilityId) return;
    const current = payments[0] || {
      facility_id: facilityId,
      sport_id: 0,
      allow_full_payment: true,
      allow_advance_payment: true,
      allow_spot_payment: false,
      advance_amount: 0,
    };
    await wrap(async () => {
      await savePayment(facilityId, { ...current, sport_id: 0 });
      setMsg('Payment settings saved.');
    });
  }

  async function onSaveServices(e: FormEvent) {
    e.preventDefault();
    if (!facilityId || !services) return;
    await wrap(async () => {
      await saveServices(facilityId, services);
      setMsg('Service flags saved.');
    });
  }

  async function onAddStaff(e: FormEvent) {
    e.preventDefault();
    if (!facilityId || !staffEmail.trim()) return;
    await wrap(async () => {
      await upsertStaff(facilityId, {
        email: staffEmail.trim(),
        name: staffName.trim() || undefined,
        role: staffRole,
      });
      setStaffEmail('');
      setStaffName('');
      setMsg('Staff membership saved. User must already exist in identity.');
    });
  }

  if (!loaded) return <p>Loading administration…</p>;
  if (!facilityId) return <p className={styles.settingsErr}>Facility not found for this account.</p>;

  const pay = payments[0] || {
    facility_id: facilityId,
    sport_id: 0,
    allow_full_payment: true,
    allow_advance_payment: true,
    allow_spot_payment: false,
    advance_amount: 0,
  };

  const tabs: { id: Tab; label: string }[] = [
    { id: 'profile', label: 'Profile' },
    { id: 'sports', label: 'Sports' },
    { id: 'courts', label: 'Courts' },
    { id: 'payments', label: 'Payments' },
    { id: 'services', label: 'Services' },
    { id: 'staff', label: 'Staff' },
  ];

  return (
    <>
      <div className={styles.detailGrid}>
        <div className={styles.detailCard}>
          <span>Sports</span>
          <strong>{sports.filter((s) => s.status === 'active').length}</strong>
        </div>
        <div className={styles.detailCard}>
          <span>Courts</span>
          <strong>{courts.filter((c) => c.status === 'active').length}</strong>
        </div>
        <div className={styles.detailCard}>
          <span>Staff</span>
          <strong>{staff.filter((s) => s.status === 'active').length}</strong>
        </div>
      </div>
      <p style={{ marginBottom: 12 }}>
        <Link href="/add-facility">Add another facility →</Link>
      </p>

      <div className={styles.tabRow}>
        {tabs.map((t) => (
          <button
            key={t.id}
            type="button"
            className={`${styles.tabBtn} ${tab === t.id ? styles.tabBtnActive : ''}`}
            onClick={() => setTab(t.id)}
          >
            {t.label}
          </button>
        ))}
      </div>

      {err && <p className={styles.settingsErr}>{err}</p>}
      {msg && <p className={styles.settingsOk}>{msg}</p>}

      {tab === 'profile' && profile && (
        <section className={styles.panel}>
          <h2>Facility profile</h2>
          <form className={styles.settingsForm} onSubmit={onSaveProfile} style={{ marginTop: 12 }}>
            <label>
              Display name
              <input
                value={profile.display_name}
                onChange={(e) => setProfile({ ...profile, display_name: e.target.value })}
                required
              />
            </label>
            <label>
              Location
              <input
                value={profile.location || ''}
                onChange={(e) => setProfile({ ...profile, location: e.target.value })}
              />
            </label>
            <label>
              City
              <input
                value={profile.city || ''}
                onChange={(e) => setProfile({ ...profile, city: e.target.value })}
              />
            </label>
            <label>
              Phone
              <input
                value={profile.phone || ''}
                onChange={(e) => setProfile({ ...profile, phone: e.target.value })}
              />
            </label>
            <label>
              WhatsApp
              <input
                value={profile.whatsapp || ''}
                onChange={(e) => setProfile({ ...profile, whatsapp: e.target.value })}
              />
            </label>
            <label>
              Email
              <input
                type="email"
                value={profile.email || ''}
                onChange={(e) => setProfile({ ...profile, email: e.target.value })}
              />
            </label>
            <label>
              Open time
              <input
                placeholder="06:00"
                value={profile.open_time || ''}
                onChange={(e) => setProfile({ ...profile, open_time: e.target.value })}
              />
            </label>
            <label>
              Close time
              <input
                placeholder="22:00"
                value={profile.close_time || ''}
                onChange={(e) => setProfile({ ...profile, close_time: e.target.value })}
              />
            </label>
            <label>
              Address
              <input
                value={profile.address || ''}
                onChange={(e) => setProfile({ ...profile, address: e.target.value })}
              />
            </label>
            <button type="submit" className={styles.settingsPrimary} disabled={busy}>
              {busy ? 'Saving…' : 'Save profile'}
            </button>
          </form>
        </section>
      )}

      {tab === 'sports' && (
        <section className={styles.panel}>
          <form className={styles.inlineForm} onSubmit={onAddSport} style={{ marginBottom: 12 }}>
            <label className={styles.fieldGrow}>
              Sport name
              <input value={sportName} onChange={(e) => setSportName(e.target.value)} required />
            </label>
            <button type="submit" className={styles.settingsPrimary} disabled={busy}>
              Add
            </button>
          </form>
          {sports.length === 0 ? (
            <p>No sports yet.</p>
          ) : (
            <div className={styles.tableWrap}>
              <table className={styles.dataTable}>
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Status</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {sports.map((s) => (
                    <tr key={s.id}>
                      <td>{s.name}</td>
                      <td>
                        <span
                          className={`${styles.statusPill} ${
                            s.status === 'active' ? styles.statusOk : styles.statusBad
                          }`}
                        >
                          {s.status}
                        </span>
                      </td>
                      <td>
                        <button
                          type="button"
                          className={s.status === 'active' ? styles.settingsDanger : styles.settingsPrimary}
                          disabled={busy}
                          onClick={() =>
                            wrap(async () => {
                              await updateSportStatus(
                                facilityId,
                                s.id,
                                s.status === 'active' ? 'inactive' : 'active',
                              );
                              setMsg(`Sport ${s.name} updated.`);
                            })
                          }
                        >
                          {s.status === 'active' ? 'Deactivate' : 'Activate'}
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

      {tab === 'courts' && (
        <section className={styles.panel}>
          <form className={styles.inlineForm} onSubmit={onAddCourt} style={{ marginBottom: 12 }}>
            <label className={styles.fieldGrow}>
              Court name
              <input value={courtName} onChange={(e) => setCourtName(e.target.value)} required />
            </label>
            <label>
              Price / hr
              <input
                type="number"
                min="0"
                value={courtPrice}
                onChange={(e) => setCourtPrice(e.target.value)}
              />
            </label>
            <button type="submit" className={styles.settingsPrimary} disabled={busy}>
              Add
            </button>
          </form>
          {courts.length === 0 ? (
            <p>No courts yet. Catalogue only — booking slots stay on booking service.</p>
          ) : (
            <div className={styles.tableWrap}>
              <table className={styles.dataTable}>
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Price</th>
                    <th>Status</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {courts.map((c) => (
                    <tr key={c.id}>
                      <td>{c.name}</td>
                      <td>
                        {c.price_per_hour != null
                          ? `₹${Number(c.price_per_hour).toLocaleString()}`
                          : '—'}
                      </td>
                      <td>
                        <span
                          className={`${styles.statusPill} ${
                            c.status === 'active' ? styles.statusOk : styles.statusBad
                          }`}
                        >
                          {c.status}
                        </span>
                      </td>
                      <td>
                        <button
                          type="button"
                          className={c.status === 'active' ? styles.settingsDanger : styles.settingsPrimary}
                          disabled={busy}
                          onClick={() =>
                            wrap(async () => {
                              await updateCourtStatus(
                                facilityId,
                                c.id,
                                c.status === 'active' ? 'inactive' : 'active',
                              );
                              setMsg(`Court ${c.name} updated.`);
                            })
                          }
                        >
                          {c.status === 'active' ? 'Deactivate' : 'Activate'}
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

      {tab === 'payments' && (
        <section className={styles.panel}>
          <h2>Booking payment modes</h2>
          <form className={styles.settingsForm} onSubmit={onSavePayment} style={{ marginTop: 12 }}>
            <label style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
              <input
                type="checkbox"
                checked={pay.allow_full_payment}
                onChange={(e) =>
                  setPayments([{ ...pay, allow_full_payment: e.target.checked }])
                }
              />
              Allow full payment
            </label>
            <label style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
              <input
                type="checkbox"
                checked={pay.allow_advance_payment}
                onChange={(e) =>
                  setPayments([{ ...pay, allow_advance_payment: e.target.checked }])
                }
              />
              Allow advance payment
            </label>
            <label style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
              <input
                type="checkbox"
                checked={pay.allow_spot_payment}
                onChange={(e) =>
                  setPayments([{ ...pay, allow_spot_payment: e.target.checked }])
                }
              />
              Allow spot payment
            </label>
            <label>
              Advance amount (₹)
              <input
                type="number"
                min="0"
                value={pay.advance_amount}
                onChange={(e) =>
                  setPayments([{ ...pay, advance_amount: Number(e.target.value) || 0 }])
                }
              />
            </label>
            <button type="submit" className={styles.settingsPrimary} disabled={busy}>
              {busy ? 'Saving…' : 'Save payments'}
            </button>
          </form>
        </section>
      )}

      {tab === 'services' && services && (
        <section className={styles.panel}>
          <h2>Full services / maintenance</h2>
          <form className={styles.settingsForm} onSubmit={onSaveServices} style={{ marginTop: 12 }}>
            <label style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
              <input
                type="checkbox"
                checked={services.staff_enabled}
                onChange={(e) => setServices({ ...services, staff_enabled: e.target.checked })}
              />
              Staff portal enabled
            </label>
            <label style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
              <input
                type="checkbox"
                checked={services.customer_enabled}
                onChange={(e) => setServices({ ...services, customer_enabled: e.target.checked })}
              />
              Customer booking enabled
            </label>
            <button type="submit" className={styles.settingsPrimary} disabled={busy}>
              {busy ? 'Saving…' : 'Save services'}
            </button>
          </form>
        </section>
      )}

      {tab === 'staff' && (
        <section className={styles.panel}>
          <form className={styles.settingsForm} onSubmit={onAddStaff} style={{ marginBottom: 16 }}>
            <label>
              Email (existing identity user)
              <input
                type="email"
                value={staffEmail}
                onChange={(e) => setStaffEmail(e.target.value)}
                required
              />
            </label>
            <label>
              Name
              <input value={staffName} onChange={(e) => setStaffName(e.target.value)} />
            </label>
            <label>
              Role
              <select value={staffRole} onChange={(e) => setStaffRole(e.target.value)}>
                <option value="admin">admin</option>
                <option value="sub_admin">sub_admin</option>
                <option value="headcoach">headcoach</option>
                <option value="coach">coach</option>
                <option value="tournament_admin">tournament_admin</option>
              </select>
            </label>
            <button type="submit" className={styles.settingsPrimary} disabled={busy}>
              Save staff
            </button>
          </form>
          {staff.length === 0 ? (
            <p>No staff memberships yet.</p>
          ) : (
            <div className={styles.tableWrap}>
              <table className={styles.dataTable}>
                <thead>
                  <tr>
                    <th>Email</th>
                    <th>Name</th>
                    <th>Role</th>
                    <th>Status</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {staff.map((s) => (
                    <tr key={s.membership_id}>
                      <td>{s.email}</td>
                      <td>{s.name || '—'}</td>
                      <td>{s.role}</td>
                      <td>
                        <span
                          className={`${styles.statusPill} ${
                            s.status === 'active' ? styles.statusOk : styles.statusBad
                          }`}
                        >
                          {s.status}
                        </span>
                      </td>
                      <td>
                        <button
                          type="button"
                          className={s.status === 'active' ? styles.settingsDanger : styles.settingsPrimary}
                          disabled={busy}
                          onClick={() =>
                            wrap(async () => {
                              await updateStaffStatus(
                                facilityId,
                                s.membership_id,
                                s.status === 'active' ? 'disabled' : 'active',
                              );
                              setMsg(`Staff ${s.email} updated.`);
                            })
                          }
                        >
                          {s.status === 'active' ? 'Disable' : 'Enable'}
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
    </>
  );
}
