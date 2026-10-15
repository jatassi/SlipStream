import { Tick02Icon } from "@hugeicons/core-free-icons";
import type { ReactNode } from "react";

import { Icon } from "./icon";

export function MetPill({ children }: { children: ReactNode }) {
  return (
    <span className="inline-flex h-7 items-center gap-1.5 rounded-capsule bg-solid px-3 text-chip whitespace-nowrap text-on-solid">
      <Icon icon={Tick02Icon} className="size-3.5 shrink-0" />
      {children}
    </span>
  );
}
