import type { ReactNode } from "react";

export function LiveDot() {
  return (
    <span
      aria-hidden="true"
      className="size-1.5 shrink-0 rounded-capsule bg-ink-1 shadow-[0_0_0_3px_hsl(var(--h)_90%_65%/.32),var(--shadow-live-glow)]"
    />
  );
}

export function LiveChip({ children }: { children: ReactNode }) {
  return (
    <span className="inline-flex h-6 items-center gap-2 rounded-chip bg-live-fill px-2 text-chip whitespace-nowrap text-ink-1 ring-1 ring-[hsl(var(--h)_90%_65%/.32)] ring-inset">
      <LiveDot />
      {children}
    </span>
  );
}
