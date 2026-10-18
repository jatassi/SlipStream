import { Link } from "@tanstack/react-router";

import { pageTab } from "../../../../ui/page-tabs";

export type View = "diagram" | "roster";

const views: readonly { readonly view: View; readonly label: string }[] = [
  { view: "diagram", label: "Diagram" },
  { view: "roster", label: "Roster" },
];

/** The landing's two views, beside the title. The one shown by default has no `view` in the address. */
export function ViewTabs({ current }: { current: View }) {
  return (
    <nav aria-label="View" className="flex gap-6">
      {views.map(({ view, label }) => (
        <Link
          key={view}
          to="/nodes"
          search={(prev) => ({ ...prev, view })}
          activeOptions={{ exact: true, includeSearch: true }}
          aria-current={view === current ? "page" : undefined}
          className={pageTab}
        >
          {label}
        </Link>
      ))}
    </nav>
  );
}
