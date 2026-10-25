import { Outlet, useMatch, useNavigate, useSearch } from "@tanstack/react-router";

import { useShell } from "../ui/shell-context";
import { menuFor } from "./actions";
import { ActionsMenu } from "./actions-menu";
import { BulkBar } from "./bulk";
import type { Row } from "./model";
import { useNames } from "./names";
import { type TorrentsSearch, TorrentsView } from "./view";

const inspected = "/_app/activity/torrents/$torrentId";

/**
 * Every Torrent, with the selected one's Inspector beside the table, or on a
 * phone in its place.
 */
export function TorrentsPage() {
  const navigate = useNavigate();
  const names = useNames();
  const search = useSearch({ from: "/_app/activity/torrents" });
  const selected = useMatch({ from: inspected, shouldThrow: false })?.params.torrentId;
  if (useShell() === "phone" && selected !== undefined) {
    return <Outlet />;
  }
  const subjectOf = (row: Row) => ({ id: row.id, name: names.work(row.workId) ?? row.name });
  return (
    <>
      <TorrentsView
        search={search}
        onSearchChange={(change: Partial<TorrentsSearch>) => {
          void navigate({ to: ".", search: (prev) => ({ ...prev, ...change }), replace: true });
        }}
        selection={{
          selected,
          onSelect: (row) => {
            void navigate({
              to: "/activity/torrents/$torrentId",
              params: { torrentId: row.id },
              search: (prev) => prev,
              replace: selected !== undefined,
            });
          },
        }}
        renderMenu={(row) => <ActionsMenu subject={subjectOf(row)} actions={menuFor(row)} />}
        renderBulk={(picked, clear) => (
          <BulkBar picked={picked} subjectOf={subjectOf} onClear={clear} />
        )}
      />
      <Outlet />
    </>
  );
}
