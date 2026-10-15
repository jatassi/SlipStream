import { Alert02Icon, CancelCircleIcon, CheckmarkCircle02Icon } from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";
import type { ReactNode } from "react";

import { Icon } from "./icon";

export type Signal = "fail" | "warn" | "ok";

export const signals: { readonly [S in Signal]: { icon: IconSvgElement; className: string } } = {
  fail: { icon: CancelCircleIcon, className: "bg-fail text-fail-deep" },
  warn: { icon: Alert02Icon, className: "bg-warn text-warn-deep" },
  ok: { icon: CheckmarkCircle02Icon, className: "bg-ok text-ok-deep" },
};

export function SignalChip({ kind, children }: { kind: Signal; children: ReactNode }) {
  const { icon, className } = signals[kind];
  return (
    <span
      className={`inline-flex h-6 shrink-0 items-center gap-1 rounded-chip px-2 text-chip font-bold whitespace-nowrap ${className}`}
    >
      <Icon icon={icon} className="size-3.5 shrink-0" />
      {children}
    </span>
  );
}
