import type { ReactNode } from "react";

import { useShell } from "../ui/shell-context";

export function DialogActions({ children }: { children: ReactNode }) {
  const stacked = useShell() === "phone";
  return (
    <div className={`mt-2 flex gap-2 ${stacked ? "flex-col-reverse" : "justify-end"}`}>
      {children}
    </div>
  );
}
