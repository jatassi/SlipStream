import { useState } from "react";

import type { MovesMoveAskOrigin, WorksContentType } from "../api/gen/model";
import { countWords } from "../format";
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
  const noun = { one: type.work.one.toLowerCase(), many: type.work.many.toLowerCase() };
  const [only] = works;
  return (
    <PreviewDialog
      title={
        works.length === 1 && only !== undefined
          ? `Move ${only.label}`
          : `Move ${countWords(works.length, noun)}`
      }
      previewing={
        rootId === ""
          ? undefined
          : { kind: "ask", ask: { origin, works: works.map(({ id }) => ({ workId: id, rootId })) } }
      }
      picker={<DestinationPicker type={type} value={rootId} onValueChange={setRootId} />}
      homedWords={(homed) =>
        `${countWords(homed, noun)} without files ${homed === 1 ? "goes" : "go"} into the new root too.`
      }
      startedWords={(started, plan) =>
        `Moving ${countWords(started, noun)} to ${plan.units.find(({ verdict }) => verdict !== "red")?.to.nodeName ?? "the new root"}.`
      }
      open={open}
      onOpenChange={onOpenChange}
    />
  );
}

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
          : `Moving ${started} back to the roots they left.`
      }
      open={open}
      onOpenChange={onOpenChange}
    />
  );
}
