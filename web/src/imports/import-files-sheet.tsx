import { Fragment, Suspense, useState } from "react";

import { sentenceOf } from "../api/errors";
import type {
  ImportsImportEntry,
  ImportsImportFile,
  ImportsImportFileOutcome,
  NodesNode,
  WorksContentType,
} from "../api/gen/model";
import {
  confirmImports,
  importFolder,
  setImportMoveFiles,
  useListImportsSuspense,
} from "../api/gen/query/imports/imports";
import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { useListRootsSuspense } from "../api/gen/query/roots/roots";
import { useListContentTypesSuspense, useListLibrarySuspense } from "../api/gen/query/works/works";
import { FolderBrowser } from "../folders/folder-browser";
import { purposeCheck } from "../roots/check";
import { Button } from "../ui/button";
// oxlint-disable-next-line import/max-dependencies -- the sheet picks a Node's folder, opens its Imports and acts on their files
import { Data } from "../ui/data";
import { Dialog } from "../ui/dialog";
import { Select } from "../ui/select";
import { Skeleton } from "../ui/skeleton";
import { Switch } from "../ui/switch";
import { toast } from "../ui/toast";
import { nameOf } from "../works/paths";
import { FileRow } from "./file-row";
import { filesWords } from "./words";

/** The folder Imports the sheet's rows show, as opening a folder made them. */
export type Opened = { readonly importIds: readonly string[] };

const outcomeWords: { readonly [O in ImportsImportFileOutcome]: string } = {
  pending: "Reading",
  left_out: "Left out",
  not_better: "Not better than the file you have",
  waiting: "Waiting for you",
  sending: "Sending",
  placing: "Importing",
  placed: "Imported",
  declined: "Not imported",
  failed: "Failed",
};

function useOpening(onOpened: (opened: Opened) => void) {
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const open = async (nodeId: string, folder: string) => {
    setBusy(true);
    setFailure(undefined);
    try {
      const { imports } = await importFolder({ nodeId, folder });
      onOpened({ importIds: imports.map(({ id }) => id) });
    } catch (error) {
      setFailure(sentenceOf(error));
    } finally {
      setBusy(false);
    }
  };
  return { busy, failure, open };
}

function Browser({
  node,
  open,
  onOpenChange,
  onPick,
}: {
  node: NodesNode;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onPick: (folder: string) => void;
}) {
  const { roots } = useListRootsSuspense().data;
  const { contentTypes } = useListContentTypesSuspense().data;
  return (
    <FolderBrowser
      nodeId={node.id}
      nodeName={node.name}
      open={open}
      onOpenChange={onOpenChange}
      startPath=""
      purpose="import"
      check={purposeCheck("import", { node: node.name, roots, types: contentTypes })}
      confirmVerb="Open"
      allowNewFolder={false}
      onPick={onPick}
    />
  );
}

function FolderStep({ onOpened }: { onOpened: (opened: Opened) => void }) {
  const { nodes } = useListNodesSuspense().data;
  const [nodeId, setNodeId] = useState(nodes[0]?.id ?? "");
  const [browsing, setBrowsing] = useState(false);
  const { busy, failure, open } = useOpening(onOpened);
  const node = nodes.find(({ id }) => id === nodeId);
  return (
    <div className="flex flex-col gap-4">
      <p>
        Pick a folder in one of the Node&apos;s library roots or download folders. SlipStream reads
        each video in it and shows what it would fill.
      </p>
      {nodes.length > 1 && (
        <Select
          label="Node"
          options={nodes.map(({ id, name }) => ({ value: id, label: name }))}
          value={nodeId}
          onValueChange={setNodeId}
        />
      )}
      {failure !== undefined && (
        <p role="alert" className="text-fail-ink">
          {failure}
        </p>
      )}
      <div className="flex justify-end">
        <Button
          look="primary"
          busy={busy}
          onClick={() => {
            setBrowsing(true);
          }}
        >
          Choose a folder
        </Button>
      </div>
      {node !== undefined && (
        <Browser
          node={node}
          open={browsing}
          onOpenChange={setBrowsing}
          onPick={(folder) => {
            setBrowsing(false);
            void open(node.id, folder);
          }}
        />
      )}
    </div>
  );
}

