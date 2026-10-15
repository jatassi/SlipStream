import { createFlag } from "./flag";
import { narrowDesktopQuery, useMedia } from "./media";
import { readPref, usePref, writePref } from "./prefs";

// Below 1100px the sidebar collapses by itself, and opening it there lasts
// for this page only, so the stored choice keeps describing wide windows.
const openWhileNarrow = createFlag();

export function toggleSidebar(): void {
  if (matchMedia(narrowDesktopQuery).matches) {
    openWhileNarrow.set(!openWhileNarrow.get());
  } else {
    writePref("sidebarCollapsed", !readPref("sidebarCollapsed"));
  }
}

export function useSidebarCollapsed(): boolean {
  const narrow = useMedia(narrowDesktopQuery);
  const stored = usePref("sidebarCollapsed");
  const open = openWhileNarrow.useValue();
  return narrow ? !open : stored;
}
