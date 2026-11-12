import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { WorksContentType, WorksItem, WorksLibraryRow } from "../api/gen/model";
import type { RootEntry } from "./root-entry";
import { useGetWorkSuspense } from "../api/gen/query/works/works";
import { Button } from "../ui/button";
import { Select } from "../ui/select";
import { codeOf } from "../works/fields-by-tier";
import { useImporting } from "./adding";
import { filesWords } from "./words";

/** The choice that leaves a file in the entry. */
const waiting = "";

type Chosen = Readonly<Record<string, string>>;

const nameOf = (path: string) => path.slice(path.lastIndexOf("/") + 1);

function startOf(
  entry: RootEntry,
  type: WorksContentType,
  row: WorksLibraryRow,
  items: readonly WorksItem[],
): Chosen {
  const suggested = entry.suggestion?.workId === row.id;
  return Object.fromEntries(
    entry.files.map(({ path, itemIds }) => {
      if (type.items === "one") {
        return [path, items[0]?.id ?? waiting];
      }
      return [path, suggested ? (itemIds[0] ?? waiting) : waiting];
    }),
  );
}

function FileChoices({
  entry,
  type,
  row,
  items,
  chosen,
  onChosen,
}: {
  entry: RootEntry;
  type: WorksContentType;
  row: WorksLibraryRow;
  items: readonly WorksItem[];
  chosen: Chosen;
  onChosen: (chosen: Chosen) => void;
}) {
  if (type.items === "one") {
    return (
      <p>
        {entry.files.length === 1
          ? `Its file joins ${row.label}.`
          : `Its ${filesWords(entry.files.length)} join ${row.label}.`}
      </p>
    );
  }
  const options = [
    { value: waiting, label: "Leave it waiting" },
    ...items.map((item) => ({ value: item.id, label: `${codeOf(type, item)} ${item.label}` })),
  ];
  return entry.files.map(({ path }) => (
    <Select
      key={path}
      label={nameOf(path)}
      options={options}
      value={chosen[path] ?? waiting}
      onValueChange={(itemId) => {
        onChosen({ ...chosen, [path]: itemId });
      }}
    />
  ));
}

function Picked({
  row,
  type,
  onBack,
}: {
  row: WorksLibraryRow;
  type: WorksContentType;
  onBack: () => void;
}) {
  return (
    <div className="flex flex-wrap items-center gap-3">
      <p className="flex-1 font-semibold">{row.label}</p>
      <Button look="ghost" size="small" onClick={onBack}>
        Pick another {type.work.one.toLowerCase()}
      </Button>
    </div>
  );
}

/** Each of the entry's files and the Item of `row`'s Work it holds; a one-Item type needs no choice. */
export function ImportFiles({
  entry,
  type,
  row,
  onBack,
  onDone,
}: {
  entry: RootEntry;
  type: WorksContentType;
  row: WorksLibraryRow;
  onBack: () => void;
  onDone: () => void;
}) {
  const { items } = useGetWorkSuspense(row.id).data;
  const [chosen, setChosen] = useState(() => startOf(entry, type, row, items));
  const files = entry.files.flatMap(({ path, position }) => {
    const itemId = chosen[path] ?? waiting;
    return itemId === waiting ? [] : [{ position, itemIds: [itemId] }];
  });
  const importing = useImporting(entry, row, onDone);
  return (
    <div className="flex flex-col gap-4">
      <Picked row={row} type={type} onBack={onBack} />
      <FileChoices
        entry={entry}
        type={type}
        row={row}
        items={items}
        chosen={chosen}
        onChosen={setChosen}
      />
      {importing.error !== null && <p className="text-fail-ink">{sentenceOf(importing.error)}</p>}
      <div className="flex justify-end">
        <Button
          look="primary"
          disabled={files.length === 0}
          busy={importing.isPending}
          onClick={() => {
            importing.mutate(files);
          }}
        >
          Import
        </Button>
      </div>
    </div>
  );
}
