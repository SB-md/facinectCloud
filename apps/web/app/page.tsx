'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import styles from './landing.module.css';

const LEFT_MENU = [
  { label: 'Features', href: '#features' },
  { label: 'Why us', href: '#why-us' },
  { label: 'Pricing', href: '#pricing' },
] as const;

export default function LandingPage() {
  const [signedIn, setSignedIn] = useState(false);

  useEffect(() => {
    setSignedIn(Boolean(sessionStorage.getItem('facinect_access_token')));
  }, []);

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <nav className={styles.menu} aria-label="Primary">
          {LEFT_MENU.map((item) => (
            <a key={item.href} href={item.href} className={styles.menuLink}>
              {item.label}
            </a>
          ))}
        </nav>

        <Link href="/" className={styles.brand} aria-label="Facinect Multisports Management">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            className={styles.brandMark}
            src="/facinectlogo.png"
            alt=""
            width={96}
            height={96}
          />
          <span className={styles.brandName}>Facinect Multisports Management</span>
        </Link>

        <nav className={styles.nav}>
          {signedIn ? (
            <Link href="/home" className={styles.loginBtn}>
              Open account
            </Link>
          ) : (
            <Link href="/login" className={styles.loginBtn}>
              Partner Login
            </Link>
          )}
        </nav>
      </header>
    </div>
  );
}
