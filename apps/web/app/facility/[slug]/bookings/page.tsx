'use client';

import { useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';

/** Legacy /bookings → /slots-setup */
export default function LegacyBookingsRedirect() {
  const params = useParams<{ slug: string }>();
  const router = useRouter();
  useEffect(() => {
    router.replace(`/facility/${params.slug}/slots-setup`);
  }, [params.slug, router]);
  return null;
}
