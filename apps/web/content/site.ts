/**
 * Site copy for the Next.js gateway home.
 * Keep in sync with Services/Facinect/website/index.html (marketing source of truth).
 */
export const site = {
  brand: {
    name: 'Facinect',
    fullName: 'Facinect Multisports Management',
    tagline: 'Sports Facility Operating System',
  },
  meta: {
    title: 'Facinect — Sports Facility Operating System',
    description:
      'Facinect automates court bookings, academy management, memberships, payments, WhatsApp messaging, and tournaments for sports facilities.',
  },
  hero: {
    brand: 'Facinect',
    headline: 'The operating system for sports facilities',
    lede: 'Bookings, academies, memberships, WhatsApp, and tournaments — one platform from court to mobile.',
    primaryCta: { label: 'Partner Login', href: '/login' },
    secondaryCta: { label: 'Open account', href: '/home' },
  },
  audience: {
    title: 'Built for the people who run the venue',
    lede: 'Facinect connects facility owners, academy coaches, and tournament admins on a single workflow.',
    items: [
      {
        title: 'Facility owners',
        body: 'Live court calendars, payments, offers, and staff roles in one admin.',
      },
      {
        title: 'Academy coaches',
        body: 'Students, plans, attendance, and batches on web or the mobile app.',
      },
      {
        title: 'Tournament admins',
        body: 'Fixtures, live scores, registrations, and shareable referee links.',
      },
    ],
  },
  features: {
    title: 'Everything your facility runs on',
    lede: 'The same capabilities your team already uses in Facinect — one product story.',
    items: [
      {
        title: 'Court & slot booking',
        body: 'Multi-court calendars with available, booked, and blocked slots.',
      },
      {
        title: 'Academy & students',
        body: 'Sport-wise roster, enrollments, plans, and coach workflows.',
      },
      {
        title: 'Payments',
        body: 'Unified ledgers across booking, coaching, membership, and tournaments.',
      },
      {
        title: 'Tournaments',
        body: 'Events, fixtures, live scores, and referee scoring links.',
      },
      {
        title: 'WhatsApp & ops',
        body: 'Bulk messaging, offers, and day-to-day facility operations.',
      },
      {
        title: 'Memberships',
        body: 'Plans, enrollments, and approve or reject membership requests.',
      },
    ],
  },
} as const;

export type SiteContent = typeof site;
