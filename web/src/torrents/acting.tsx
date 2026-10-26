import { type ReactNode, useState } from "react";

import { sentenceOf } from "../api/errors";
import { actOnTorrents } from "../api/gen/query/downloadclients/downloadclients";
import { useProjection } from "../library/projection";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { toast } from "../ui/toast";
import { type Subject, type TorrentAction, outcomeWords } from "./actions";
import { RemoveDialog } from "./remove-dialog";
import type { Parting } from "./remove-preview";

type Asking = {
  readonly action: Parting | "recheck";
  readonly subjects: readonly Subject[];
};

function useWorkNames(): (id: string) => string | undefined {
  const works = useProjection().byId;
  return (id) => works.get(id)?.label;
}

async function actAndSay(
  action: TorrentAction,
  subjects: readonly Subject[],
  deleteFiles: boolean,
  work: (id: string) => string | undefined,
) {
  const { results } = await actOnTorrents({
    action,
    ids: subjects.map(({ id }) => id),
    ...(deleteFiles && { deleteFiles }),
  });
  const { done, refused } = outcomeWords(results, subjects, {
    action,
    work,
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
 * first, and so does a recheck that would read library files; `dialogs` is
 * where those questions render.
 */
export function useActing(): {
  start: (action: TorrentAction, subjects: readonly Subject[]) => void;
  dialogs: ReactNode;
} {
  const work = useWorkNames();
  const [asking, setAsking] = useState<Asking>();
  const [open, setOpen] = useState(false);

  const act = (action: TorrentAction, subjects: readonly Subject[], deleteFiles = false) =>
    actAndSay(action, subjects, deleteFiles, work);

  const ask = (next: Asking) => {
    setAsking(next);
    setOpen(true);
  };

  const begin = async (action: TorrentAction, subjects: readonly Subject[]) => {
    if (action === "remove" || action === "mark-failed") {
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
  act: (
    action: TorrentAction,
    subjects: readonly Subject[],
    deleteFiles?: boolean,
  ) => Promise<void>;
  work: (id: string) => string | undefined;
}) {
  const parting = asking.action;
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
