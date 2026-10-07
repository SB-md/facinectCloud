'use client';

import { Suspense, useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter, useSearchParams } from 'next/navigation';
import FacilityShell from './FacilityShell';
import { FacilityMembership, fetchSession, findFacilityBySlug, getCachedUser } from '../../../lib/auth';
import {
  BookingRow,
  listBookings,
  monthKeyNow,
  shiftMonth,
  summarizeMonthBookings,
} from '../../../lib/booking';
import styles from './portal.module.css';

export default function FacilityDashboardPage() {
  return (
    <FacilityShell pageKey="dashboard" title="Dashboard" description="">
      <Suspense fallback={<p className={styles.dashHint}>Loading…</p>}>
        <DashboardBody />
      </Suspense>
    </FacilityShell>
  );
}

function DashboardBody() {
  const params = useParams<{ slug: string }>();
  const router = useRouter();
  const search = useSearchParams();
  const month = search.get('month') || monthKeyNow();

  const [facility, setFacility] = useState<FacilityMembership | null>(null);
  const [confirmed, setConfirmed] = useState<BookingRow[]>([]);
  const [cancelled, setCancelled] = useState<BookingRow[]>([]);
  const [err, setErr] = useState('');
  const [loaded, setLoaded] = useState(false);
  const [busy, setBusy] = useState(false);

  const facilityId = facility?.facilityId;
  const slug = params.slug;

  const load = useCallback(async (fid: number, monthKey: string) => {
    const [y, m] = monthKey.split('-').map(Number);
    const [c, x] = await Promise.all([
      listBookings(fid, { year: y, month: m, status: 'confirmed' }),
      listBookings(fid, { year: y, month: m, status: 'cancelled' }),
    ]);
    setConfirmed(c.bookings || []);
    setCancelled(x.bookings || []);
  }, []);

  useEffect(() => {
    (async () => {
      setErr('');
      let user = getCachedUser();
      if (!user) {
        const session = await fetchSession();
        user = session?.user || null;
      }
      const f = findFacilityBySlug(user, slug);
      setFacility(f);
      if (f?.facilityId) {
        setBusy(true);
        try {
          await load(f.facilityId, month);
        } catch (e) {
          setErr(e instanceof Error ? e.message : 'load_failed');
        } finally {
          setBusy(false);
        }
      }
      setLoaded(true);
    })();
  }, [slug, month, load]);

  const confSum = useMemo(() => summarizeMonthBookings(confirmed), [confirmed]);
  const canSum = useMemo(() => summarizeMonthBookings(cancelled), [cancelled]);
  const total = confSum.count + canSum.count;
  const confPct = total > 0 ? Math.round((confSum.count / total) * 1000) / 10 : 0;
  const canPct = total > 0 ? Math.round((canSum.count / total) * 1000) / 10 : 0;

  const monthLabel = useMemo(() => {
    const [y, m] = month.split('-').map(Number);
    return new Date(y, m - 1, 1).toLocaleString(undefined, { month: 'long', year: 'numeric' });
  }, [month]);

  function goMonth(delta: number) {
    router.replace(`/facility/${encodeURIComponent(slug)}?month=${shiftMonth(month, delta)}`);
  }

  if (!loaded) {
    return <p className={styles.dashHint}>Loading…</p>;
  }

  if (!facilityId) {
    return <p className={styles.settingsErr}>Facility not found for this account.</p>;
  }

  const viewBase = `/facility/${encodeURIComponent(slug)}/view-bookings`;
  const fromDash = `from=dashboard&month=${month}`;
  const initial = (facility?.facilityName || slug).trim().charAt(0).toUpperCase() || 'F';

  return (
    <div className={styles.dash}>
      <header className={styles.dashBar}>
        <div className={styles.dashBrand}>
          <span className={styles.dashLogo} aria-hidden>
            {initial}
          </span>
          <strong className={styles.dashFacName}>{facility?.facilityName || slug}</strong>
        </div>
        {busy ? <span className={styles.dashBusy}>Updating…</span> : null}
      </header>

      <div className={styles.dashNav} role="group" aria-label="Select month">
        <button type="button" className={styles.dashNavBtn} aria-label="Previous month" onClick={() => goMonth(-1)}>
          <i className="fa-solid fa-chevron-left" />
        </button>
        <div className={styles.dashNavTitle}>{monthLabel}</div>
        <button type="button" className={styles.dashNavBtn} aria-label="Next month" onClick={() => goMonth(1)}>
          <i className="fa-solid fa-chevron-right" />
        </button>
      </div>

      {err && <p className={styles.settingsErr}>{err}</p>}

      <div className={styles.dashGrid}>
        <article className={`${styles.statCard} ${styles.statOk}`}>
          <div className={styles.statHead}>
            <span className={styles.statLabel}>Confirmed bookings</span>
            <span className={`${styles.statBadge} ${styles.badgeOk}`}>{confPct}%</span>
          </div>
          <div className={styles.statValue}>{confSum.count}</div>
          <Sparkline values={confSum.weekly} stroke="rgba(21, 154, 127, 1)" fill="rgba(21, 154, 127, 0.16)" />
          <div className={styles.statFoot}>
            <Link href={`${viewBase}?${fromDash}&status=confirmed`} className={styles.statDetails}>
              View details <i className="fa-solid fa-arrow-right" />
            </Link>
          </div>
        </article>

        <article className={`${styles.statCard} ${styles.statBad}`}>
          <div className={styles.statHead}>
            <span className={styles.statLabel}>Cancellations</span>
            <span className={`${styles.statBadge} ${styles.badgeBad}`}>{canPct}%</span>
          </div>
          <div className={styles.statValue}>{canSum.count}</div>
          <Sparkline values={canSum.weekly} stroke="rgba(196, 92, 92, 1)" fill="rgba(196, 92, 92, 0.12)" />
          <div className={styles.statFoot}>
            <Link href={`${viewBase}?${fromDash}&status=cancelled`} className={`${styles.statDetails} ${styles.statDetailsBad}`}>
              View details <i className="fa-solid fa-arrow-right" />
            </Link>
          </div>
        </article>

        <article className={`${styles.statCard} ${styles.statWarn}`}>
          <div className={styles.statHead}>
            <span className={styles.statLabel}>Students</span>
            <span className={`${styles.statBadge} ${styles.badgeWarn}`}>Soon</span>
          </div>
          <div className={styles.statValue}>—</div>
          <div className={styles.statPlaceholder}>Students service not wired yet</div>
          <div className={styles.statFoot}>
            <Link href={`/facility/${encodeURIComponent(slug)}/students`} className={`${styles.statDetails} ${styles.statDetailsWarn}`}>
              Analytics <i className="fa-solid fa-arrow-right" />
            </Link>
          </div>
        </article>

        <article className={`${styles.statCard} ${styles.statInfo}`}>
          <div className={styles.statHead}>
            <span className={styles.statLabel}>Registered members</span>
            <span className={`${styles.statBadge} ${styles.badgeInfo}`}>Soon</span>
          </div>
          <div className={styles.statValue}>—</div>
          <div className={styles.statPlaceholder}>Members service not wired yet</div>
          <div className={styles.statFoot}>
            <Link href={`/facility/${encodeURIComponent(slug)}/members`} className={`${styles.statDetails} ${styles.statDetailsInfo}`}>
              Manage pool <i className="fa-solid fa-arrow-right" />
            </Link>
          </div>
        </article>
      </div>
    </div>
  );
}

function Sparkline({
  values,
  stroke,
  fill,
}: {
  values: number[];
  stroke: string;
  fill: string;
}) {
  const w = 220;
  const h = 64;
  const pad = 4;
  const max = Math.max(1, ...values);
  const step = (w - pad * 2) / Math.max(1, values.length - 1);
  const points = values.map((v, i) => {
    const x = pad + i * step;
    const y = h - pad - (v / max) * (h - pad * 2);
    return `${x},${y}`;
  });
  const line = points.join(' ');
  const area = `${pad},${h - pad} ${line} ${w - pad},${h - pad}`;

  return (
    <svg className={styles.statChart} viewBox={`0 0 ${w} ${h}`} role="img" aria-label="Weekly trend">
      <polygon points={area} fill={fill} />
      <polyline
        points={line}
        fill="none"
        stroke={stroke}
        strokeWidth="2.5"
        strokeLinejoin="round"
        strokeLinecap="round"
      />
    </svg>
  );
}
