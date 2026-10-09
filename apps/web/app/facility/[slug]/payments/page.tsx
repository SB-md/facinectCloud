'use client';

import { FormEvent, useCallback, useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import FacilityShell from '../FacilityShell';
import { FacilityMembership, fetchSession, findFacilityBySlug, getCachedUser } from '../../../../lib/auth';
import {
  CreatePaymentInput,
  LedgerRow,
  PaymentSummary,
  createPayment,
  listLedger,
  markPaymentPaid,
  paymentsSummary,
  todayISO,
} from '../../../../lib/payments';
import styles from '../portal.module.css';

export default function PaymentsPage() {
  return (
    <FacilityShell
      pageKey="payments"
      title="Payments"
      description="Court, membership, coaching, and tournament payment ledger."
    >
      <PaymentsBody />
    </FacilityShell>
  );
}

function PaymentsBody() {
  const params = useParams<{ slug: string }>();
  const [facility, setFacility] = useState<FacilityMembership | null>(null);
  const [tab, setTab] = useState<'ledger' | 'record'>('ledger');
  const [rows, setRows] = useState<LedgerRow[]>([]);
  const [summary, setSummary] = useState<PaymentSummary | null>(null);
  const [category, setCategory] = useState('ALL');
  const [status, setStatus] = useState('all');
  const [from, setFrom] = useState(todayISO());
  const [to, setTo] = useState(todayISO());
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');
  const [msg, setMsg] = useState('');
  const [loaded, setLoaded] = useState(false);

  const [form, setForm] = useState<CreatePaymentInput>({
    category: 'BOOKING',
    title: '',
    customer_name: '',
    customer_phone: '',
    amount: 0,
    paid_amount: 0,
    payment_method: 'cash',
  });
  const [amountText, setAmountText] = useState('');

  const facilityId = facility?.facilityId;

  const reload = useCallback(
    async (fid: number) => {
      const opts = { category, status, from, to };
      const [ledger, sum] = await Promise.all([listLedger(fid, opts), paymentsSummary(fid, opts)]);
      setRows(ledger.ledger || []);
      setSummary(sum);
    },
    [category, status, from, to],
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

  async function onRecord(e: FormEvent) {
    e.preventDefault();
    if (!facilityId) return;
    const amount = Number(amountText);
    if (!form.title.trim() || !Number.isFinite(amount) || amount < 0) {
      setErr('title_and_amount_required');
      return;
    }
    await wrap(async () => {
      await createPayment(facilityId, {
        ...form,
        title: form.title.trim(),
        customer_name: form.customer_name?.trim(),
        customer_phone: form.customer_phone?.trim(),
        amount,
        paid_amount: 0,
        status: 'pending',
      });
      setMsg('Payment recorded.');
      setForm({
        category: 'BOOKING',
        title: '',
        customer_name: '',
        customer_phone: '',
        amount: 0,
        paid_amount: 0,
        payment_method: 'cash',
      });
      setAmountText('');
      setTab('ledger');
    });
  }

  if (!loaded) return <p>Loading payments…</p>;
  if (!facilityId) return <p className={styles.settingsErr}>Facility not found for this account.</p>;

  return (
    <>
      <div className={styles.detailGrid}>
        <div className={styles.detailCard}>
          <span>Entries</span>
          <strong>{summary?.ledger_count ?? 0}</strong>
        </div>
        <div className={styles.detailCard}>
          <span>Pending</span>
          <strong>{summary?.pending_count ?? 0}</strong>
        </div>
        <div className={styles.detailCard}>
          <span>Pending ₹</span>
          <strong>₹{Number(summary?.pending_amount || 0).toLocaleString()}</strong>
        </div>
        <div className={styles.detailCard}>
          <span>Collected ₹</span>
          <strong>₹{Number(summary?.total_paid || 0).toLocaleString()}</strong>
        </div>
      </div>

      <div className={styles.tabRow}>
        <button
          type="button"
          className={`${styles.tabBtn} ${tab === 'ledger' ? styles.tabBtnActive : ''}`}
          onClick={() => setTab('ledger')}
        >
          Ledger
        </button>
        <button
          type="button"
          className={`${styles.tabBtn} ${tab === 'record' ? styles.tabBtnActive : ''}`}
          onClick={() => setTab('record')}
        >
          Record
        </button>
      </div>

      {err && <p className={styles.settingsErr}>{err}</p>}
      {msg && <p className={styles.settingsOk}>{msg}</p>}

      {tab === 'ledger' && (
        <section className={styles.panel}>
          <div className={styles.inlineForm} style={{ marginBottom: 12, flexWrap: 'wrap' }}>
            <label>
              Category
              <select value={category} onChange={(e) => setCategory(e.target.value)}>
                <option value="ALL">All</option>
                <option value="BOOKING">Booking</option>
                <option value="MEMBERSHIP">Membership</option>
                <option value="COACHING">Coaching</option>
                <option value="TOURNAMENT">Tournament</option>
                <option value="OTHER">Other</option>
              </select>
            </label>
            <label>
              Status
              <select value={status} onChange={(e) => setStatus(e.target.value)}>
                <option value="all">All</option>
                <option value="pending">Pending</option>
                <option value="partial">Partial</option>
                <option value="paid">Paid</option>
                <option value="failed">Failed</option>
              </select>
            </label>
            <label>
              From
              <input type="date" value={from} onChange={(e) => setFrom(e.target.value)} />
            </label>
            <label>
              To
              <input type="date" value={to} onChange={(e) => setTo(e.target.value)} />
            </label>
            <button
              type="button"
              className={styles.settingsPrimary}
              disabled={busy}
              onClick={() =>
                wrap(async () => {
                  setMsg('Refreshed.');
                })
              }
            >
              {busy ? 'Loading…' : 'Refresh'}
            </button>
          </div>

          {rows.length === 0 ? (
            <p>No ledger rows for this filter. Use Record to add one.</p>
          ) : (
            <div className={styles.tableWrap}>
              <table className={styles.dataTable}>
                <thead>
                  <tr>
                    <th>Title</th>
                    <th>Category</th>
                    <th>Customer</th>
                    <th>Amount</th>
                    <th>Paid</th>
                    <th>Status</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {rows.map((r) => (
                    <tr key={r.id}>
                      <td>{r.title}</td>
                      <td>{r.category}</td>
                      <td>{r.customer_name || r.customer_phone || '—'}</td>
                      <td>₹{Number(r.amount).toLocaleString()}</td>
                      <td>₹{Number(r.paid_amount).toLocaleString()}</td>
                      <td>
                        <span
                          className={`${styles.statusPill} ${
                            r.status === 'paid' ? styles.statusOk : styles.statusBad
                          }`}
                        >
                          {r.status}
                        </span>
                      </td>
                      <td>
                        {(r.status === 'pending' || r.status === 'partial') && (
                          <button
                            type="button"
                            className={styles.settingsPrimary}
                            disabled={busy}
                            onClick={() =>
                              wrap(async () => {
                                await markPaymentPaid(facilityId, r.id, {
                                  payment_method: 'cash',
                                });
                                setMsg(`Payment #${r.id} marked paid.`);
                              })
                            }
                          >
                            Mark paid
                          </button>
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

      {tab === 'record' && (
        <section className={styles.panel}>
          <h2>Record payment</h2>
          <form className={styles.settingsForm} onSubmit={onRecord} style={{ marginTop: 12 }}>
            <label>
              Title
              <input
                value={form.title}
                onChange={(e) => setForm({ ...form, title: e.target.value })}
                placeholder="Court booking — Court 1"
                required
              />
            </label>
            <label>
              Category
              <select
                value={form.category || 'BOOKING'}
                onChange={(e) => setForm({ ...form, category: e.target.value })}
              >
                <option value="BOOKING">Booking</option>
                <option value="MEMBERSHIP">Membership</option>
                <option value="COACHING">Coaching</option>
                <option value="TOURNAMENT">Tournament</option>
                <option value="OTHER">Other</option>
              </select>
            </label>
            <label>
              Customer name
              <input
                value={form.customer_name || ''}
                onChange={(e) => setForm({ ...form, customer_name: e.target.value })}
              />
            </label>
            <label>
              Customer phone
              <input
                value={form.customer_phone || ''}
                onChange={(e) => setForm({ ...form, customer_phone: e.target.value })}
              />
            </label>
            <label>
              Amount (₹)
              <input
                type="number"
                min="0"
                step="1"
                value={amountText}
                onChange={(e) => setAmountText(e.target.value)}
                required
              />
            </label>
            <label>
              Method
              <select
                value={form.payment_method || 'cash'}
                onChange={(e) => setForm({ ...form, payment_method: e.target.value })}
              >
                <option value="cash">Cash</option>
                <option value="upi">UPI</option>
                <option value="card">Card</option>
                <option value="online">Online</option>
              </select>
            </label>
            <button type="submit" className={styles.settingsPrimary} disabled={busy}>
              {busy ? 'Saving…' : 'Record pending'}
            </button>
          </form>
        </section>
      )}
    </>
  );
}
