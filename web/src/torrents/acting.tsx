import { type ReactNode, useState } from "react";

import { sentenceOf } from "../api/errors";
import { actOnTorrents } from "../api/gen/query/downloadclients/downloadclients";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { toast } from "../ui/toast";
import { TorrentMoveDialog } from "../moves/client-move-dialog";
import { type Acted, type Subject, type TorrentAction, outcomeWords } from "./actions";
import { PauseDialog } from "./pause-dialog";
import { RemoveDialog } from "./remove-dialog";
import type { Parting, WorkNames } from "./remove-preview";

type Asking = {
  readonly action: Parting | "recheck" | "pause" | "move";
  readonly subjects: readonly Subject[];
};

async function actAndSay(
  action: Acted,
  subjects: readonly Subject[],
  deleteFiles: boolean,
  work: WorkNames,
) {
  const { results } = await actOnTorrents({
    action,
    ids: subjects.map(({ id }) => id),
    ...(deleteFiles && { deleteFiles }),
  });
  const { done, refused } = outcomeWords(results, subjects, {
    action,
    work: work.of,
    clock: { now: new Date(), zone: undefined },
  });
  if (done !== undefined) {
    toast(done);
  }
  for (const sentence of refused) {
    toast(sentence, { kind: "fail" });
  }
}

function failed(error: unknown) {
  toast(sentenceOf(error), { kind: "fail" });
}

/**
 * Runs Torrent actions and says how each went. Remove and mark failed ask
 * first, and so do a recheck that would read library files and a pause of
 * a torrent that still owes its Tracker; `dialogs` is where those
 * questions render.
 */
export function useActing(work: WorkNames): {
  start: (action: TorrentAction, subjects: readonly Subject[]) => void;
  dialogs: ReactNode;
} {
  const [asking, setAsking] = useState<Asking>();
  const [open, setOpen] = useState(false);

  const act = (action: Acted, subjects: readonly Subject[], deleteFiles = false) =>
    actAndSay(action, subjects, deleteFiles, work);

  const ask = (next: Asking) => {
    setAsking(next);
    setOpen(true);
  };

  const begin = async (action: TorrentAction, subjects: readonly Subject[]) => {
    if (
      action === "move" ||
      action === "remove" ||
      action === "mark-failed" ||
      (action === "pause" && subjects.some(({ owing }) => owing !== undefined))
    ) {
      ask({ action, subjects });
      return;
    }
    if (action === "recheck") {
      const { preview } = await actOnTorrents({
        action,
        ids: subjects.map(({ id }) => id),
        dryRun: true,
      });
      if (preview?.linked === true) {
        ask({ action, subjects });
        return;
      }
    }
    await act(action, subjects);
  };

  return {
    start: (action, subjects) => {
      begin(action, subjects).catch(failed);
    },
    dialogs: asking !== undefined && (
      <Asked asking={asking} open={open} onOpenChange={setOpen} act={act} work={work} />
    ),
  };
}

function Asked({
  asking,
  open,
  onOpenChange,
  act,
  work,
}: {
  asking: Asking;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  act: (action: Acted, subjects: readonly Subject[], deleteFiles?: boolean) => Promise<void>;
  work: WorkNames;
}) {
  const parting = asking.action;
  if (parting === "move") {
    return <MoveAsked subjects={asking.subjects} open={open} onOpenChange={onOpenChange} />;
  }
  if (parting === "pause") {
    return (
      <PauseDialog
        subjects={asking.subjects}
        open={open}
        onOpenChange={onOpenChange}
        onConfirm={() => act("pause", asking.subjects)}
      />
    );
  }
  return parting === "recheck" ? (
    <RecheckDialog
      subjects={asking.subjects}
      open={open}
      onOpenChange={onOpenChange}
      onConfirm={() => act("recheck", asking.subjects)}
    />
  ) : (
    <RemoveDialog
      key={asking.subjects.map(({ id }) => id).join()}
      parting={parting}
      subjects={asking.subjects}
      work={work}
      open={open}
      onOpenChange={onOpenChange}
      onConfirm={(going, deleteFiles) => act(parting, going, deleteFiles)}
    />
  );
}

function MoveAsked({
  subjects,
  ...props
}: {
  subjects: readonly Subject[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const torrents = subjects.map(({ id, name, nodeId, clientId }) => ({
    id,
    name,
    nodeId: nodeId ?? "",
    clientId,
  }));
  return <TorrentMoveDialog torrents={torrents} {...props} />;
}

function RecheckDialog({
  subjects,
  open,
  onOpenChange,
  onConfirm,
}: {
  subjects: readonly Subject[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => Promise<void>;
}) {
  const names = new Intl.ListFormat("en", { type: "conjunction" }).format(
    subjects.map(({ name }) => name),
  );
  return (
    <ConfirmDialog
      title={`Recheck ${names}?`}
      confirmLabel="Recheck"
      open={open}
      onOpenChange={onOpenChange}
      onConfirm={onConfirm}
    >
      <p>
        Some of its files are also files in your library. The recheck reads them, and any piece that
        no longer matches downloads again, which changes the library file too.
      </p>
    </ConfirmDialog>
  );
}
