import { ArrowRight01Icon, Search01Icon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";
import { type ComponentType, createContext, use } from "react";

import { TasksCapsule } from "../../tasks/capsule";
import { Icon } from "../../ui/icon";
import { ConnectionLine } from "../connection-line";
import { openPalette, paletteShortcut } from "../palette-state";
import { useBreadcrumbs } from "../titles";

export const ActionSlotTarget = createContext<(element: HTMLElement | null) => void>(() => {});

function PageAction() {
  return <div ref={use(ActionSlotTarget)} className="contents" />;
}

type TopBarSlot = { readonly id: string; readonly Slot: ComponentType };

// The right side's order is fixed: the connection line first, the page's
// main action last, and each later spec's control as a row between them.
const topBarSlots = [
  { id: "connection", Slot: ConnectionLine },
  { id: "tasks", Slot: TasksCapsule },
  { id: "page-action", Slot: PageAction },
] as const satisfies readonly TopBarSlot[];

export function TopBar() {
  return (
    <header className="sticky top-0 z-20 grid h-14 grid-cols-[1fr_minmax(320px,440px)_1fr] items-center gap-4 bg-chrome px-7 shadow-[inset_0_-1px_0_var(--color-hairline)] backdrop-blur-[24px] backdrop-saturate-[1.6] [view-transition-name:top-bar]">
      <Breadcrumbs />
      <PaletteCapsule />
      <div className="flex min-w-0 items-center justify-end gap-3">
        {topBarSlots.map(({ id, Slot }) => (
          <Slot key={id} />
        ))}
      </div>
    </header>
  );
}

function Breadcrumbs() {
  const crumbs = useBreadcrumbs();
  return (
    <nav aria-label="Breadcrumbs" className="min-w-0">
      <ol className="flex min-w-0 items-center gap-1.5 text-body text-ink-3">
        {crumbs.map((crumb, index) => (
          <li key={crumb.pathname} className="flex min-w-0 items-center gap-1.5">
            {index > 0 && <Icon icon={ArrowRight01Icon} className="size-3.5 shrink-0 text-ink-4" />}
            {index === crumbs.length - 1 ? (
              <span aria-current="page" className="truncate font-medium text-ink-1">
                {crumb.title}
              </span>
            ) : (
              <Link
                to={crumb.pathname}
                viewTransition={{ types: ["pop"] }}
                className="truncate rounded-chip hover:text-ink-1"
              >
                {crumb.title}
              </Link>
            )}
          </li>
        ))}
      </ol>
    </nav>
  );
}

function PaletteCapsule() {
  return (
    <button
      type="button"
      aria-label="Search Works or run a command"
      aria-keyshortcuts={paletteShortcut.keys}
      onClick={openPalette}
      className="flex h-8 w-full min-w-0 items-center gap-2.5 rounded-capsule bg-fill px-3.5 text-body text-ink-3 ring-1 ring-hairline ring-inset transition-colors duration-(--duration-control) hover:bg-fill-hover hover:text-ink-2"
    >
      <Icon icon={Search01Icon} className="size-4 shrink-0" />
      <span className="min-w-0 flex-1 truncate text-left">Search Works or run a command</span>
      <kbd className="shrink-0 font-sans text-label text-ink-3">{paletteShortcut.label}</kbd>
    </button>
  );
}
