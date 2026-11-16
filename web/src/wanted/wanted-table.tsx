import { MoreHorizontalIcon } from "@hugeicons/core-free-icons";
import { Link, useNavigate } from "@tanstack/react-router";

import type { GrabsWantedRow } from "../api/gen/model";
import type { Clock } from "../format";
import { openManualSearch } from "../manual/sheet-state";
import { Button } from "../ui/button";
import { IconButton } from "../ui/icon-button";
import { Menu } from "../ui/menu";
import { Poster } from "../ui/poster";
import { Spinner } from "../ui/spinner";
// oxlint-disable-next-line import/max-dependencies -- each column draws its own part of a row
import { type Column, Table } from "../ui/table";
import { type RowSearch, useRowSearch, workLink } from "./row-search";
import { type Headers, lastWords, nextWords, rangesWords, reasonWords } from "./wanted-words";

function WorkCell({ row }: { row: GrabsWantedRow }) {
  return (
    <span className="flex min-w-0 items-center gap-3 py-1.5">
      <Poster url={row.work.posterUrl ?? undefined} />
      <Link {...workLink(row)} className="truncate font-semibold">
        {row.work.label}
      </Link>
    </span>
  );
}

function NextCell({ row, clock }: { row: GrabsWantedRow; clock: Clock }) {
  return (
    <span className="inline-flex items-center gap-2">
      {row.state === "searching" && <Spinner />}
      {nextWords(row, clock)}
    </span>
  );
}

function RowActions({ row, search }: { row: GrabsWantedRow; search: RowSearch }) {
  const navigate = useNavigate();
  const label = `More for ${row.work.label}`;
  return (
    <span className="inline-flex items-center gap-1">
      <Button
        size="small"
        aria-label={`Search ${row.work.label}`}
        busy={search.busy === row.work.id}
        onClick={() => {
          search.start(row);
        }}
      >
        Search
      </Button>
      <Menu
        label={label}
        trigger={<IconButton icon={MoreHorizontalIcon} label={label} />}
        items={[
          {
            label: "Manual search",
            onSelect: () => {
              openManualSearch({ workId: row.work.id });
            },
          },
          {
            label: "Open Work",
            onSelect: () => {
              void navigate(workLink(row));
            },
          },
        ]}
      />
    </span>
  );
}

type Columns = readonly [Column<GrabsWantedRow>, ...Column<GrabsWantedRow>[]];

const columnsOf = (headers: Headers, clock: Clock, search: RowSearch): Columns => [
  { id: "work", header: headers.work, cell: (row) => <WorkCell row={row} /> },
  ...(headers.items === undefined
    ? []
    : [
        {
          id: "items",
          header: headers.items,
          hideBelow: 900,
          cell: (row: GrabsWantedRow) => rangesWords(row.ranges),
        },
      ]),
  { id: "reason", header: "Reason", cell: reasonWords },
  { id: "last", header: "Last searched", hideBelow: 760, cell: (row) => lastWords(row, clock) },
  { id: "next", header: "Next search", cell: (row) => <NextCell row={row} clock={clock} /> },
  {
    id: "actions",
    header: "Actions",
    align: "end",
    cell: (row) => <RowActions row={row} search={search} />,
  },
];

/** Wanted's rows as a table: the Work, its Items, why, the last search and the next. */
export function WantedTable({
  rows,
  headers,
  clock,
}: {
  rows: readonly GrabsWantedRow[];
  headers: Headers;
  clock: Clock;
}) {
  const search = useRowSearch();
  return (
    <Table
      label="Wanted"
      columns={columnsOf(headers, clock, search)}
      rows={rows}
      rowKey={(row) => row.work.id}
      lines={2}
    />
  );
}
