import { type ReactNode, useId } from "react";

import { useShell } from "./shell-context";

const shellLook = {
  desktop: { section: "[section+&]:mt-6", title: "text-heading" },
  phone: { section: "[section+&]:mt-7", title: "text-title" },
} as const;

export function Section({
  id,
  title,
  count,
  children,
}: {
  id?: string;
  title: string;
  count?: number;
  children: ReactNode;
}) {
  const look = shellLook[useShell()];
  const titleId = useId();
  return (
    <section id={id} aria-labelledby={titleId} className={`scroll-mt-20 ${look.section}`}>
      <h2 className={`mb-3 flex items-baseline gap-2 text-ink-1 ${look.title}`}>
        <span id={titleId}>{title}</span>
        {count !== undefined && (
          <span className="font-normal text-ink-3 tabular-nums">{count}</span>
        )}
      </h2>
      {children}
    </section>
  );
}
