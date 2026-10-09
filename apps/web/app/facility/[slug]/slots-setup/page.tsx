'use client';

import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { useParams } from 'next/navigation';
import FacilityShell from '../FacilityShell';
import { FacilityMembership, findFacilityBySlug, getCachedUser, fetchSession } from '../../../../lib/auth';
import {
  BookingCourt,
  BookingSlot,
  blockSlot,
  createBooking,
  createCourt,
  formatTimeRange,
  generateSlots,
  groupSlotsByCourt,
  listCourts,
  listSlots,
  todayISO,
  unblockSlot,
} from '../../../../lib/booking';
import styles from '../portal.module.css';

export default function SlotsSetupPage() {
  return (
    <FacilityShell
      pageKey="bookings"
      title="Slots Setup"
      description="Configure court slots, book customers, and block availability."
    >
      <SlotsSetupBody />
    </FacilityShell>
  );
}

function SlotsSetupBody() {
  const params = useParams<{ slug: string }>();
  const [facility, setFacility] = useState<FacilityMembership | null>(null);
  const [date, setDate] = useState(todayISO());
  const [courts, setCourts] = useState<BookingCourt[]>([]);
  const [slots, setSlots] = useState<BookingSlot[]>([]);
  const [selected, setSelected] = useState<number[]>([]);
  const [tab, setTab] = useState<'book' | 'block'>('book');
  const [notes, setNotes] = useState('');
  const [courtName, setCourtName] = useState('');
  const [customerName, setCustomerName] = useState('');
  const [customerPhone, setCustomerPhone] = useState('');
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState('');
  const [err, setErr] = useState('');
  const [loaded, setLoaded] = useState(false);

  const facilityId = facility?.facilityId;

  const load = useCallback(async (fid: number, day: string) => {
    const [c, s] = await Promise.all([listCourts(fid), listSlots(fid, day)]);
    setCourts(c.courts || []);
    setSlots(s.slots || []);
    setSelected([]);
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
          await load(f.facilityId, date);
        } catch (e) {
          setErr(e instanceof Error ? e.message : 'load_failed');
        }
      }
      setLoaded(true);
    })();
  }, [params.slug, date, load]);

  const grouped = useMemo(() => groupSlotsByCourt(slots), [slots]);

  const counts = useMemo(() => {
    let available = 0;
    let booked = 0;
    let blocked = 0;
    for (const s of slots) {
      if (s.status === 'available') available += 1;
      else if (s.status === 'booked') booked += 1;
      else if (s.status === 'blocked') blocked += 1;
    }
    return { available, booked, blocked };
  }, [slots]);

  function toggleSlot(id: number, status: string) {
    if (tab === 'book' && status !== 'available') return;
    if (tab === 'block' && status !== 'available' && status !== 'blocked') return;
    setSelected((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));
  }

  async function refresh() {
    if (!facilityId) return;
    await load(facilityId, date);
  }

  async function onCreateCourt(e: FormEvent) {
    e.preventDefault();
    if (!facilityId || !courtName.trim()) return;
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      await createCourt(facilityId, courtName.trim());
      setCourtName('');
      setMsg('Court created.');
      await refresh();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'create_court_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onGenerate() {
    if (!facilityId) return;
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      const res = await generateSlots(facilityId, {
        date,
        open_hour: 6,
        close_hour: 22,
        slot_minutes: 60,
      });
      setMsg(`Generated ${res.created} slot(s).`);
      await refresh();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'generate_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onBook() {
    if (!facilityId || selected.length === 0) return;
    if (!customerName.trim() || !customerPhone.trim()) {
      setErr('Customer name and phone are required to book.');
      return;
    }
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      let ok = 0;
      for (const slotId of selected) {
        const slot = slots.find((s) => s.id === slotId);
        if (!slot || slot.status !== 'available') continue;
        await createBooking(facilityId, {
          slot_id: slotId,
          customer_name: customerName.trim(),
          customer_phone: customerPhone.trim(),
          notes: notes.trim() || undefined,
        });
        ok += 1;
      }
      setMsg(`Booked ${ok} slot(s).`);
      setSelected([]);
      setNotes('');
      await refresh();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'book_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onBlock() {
    if (selected.length === 0) return;
    if (!notes.trim()) {
      setErr('Notes are required to block slots.');
      return;
    }
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      let ok = 0;
      for (const slotId of selected) {
        const slot = slots.find((s) => s.id === slotId);
        if (!slot || slot.status !== 'available') continue;
        await blockSlot(slotId, notes.trim());
        ok += 1;
      }
      setMsg(`Blocked ${ok} slot(s).`);
      setSelected([]);
      setNotes('');
      await refresh();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'block_failed');
    } finally {
      setBusy(false);
    }
  }

  async function onUnblock() {
    if (selected.length === 0) return;
    setBusy(true);
    setErr('');
    setMsg('');
    try {
      let ok = 0;
      for (const slotId of selected) {
        const slot = slots.find((s) => s.id === slotId);
        if (!slot || slot.status !== 'blocked') continue;
        await unblockSlot(slotId);
        ok += 1;
      }
      setMsg(`Unblocked ${ok} slot(s).`);
      setSelected([]);
      await refresh();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'unblock_failed');
    } finally {
      setBusy(false);
    }
  }

  if (!loaded) {
    return <p>Loading slots…</p>;
  }

  if (!facilityId) {
    return <p className={styles.settingsErr}>Facility not found for this account.</p>;
  }

  return (
    <>
      <div className={styles.detailGrid}>
        <div className={styles.detailCard}>
          <span>Available</span>
          <strong>{counts.available}</strong>
        </div>
        <div className={styles.detailCard}>
          <span>Booked</span>
          <strong>{counts.booked}</strong>
        </div>
        <div className={styles.detailCard}>
          <span>Blocked</span>
          <strong>{counts.blocked}</strong>
        </div>
        <div className={styles.detailCard}>
          <span>Courts</span>
          <strong>{courts.length}</strong>
        </div>
      </div>

      <div className={styles.tabRow} role="tablist">
        <button
          type="button"
          className={`${styles.tabBtn} ${tab === 'book' ? styles.tabBtnActive : ''}`}
          onClick={() => {
            setTab('book');
            setSelected([]);
          }}
        >
          Book
        </button>
        <button
          type="button"
          className={`${styles.tabBtn} ${tab === 'block' ? styles.tabBtnActive : ''}`}
          onClick={() => {
            setTab('block');
            setSelected([]);
          }}
        >
          Block slots
        </button>
      </div>

      <section className={styles.panel} style={{ marginBottom: 16 }}>
        <h2>System tools</h2>
        <p style={{ marginBottom: 14 }}>Create courts and generate the day&apos;s time slots.</p>
        <form onSubmit={onCreateCourt} className={styles.inlineForm}>
          <input
            type="text"
            placeholder="New court name"
            value={courtName}
            onChange={(e) => setCourtName(e.target.value)}
          />
          <button type="submit" className={styles.settingsPrimary} disabled={busy || !courtName.trim()}>
            Add court
          </button>
          <button type="button" className={styles.settingsPrimary} disabled={busy} onClick={onGenerate}>
            Generate slots
          </button>
        </form>
      </section>

      <section className={styles.panel} style={{ marginBottom: 16 }}>
        <h2>Manage slots</h2>
        <div className={styles.inlineForm} style={{ marginBottom: 14 }}>
          <label className={styles.fieldGrow}>
            Date
            <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </label>
          <button type="button" className={styles.settingsPrimary} disabled={busy} onClick={() => refresh()}>
            Load slots
          </button>
        </div>

        {grouped.length === 0 ? (
          <p>No slots for this date. Add a court and generate slots.</p>
        ) : (
          <div className={styles.slotBoard}>
            {grouped.map((group) => (
              <div key={group.courtName} className={styles.slotCourt}>
                <h3>{group.courtName}</h3>
                <div className={styles.slotGrid}>
                  {group.slots.map((slot) => {
                    const active = selected.includes(slot.id);
                    return (
                      <button
                        key={slot.id}
                        type="button"
                        className={`${styles.slotCell} ${styles[`slot_${slot.status}`] || ''} ${
                          active ? styles.slotSelected : ''
                        }`}
                        onClick={() => toggleSlot(slot.id, slot.status)}
                        title={slot.notes || slot.status}
                      >
                        <strong>{formatTimeRange(slot.start_time, slot.end_time)}</strong>
                        <span>{slot.status}</span>
                      </button>
                    );
                  })}
                </div>
              </div>
            ))}
          </div>
        )}

        <div className={styles.slotLegend}>
          <span>
            <i className={styles.legAvailable} /> Available
          </span>
          <span>
            <i className={styles.legBooked} /> Booked
          </span>
          <span>
            <i className={styles.legBlocked} /> Blocked
          </span>
        </div>

        {tab === 'book' ? (
          <div className={styles.settingsForm} style={{ marginTop: 16 }}>
            <label>
              Customer name
              <input value={customerName} onChange={(e) => setCustomerName(e.target.value)} />
            </label>
            <label>
              WhatsApp / phone
              <input value={customerPhone} onChange={(e) => setCustomerPhone(e.target.value)} />
            </label>
            <label>
              Notes
              <input value={notes} onChange={(e) => setNotes(e.target.value)} />
            </label>
            <div className={styles.actionRow}>
              <button
                type="button"
                className={styles.settingsPrimary}
                disabled={busy || selected.length === 0}
                onClick={onBook}
              >
                Book selected ({selected.length})
              </button>
              <button type="button" className={styles.settingsDanger} disabled={!selected.length} onClick={() => setSelected([])}>
                Clear
              </button>
            </div>
          </div>
        ) : (
          <div className={styles.settingsForm} style={{ marginTop: 16 }}>
            <label>
              Block notes <span style={{ color: '#b91c1c' }}>*</span>
              <input value={notes} onChange={(e) => setNotes(e.target.value)} placeholder="Reason for blocking" />
            </label>
            <div className={styles.actionRow}>
              <button
                type="button"
                className={styles.settingsPrimary}
                disabled={busy || selected.length === 0}
                onClick={onBlock}
              >
                Block selected
              </button>
              <button
                type="button"
                className={styles.settingsPrimary}
                disabled={busy || selected.length === 0}
                onClick={onUnblock}
              >
                Unblock selected
              </button>
              <button type="button" className={styles.settingsDanger} disabled={!selected.length} onClick={() => setSelected([])}>
                Clear
              </button>
            </div>
          </div>
        )}

        {err && <p className={styles.settingsErr}>{err}</p>}
        {msg && <p className={styles.settingsOk}>{msg}</p>}
      </section>
    </>
  );
}
