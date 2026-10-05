'use client';

import { ReactNode, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  FacilityMembership,
  SessionUser,
  canAccessPage,
  clearSession,
  fetchSession,
  findFacilityBySlug,
} from './auth';
import styles from '../app/login/login.module.css';

type Props = {
  children: (ctx: {
    user: SessionUser;
    facility: FacilityMembership | null;
    claims: Record<string, unknown> | null;
  }) => ReactNode;
  facilitySlug?: string;
  pageKey?: string;
};

export default function AuthGuard({ children, facilitySlug, pageKey }: Props) {
  const router = useRouter();
  const [user, setUser] = useState<SessionUser | null>(null);
  const [facility, setFacility] = useState<FacilityMembership | null>(null);
  const [claims, setClaims] = useState<Record<string, unknown> | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const data = await fetchSession();
      if (cancelled) return;
      if (!data || data.error || !data.user) {
        clearSession();
        router.replace('/login?error=not_logged_in');
        return;
      }
      if ((data.user.facilities || []).length === 0) {
        setError('no_facilities_assigned');
        setLoading(false);
        return;
      }

      let current: FacilityMembership | null = null;
      if (facilitySlug) {
        current = findFacilityBySlug(data.user, facilitySlug);
        if (!current) {
          setError('facility_denied');
          setLoading(false);
          return;
        }
      }

      const accessUser: SessionUser = current
        ? { ...data.user, role: current.role, page_access: current.page_access }
        : data.user;

      if (pageKey && !canAccessPage(accessUser, pageKey)) {
        setError('page_denied');
        setLoading(false);
        return;
      }

      setUser(data.user);
      setFacility(current);
      setClaims((data.claims as Record<string, unknown>) || null);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
    };
  }, [router, facilitySlug, pageKey]);

  if (loading) {
    return (
      <main className={styles.wrap}>
        <div className={styles.box}>
          <p className={styles.msg}>Verifying session…</p>
        </div>
      </main>
    );
  }

  if (error) {
    return (
      <main className={styles.wrap}>
        <div className={styles.box}>
          <h1>Access denied</h1>
          <p className={`${styles.msg} ${styles.err}`}>
            {error === 'no_facilities_assigned' && 'No facilities assigned to this account.'}
            {error === 'facility_denied' && 'You do not have access to this facility.'}
            {error === 'page_denied' && 'You do not have permission to open this page.'}
            {!['no_facilities_assigned', 'facility_denied', 'page_denied'].includes(error) && error}
          </p>
          <button type="button" className={styles.btn} onClick={() => router.replace('/login')}>
            Back to login
          </button>
        </div>
      </main>
    );
  }

  if (!user) return null;
  return <>{children({ user, facility, claims })}</>;
}
