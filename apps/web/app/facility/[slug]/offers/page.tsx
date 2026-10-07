'use client';

import { FormEvent, useCallback, useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import FacilityShell from '../FacilityShell';
import { FacilityMembership, fetchSession, findFacilityBySlug, getCachedUser } from '../../../../lib/auth';
import {
  CreateOfferInput,
  OfferRow,
  createOffer,
  listOffers,
  todayISO,
  updateOfferStatus,
} from '../../../../lib/offers';
import styles from '../portal.module.css';

export default function OffersPage() {
  return (
    <FacilityShell
      pageKey="offers"
      title="Offers"
      description="Promotions and discount coupons for bookings."
    >
      <OffersBody />
    </FacilityShell>
  );
}

function OffersBody() {
  const params = useParams<{ slug: string }>();
  const [facility, setFacility] = useState<FacilityMembership | null>(null);
  const [tab, setTab] = useState<'list' | 'create'>('list');
  const [offers, setOffers] = useState<OfferRow[]>([]);
  const [statusFilter, setStatusFilter] = useState<'active' | 'all' | 'inactive'>('active');
  const [kindFilter, setKindFilter] = useState<'all' | 'promotion' | 'discount'>('all');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');
  const [msg, setMsg] = useState('');
  const [loaded, setLoaded] = useState(false);

  const [form, setForm] = useState<CreateOfferInput>({
    code: '',
    title: '',
    description: '',
    offer_kind: 'discount',
    discount_type: 'flat',
    discount_value: 0,
    valid_from: todayISO(),
    duration_days: 30,
  });
  const [valueText, setValueText] = useState('100');
  const [limitText, setLimitText] = useState('');
  const [daysText, setDaysText] = useState('30');

  const facilityId = facility?.facilityId;

  const loadList = useCallback(async (fid: number, status: string, kind: string) => {
    const res = await listOffers(fid, {
      status: status === 'all' ? undefined : status,
      kind: kind === 'all' ? undefined : kind,
    });
    setOffers(res.offers || []);
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
          await loadList(f.facilityId, statusFilter, kindFilter);
        } catch (e) {
          setErr(e instanceof Error ? e.message : 'load_failed');
        }
      }
      setLoaded(true);
    })();
  }, [params.slug, loadList, statusFilter, kindFilter]);

  async function onRefresh() {
    if (!facilityId) return;
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      await loadList(facilityId, statusFilter, kindFilter);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'load_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onToggleStatus(row: OfferRow) {
    if (!facilityId) return;
    const next = row.status === 'active' ? 'inactive' : 'active';
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      await updateOfferStatus(facilityId, row.id, next);
      setMsg(`Offer ${row.code} set to ${next}.`);
      await loadList(facilityId, statusFilter, kindFilter);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'status_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onCreate(e: FormEvent) {
    e.preventDefault();
    if (!facilityId) return;
    if (!form.code.trim() || !form.title.trim()) {
      setErr('code_and_title_required');
      return;
    }
    const value = Number(valueText);
    if (!Number.isFinite(value) || value < 0) {
      setErr('invalid_discount_value');
      return;
    }
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      const days = daysText.trim() ? Number(daysText) : undefined;
      const limit = limitText.trim() ? Number(limitText) : undefined;
      await createOffer(facilityId, {
        ...form,
        code: form.code.trim(),
        title: form.title.trim(),
        description: form.description?.trim(),
        discount_value: value,
        duration_days: Number.isFinite(days as number) && (days as number) > 0 ? days : undefined,
        usage_limit: Number.isFinite(limit as number) ? limit : undefined,
      });
      setMsg('Offer published.');
      setForm({
        code: '',
        title: '',
        description: '',
        offer_kind: 'discount',
        discount_type: 'flat',
        discount_value: 0,
        valid_from: todayISO(),
        duration_days: 30,
      });
      setValueText('100');
      setLimitText('');
      setDaysText('30');
      await loadList(facilityId, statusFilter, kindFilter);
      setTab('list');
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : 'create_failed');
    } finally {
      setBusy(false);
    }
  }

  if (!loaded) {
    return <p>Loading offers…</p>;
  }

  if (!facilityId) {
    return <p className={styles.settingsErr}>Facility not found for this account.</p>;
  }

  return (
    <>
      <div className={styles.detailGrid}>
        <div className={styles.detailCard}>
          <span>Shown</span>
          <strong>{offers.length}</strong>
        </div>
        <div className={styles.detailCard}>
          <span>Filter</span>
          <strong>
            {kindFilter}/{statusFilter}
          </strong>
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
          className={`${styles.tabBtn} ${tab === 'create' ? styles.tabBtnActive : ''}`}
          onClick={() => setTab('create')}
        >
          Create
        </button>
      </div>

      {err && <p className={styles.settingsErr}>{err}</p>}
      {msg && <p className={styles.settingsOk}>{msg}</p>}

      {tab === 'list' && (
        <section className={styles.panel}>
          <div className={styles.inlineForm} style={{ marginBottom: 12 }}>
            <label className={styles.fieldGrow}>
              Kind
              <select
                value={kindFilter}
                onChange={(e) => setKindFilter(e.target.value as typeof kindFilter)}
              >
                <option value="all">All</option>
                <option value="promotion">Promotion</option>
                <option value="discount">Discount</option>
              </select>
            </label>
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
          {offers.length === 0 ? (
            <p>No offers for this filter. Use Create to publish one.</p>
          ) : (
            <div className={styles.tableWrap}>
              <table className={styles.dataTable}>
                <thead>
                  <tr>
                    <th>Code</th>
                    <th>Title</th>
                    <th>Kind</th>
                    <th>Discount</th>
                    <th>Valid to</th>
                    <th>Usage</th>
                    <th>Status</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {offers.map((o) => (
                    <tr key={o.id}>
                      <td>
                        <strong>{o.code}</strong>
                      </td>
                      <td>{o.title}</td>
                      <td>{o.offer_kind}</td>
                      <td>
                        {o.discount_type === 'percent'
                          ? `${o.discount_value}%`
                          : `₹${Number(o.discount_value).toLocaleString()}`}
                      </td>
                      <td>
                        {o.valid_to || '—'}
                        {o.is_expired ? ' (expired)' : ''}
                      </td>
                      <td>
                        {o.used_count}
                        {o.usage_limit != null ? ` / ${o.usage_limit}` : ''}
                      </td>
                      <td>
                        <span
                          className={`${styles.statusPill} ${
                            o.status === 'active' && !o.is_expired ? styles.statusOk : styles.statusBad
                          }`}
                        >
                          {o.status}
                        </span>
                      </td>
                      <td>
                        <button
                          type="button"
                          className={o.status === 'active' ? styles.settingsDanger : styles.settingsPrimary}
                          disabled={busy}
                          onClick={() => onToggleStatus(o)}
                        >
                          {o.status === 'active' ? 'Deactivate' : 'Activate'}
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

      {tab === 'create' && (
        <section className={styles.panel}>
          <h2>Create offer</h2>
          <form className={styles.settingsForm} onSubmit={onCreate} style={{ marginTop: 12 }}>
            <label>
              Coupon code
              <input
                value={form.code}
                onChange={(e) => setForm((f) => ({ ...f, code: e.target.value.toUpperCase() }))}
                placeholder="e.g. WEEKEND100"
                required
              />
            </label>
            <label>
              Title
              <input
                value={form.title}
                onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))}
                placeholder="Weekend flat discount"
                required
              />
            </label>
            <label>
              Description
              <input
                value={form.description || ''}
                onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
              />
            </label>
            <label>
              Kind
              <select
                value={form.offer_kind || 'discount'}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    offer_kind: e.target.value as 'promotion' | 'discount',
                  }))
                }
              >
                <option value="discount">Discount</option>
                <option value="promotion">Promotion</option>
              </select>
            </label>
            <label>
              Discount type
              <select
                value={form.discount_type || 'flat'}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    discount_type: e.target.value as 'flat' | 'percent',
                  }))
                }
              >
                <option value="flat">Flat (₹)</option>
                <option value="percent">Percent (%)</option>
              </select>
            </label>
            <label>
              Discount value
              <input
                type="number"
                min="0"
                step="1"
                value={valueText}
                onChange={(e) => setValueText(e.target.value)}
                required
              />
            </label>
            <label>
              Valid from
              <input
                type="date"
                value={form.valid_from || ''}
                onChange={(e) => setForm((f) => ({ ...f, valid_from: e.target.value }))}
              />
            </label>
            <label>
              Duration (days)
              <input
                type="number"
                min="1"
                step="1"
                value={daysText}
                onChange={(e) => setDaysText(e.target.value)}
              />
            </label>
            <label>
              Usage limit
              <input
                type="number"
                min="0"
                step="1"
                placeholder="Unlimited"
                value={limitText}
                onChange={(e) => setLimitText(e.target.value)}
              />
            </label>
            <button type="submit" className={styles.settingsPrimary} disabled={busy}>
              {busy ? 'Saving…' : 'Publish offer'}
            </button>
          </form>
        </section>
      )}
    </>
  );
}
