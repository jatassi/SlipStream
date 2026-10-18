import type { ReactNode } from "react";

const looks = {
  fill: "bg-fill",
  outline: "ring-1 ring-hairline-strong ring-inset",
} as const;

export function Chip({
  look = "fill",
  children,
}: {
  look?: keyof typeof looks;
  children: ReactNode;
}) {
  return (
    <span
      className={`inline-flex h-6 items-center gap-1 rounded-chip px-2 text-chip whitespace-nowrap text-ink-2 ${looks[look]}`}
    >
      {children}
    </span>
  );
}
