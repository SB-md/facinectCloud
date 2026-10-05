'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import styles from '../login/login.module.css';
import {
  MeResponse,
  SessionUser,
  fetchSession,
  logout,
  postLoginPath,
} from '../../lib/auth';

export default function HomePage() {
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [user, setUser] = useState<SessionUser | null>(null);
  const [claims, setClaims] = useState<MeResponse['claims'] | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    (async () => {
      const data = await fetchSession();
      if (!data || data.error || !data.user) {
        setError(data?.error || 'not_logged_in');
        setTimeout(() => router.replace('/login'), 800);
        setLoading(false);
        return;
      }
      // Facinect-style: bounce partners into their facility home.
      if ((data.user.facilities || []).length > 0) {
        router.replace(postLoginPath(data));
        return;
      }
      setUser(data.user);
      setClaims(data.claims || null);
      setLoading(false);
    })();
  }, [router]);

  async function onLogout() {
    await logout();
    router.replace('/login');
  }

  return (
    <main className={styles.wrap}>
      <div className={styles.box}>
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img className={styles.logo} src="/facinectlogo.png" alt="Facinect Multisports Management" width={96} height={96} />
        <h1>Signed in</h1>
        {loading && <p className={styles.msg}>Verifying JWT via /v1/auth/me…</p>}
        {error && <p className={`${styles.msg} ${styles.err}`}>{error}</p>}
        {!loading && user && (
          <>
            <p className={`${styles.msg} ${styles.ok}`}>
              Welcome, {user.full_name || user.email || 'user'}
            </p>
            <div className={styles.sessionCard}>
              <div>
                <span>Email</span>
                <strong>{user.email || '—'}</strong>
              </div>
              <div>
                <span>User ID</span>
                <strong>{user.id ?? '—'}</strong>
              </div>
              <div>
                <span>Role</span>
                <strong>{user.role || '—'}</strong>
              </div>
              <div>
                <span>Status</span>
                <strong>{user.status || '—'}</strong>
              </div>
              <div>
                <span>JWT sub</span>
                <strong>{String(claims?.sub || '—')}</strong>
              </div>
              <div>
                <span>Expires</span>
                <strong>
                  {claims?.exp ? new Date(Number(claims.exp) * 1000).toLocaleString() : '—'}
                </strong>
              </div>
            </div>
            <p className={styles.hint}>No facilities assigned yet — complete onboarding next.</p>
            <button type="button" className={styles.btn} onClick={onLogout}>
              Sign out
            </button>
          </>
        )}
      </div>
    </main>
  );
}
