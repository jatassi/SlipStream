import { useRouter } from "@tanstack/react-router";

/** Reads the page again after a change made here, before its live event arrives. */
export function useRefresh(): () => Promise<void> {
  const router = useRouter();
  return () => router.invalidate();
}
