import { Link } from "@tanstack/react-router";
import { type ReactNode, Suspense, useState } from "react";

import type {
  ImportsImportEntry,
  ImportsLead,
  ListImportsParams,
  WorksContentType,
} from "../api/gen/model";
import { type RootEntry, usePendingImports } from "./root-entry";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { Empty } from "../shell/empty";
import { Page } from "../shell/page";
import { Button } from "../ui/button";
import { Data } from "../ui/data";
import { List, ListRow } from "../ui/list";
// oxlint-disable-next-line import/max-dependencies -- the page lists downloads, roots and folders, each kind with its own entry
import { Section } from "../ui/section";
import { useShell } from "../ui/shell-context";
import { Skeleton } from "../ui/skeleton";
import { useAdding } from "./adding";
import { DownloadEntry, waitingFiles } from "./download-entry";
import { type FolderGroup, FolderEntry, foldersOf } from "./folder-entry";
import { ImportFilesSheet } from "./import-files-sheet";
import { type Group, RootGroup } from "./root-group";
import { filesWords, isAddable } from "./words";

export type PendingView = "needs-you" | "ignored";

export const pendingParams = (view: PendingView): ListImportsParams =>
  view === "ignored" ? { source: "root", ignored: true } : { waiting: true };

type Download = { readonly entry: ImportsImportEntry; readonly type: WorksContentType };

/** Pending imports' groups in their fixed order: downloads, then each root, then folders imported by hand. */
export type PendingGroup =
  | { readonly kind: "downloads"; readonly downloads: readonly Download[] }
  | { readonly kind: "root"; readonly group: Group }
  | { readonly kind: "folders"; readonly folders: readonly FolderGroup[] };

function rootGroupsOf(
  leads: readonly ImportsLead[],
  entries: readonly RootEntry[],
  types: readonly WorksContentType[],
): Group[] {
  const roots = [
    ...new Map(
      [
        ...leads.map(({ root, contentType }) => ({ root, contentType })),
        ...entries.map(({ root, contentType }) => ({ root, contentType })),
      ].map((found) => [found.root.id, found]),
    ).values(),
  ];
  return roots.flatMap(({ root, contentType }) =>
    types
      .filter(({ id }) => id === contentType)
      .map((type) => ({
        root,
        type,
        lead: leads.find((lead) => lead.root.id === root.id),
        entries: entries.filter((entry) => entry.root.id === root.id),
      })),
  );
}

export function groupsOf(
  view: PendingView,
  { leads, imports, entries }: ReturnType<typeof usePendingImports>,
  types: readonly WorksContentType[],
): PendingGroup[] {
  const downloads = imports.flatMap((entry) =>
    entry.source === "grab"
      ? types.filter(({ id }) => id === entry.contentType).map((type) => ({ entry, type }))
      : [],
  );
  const folders = foldersOf(imports.filter(({ source }) => source === "folder"));
  return [
    ...(downloads.length === 0 ? [] : [{ kind: "downloads", downloads } as const]),
    ...rootGroupsOf(view === "ignored" ? [] : leads, entries, types).map(
      (group) => ({ kind: "root", group }) as const,
    ),
    ...(folders.length === 0 ? [] : [{ kind: "folders", folders } as const]),
  ];
}

function AddAll({ entries }: { entries: readonly RootEntry[] }) {
  const { add, busy } = useAdding();
  const addable = entries.filter((entry) => isAddable(entry));
  if (addable.length < 2) {
    return null;
  }
  return (
    <Button
      busy={busy}
      onClick={() => {
        add(addable);
      }}
    >
      Add all {addable.length}
    </Button>
  );
}

function ImportFiles() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        onClick={() => {
          setOpen(true);
        }}
      >
        Import files
      </Button>
      {open && (
        <ImportFilesSheet
          onClose={() => {
            setOpen(false);
          }}
        />
      )}
    </>
  );
}

function waitWords(entry: ImportsImportEntry): string {
  const count = waitingFiles(entry).length;
  return `${filesWords(count)} ${count === 1 ? "waits" : "wait"} for you`;
}

function Downloads({ downloads }: { downloads: readonly Download[] }) {
  const phone = useShell() === "phone";
  return (
    <Section title="From downloads">
      {phone ? (
        <List label="Pending imports from downloads">
          {downloads.map(({ entry }) => (
            <ListRow
              key={entry.id}
              title={<Data>{entry.grab?.release}</Data>}
              meta={waitWords(entry)}
              render={<Link to="/library/pending/$importId" params={{ importId: entry.id }} />}
            />
          ))}
        </List>
      ) : (
        <ul aria-label="Pending imports from downloads" className="flex flex-col gap-2">
          {downloads.map(({ entry, type }) => (
            <Suspense key={entry.id} fallback={<Skeleton label="Loading the download" />}>
              <DownloadEntry entry={entry} type={type} />
            </Suspense>
          ))}
        </ul>
      )}
    </Section>
  );
}

function Folders({ folders }: { folders: readonly FolderGroup[] }) {
  return (
    <Section title="Imported by hand">
      <ul aria-label="Pending imports by hand" className="flex flex-col gap-2">
        {folders.map((group) => (
          <FolderEntry key={JSON.stringify([group.nodeId, group.folder])} group={group} />
        ))}
      </ul>
    </Section>
  );
}

function PendingGroupOf({ group }: { group: PendingGroup }) {
  switch (group.kind) {
    case "downloads":
      return <Downloads downloads={group.downloads} />;
    case "root":
      return <RootGroup group={group.group} />;
    case "folders":
      return <Folders folders={group.folders} />;
    default:
      return group satisfies never;
  }
}

const keyOf = (group: PendingGroup) => (group.kind === "root" ? group.group.root.id : group.kind);

const empties: { readonly [V in PendingView]: string } = {
  "needs-you":
    "Nothing needs you. A download SlipStream can't import without you, and each folder a library scan can't match to a Work with confidence, waits here.",
  ignored: "No folders are ignored.",
};

/**
 * Pending imports: the downloads that wait for the Admin, then what each
 * root's last library scan added and the folders it couldn't match, then the
 * folders the Admin opened with Import files. The Ignored view lists the
 * folders the Admin set aside.
 */
export function PendingPage({ view, beside }: { view: PendingView; beside: ReactNode }) {
  const pending = usePendingImports(pendingParams(view));
  const { contentTypes } = useListContentTypesSuspense().data;
  const groups = groupsOf(view, pending, contentTypes);
  return (
    <Page
      beside={beside}
      action={
        view === "ignored" ? undefined : (
          <div className="flex gap-2">
            <AddAll entries={pending.entries} />
            <ImportFiles />
          </div>
        )
      }
    >
      {groups.length === 0 ? (
        <Empty>{empties[view]}</Empty>
      ) : (
        groups.map((group) => <PendingGroupOf key={keyOf(group)} group={group} />)
      )}
    </Page>
  );
}
