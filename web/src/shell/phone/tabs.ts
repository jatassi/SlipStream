import { useLocation, useMatches, useRouter } from "@tanstack/react-router";
import { useState } from "react";

import { type NavId, type ShownRow, navPaths } from "../nav";
import { type Crumb, placeOf, useBreadcrumbs } from "../titles";

export type PhonePlace = "top" | "push";

/** A nav destination opens on a large title; any other page is pushed. */
export function usePhonePlace(): PhonePlace {
  const fullPath = useMatches({ select: (matches) => matches.at(-1)?.fullPath ?? "/" });
  return navPaths.has(placeOf(fullPath)) ? "top" : "push";
}

const owns = (row: ShownRow, pathname: string) =>
  pathname === row.to || pathname.startsWith(`${row.to}/`);

type Visit = { href: string; scroll: number };

// A tab remembers the page and scroll it was left at, so switching back
// returns there.
const visits = new Map<NavId, Visit>();

/**
 * The tab the current page belongs to: the tab whose root holds the path, or,
 * for a page outside every tab's root, the tab it was opened from.
 */
export function useTabs(tabs: readonly ShownRow[]) {
  const router = useRouter();
  const { pathname, href } = useLocation();
  const owner = tabs.find((row) => owns(row, pathname));
  const [current, setCurrent] = useState<NavId | undefined>(owner?.id);
  if (owner !== undefined && owner.id !== current) {
    setCurrent(owner.id);
  }

  const open = async (row: ShownRow) => {
    if (row.id === current) {
      await router.navigate({ to: row.to });
      return;
    }
    if (current !== undefined) {
      visits.set(current, { href, scroll: window.scrollY });
    }
    setCurrent(row.id);
    const visit = visits.get(row.id);
    await router.navigate({ href: visit?.href ?? row.to, resetScroll: visit === undefined });
    window.scrollTo({ top: visit?.scroll ?? 0 });
  };

  return { current: tabs.find((row) => row.id === current), open };
}

/** Where a pushed page's back button goes: the page above it, else its tab. */
function backOf(crumbs: readonly Crumb[], tab: ShownRow | undefined): Crumb | undefined {
  const above = crumbs.at(-2);
  if (above !== undefined) {
    return above;
  }
  return tab === undefined ? undefined : { title: tab.label, pathname: tab.to };
}

export const useBack = (tab: ShownRow | undefined) => backOf(useBreadcrumbs(), tab);
