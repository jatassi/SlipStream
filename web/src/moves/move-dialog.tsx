import { useState } from "react";

import type {
  MovesMoveAskOrigin,
  MovesMovePreview,
  RootsRoot,
  WorksContentType,
} from "../api/gen/model";
import { countWords } from "../format";
import { Checkbox } from "../ui/checkbox";
import type { Subject } from "../works/actions/actions";
import { DestinationPicker, useRootsOf } from "./destination-picker";
import { PreviewDialog } from "./preview-dialog";

type MoveDialogProps = {
  readonly type: WorksContentType;
  readonly works: readonly Subject[];
  /** The root they're in now, which the picker starts past; absent for a bulk pick. */
  readonly from?: string;
  readonly origin: MovesMoveAskOrigin;
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
};

/**
 * Moves Works with their files to another library root of their Content
 * Type, after a preview that gives each one a verdict.
 */
export function MoveDialog({ type, works, from, origin, open, onOpenChange }: MoveDialogProps) {
  const roots = useRootsOf(type);
  const [rootId, setRootId] = useState(
    (
      roots.find((root) => root.id !== from && root.default) ??
      roots.find((root) => root.id !== from)
    )?.id ?? "",
  );
  const [withTorrents, setWithTorrents] = useState(true);
  const noun = { one: type.work.one.toLowerCase(), many: type.work.many.toLowerCase() };
  return (
    <PreviewDialog
      title={titleOf(works, noun)}
      previewing={
        rootId === ""
          ? undefined
          : {
              kind: "ask",
              ask: {
                origin,
                works: works.map(({ id }) => ({ workId: id, rootId })),
                options: { includeLinkedTorrents: withTorrents },
              },
            }
      }
      picker={
        <>
          <DestinationPicker type={type} value={rootId} onValueChange={setRootId} />
          <Checkbox
            label={works.length === 1 ? "Also move its torrent" : "Also move their torrents"}
            checked={withTorrents}
            onCheckedChange={setWithTorrents}
          />
        </>
      }
      homedWords={(homed) =>
        `${countWords(homed, noun)} without files ${homed === 1 ? "goes" : "go"} into the new root too.`
      }
      startedWords={(started, plan) =>
        `Moving ${countWords(started - torrentsOf(plan), noun)} to ${plan.units.find(({ verdict }) => verdict !== "red")?.to.nodeName ?? "the new root"}.`
      }
      open={open}
      onOpenChange={onOpenChange}
    />
  );
}

const titleOf = (works: readonly Subject[], noun: { one: string; many: string }) => {
  const [only] = works;
  return works.length === 1 && only !== undefined
    ? `Move ${only.label}`
    : `Move ${countWords(works.length, noun)}`;
};

/** The movable torrent units a Work Move's plan took for its Works' linked torrents. */
const torrentsOf = (plan: MovesMovePreview) =>
  plan.units.filter(({ kind, verdict }) => kind === "torrent" && verdict !== "red").length;

/** The reverse of a finished batch: each Work it moved back to the root it left. */
export function MoveBackDialog({
  batchId,
  open,
  onOpenChange,
}: {
  batchId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <PreviewDialog
      title="Move back"
      previewing={{ kind: "back", batchId }}
      startedWords={(started, plan) =>
        started === 1
          ? `Moving ${plan.units.find(({ verdict }) => verdict !== "red")?.name ?? "it"} back.`
          : `Started ${countWords(started, { one: "Move", many: "Moves" })} back.`
      }
      open={open}
      onOpenChange={onOpenChange}
    />
  );
}

/** Moves every Work of a root to another root of its Content Type, deleting the emptied root when asked. */
export function RootMoveDialog({
  type,
  from,
  open,
  onOpenChange,
}: {
  type: WorksContentType;
  from: RootsRoot;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const roots = useRootsOf(type).filter((root) => root.id !== from.id);
  const [rootId, setRootId] = useState((roots.find((root) => root.default) ?? roots[0])?.id ?? "");
  const [deleteEmptied, setDeleteEmptied] = useState(false);
  const noun = { one: type.work.one.toLowerCase(), many: type.work.many.toLowerCase() };
  return (
    <PreviewDialog
      title={`Move the ${noun.many} in ${from.path}`}
      previewing={
        rootId === ""
          ? undefined
          : {
              kind: "ask",
              ask: {
                origin: "root",
                works: [],
                roots: [{ rootId: from.id, to: rootId }],
                options: { deleteEmptiedRoots: deleteEmptied },
              },
            }
      }
      picker={
        <>
          <DestinationPicker type={type} value={rootId} onValueChange={setRootId} />
          <Checkbox
            label="Delete this root when it's empty"
            checked={deleteEmptied}
            onCheckedChange={setDeleteEmptied}
          />
        </>
      }
      homedWords={(homed) =>
        `${countWords(homed, noun)} without files ${homed === 1 ? "goes" : "go"} into the new root too.`
      }
      startedWords={(started) => `Moving ${countWords(started, noun)} out of ${from.path}.`}
      open={open}
      onOpenChange={onOpenChange}
    />
  );
}
