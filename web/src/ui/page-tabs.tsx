import type { ReactNode } from "react";

/** A link in PageTabs. The router marks the current one `aria-current="page"`. */
export const pageTab =
  "flex h-9 items-center rounded-t-item text-body font-medium whitespace-nowrap text-ink-3 transition-colors duration-(--duration-control) ease-standard hover:text-ink-1 aria-[current=page]:text-ink-1 aria-[current=page]:shadow-[inset_0_-2px_0_var(--color-glow)] motion-reduce:transition-none";

/**
 * The row of links under a page's title that moves between the page's
 * sections. Each section is a route, so these are links in a named nav
 * landmark, not ARIA tabs.
 */
export function PageTabs({ label, children }: { label: string; children: ReactNode }) {
  return (
    <nav
      aria-label={label}
      className="mb-6 flex gap-6 overflow-x-auto shadow-[inset_0_-1px_0_var(--color-hairline)]"
    >
      {children}
    </nav>
  );
}
