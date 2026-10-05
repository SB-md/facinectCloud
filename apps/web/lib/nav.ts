export type NavItem = {
  key: string;
  label: string;
  path: string; // relative to /facility/{slug}
  icon: string; // font-awesome class without fa-solid prefix piece
};

/** Mirrors Facinect PageAccess::catalog with icons. */
export const NAV_CATALOG: NavItem[] = [
  { key: 'dashboard', label: 'Dashboard', path: '', icon: 'fa-gauge-high' },
  { key: 'bookings', label: 'Slots Setup', path: '/slots-setup', icon: 'fa-table-cells' },
  { key: 'view_bookings', label: 'View Bookings', path: '/view-bookings', icon: 'fa-calendar-check' },
  { key: 'students', label: 'Students', path: '/students', icon: 'fa-user-graduate' },
  { key: 'members', label: 'Members', path: '/members', icon: 'fa-users' },
  { key: 'payments', label: 'Payments', path: '/payments', icon: 'fa-credit-card' },
  { key: 'tournaments', label: 'Tournaments', path: '/tournaments', icon: 'fa-trophy' },
  { key: 'enquiry', label: 'Enquiry', path: '/enquiry', icon: 'fa-comments' },
  { key: 'offers', label: 'Offers', path: '/offers', icon: 'fa-tags' },
  { key: 'administration', label: 'Administration', path: '/administration', icon: 'fa-gear' },
];

export function navHref(slug: string, item: NavItem): string {
  return `/facility/${encodeURIComponent(slug)}${item.path}`;
}

export function findNavByKey(key: string): NavItem | undefined {
  return NAV_CATALOG.find((n) => n.key === key);
}
