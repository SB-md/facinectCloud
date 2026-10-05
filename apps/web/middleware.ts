import { NextRequest, NextResponse } from 'next/server';

/**
 * Legacy Facinect-style URLs:
 *   /facility/{id}/{slug} → /facility/{slug}
 *   /facility/{id}/{slug}/view-bookings → /facility/{slug}/view-bookings
 */
export function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl;
  const m = pathname.match(/^\/facility\/(\d+)\/([^/]+)(\/.*)?$/);
  if (!m) return NextResponse.next();

  const slug = m[2];
  const rest = m[3] || '';
  const url = req.nextUrl.clone();
  url.pathname = `/facility/${slug}${rest}`;
  return NextResponse.redirect(url);
}

export const config = {
  matcher: ['/facility/:path*'],
};