function SettledFile({ file }: { file: ImportsImportFile }) {
  return (
    <li className="flex min-w-0 flex-col gap-0.5">
      <span className="font-semibold text-ink-1 wrap-anywhere">
        <Data>{nameOf(file.path)}</Data>
      </span>
      <span className="text-ink-2">
        {outcomeWords[file.outcome]}
        {file.reason?.sentence === undefined || file.reason.sentence === null
          ? ""
          : `. ${file.reason.sentence}`}
      </span>
    </li>
  );
}

function ImportRows({ entry, type }: { entry: ImportsImportEntry; type: WorksContentType }) {
  const { rows } = useListLibrarySuspense({ contentType: type.id }).data;
  const work = rows.find(({ id }) => id === entry.workId)?.label ?? `No ${type.work.one}`;
  return (
    <section aria-label={work} className="flex flex-col gap-3">
      <h3 className="font-semibold text-ink-1">{work}</h3>
      <ul className="flex flex-col gap-4">
        {entry.files.map((file) =>
          file.outcome === "waiting" ? (
            <Fragment key={file.position}>
              <FileRow entry={entry} file={file} type={type} items={[]} />
              {!file.movable && (
                <li className="text-meta text-ink-3">
                  A torrent seeds {nameOf(file.path)}. Move files can&apos;t take a file while it
                  seeds.
                </li>
              )}
            </Fragment>
          ) : (
            <SettledFile key={file.position} file={file} />
          ),
        )}
      </ul>
    </section>
  );
}

function useBatch(entries: readonly ImportsImportEntry[], onDone: () => void) {
  const importIds = entries.map(({ id }) => id);
  const [busy, setBusy] = useState(false);
  const confirming = entries.flatMap(({ files }) =>
    files.filter(({ outcome, recommended }) => outcome === "waiting" && recommended === "confirm"),
  ).length;
  const run = async (send: () => Promise<unknown>, done: () => void) => {
    setBusy(true);
    try {
      await send();
      done();
    } catch (error) {
      toast(sentenceOf(error), { kind: "fail" });
    } finally {
      setBusy(false);
    }
  };
  return {
    busy,
    confirming,
    importAll: () =>
      run(
        () => confirmImports({ importIds }),
        () => {
          toast(`Importing ${filesWords(confirming)}.`);
          onDone();
        },
      ),
    moveFiles: (on: boolean) =>
      run(
        () => setImportMoveFiles({ importIds, on }),
        () => {},
      ),
  };
}

function RowsStep({ opened, onDone }: { opened: Opened; onDone: () => void }) {
  const { imports } = useListImportsSuspense({ source: "folder" }).data;
  const { contentTypes } = useListContentTypesSuspense().data;
  const entries = imports.filter(({ id }) => opened.importIds.includes(id));
  const { busy, confirming, importAll, moveFiles } = useBatch(entries, onDone);
  return (
    <div className="flex flex-col gap-6">
      {entries.map((entry) =>
        contentTypes
          .filter(({ id }) => id === entry.contentType)
          .map((type) => <ImportRows key={entry.id} entry={entry} type={type} />),
      )}
      <div className="flex flex-col items-start gap-1">
        <Switch
          label="Move files"
          checked={entries.length > 0 && entries.every(({ moveFiles: on }) => on)}
          disabled={busy}
          onCheckedChange={(on) => void moveFiles(on)}
        />
        <p className="text-meta text-ink-3">
          On moves each file into the library and deletes it here. Off keeps the files in this
          folder.
        </p>
      </div>
      <div className="flex justify-end">
        <Button
          look="primary"
          busy={busy}
          disabled={confirming === 0}
          onClick={() => void importAll()}
        >
          Import all
        </Button>
      </div>
    </div>
  );
}

/**
 * Import files: pick a Node's folder, then each video in it with what it
 * would fill, Import on each and Import all. Closing it keeps the folder's
 * Imports in Pending imports.
 */
export function ImportFilesSheet({
  start,
  onClose,
}: {
  start?: Opened | undefined;
  onClose: () => void;
}) {
  const [opened, setOpened] = useState(start);
  return (
    <Dialog
      title="Import files"
      size={680}
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      <Suspense fallback={<Skeleton label="Loading the files" />}>
        {opened === undefined ? (
          <FolderStep onOpened={setOpened} />
        ) : (
          <RowsStep opened={opened} onDone={onClose} />
        )}
      </Suspense>
    </Dialog>
  );
}
