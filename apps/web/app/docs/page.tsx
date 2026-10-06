'use client';

import { Suspense } from 'react';
import DocsClient from './DocsClient';
import styles from './docs.module.css';

export default function DocsPage() {
  return (
    <Suspense
      fallback={
        <main className={styles.wrap}>
          <header className={styles.header}>
            <a href="/" className={styles.brand}>
              Facinect
            </a>
            <h1>API docs</h1>
            <p>Loading…</p>
          </header>
        </main>
      }
    >
      <DocsClient />
    </Suspense>
  );
}
