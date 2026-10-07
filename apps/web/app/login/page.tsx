'use client';

import { FormEvent, useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import styles from './login.module.css';
import {
  TokenResponse,
  getAccessToken,
  postLoginPath,
  storeSession,
} from '../../lib/auth';

const AUTH_ERRORS: Record<string, string> = {
  oauth_denied: 'Google sign-in was cancelled.',
  oauth_failed: 'Google sign-in failed. Please try again.',
  oauth_bad_secret: 'Server Google client secret is missing or wrong.',
  oauth_redirect_mismatch:
    'Redirect URI mismatch. Add exactly:\nhttp://localhost:8080/v1/auth/google/callback',
  oauth_expired: 'Sign-in session expired. Please try again.',
  invalid_state: 'Security check failed. Please try again.',
  email_not_verified: 'Your Google email is not verified.',
  oauth_config: 'Google login is not configured.',
  invalid_credentials: 'Incorrect email or password.',
  unauthorized: 'This account is not allowed to sign in.',
  invalid_handoff: 'Google sign-in handoff expired. Please try again.',
  no_facilities_assigned: 'No facilities assigned to this account. Contact support.',
  not_logged_in: 'Please sign in to continue.',
};

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [msg, setMsg] = useState('');
  const [msgType, setMsgType] = useState<'ok' | 'error' | ''>('');
  const [busy, setBusy] = useState(false);
  const [showNotReg, setShowNotReg] = useState(false);
  const [showPwErr, setShowPwErr] = useState(false);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    setReady(true);
    const existing = getAccessToken();
    if (existing && !window.location.search) {
      router.replace('/home');
      return;
    }

    const params = new URLSearchParams(window.location.search);
    const error = params.get('error');
    if (error) {
      const message = AUTH_ERRORS[error] || `Login failed: ${error}`;
      setMsg(message);
      setMsgType('error');
      window.history.replaceState({}, '', '/login');
      return;
    }
    const handoff = params.get('handoff');
    if (params.get('google') === '1' && handoff) {
      (async () => {
        setMsg('Completing Google sign-in…');
        try {
          const res = await fetch('/v1/auth/google/handoff', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
            body: JSON.stringify({ handoff }),
          });
          const data = (await res.json()) as TokenResponse;
          if (!res.ok) throw new Error(data.error || 'oauth_failed');
          storeSession(data);
          window.history.replaceState({}, '', '/login');
          router.replace(postLoginPath(data));
        } catch (e) {
          const code = e instanceof Error ? e.message : 'oauth_failed';
          setMsg(AUTH_ERRORS[code] || code);
          setMsgType('error');
          window.history.replaceState({}, '', '/login');
        }
      })();
    }
  }, [router]);

  async function onNext() {
    const value = email.trim();
    if (!value) return;
    setBusy(true);
    setMsg('Checking email…');
    setMsgType('');
    try {
      const res = await fetch('/v1/auth/check-email', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify({ email: value }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'check_failed');
      if (!data.exists) {
        setShowNotReg(true);
        setMsg('Email not found for password login. Use Google, or create the user first.');
        setMsgType('error');
        return;
      }
      setShowPassword(true);
      setMsg('');
    } catch (e) {
      setMsg(e instanceof Error ? e.message : 'check_failed');
      setMsgType('error');
    } finally {
      setBusy(false);
    }
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setMsg('Signing in…');
    setMsgType('');
    try {
      const res = await fetch('/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify({ email: email.trim(), password }),
      });
      const data = (await res.json()) as TokenResponse;
      if (!res.ok) {
        if (data.error === 'invalid_credentials') {
          setShowPwErr(true);
        }
        setMsg(AUTH_ERRORS[data.error || ''] || data.error || 'invalid_credentials');
        setMsgType('error');
        return;
      }
      storeSession(data);
      router.replace(postLoginPath(data));
    } catch (err) {
      setMsg(err instanceof Error ? err.message : 'login_failed');
      setMsgType('error');
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className={`${styles.wrap} ${ready ? styles.ready : ''}`}>
      <div className={styles.atmosphere} aria-hidden>
        <div className={styles.glowA} />
        <div className={styles.glowB} />
        <div className={styles.grid} />
      </div>

      <Link href="/" className={styles.backLink}>
        ← Back to home
      </Link>

      {showNotReg && (
        <div className={styles.modal} role="dialog">
          <div className={styles.modalCard}>
            <i className="fa-solid fa-circle-exclamation" style={{ color: '#d9534f', fontSize: 48 }} />
            <h3>Not Registered</h3>
            <p>This email is not registered. Use Google sign-in or contact support.</p>
            <button type="button" className={styles.modalClose} onClick={() => setShowNotReg(false)}>
              OK
            </button>
          </div>
        </div>
      )}
      {showPwErr && (
        <div className={styles.modal} role="dialog">
          <div className={styles.modalCard}>
            <i className="fa-solid fa-key" style={{ color: '#f0ad4e', fontSize: 48 }} />
            <h3>Incorrect Password</h3>
            <p>The password you entered is incorrect. Please try again.</p>
            <button
              type="button"
              className={`${styles.modalClose} ${styles.modalWarn}`}
              onClick={() => setShowPwErr(false)}
            >
              Try Again
            </button>
          </div>
        </div>
      )}

      <div className={styles.cardShell}>
        <div className={styles.box}>
          <div className={styles.logoWrap}>
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              className={styles.logo}
              src="/facinectlogo.png"
              alt="Facinect Multisports Management"
              width={112}
              height={112}
            />
          </div>
          <p className={styles.eyebrow}>Partner access</p>
          <h1>Partner Login</h1>
          <p className={styles.sub}>Sign in to Facinect Multisports Management</p>

          <form onSubmit={onSubmit} className={styles.form}>
            <div className={styles.group}>
              <label htmlFor="email">Email Address</label>
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="Enter your email"
                required
                autoComplete="email"
              />
              {showPassword && <i className={`fa-solid fa-circle-check ${styles.check}`} />}
            </div>
            {!showPassword ? (
              <button type="button" className={styles.btn} onClick={onNext} disabled={busy}>
                {busy ? 'Please wait…' : 'Continue'}
              </button>
            ) : (
              <div className={styles.passwordStep}>
                <div className={styles.group}>
                  <label htmlFor="password">Password</label>
                  <input
                    id="password"
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="••••••••"
                    autoComplete="current-password"
                  />
                </div>
                <button type="submit" className={styles.btn} disabled={busy}>
                  {busy ? 'Signing in…' : 'Sign In'}
                </button>
              </div>
            )}
          </form>

          <div className={styles.divider}>
            <span>or</span>
          </div>

          <a href="/v1/auth/google/start" className={`${styles.btn} ${styles.google}`}>
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src="https://www.gstatic.com/firebasejs/ui/2.0.0/images/auth/google.svg"
              alt=""
              width={20}
              height={20}
            />
            Continue with Google
          </a>

          <p className={`${styles.msg} ${msgType === 'ok' ? styles.ok : ''} ${msgType === 'error' ? styles.err : ''}`}>
            {msg}
          </p>
        </div>
      </div>
    </main>
  );
}
