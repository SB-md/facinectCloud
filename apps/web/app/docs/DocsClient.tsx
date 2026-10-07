'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import styles from './docs.module.css';

declare global {
  interface Window {
    SwaggerUIBundle?: (opts: Record<string, unknown>) => unknown;
  }
}

const SPECS = [
  {
    id: 'identity',
    label: 'Identity',
    url: '/v1/auth/openapi.json',
    blurb: 'Auth, session, Google OAuth, password — /v1/auth',
  },
  {
    id: 'notifications',
    label: 'Notifications',
    url: '/v1/notifications/openapi.json',
    blurb: 'WhatsApp + push send, facility config — /v1/notifications',
  },
  {
    id: 'booking',
    label: 'Booking',
    url: '/v1/booking/openapi.json',
    blurb: 'Courts, slots, view/book/block — /v1/booking',
  },
  {
    id: 'students',
    label: 'Students',
    url: '/v1/students/openapi.json',
    blurb: 'Enroll, list, attendance — /v1/students',
  },
  {
    id: 'members',
    label: 'Members',
    url: '/v1/members/openapi.json',
    blurb: 'Register, list, membership status — /v1/members',
  },
  {
    id: 'tournaments',
    label: 'Tournaments',
    url: '/v1/tournaments/openapi.json',
    blurb: 'List, create, update tournaments — /v1/tournaments',
  },
  {
    id: 'enquiry',
    label: 'Enquiry',
    url: '/v1/enquiry/openapi.json',
    blurb: 'Inbox + AI analyze/notify — /v1/enquiry',
  },
  {
    id: 'offers',
    label: 'Offers',
    url: '/v1/offers/openapi.json',
    blurb: 'Promotions and discount coupons — /v1/offers',
  },
  {
    id: 'administration',
    label: 'Administration',
    url: '/v1/administration/openapi.json',
    blurb: 'Facility profile, sports, staff, payments — /v1/administration',
  },
  {
    id: 'add-facility',
    label: 'Add facility',
    url: '/v1/add-facility/openapi.json',
    blurb: 'Onboard / additional facility requests — /v1/add-facility',
  },
  {
    id: 'payments',
    label: 'Payments',
    url: '/v1/payments/openapi.json',
    blurb: 'Ledger + summary — /v1/payments',
  },
  {
    id: 'ai',
    label: 'AI',
    url: '/v1/ai/openapi.json',
    blurb: 'Shared AI skills — /v1/ai',
  },
] as const;

type SpecId = (typeof SPECS)[number]['id'];

function resolveSpecId(raw: string | null): SpecId {
  if (raw === 'notifications') return 'notifications';
  if (raw === 'booking') return 'booking';
  if (raw === 'students') return 'students';
  if (raw === 'members') return 'members';
  if (raw === 'tournaments') return 'tournaments';
  if (raw === 'enquiry') return 'enquiry';
  if (raw === 'offers') return 'offers';
  if (raw === 'administration') return 'administration';
  if (raw === 'add-facility' || raw === 'addfacility') return 'add-facility';
  if (raw === 'payments') return 'payments';
  if (raw === 'ai') return 'ai';
  return 'identity';
}

export default function DocsClient() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const specId = resolveSpecId(searchParams.get('spec'));
  const spec = useMemo(() => SPECS.find((s) => s.id === specId) || SPECS[0], [specId]);
  const [ready, setReady] = useState(false);

  const mountSwagger = useCallback((url: string) => {
    const el = document.getElementById('swagger-ui');
    if (el) el.innerHTML = '';
    if (!window.SwaggerUIBundle) return;
    window.SwaggerUIBundle({
      url,
      dom_id: '#swagger-ui',
      deepLinking: true,
      persistAuthorization: true,
      displayRequestDuration: true,
      tryItOutEnabled: true,
    });
  }, []);

  useEffect(() => {
    const cssId = 'swagger-ui-css';
    if (!document.getElementById(cssId)) {
      const link = document.createElement('link');
      link.id = cssId;
      link.rel = 'stylesheet';
      link.href = 'https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css';
      document.head.appendChild(link);
    }

    const existing = document.getElementById('swagger-ui-bundle') as HTMLScriptElement | null;
    if (existing && window.SwaggerUIBundle) {
      setReady(true);
      return;
    }
    if (existing) {
      existing.addEventListener('load', () => setReady(true));
      return;
    }

    const script = document.createElement('script');
    script.id = 'swagger-ui-bundle';
    script.src = 'https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js';
    script.async = true;
    script.onload = () => setReady(true);
    document.body.appendChild(script);
  }, []);

  useEffect(() => {
    if (!ready) return;
    mountSwagger(spec.url);
  }, [ready, spec.url, mountSwagger]);

  function onSpecChange(next: string) {
    const id = resolveSpecId(next);
    const q = new URLSearchParams(searchParams.toString());
    q.set('spec', id);
    router.replace(`${pathname}?${q.toString()}`);
  }

  return (
    <main className={styles.wrap}>
      <header className={styles.header}>
        <a href="/" className={styles.brand}>
          Facinect
        </a>
        <h1>API docs</h1>
        <p>{spec.blurb}</p>
        <label className={styles.specPicker}>
          <span>Service</span>
          <select value={spec.id} onChange={(e) => onSpecChange(e.target.value)}>
            {SPECS.map((s) => (
              <option key={s.id} value={s.id}>
                {s.label}
              </option>
            ))}
          </select>
        </label>
      </header>
      <div id="swagger-ui" className={styles.swagger} />
    </main>
  );
}
