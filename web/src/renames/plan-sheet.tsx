import { useQuery } from "@tanstack/react-query";
import { useId, useState } from "react";

import type { FilesRenameScope } from "../api/gen/model";
import { getPlanRenamesUrl, planRenames } from "../api/gen/query/files/files";
import { Dialog } from "../ui/dialog";
import { useShell } from "../ui/shell-context";
import { Sheet } from "../ui/sheet";
import { PlanBody } from "./plan-body";
import { Reading } from "./reading";
import { type Saving, firstRound } from "./round";

/** What a rename plan sheet opens on: a Work, a Content Type or the library, and in the save flow the draft. */
export type Opening = {
  readonly title: string;
  readonly scope: FilesRenameScope;
  readonly saving?: Saving;
};

function PlanRound({ opening, onClose }: { opening: Opening; onClose: () => void }) {
  const sheet = useId();
  const [round, setRound] = useState(() => firstRound(sheet, opening.saving));
  const body = {
    scope: opening.scope,
    ...(round.draft !== undefined && { draft: round.draft }),
  };
  const plan = useQuery({
    queryKey: [getPlanRenamesUrl(), body, round.sheet, round.n],
    queryFn: ({ signal }) => planRenames(body, { signal }),
    ...(round.plan !== undefined && { initialData: round.plan }),
  });
  return plan.data === undefined ? (
    <Reading label="Reading the rename plan" pending={plan.isPending} error={plan.error} />
  ) : (
    <PlanBody
      key={round.n}
      scope={opening.scope}
      plan={plan.data}
      round={round}
      onRound={setRound}
      onClose={onClose}
    />
  );
}

/**
 * A rename plan: the Works whose files' paths differ from what the naming
 * templates make, ticked to rename, each opening to its files' old and new
 * paths. A dialog on desktop and a full sheet on phone.
 */
export function PlanSheet({
  opening,
  onClose,
}: {
  opening: Opening | undefined;
  onClose: () => void;
}) {
  const shell = useShell();
  const title = opening?.title ?? "Rename files";
  const onOpenChange = (open: boolean) => {
    if (!open) {
      onClose();
    }
  };
  const body = opening !== undefined && <PlanRound opening={opening} onClose={onClose} />;
  return shell === "phone" ? (
    <Sheet title={title} fill open={opening !== undefined} onOpenChange={onOpenChange}>
      {body}
    </Sheet>
  ) : (
    <Dialog title={title} size={940} open={opening !== undefined} onOpenChange={onOpenChange}>
      {body}
    </Dialog>
  );
}
