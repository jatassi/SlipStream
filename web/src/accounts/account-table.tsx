import { Link } from "@tanstack/react-router";

import type { IdentityAccountRow } from "../api/gen/model";
import { clock } from "../format";
import { type Column, Table } from "../ui/table";
import { pinLabel } from "./pin-state";

const columns: readonly [Column<IdentityAccountRow>, ...Column<IdentityAccountRow>[]] = [
  {
    id: "name",
    header: "Name",
    cell: (account) => (
      <Link
        to="/accounts/$accountId"
        params={{ accountId: account.id }}
        className="hover:underline"
      >
        {account.displayName}
      </Link>
    ),
  },
  { id: "email", header: "Email", cell: (account) => account.email, hideBelow: 760 },
  { id: "role", header: "Role", cell: (account) => (account.admin ? "Admin" : "Member") },
  { id: "pin", header: "PIN", cell: (account) => pinLabel[account.pin.state] },
  {
    id: "passkeys",
    header: "Passkeys",
    cell: (account) => account.passkeyCount,
    align: "end",
    hideBelow: 900,
  },
  {
    id: "last-sign-in",
    header: "Last sign-in",
    cell: (account) => (account.lastSignInAt === undefined ? "Never" : clock(account.lastSignInAt)),
    hideBelow: 900,
  },
  { id: "status", header: "Sign-in", cell: (account) => (account.disabled ? "Turned off" : "On") },
];

export function AccountTable({ accounts }: { accounts: readonly IdentityAccountRow[] }) {
  return (
    <Table
      label="Accounts"
      columns={columns}
      rows={accounts}
      rowKey={(account) => account.id}
      stackBelow={620}
    />
  );
}
