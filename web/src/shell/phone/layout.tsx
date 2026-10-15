import type { ReactNode } from "react";

import { useAdminNav } from "../nav";
import { AdminBarActions, adminTabActions } from "./admin-actions";
import { PhoneFrame } from "./frame";
import { MoreSheet } from "./more-sheet";

export function PhoneShell({ children }: { children: ReactNode }) {
  const rows = useAdminNav();
  return (
    <>
      <PhoneFrame rows={rows} barActions={<AdminBarActions />} tabActions={adminTabActions}>
        {children}
      </PhoneFrame>
      <MoreSheet rows={rows.filter((row) => row.tab === undefined)} />
    </>
  );
}
