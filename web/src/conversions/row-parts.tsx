import { MoreHorizontalIcon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";

import type { ConversionsConversionRow } from "../api/gen/model";
import { IconButton } from "../ui/icon-button";
import { Menu, type MenuEntry } from "../ui/menu";

export const listed = new Intl.ListFormat("en", { type: "conjunction" });

/** The Work and its episodes, such as "The Office S02E01, S02E02". */
export const subjectOf = ({ work, items }: ConversionsConversionRow) =>
  items.length === 0 ? work.title : `${work.title} ${items.map((item) => item.label).join(", ")}`;

export const rowLook =
  "flex min-h-(--row-height) items-center gap-3 rounded-card bg-glass px-4 py-2 ring-1 ring-hairline ring-inset in-data-[density=touch]:min-h-(--row-height-two)";

export function Lines({ first, second }: { first: ReactNode; second: ReactNode }) {
  return (
    <span className="flex min-w-0 flex-1 flex-col gap-0.5">
      <span className="truncate text-ink-1">{first}</span>
      <span className="text-meta text-ink-3">{second}</span>
    </span>
  );
}

export function WorkLink({ row }: { row: ConversionsConversionRow }) {
  return (
    <Link
      to="/works/$workId"
      params={{ workId: row.work.id }}
      className="rounded-item underline-offset-4 hover:underline"
    >
      {subjectOf(row)}
    </Link>
  );
}

export function RowMenu({ label, items }: { label: string; items: readonly MenuEntry[] }) {
  return (
    <Menu
      label={label}
      trigger={<IconButton icon={MoreHorizontalIcon} label={`Actions for ${label}`} />}
      items={items}
    />
  );
}
