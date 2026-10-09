'use client';

import { FormEvent, useCallback, useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import FacilityShell from '../FacilityShell';
import { FacilityMembership, fetchSession, findFacilityBySlug, getCachedUser } from '../../../../lib/auth';
import {
  CreateEnquiryInput,
  EnquiryRow,
  createEnquiry,
  listEnquiries,
  notifyEnquiry,
  reanalyzeEnquiry,
  suggestReply,
  updateEnquiryStatus,
} from '../../../../lib/enquiry';
import styles from '../portal.module.css';

export default function EnquiryPage() {
  return (
    <FacilityShell
      pageKey="enquiry"
      title="Enquiry"
      description="Lead inbox with shared AI analyze and reply assist."
    >
      <EnquiryBody />
    </FacilityShell>
  );
}

function EnquiryBody() {
  const params = useParams<{ slug: string }>();
  const [facility, setFacility] = useState<FacilityMembership | null>(null);
  const [rows, setRows] = useState<EnquiryRow[]>([]);
  const [selected, setSelected] = useState<EnquiryRow | null>(null);
  const [statusFilter, setStatusFilter] = useState('New');
  const [search, setSearch] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');
  const [msg, setMsg] = useState('');
  const [loaded, setLoaded] = useState(false);
  const [reply, setReply] = useState('');
  const [form, setForm] = useState<CreateEnquiryInput>({
    customer_phone: '',
    customer_name: '',
    enquiry_details: '',
    run_ai: true,
  });

  const facilityId = facility?.facilityId;

  const loadList = useCallback(
    async (fid: number) => {
      const res = await listEnquiries(fid, {
        status: statusFilter === 'All' ? undefined : statusFilter,
        search: search.trim() || undefined,
      });
      setRows(res.enquiries || []);
    },
    [statusFilter, search],
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
          await loadList(f.facilityId);
        } catch (e) {
          setErr(e instanceof Error ? e.message : 'load_failed');
        }
      }
      setLoaded(true);
    })();
  }, [params.slug, loadList]);

  async function onRefresh() {
    if (!facilityId) return;
    setBusy(true);
    setErr('');
    try {
      await loadList(facilityId);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'load_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onCreate(e: FormEvent) {
    e.preventDefault();
    if (!facilityId) return;
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      const row = await createEnquiry(facilityId, {
        ...form,
        customer_phone: form.customer_phone.trim(),
        enquiry_details: form.enquiry_details.trim(),
        customer_name: form.customer_name?.trim(),
      });
      setMsg(`Enquiry #${row.id} saved.`);
      setForm({ customer_phone: '', customer_name: '', enquiry_details: '', run_ai: true });
      await loadList(facilityId);
      setSelected(row);
      setReply('');
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : 'create_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onAnalyze() {
    if (!selected || !facilityId) return;
    setBusy(true);
    setErr('');
    try {
      const row = await reanalyzeEnquiry(selected.id);
      setSelected(row);
      setMsg('AI analysis updated.');
      await loadList(facilityId);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'analyze_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onSuggest() {
    if (!selected) return;
    setBusy(true);
    setErr('');
    try {
      const res = await suggestReply(selected.id, facility?.facilityName);
      setReply(res.reply || '');
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'suggest_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onNotify() {
    if (!selected || !facilityId) return;
    if (!reply.trim()) {
      setErr('reply_required');
      return;
    }
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      const res = await notifyEnquiry(selected.id, reply.trim());
      setMsg('Notification sent (or queued).');
      if (res.enquiry) setSelected(res.enquiry);
      await loadList(facilityId);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'notify_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onStatus(status: string) {
    if (!selected || !facilityId) return;
    setBusy(true);
    setErr('');
    try {
      const row = await updateEnquiryStatus(selected.id, status);
      setSelected(row);
      await loadList(facilityId);
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'status_failed');
    } finally {
      setBusy(false);
    }
  }

  if (!loaded) return <p>Loading enquiries…</p>;
  if (!facilityId) {
    return <p className={styles.settingsErr}>Facility not found for this account.</p>;
  }

  const intent =
    selected?.ai_data && typeof selected.ai_data.intent_summary === 'string'
      ? selected.ai_data.intent_summary
      : '';
  const category =
    selected?.ai_data && typeof selected.ai_data.category === 'string' ? selected.ai_data.category : '';

  return (
    <>
      {err && <p className={styles.settingsErr}>{err}</p>}
      {msg && <p className={styles.settingsOk}>{msg}</p>}

      <div className={styles.enquiryLayout}>
        <aside className={styles.enquirySide}>
          <section className={styles.panel} style={{ marginBottom: 12 }}>
            <h2>New enquiry</h2>
            <form className={styles.settingsForm} onSubmit={onCreate} style={{ marginTop: 10 }}>
              <label>
                Phone
                <input
                  value={form.customer_phone}
                  onChange={(e) => setForm((f) => ({ ...f, customer_phone: e.target.value }))}
                  required
                />
              </label>
              <label>
                Name
                <input
                  value={form.customer_name || ''}
                  onChange={(e) => setForm((f) => ({ ...f, customer_name: e.target.value }))}
                />
              </label>
              <label>
                Transcript / notes
                <textarea
                  value={form.enquiry_details}
                  onChange={(e) => setForm((f) => ({ ...f, enquiry_details: e.target.value }))}
                  required
                  rows={5}
                  className={styles.enquiryTextarea}
                />
              </label>
              <label style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
                <input
                  type="checkbox"
                  checked={form.run_ai !== false}
                  onChange={(e) => setForm((f) => ({ ...f, run_ai: e.target.checked }))}
                />
                Run AI analyze
              </label>
              <button type="submit" className={styles.settingsPrimary} disabled={busy}>
                {busy ? 'Saving…' : 'Save'}
              </button>
            </form>
          </section>

          <section className={styles.panel}>
            <div className={styles.inlineForm} style={{ marginBottom: 10 }}>
              <label className={styles.fieldGrow}>
                Status
                <select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
                  <option value="New">New</option>
                  <option value="Follow-up">Follow-up</option>
                  <option value="Resolved">Resolved</option>
                  <option value="Closed">Closed</option>
                  <option value="All">All</option>
                </select>
              </label>
              <label className={styles.fieldGrow}>
                Search phone
                <input value={search} onChange={(e) => setSearch(e.target.value)} />
              </label>
              <button type="button" className={styles.settingsPrimary} disabled={busy} onClick={onRefresh}>
                Refresh
              </button>
            </div>
            <div className={styles.enquiryList}>
              {rows.length === 0 ? (
                <p>No enquiries.</p>
              ) : (
                rows.map((r) => (
                  <button
                    key={r.id}
                    type="button"
                    className={`${styles.enquiryItem} ${selected?.id === r.id ? styles.enquiryItemActive : ''}`}
                    onClick={() => {
                      setSelected(r);
                      setReply('');
                    }}
                  >
                    <strong>{r.customer_phone}</strong>
                    <span className={styles.statusPill}>{r.status}</span>
                    <div className={styles.enquiryItemMeta}>
                      {typeof r.ai_data?.short_enquiry_tag === 'string'
                        ? r.ai_data.short_enquiry_tag
                        : '—'}
                    </div>
                  </button>
                ))
              )}
            </div>
          </section>
        </aside>

        <section className={styles.panel} style={{ minHeight: 420 }}>
          {!selected ? (
            <p style={{ color: 'var(--portal-muted)' }}>Select an enquiry to view AI summary and reply.</p>
          ) : (
            <>
              <div className={styles.enquiryDetailHead}>
                <div>
                  <h2 style={{ margin: 0 }}>
                    {selected.customer_name || selected.customer_phone}
                  </h2>
                  <p style={{ margin: '4px 0 0', color: 'var(--portal-muted)', fontSize: 13 }}>
                    #{selected.id} · {selected.created_at || ''}
                  </p>
                </div>
                <div className={styles.actionRow}>
                  <button type="button" className={styles.tabBtn} disabled={busy} onClick={onAnalyze}>
                    Re-analyze
                  </button>
                  <button
                    type="button"
                    className={styles.settingsPrimary}
                    disabled={busy}
                    onClick={() => onStatus('Resolved')}
                  >
                    Resolve
                  </button>
                </div>
              </div>

              <div style={{ marginTop: 16 }}>
                <p className={styles.enquiryLabel}>AI summary</p>
                <p style={{ fontWeight: 600, margin: '6px 0 10px' }}>{intent || 'No AI data yet'}</p>
                {category ? (
                  <span className={`${styles.statusPill} ${styles.statusOk}`}>{category}</span>
                ) : null}
              </div>

              <div style={{ marginTop: 18 }}>
                <p className={styles.enquiryLabel}>Transcript</p>
                <pre className={styles.enquiryTranscript}>{selected.enquiry_details}</pre>
              </div>

              <div style={{ marginTop: 18 }}>
                <div className={styles.actionRow} style={{ marginBottom: 8 }}>
                  <button type="button" className={styles.settingsPrimary} disabled={busy} onClick={onSuggest}>
                    Suggest reply
                  </button>
                  <button type="button" className={styles.tabBtn} disabled={busy || !reply} onClick={onNotify}>
                    Send WhatsApp
                  </button>
                </div>
                <textarea
                  className={styles.enquiryTextarea}
                  rows={6}
                  value={reply}
                  onChange={(e) => setReply(e.target.value)}
                  placeholder="Reply draft…"
                />
              </div>
            </>
          )}
        </section>
      </div>
    </>
  );
}
