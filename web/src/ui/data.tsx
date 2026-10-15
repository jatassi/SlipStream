import type { ReactNode } from "react";

export const dataFont = "font-mono tabular-nums";

export function Data({ children, className }: { children: ReactNode; className?: string }) {
  return <span className={[dataFont, "text-data", className].join(" ")}>{children}</span>;
}
