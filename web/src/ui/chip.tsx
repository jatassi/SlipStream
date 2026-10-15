import type { ReactNode } from "react";

export function Chip({ children }: { children: ReactNode }) {
  return (
    <span className="inline-flex h-6 items-center gap-1 rounded-chip bg-fill px-2 text-chip whitespace-nowrap text-ink-2">
      {children}
    </span>
  );
}
