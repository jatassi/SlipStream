import type { IconSvgElement } from "@hugeicons/react";
import { Link, type LinkProps } from "@tanstack/react-router";

import { AlertDot, type AlertTone, Count, type CountDot, dotOf } from "../../ui/count";
import { Icon } from "../../ui/icon";
import { Tooltip } from "../../ui/tooltip";
import { alertOf, countDots, useCounts } from "../counts";
import type { ShownRow } from "../nav";

export function NavRows({ rows, collapsed }: { rows: readonly ShownRow[]; collapsed: boolean }) {
  const counts = useCounts();
  return (
    <ul className="flex flex-col gap-0.5">
      {rows.map((row) => (
        <li key={row.id}>
          <Row
            link={{ to: row.to }}
            within={row.lands !== undefined}
            label={row.label}
            icon={row.icon}
            alert={alertOf(counts, row.alert)}
            count={
              row.count === undefined
                ? undefined
                : { value: counts?.[row.count] ?? 0, dot: countDots[row.count] }
            }
            collapsed={collapsed}
          />
          {row.children.length > 0 && (
            <ul className="mt-0.5 flex flex-col gap-0.5">
              {row.children.map((child) => (
                <li key={child.id}>
                  <Row
                    link={{ to: child.to, params: child.params }}
                    label={child.label}
                    icon={child.icon}
                    collapsed={collapsed}
                    nested
                  />
                </li>
              ))}
            </ul>
          )}
        </li>
      ))}
    </ul>
  );
}

type RowCount = { value: number; dot: CountDot };

// The rail has no room for the count, so the icon carries its dot.
function RowIcon({ icon, dot }: { icon: IconSvgElement; dot: CountDot | undefined }) {
  return (
    <span className="relative flex shrink-0">
      <Icon icon={icon} className="size-4" />
      {dot !== undefined && <span className="absolute -top-0.5 -right-0.5 flex">{dotOf(dot)}</span>}
    </span>
  );
}

function Row({
  link,
  within = false,
  label,
  icon,
  alert,
  count,
  collapsed,
  nested = false,
}: {
  link: LinkProps;
  within?: boolean;
  label: string;
  icon: IconSvgElement;
  alert?: AlertTone | undefined;
  count?: RowCount | undefined;
  collapsed: boolean;
  nested?: boolean;
}) {
  const shown = count !== undefined && count.value > 0 ? count : undefined;
  const anchor = (
    <Link
      {...link}
      activeOptions={{ exact: !within }}
      className={`flex h-8 items-center gap-2.5 rounded-item text-body text-ink-2 transition-colors duration-(--duration-control) hover:bg-fill hover:text-ink-1 aria-[current=page]:bg-selected aria-[current=page]:font-medium aria-[current=page]:text-ink-1 ${collapsed ? "justify-center" : nested ? "pr-2.5 pl-8" : "px-2.5"}`}
    >
      <RowIcon icon={icon} dot={collapsed ? (alert ?? shown?.dot) : undefined} />
      {!collapsed && <span className="min-w-0 flex-1 truncate">{label}</span>}
      {!collapsed && shown !== undefined && <Count value={shown.value} dot={shown.dot} />}
      {!collapsed && alert !== undefined && <AlertDot tone={alert} />}
    </Link>
  );
  return collapsed ? (
    <Tooltip
      label={label}
      side="right"
      {...(shown === undefined ? {} : { count: shown.value })}
      {...(alert === undefined ? {} : { alert })}
    >
      {anchor}
    </Tooltip>
  ) : (
    anchor
  );
}
