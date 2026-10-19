import { FolderAddIcon } from "@hugeicons/core-free-icons";
import { type KeyboardEvent, type ReactNode, useId } from "react";

import { Button } from "../ui/button";
import { Icon } from "../ui/icon";
import type { NewFolder } from "./browser-state";
import { row, ruled } from "./folder-rows";

export function SkeletonRows() {
  return (
    <output aria-label="Loading folders" className="flex flex-col">
      {["a", "b", "c", "d", "e", "f"].map((key) => (
        <span key={key} className={`${row} ${ruled} min-h-(--row-height)`}>
          <span className="size-4.5 rounded-chip bg-fill" />
          <span className="h-3 w-48 max-w-[60%] rounded-chip bg-fill" />
        </span>
      ))}
    </output>
  );
}

export function Quiet({ children }: { children: ReactNode }) {
  return (
    <p className="px-4 py-6 text-body text-ink-2 in-data-[density=touch]:text-body-touch">
      {children}
    </p>
  );
}

const selectOnMount = (element: HTMLInputElement | null) => {
  element?.focus();
  element?.select();
};

export function NewFolderRow({
  folder,
  busy,
  onName,
  onKeyDown,
  onCreate,
  onCancel,
}: {
  folder: NewFolder;
  busy: boolean;
  onName: (name: string) => void;
  onKeyDown: (event: KeyboardEvent<HTMLInputElement>) => void;
  onCreate: () => void;
  onCancel: () => void;
}) {
  const errorId = useId();
  return (
    <div className="shadow-[inset_0_-1px_0_var(--color-hairline)]">
      <div className={`${row} min-h-(--row-height)`}>
        <Icon icon={FolderAddIcon} className="size-4.5 shrink-0 text-ink-2" />
        <input
          ref={selectOnMount}
          aria-label="New folder name"
          aria-invalid={folder.error !== null}
          aria-describedby={folder.error === null ? undefined : errorId}
          value={folder.name}
          autoComplete="off"
          spellCheck={false}
          onChange={(event) => {
            onName(event.target.value);
          }}
          onKeyDown={onKeyDown}
          className="h-7 min-w-0 flex-1 rounded-item bg-well px-2 text-body text-ink-1 ring-1 ring-hairline ring-inset in-data-[density=touch]:h-(--control-height) in-data-[density=touch]:text-input-touch"
        />
        <Button size="small" look="ghost" onClick={onCancel}>
          Cancel
        </Button>
        <Button size="small" busy={busy} onClick={onCreate}>
          Create
        </Button>
      </div>
      {folder.error !== null && (
        <p id={errorId} className="px-4 pb-2 text-meta text-fail-ink">
          {folder.error}
        </p>
      )}
    </div>
  );
}
