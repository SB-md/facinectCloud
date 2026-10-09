'use client';

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { useParams, usePathname, useRouter } from 'next/navigation';
import AuthGuard from '../../../lib/AuthGuard';
import {
  FacilityMembership,
  SessionUser,
  canAccessPage,
  facilityPath,
} from '../../../lib/auth';
import { NAV_CATALOG, navHref } from '../../../lib/nav';
import styles from './portal.module.css';

type Props = {
  pageKey?: string;
  title: string;
  description?: string;
  children?: React.ReactNode;
};

export default function FacilityShell({ pageKey, title, description, children }: Props) {
  const params = useParams<{ slug: string }>();
  const slug = params.slug;

  return (
    <AuthGuard facilitySlug={slug} pageKey={pageKey}>
      {({ user, facility }) => (
        <PortalLayout
          user={user}
          facility={facility}
          slug={slug}
          pageKey={pageKey}
          title={title}
          description={description}
        >
          {children}
        </PortalLayout>
      )}
    </AuthGuard>
  );
}

function PortalLayout({
  user,
  facility,
  slug,
  pageKey,
  title,
  description,
  children,
}: {
  user: SessionUser;
  facility: FacilityMembership | null;
  slug: string;
  pageKey?: string;
  title: string;
  description?: string;
  children?: React.ReactNode;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const current = facility || findBySlug(user, slug);
  const [menuOpen, setMenuOpen] = useState(false);
  const [switchOpen, setSwitchOpen] = useState(false);

  const accessUser = useMemo(
    () =>
      current
        ? { ...user, role: current.role, page_access: current.page_access }
        : user,
    [user, current],
  );

  useEffect(() => {
    setMenuOpen(false);
    setSwitchOpen(false);
  }, [pathname]);

  function switchFacility(next: FacilityMembership) {
    setSwitchOpen(false);
    if (next.slug === slug) return;
    if (!pageKey) {
      router.push(`/facility/${encodeURIComponent(next.slug)}/settings`);
      return;
    }
    const item = NAV_CATALOG.find((n) => n.key === pageKey);
    const path = item ? navHref(next.slug, item) : facilityPath(next.slug, next.role);
    if (!canAccessPage({ ...user, role: next.role, page_access: next.page_access }, pageKey)) {
      router.push(facilityPath(next.slug, next.role));
      return;
    }
    router.push(path);
  }

  const facilities = user.facilities || [];
  const settingsHref = `/facility/${encodeURIComponent(slug)}/settings`;
  const settingsActive = pathname?.endsWith('/settings');

  return (
    <div className={`${styles.shell} ${pageKey === 'dashboard' ? styles.shellDash : ''}`}>
      <header className={styles.topbar}>
        <div className={styles.topLeft}>
          <button
            type="button"
            className={styles.menuToggle}
            aria-label={menuOpen ? 'Close menu' : 'Open menu'}
            aria-expanded={menuOpen}
            onClick={() => setMenuOpen((v) => !v)}
          >
            <i className={`fa-solid ${menuOpen ? 'fa-xmark' : 'fa-bars'}`} />
          </button>
        </div>

        <Link href={navHref(slug, NAV_CATALOG[0])} className={styles.brand}>
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img className={styles.brandLogo} src="/facinectlogo.png" alt="" width={52} height={52} />
          <p className={styles.brandName}>Facinect Multisports Management</p>
          <p className={styles.brandSub}>{current?.facilityName || slug}</p>
        </Link>

        <div className={styles.topRight}>
          <div className={styles.switchWrap}>
            <button
              type="button"
              className={styles.switchBtn}
              onClick={() => setSwitchOpen((v) => !v)}
              aria-expanded={switchOpen}
            >
              <i className="fa-solid fa-building" />
              <strong>{current?.facilityName || 'Facility'}</strong>
              <i className="fa-solid fa-chevron-down" style={{ fontSize: 11 }} />
            </button>
            {switchOpen && (
              <div className={styles.switchMenu} role="listbox">
                {facilities.map((f) => (
                  <button
                    key={f.facilityId}
                    type="button"
                    className={`${styles.switchItem} ${f.slug === slug ? styles.switchItemActive : ''}`}
                    onClick={() => switchFacility(f)}
                  >
                    {f.facilityName}
                    <div style={{ fontSize: 11, color: '#5a6f68', marginTop: 2 }}>
                      {f.role} · {f.slug}
                    </div>
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
      </header>

      {menuOpen && <div className={styles.backdrop} onClick={() => setMenuOpen(false)} aria-hidden />}

      <aside className={`${styles.sidebar} ${menuOpen ? styles.sidebarOpen : ''}`}>
        <p className={styles.sideLabel}>Features</p>
        <nav className={styles.sideNav}>
          {NAV_CATALOG.map((item) => {
            const allowed = canAccessPage(accessUser, item.key);
            const href = navHref(slug, item);
            const active = pageKey === item.key;
            if (!allowed) {
              return (
                <span key={item.key} className={`${styles.navLink} ${styles.navLocked}`} title="No access">
                  <i className={`fa-solid ${item.icon}`} />
                  {item.label}
                  <i className="fa-solid fa-lock" style={{ marginLeft: 'auto', fontSize: 11 }} />
                </span>
              );
            }
            return (
              <Link
                key={item.key}
                href={href}
                className={`${styles.navLink} ${active ? styles.navActive : ''}`}
                onClick={() => setMenuOpen(false)}
              >
                <i className={`fa-solid ${item.icon}`} />
                {item.label}
              </Link>
            );
          })}
        </nav>

        <div className={styles.profileFoot}>
          {user.avatar_url ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img className={styles.avatarImg} src={user.avatar_url} alt="" width={40} height={40} referrerPolicy="no-referrer" />
          ) : (
            <div className={styles.avatar} aria-hidden>
              {userInitials(user)}
            </div>
          )}
          <div className={styles.profileMeta}>
            <strong>{user.full_name || user.email || 'User'}</strong>
            <span>{user.email || current?.role || '—'}</span>
          </div>
          <Link
            href={settingsHref}
            className={`${styles.profileLogout} ${settingsActive ? styles.profileSettingsActive : ''}`}
            title="Settings"
            onClick={() => setMenuOpen(false)}
          >
            <i className="fa-solid fa-gear" />
          </Link>
        </div>
      </aside>

      <main className={`${styles.main} ${pageKey === 'dashboard' ? styles.mainDash : ''}`}>
        {pageKey === 'dashboard' ? null : (
          <div className={styles.pageHead}>
            <h1>{title}</h1>
            <p>{description || `Manage ${title.toLowerCase()} for ${current?.facilityName || slug}.`}</p>
          </div>
        )}

        {children ? (
          children
        ) : (
          <>
            <div className={styles.detailGrid}>
              <div className={styles.detailCard}>
                <span>Facility</span>
                <strong>{current?.facilityName || '—'}</strong>
              </div>
              <div className={styles.detailCard}>
                <span>Slug</span>
                <strong>{current?.slug || slug}</strong>
              </div>
              <div className={styles.detailCard}>
                <span>Role</span>
                <strong>{current?.role || user.role || '—'}</strong>
              </div>
              <div className={styles.detailCard}>
                <span>Signed in</span>
                <strong>{user.full_name || user.email || '—'}</strong>
              </div>
              <div className={styles.detailCard}>
                <span>Facility ID</span>
                <strong>{current?.facilityId ?? '—'}</strong>
              </div>
              <div className={styles.detailCard}>
                <span>Pages enabled</span>
                <strong>{(current?.page_access || []).length}</strong>
              </div>
            </div>

            <section className={styles.panel}>
              <h2>{title}</h2>
              <p>
                This is the <strong>{title}</strong> workspace for{' '}
                <strong>{current?.facilityName || slug}</strong>. Feature modules from Facinect are
                wired into this portal shell; connect booking / payments / WhatsApp services next.
              </p>
              <ul className={styles.featureList}>
                {(current?.page_access || []).map((key) => {
                  const item = NAV_CATALOG.find((n) => n.key === key);
                  return (
                    <li key={key}>
                      <i className={`fa-solid ${item?.icon || 'fa-circle'}`} />
                      {item?.label || key}
                    </li>
                  );
                })}
              </ul>
            </section>
          </>
        )}
      </main>
    </div>
  );
}

function findBySlug(user: SessionUser, slug: string): FacilityMembership | null {
  const key = decodeURIComponent(slug).toLowerCase();
  return (user.facilities || []).find((f) => f.slug.toLowerCase() === key) || null;
}

function userInitials(user: SessionUser): string {
  const name = (user.full_name || '').trim();
  if (name) {
    const parts = name.split(/\s+/).filter(Boolean);
    if (parts.length >= 2) {
      return (parts[0][0] + parts[1][0]).toUpperCase();
    }
    return name.slice(0, 2).toUpperCase();
  }
  const email = (user.email || '').trim();
  if (email) return email.slice(0, 2).toUpperCase();
  return 'U';
}
