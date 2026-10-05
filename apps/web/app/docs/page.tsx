'use client';

import { useEffect } from 'react';
import styles from './docs.module.css';

declare global {
  interface Window {
    SwaggerUIBundle?: (opts: Record<string, unknown>) => void;
  }
}

export default function DocsPage() {
  useEffect(() => {
    const cssId = 'swagger-ui-css';
    if (!document.getElementById(cssId)) {
      const link = document.createElement('link');
      link.id = cssId;
      link.rel = 'stylesheet';
      link.href = 'https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css';
      document.head.appendChild(link);
    }

    const boot = () => {
      if (!window.SwaggerUIBundle) return;
      window.SwaggerUIBundle({
        url: '/v1/auth/openapi.json',
        dom_id: '#swagger-ui',
        deepLinking: true,
        persistAuthorization: true,
        displayRequestDuration: true,
        tryItOutEnabled: true,
      });
    };

    const existing = document.getElementById('swagger-ui-bundle') as HTMLScriptElement | null;
    if (existing) {
      boot();
      return;
    }

    const script = document.createElement('script');
    script.id = 'swagger-ui-bundle';
    script.src = 'https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js';
    script.async = true;
    script.onload = boot;
    document.body.appendChild(script);
  }, []);

  return (
    <main className={styles.wrap}>
      <header className={styles.header}>
        <a href="/" className={styles.brand}>
          Facinect
        </a>
        <h1>API docs</h1>
        <p>Identity OpenAPI — try requests against the local gateway.</p>
      </header>
      <div id="swagger-ui" className={styles.swagger} />
    </main>
  );
}
