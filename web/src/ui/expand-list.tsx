import { Collapsible } from "@base-ui/react/collapsible";
import type { ReactNode } from "react";

import { Chevron } from "./chevron";
import "./rows.css";

export function ExpandList({ label, children }: { label: string; children: ReactNode }) {
  return (
    <ul
      aria-label={label}
      className="@container rounded-card bg-glass ring-1 ring-hairline ring-inset"
    >
      {children}
    </ul>
  );
}

export function ExpandRow({ head, children }: { head: ReactNode; children: ReactNode }) {
  return (
    <li className="first:rounded-t-card last:rounded-b-card not-first:shadow-[inset_0_1px_0_var(--color-hairline)]">
      <Collapsible.Root>
        <Collapsible.Trigger className="group flex min-h-(--row-height-two) w-full items-center gap-3 rounded-[inherit] px-4 py-2.5 text-left text-(length:--body-size) leading-(--body-line-height) text-ink-1 transition-colors duration-350 ease-standard hover:bg-(--row-wash) focus-visible:bg-(--row-wash)">
          <span className="flex text-ink-4">
            <Chevron />
          </span>
          {head}
        </Collapsible.Trigger>
        <Collapsible.Panel className="pr-4 pb-3 pl-11">{children}</Collapsible.Panel>
      </Collapsible.Root>
    </li>
  );
}
