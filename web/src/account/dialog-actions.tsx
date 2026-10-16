import type { ReactNode } from "react";

import { useShell } from "../ui/shell-context";

/** A dialog's buttons: right-aligned on desktop, stacked with the main one on top on phone. */
export function DialogActions({ children }: { children: ReactNode }) {
  const stacked = useShell() === "phone";
  return (
    <div className={`mt-2 flex gap-2 ${stacked ? "flex-col-reverse" : "justify-end"}`}>
      {children}
    </div>
  );
}
