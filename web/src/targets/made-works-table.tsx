import { useState } from "react";

import type { TargetsMadeRow } from "../api/gen/model";
import { bytes, spanWords } from "../format";
import { Checkbox } from "../ui/checkbox";
import { useNow } from "../ui/clock";
import { PlainTime } from "../ui/relative-time";
import { SearchField } from "../ui/search-field";
import { Select } from "../ui/select";
import { Switch } from "../ui/switch";
import { type Column, Table } from "../ui/table";

type Sort = "title" | "added" | "seconds" | "bytes";

const sorts: readonly { value: Sort; label: string }[] = [
  { value: "title", label: "Title" },
  { value: "added", label: "Recently added" },
  { value: "seconds", label: "Longest to make" },
  { value: "bytes", label: "Largest" },
];

const order: Readonly<Record<Sort, (a: TargetsMadeRow, b: TargetsMadeRow) => number>> = {
  title: (a, b) => a.title.localeCompare(b.title),
  added: (a, b) => b.addedAt.localeCompare(a.addedAt),
  seconds: (a, b) => b.seconds - a.seconds,
  bytes: (a, b) => b.bytes - a.bytes,
};

const recentDays = 30;
const dayMs = 86_400_000;

const grouped = new Intl.NumberFormat("en");

/** The picked Works' totals, such as "3 Works picked, about 2 h 10 min and 4.2 GB." */
export function pickedWords(rows: readonly TargetsMadeRow[], picked: ReadonlySet<string>): string {
  const chosen = rows.filter((row) => picked.has(row.workId));
  if (chosen.length === 0) {
    return "No Works picked.";
  }
  const seconds = chosen.reduce((sum, row) => sum + row.seconds, 0);
  const size = chosen.reduce((sum, row) => sum + row.bytes, 0);
  const works = chosen.length === 1 ? "1 Work" : `${grouped.format(chosen.length)} Works`;
  return `${works} picked, about ${spanWords(seconds)} and ${bytes(size)}.`;
}

type Pick = (ids: readonly string[], on: boolean) => void;

const columnsOf = (
  picked: ReadonlySet<string>,
  pick: Pick,
): readonly [Column<TargetsMadeRow>, ...Column<TargetsMadeRow>[]] => [
  {
    id: "work",
    header: "Work",
    cell: (row) => (
      <Checkbox
        label={row.title}
        checked={picked.has(row.workId)}
        onCheckedChange={(on) => {
          pick([row.workId], on);
        }}
      />
    ),
  },
  { id: "added", header: "Added", cell: (row) => <PlainTime at={row.addedAt} form="age" /> },
  { id: "time", header: "Time", align: "end", cell: (row) => spanWords(row.seconds) },
  { id: "size", header: "Size", align: "end", cell: (row) => bytes(row.bytes) },
];

type View = { readonly query: string; readonly sort: Sort; readonly recent: boolean };

function Controls({ view, onView }: { view: View; onView: (view: View) => void }) {
  return (
    <div className="flex flex-wrap items-end gap-3">
      <SearchField
        label="Search Works"
        value={view.query}
        onValueChange={(query) => {
          onView({ ...view, query });
        }}
      />
      <div className="w-48">
        <Select
          label="Sort by"
          options={sorts}
          value={view.sort}
          onValueChange={(sort) => {
            onView({ ...view, sort });
          }}
        />
      </div>
      <Switch
        label={`Added in the last ${String(recentDays)} days`}
        checked={view.recent}
        onCheckedChange={(recent) => {
          onView({ ...view, recent });
        }}
      />
    </div>
  );
}

function PickShown({
  shown,
  picked,
  pick,
}: {
  shown: readonly TargetsMadeRow[];
  picked: ReadonlySet<string>;
  pick: Pick;
}) {
  const count = shown.filter((row) => picked.has(row.workId)).length;
  return (
    <Checkbox
      label="Pick every Work shown"
      checked={shown.length > 0 && count === shown.length}
      indeterminate={count > 0 && count < shown.length}
      onCheckedChange={(on) => {
        pick(
          shown.map((row) => row.workId),
          on,
        );
      }}
    />
  );
}

/**
 * The Works a save would make files for, to pick from: searchable, sortable,
 * narrowed to recently added Works on request, with the picked Works' totals.
 */
export function MadeWorksTable({
  label,
  rows,
  picked,
  onPicked,
}: {
  label: string;
  rows: readonly TargetsMadeRow[];
  picked: ReadonlySet<string>;
  onPicked: (picked: ReadonlySet<string>) => void;
}) {
  const since = useNow().getTime() - recentDays * dayMs;
  const [view, setView] = useState<View>({ query: "", sort: "title", recent: false });
  const shown = rows
    .filter((row) => row.title.toLowerCase().includes(view.query.trim().toLowerCase()))
    .filter((row) => !view.recent || Date.parse(row.addedAt) >= since)
    .toSorted(order[view.sort]);
  const pick: Pick = (ids, on) => {
    const next = new Set(picked);
    for (const id of ids) {
      if (on) {
        next.add(id);
      } else {
        next.delete(id);
      }
    }
    onPicked(next);
  };
  return (
    <div className="flex flex-col gap-3">
      <Controls view={view} onView={setView} />
      <PickShown shown={shown} picked={picked} pick={pick} />
      <Table
        label={label}
        columns={columnsOf(picked, pick)}
        rows={shown}
        rowKey={(row) => row.workId}
        stackBelow={520}
      />
      <output className="text-ink-1">{pickedWords(rows, picked)}</output>
    </div>
  );
}
