import { Suspense, useState } from "react";

import type { ImportsEntry, WorksContentType } from "../api/gen/model";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { SearchField } from "../ui/search-field";
import { Skeleton } from "../ui/skeleton";
import { MatchResults } from "./match-results";

/** Searches the type's source for the record an entry holds, and makes the one picked its suggestion. */
export function MatchSheet({
  entry,
  type,
  onClose,
}: {
  entry: ImportsEntry;
  type: WorksContentType;
  onClose: () => void;
}) {
  const start =
    entry.hint.year === null ? entry.hint.title : `${entry.hint.title} ${entry.hint.year}`;
  const [draft, setDraft] = useState(start);
  const [query, setQuery] = useState(start);
  return (
    <Dialog
      title={`Change the match for ${entry.folder}`}
      size={680}
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      <div className="flex flex-col gap-4">
        <p>
          Search {type.source} for the {type.work.one.toLowerCase()} in this folder.
        </p>
        <search>
          <form
            className="flex items-center gap-2"
            onSubmit={(event) => {
              event.preventDefault();
              setQuery(draft.trim());
            }}
          >
            <SearchField label="Title" value={draft} onValueChange={setDraft} className="flex-1" />
            <Button type="submit" disabled={draft.trim() === ""}>
              Search
            </Button>
          </form>
        </search>
        <Suspense fallback={<Skeleton label={`Searching ${type.source}`} />}>
          <MatchResults entry={entry} type={type} query={query} onDone={onClose} />
        </Suspense>
      </div>
    </Dialog>
  );
}
