#!/usr/bin/env node
/**
 * Pull landing copy from Facinect website/index.html into apps/web/lib/siteContent.ts
 */
const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '..');
const srcHtml = path.resolve(
  root,
  '../../Downloads/serv/Services/Facinect/website/index.html'
);
// Prefer sibling StudioProjects layout; fall back to absolute common path
const candidates = [
  path.resolve(root, '../Downloads/serv/Services/Facinect/website/index.html'),
  path.resolve('/home/dell/Downloads/serv/Services/Facinect/website/index.html'),
  path.resolve(process.env.HOME || '', 'Downloads/serv/Services/Facinect/website/index.html'),
];

let htmlPath = candidates.find((p) => fs.existsSync(p));
if (!htmlPath) {
  console.error('Facinect website/index.html not found. Checked:\n' + candidates.join('\n'));
  process.exit(1);
}

const html = fs.readFileSync(htmlPath, 'utf8');

function pick(re, fallback = '') {
  const m = html.match(re);
  return m ? m[1].replace(/\s+/g, ' ').trim() : fallback;
}

const title = pick(/<title>([^<]+)<\/title>/i, 'Facinect — Sports Facility Operating System');
const description = pick(
  /<meta\s+name="description"\s+content="([^"]+)"/i,
  'Facinect automates court bookings, academy management, memberships, payments, WhatsApp messaging, and tournaments for sports facilities.'
);
const headline = pick(
  /<h1[^>]*>\s*([\s\S]*?)\s*<\/h1>/i,
  'The operating system for sports facilities'
).replace(/<[^>]+>/g, '');
const lede = pick(
  /<p class="hero-lede[^"]*"[^>]*>\s*([\s\S]*?)\s*<\/p>/i,
  'Bookings, academies, memberships, WhatsApp, and tournaments — one platform from court to mobile.'
).replace(/<[^>]+>/g, '');
const productLine = pick(
  /<p class="hero-brand[^"]*"[^>]*>\s*([\s\S]*?)\s*<\/p>/i,
  'Facinect'
);

const out = `/**
 * Landing copy — keep in sync with Facinect marketing site.
 * Source: ${htmlPath}
 * Generated: ${new Date().toISOString()}
 * Refresh: node scripts/sync-landing-copy.cjs
 */
export const siteContent = {
  brand: 'Facinect',
  brandFull: 'Facinect Multisports Management',
  productLine: ${JSON.stringify(productLine === 'Facinect' ? 'Sports Facility Operating System' : productLine)},
  headline: ${JSON.stringify(headline)},
  lede: ${JSON.stringify(lede)},
  primaryCta: { label: 'Login', href: '/login' },
  secondaryCta: { label: 'Open account', href: '/home' },
  meta: {
    title: ${JSON.stringify(title)},
    description: ${JSON.stringify(description)},
  },
} as const;
`;

const dest = path.resolve(root, 'apps/web/lib/siteContent.ts');
fs.mkdirSync(path.dirname(dest), { recursive: true });
fs.writeFileSync(dest, out);
console.log('Updated', dest);
console.log({ title, headline, lede: lede.slice(0, 80) + '…' });
