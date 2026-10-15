import { MoreHorizontalIcon, Search01Icon } from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";
import { Link, useLocation } from "@tanstack/react-router";
import type { ReactNode } from "react";

import { Icon } from "../../ui/icon";
import { useCounts } from "../counts";
import type { ShownRow } from "../nav";
import { openPalette } from "../palette-state";
import { openMore } from "./more-state";

const cell =
  "relative flex h-[49px] w-full flex-col items-center justify-center gap-0.5 text-label transition-colors duration-(--duration-control)";

export function TabBar({
  tabs,
  current,
  onOpen,
}: {
  tabs: readonly ShownRow[];
  current: ShownRow | undefined;
  onOpen: (row: ShownRow) => void;
}) {
  const counts = useCounts();
  return (
    <nav
      aria-label="Tabs"
      className="fixed inset-x-0 bottom-0 z-20 bg-chrome pr-[env(safe-area-inset-right)] pb-[env(safe-area-inset-bottom)] pl-[env(safe-area-inset-left)] shadow-[inset_0_1px_0_var(--color-hairline)] backdrop-blur-[24px] backdrop-saturate-[1.6] [view-transition-name:tab-bar]"
    >
      <ul className="grid auto-cols-fr grid-flow-col">
        {tabs.map((row) => (
          <li key={row.id}>
            <Tab
              row={row}
              current={row.id === current?.id}
              badge={row.count === undefined ? 0 : (counts?.[row.count] ?? 0)}
              onOpen={onOpen}
            />
          </li>
        ))}
        <li>
          <button type="button" onClick={openPalette} className={`${cell} text-ink-3`}>
            <TabFace icon={Search01Icon} label="Search" />
          </button>
        </li>
        <li>
          <button type="button" onClick={openMore} className={`${cell} text-ink-3`}>
            <TabFace icon={MoreHorizontalIcon} label="More" />
          </button>
        </li>
      </ul>
    </nav>
  );
}

function Tab({
  row,
  current,
  badge,
  onOpen,
}: {
  row: ShownRow;
  current: boolean;
  badge: number;
  onOpen: (row: ShownRow) => void;
}) {
  const { pathname } = useLocation();
  return (
    <Link
      to={row.to}
      aria-current={current ? (pathname === row.to ? "page" : "true") : undefined}
      onClick={(event) => {
        event.preventDefault();
        onOpen(row);
      }}
      className={`${cell} ${current ? "text-glow" : "text-ink-3"}`}
    >
      <TabFace icon={row.icon} label={row.label} />
      {badge > 0 && (
        <span className="absolute top-1 left-[calc(50%+6px)] grid h-4.5 min-w-4.5 place-items-center rounded-capsule bg-solid px-1 text-label font-semibold text-on-solid tabular-nums">
          {badge}
        </span>
      )}
    </Link>
  );
}

function TabFace({ icon, label }: { icon: IconSvgElement; label: string }): ReactNode {
  return (
    <>
      <Icon icon={icon} className="size-6" />
      <span>{label}</span>
    </>
  );
}
