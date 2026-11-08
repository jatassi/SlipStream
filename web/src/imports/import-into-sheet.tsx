import { Suspense, useState } from "react";

import type { ImportsEntry, WorksContentType, WorksLibraryRow } from "../api/gen/model";
import { useListLibrarySuspense } from "../api/gen/query/works/works";
import { Dialog } from "../ui/dialog";
import { List, ListRow } from "../ui/list";
import { SearchField } from "../ui/search-field";
import { Skeleton } from "../ui/skeleton";
import { ImportFiles } from "./import-files";

const shown = 25;

const letters = (text: string) => text.toLowerCase().replaceAll(/[^\p{L}\p{N}]/gu, "");

function WorkPick({
  rows,
  type,
  onPick,
}: {
  rows: readonly WorksLibraryRow[];
  type: WorksContentType;
  onPick: (row: WorksLibraryRow) => void;
}) {
  const [query, setQuery] = useState("");
  const found = rows
    .filter((row) => letters(row.label).includes(letters(query)))
    .toSorted((a, b) => a.sortLabel.localeCompare(b.sortLabel))
    .slice(0, shown);
  return (
    <div className="flex flex-col gap-4">
      <SearchField
        label={`Find a ${type.work.one.toLowerCase()}`}
        value={query}
        onValueChange={setQuery}
      />
      {found.length === 0 ? (
        <p className="text-ink-2">
          No {type.work.many.toLowerCase()} in this library root match &quot;{query}&quot;.
        </p>
      ) : (
        <List label={type.work.many}>
          {found.map((row) => (
            <ListRow
              key={row.id}
              title={row.label}
              render={
                <button
                  type="button"
                  aria-label={row.label}
                  onClick={() => {
                    onPick(row);
                  }}
                />
              }
            />
          ))}
        </List>
      )}
    </div>
  );
}

function Steps({
  entry,
  type,
  onDone,
}: {
  entry: ImportsEntry;
  type: WorksContentType;
  onDone: () => void;
}) {
  const rows = useListLibrarySuspense({ contentType: type.id }).data.rows.filter(
    ({ rootId }) => rootId === entry.root.id,
  );
  const [picked, setPicked] = useState(rows.find(({ id }) => id === entry.suggestion?.workId));
  if (picked === undefined) {
    if (rows.length === 0) {
      return (
        <p>
          No {type.work.many.toLowerCase()} are in the {entry.root.label} yet. Add the{" "}
          {type.work.one.toLowerCase()} first.
        </p>
      );
    }
    return <WorkPick rows={rows} type={type} onPick={setPicked} />;
  }
  return (
    <Suspense fallback={<Skeleton label={`Loading ${picked.label}`} />}>
      <ImportFiles
        entry={entry}
        type={type}
        row={picked}
        onBack={() => {
          setPicked(undefined);
        }}
        onDone={onDone}
      />
    </Suspense>
  );
}

/** Attaches an entry's files to a Work already in its root, at the Item each file holds. */
export function ImportIntoSheet({
  entry,
  type,
  onClose,
}: {
  entry: ImportsEntry;
  type: WorksContentType;
  onClose: () => void;
}) {
  return (
    <Dialog
      title={`Import into a ${type.work.one}`}
      size={680}
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      <Suspense fallback={<Skeleton label={`Loading the ${type.work.many.toLowerCase()}`} />}>
        <Steps entry={entry} type={type} onDone={onClose} />
      </Suspense>
    </Dialog>
  );
}
