import { useState } from "react";

import type { ImportsImportEntry } from "../api/gen/model";
import { useListLibrarySuspense } from "../api/gen/query/works/works";
import { Button } from "../ui/button";
import { Data } from "../ui/data";
import { useNodeName, waitingFiles } from "./download-entry";
import { ImportFilesSheet } from "./import-files-sheet";
import { filesWords } from "./words";

const list = new Intl.ListFormat("en", { type: "conjunction" });

/** One folder an Admin opened with Import files, as one Import per Work. */
export type FolderGroup = {
  readonly nodeId: string | null;
  readonly folder: string;
  readonly entries: readonly ImportsImportEntry[];
};

/** A folder's Imports, gathered by the Node and folder they came from. */
export function foldersOf(entries: readonly ImportsImportEntry[]): FolderGroup[] {
  const groups = new Map<string, FolderGroup>();
  for (const entry of entries) {
    const key = JSON.stringify([entry.nodeId, entry.folder]);
    const seen = groups.get(key);
    groups.set(key, {
      nodeId: entry.nodeId,
      folder: entry.folder,
      entries: [...(seen?.entries ?? []), entry],
    });
  }
  return [...groups.values()];
}

/** A manual import left open: its folder, Node and Works, with Continue opening its rows. */
export function FolderEntry({ group }: { group: FolderGroup }) {
  const { rows } = useListLibrarySuspense().data;
  const nodeName = useNodeName();
  const [open, setOpen] = useState(false);
  const works = group.entries.flatMap(({ workId }) =>
    rows.filter(({ id }) => id === workId).map(({ label }) => label),
  );
  const waiting = group.entries.flatMap((entry) => waitingFiles(entry)).length;
  return (
    <li
      aria-label={group.folder}
      className="flex flex-col items-start gap-2 rounded-card bg-glass p-4 ring-1 ring-hairline ring-inset"
    >
      <div className="flex min-w-0 flex-col gap-0.5">
        <span className="font-semibold text-ink-1 wrap-anywhere">
          <Data>{group.folder}</Data>
        </span>
        <span className="text-meta text-ink-3">
          {[
            ...(works.length === 0 ? [] : [list.format(works)]),
            `On ${nodeName(group.nodeId)}`,
          ].join(" · ")}
        </span>
      </div>
      <p>
        {filesWords(waiting)} {waiting === 1 ? "waits" : "wait"} for you to import{" "}
        {waiting === 1 ? "it" : "them"}.
      </p>
      <Button
        size="small"
        look="primary"
        onClick={() => {
          setOpen(true);
        }}
      >
        Continue
      </Button>
      {open && (
        <ImportFilesSheet
          start={{ importIds: group.entries.map(({ id }) => id) }}
          onClose={() => {
            setOpen(false);
          }}
        />
      )}
    </li>
  );
}
