import { Outlet } from "@tanstack/react-router";

import { useShell } from "../ui/shell-context";
import { menuFor, subjectOf } from "./actions";
import { ActionsMenu } from "./actions-menu";
import { BulkBar } from "./bulk";
import type { Row } from "./model";
import { useNames } from "./names";
import type { Scope } from "./scope";
import { type TorrentsSearch, TorrentsView } from "./view";

/** Where the page sits: its address's search, the Torrent its child route inspects, and how to change either. */
export type TorrentsPlace = {
  search: TorrentsSearch;
  selected: string | undefined;
  onSearchChange: (change: Partial<TorrentsSearch>) => void;
  onSelect: (row: Row) => void;
  scope?: Scope | undefined;
};

/**
 * The Torrents, with the selected one's Inspector (the child route) beside the
 * table, or on a phone in its place.
 */
export function TorrentsPage({ search, selected, onSearchChange, onSelect, scope }: TorrentsPlace) {
  const names = useNames();
  if (useShell() === "phone" && selected !== undefined) {
    return <Outlet />;
  }
  const subject = (row: Row) => subjectOf(row, names.work(row.workId) ?? row.name);
  return (
    <>
      <TorrentsView
        search={search}
        onSearchChange={onSearchChange}
        scope={scope}
        selection={{ selected, onSelect }}
        renderMenu={(row) => <ActionsMenu subject={subject(row)} actions={menuFor(row)} />}
        renderBulk={(picked, clear) => (
          <BulkBar picked={picked} subjectOf={subject} onClear={clear} />
        )}
      />
      <Outlet />
    </>
  );
}
