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
    <FacilityShell
      pageKey="dashboard"
      title="Overview"
      description="This month at a glance."
    >
      <Suspense fallback={<p className={styles.dashHint}>Loading overview…</p>}>
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
    return <p className={styles.dashHint}>Loading overview…</p>;
  }

  if (!facilityId) {
    return <p className={styles.settingsErr}>Facility not found for this account.</p>;
  }

  const viewBase = `/facility/${encodeURIComponent(slug)}/view-bookings`;
  const fromDash = `from=dashboard&month=${month}`;

  return (
    <div className={styles.dash}>
      <header className={styles.dashHero}>
        <div className={styles.dashHeroCopy}>
          <p className={styles.dashKicker}>Facility pulse</p>
          <h2 className={styles.dashHeroTitle}>{facility?.facilityName || slug}</h2>
        </div>
        <div className={styles.dashMonthBar} role="group" aria-label="Select month">
          <button type="button" className={styles.dashMonthBtn} aria-label="Previous month" onClick={() => goMonth(-1)}>
            <i className="fa-solid fa-chevron-left" />
          </button>
          <div className={styles.dashMonthLabel}>
            <span>{monthLabel}</span>
            {busy ? <em>Updating…</em> : null}
          </div>
          <button type="button" className={styles.dashMonthBtn} aria-label="Next month" onClick={() => goMonth(1)}>
            <i className="fa-solid fa-chevron-right" />
          </button>
        </div>
      </header>

      {err && <p className={styles.settingsErr}>{err}</p>}

      <div className={styles.dashMetrics}>
        <article className={`${styles.dashMetric} ${styles.dashMetricOk}`}>
          <div className={styles.dashMetricTop}>
            <span className={styles.dashMetricIcon} aria-hidden>
              <i className="fa-solid fa-calendar-check" />
            </span>
            <span className={styles.dashMetricShare}>{confPct}%</span>
          </div>
          <p className={styles.dashMetricLabel}>Confirmed</p>
          <p className={styles.dashMetricValue}>{confSum.count}</p>
          <Sparkline
            values={confSum.weekly}
            stroke="var(--portal-teal)"
            fill="rgba(21, 154, 127, 0.14)"
          />
          <Link href={`${viewBase}?${fromDash}&status=confirmed`} className={styles.dashMetricCta}>
            View details
            <i className="fa-solid fa-arrow-right" />
          </Link>
        </article>

        <article className={`${styles.dashMetric} ${styles.dashMetricBad}`}>
          <div className={styles.dashMetricTop}>
            <span className={styles.dashMetricIcon} aria-hidden>
              <i className="fa-solid fa-calendar-xmark" />
            </span>
            <span className={styles.dashMetricShare}>{canPct}%</span>
          </div>
          <p className={styles.dashMetricLabel}>Cancelled</p>
          <p className={styles.dashMetricValue}>{canSum.count}</p>
          <Sparkline
            values={canSum.weekly}
            stroke="#c45c5c"
            fill="rgba(196, 92, 92, 0.12)"
          />
          <Link href={`${viewBase}?${fromDash}&status=cancelled`} className={styles.dashMetricCta}>
            View details
            <i className="fa-solid fa-arrow-right" />
          </Link>
        </article>
      </div>

      <div className={styles.dashSecondary}>
        <article className={styles.dashSoft}>
          <div className={styles.dashSoftHead}>
            <i className="fa-solid fa-user-graduate" />
            <span>Students</span>
          </div>
          <p className={styles.dashSoftValue}>—</p>
          <p className={styles.dashSoftNote}>Coming soon</p>
          <Link href={`/facility/${encodeURIComponent(slug)}/students`} className={styles.dashSoftLink}>
            Open
          </Link>
        </article>
        <article className={styles.dashSoft}>
          <div className={styles.dashSoftHead}>
            <i className="fa-solid fa-users" />
            <span>Members</span>
          </div>
          <p className={styles.dashSoftValue}>—</p>
          <p className={styles.dashSoftNote}>Coming soon</p>
          <Link href={`/facility/${encodeURIComponent(slug)}/members`} className={styles.dashSoftLink}>
            Open
          </Link>
        </article>
        <article className={styles.dashSoft}>
          <div className={styles.dashSoftHead}>
            <i className="fa-solid fa-table-cells" />
            <span>Slots</span>
          </div>
          <p className={styles.dashSoftValue}>Setup</p>
          <p className={styles.dashSoftNote}>Courts & availability</p>
          <Link href={`/facility/${encodeURIComponent(slug)}/slots-setup`} className={styles.dashSoftLink}>
            Open
          </Link>
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
  const w = 240;
  const h = 56;
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
    <svg className={styles.dashSpark} viewBox={`0 0 ${w} ${h}`} role="img" aria-label="Weekly trend">
      <polygon points={area} fill={fill} />
      <polyline
        points={line}
        fill="none"
        stroke={stroke}
        strokeWidth="2.25"
        strokeLinejoin="round"
        strokeLinecap="round"
      />
    </svg>
  );
}
