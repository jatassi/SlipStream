import { Link } from "@tanstack/react-router";

import type { GrabsBlockEntry } from "../api/gen/model";
import { type Clock, agoWords, sinceWords } from "../format";
import { ReleaseName } from "../release/row";
import { Checkbox } from "../ui/checkbox";
import { Data } from "../ui/data";
import { type Column, Table } from "../ui/table";

export type Picking = {
  readonly has: (hash: string) => boolean;
  readonly toggle: (hash: string, range: boolean) => void;
};

type Row = { readonly entry: GrabsBlockEntry; readonly picking: Picking; readonly clock: Clock };

const columns: readonly [Column<Row>, ...Column<Row>[]] = [
  {
    id: "pick",
    header: "",
    cell: ({ entry, picking }) => (
      <Checkbox
        label={`Select ${entry.title}`}
        hideLabel
        checked={picking.has(entry.infoHash)}
        onCheckedChange={(_, by) => {
          picking.toggle(entry.infoHash, by.shiftKey);
        }}
      />
    ),
  },
  { id: "release", header: "Release", cell: ({ entry }) => <ReleaseName release={entry} /> },
  { id: "cause", header: "Cause", cell: ({ entry }) => entry.because },
  { id: "indexer", header: "Indexer", cell: ({ entry }) => entry.indexer },
  {
    id: "work",
    header: "Work",
    cell: ({ entry: { work } }) =>
      work !== null && (
        <Link
          to="/works/$workId"
          params={{ workId: work.id }}
          className="rounded-item hover:underline"
        >
          {work.title}
        </Link>
      ),
  },
  {
    id: "added",
    header: "Added",
    cell: ({ entry, clock }) => (
      <time dateTime={entry.addedAt} title={sinceWords(entry.addedAt, clock)}>
        <Data>{agoWords(entry.addedAt, clock.now)}</Data>
      </time>
    ),
  },
];

export function BlocklistTable({
  entries,
  picking,
  clock,
}: {
  entries: readonly GrabsBlockEntry[];
  picking: Picking;
  clock: Clock;
}) {
  return (
    <Table
      label="Blocklist"
      lines={2}
      stackBelow={620}
      rows={entries.map((entry) => ({ entry, picking, clock }))}
      rowKey={({ entry }) => entry.infoHash}
      columns={columns}
    />
  );
}
