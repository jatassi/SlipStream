import { DialogActions } from "../account/dialog-actions";
import type { FilesRenamePlan, FilesRenameScope } from "../api/gen/model";
import type { Attempt } from "../targets/attempt";
import { Button } from "../ui/button";
import { useShell } from "../ui/shell-context";
import { SignalChip } from "../ui/signal-chip";
import { tickedCounts } from "./plan-model";
import { PlanList } from "./plan-list";
import {
  blockedWords,
  changedWords,
  filesWords,
  leadWords,
  movingWords,
  nothingWords,
} from "./renames-words";
import { type Round, useConfirm, useReview } from "./round";

function Changed({ count }: { count: number }) {
  return (
    <output className="flex flex-wrap items-center gap-x-3 gap-y-2 rounded-card bg-warn-card px-4 py-3 text-warn-text ring-1 ring-warn-line ring-inset">
      <SignalChip kind="warn">Changed</SignalChip>
      <span className="min-w-0 flex-[1_1_24ch]">{changedWords(count)}</span>
    </output>
  );
}

/** What confirming does as ticked, then what the plan leaves out and why. */
function Lead({ plan, lead }: { plan: FilesRenamePlan; lead: string }) {
  const moving = plan.works.reduce((sum, line) => sum + line.moving, 0);
  return (
    <div className="flex flex-col gap-1">
      <p className="text-ink-1">{lead}</p>
      {plan.blocked > 0 && <p className="text-ink-3">{blockedWords(plan.blocked)}</p>}
      {moving > 0 && <p className="text-ink-3">{movingWords(moving)}</p>}
    </div>
  );
}

function Actions({
  lines,
  ticked,
  saving,
  confirming,
  savingOnly,
  onClose,
}: {
  lines: number;
  ticked: number;
  saving: boolean;
  confirming: Attempt;
  savingOnly: Attempt;
  onClose: () => void;
}) {
  return (
    <>
      {confirming.failure}
      {savingOnly.failure}
      <DialogActions>
        <Button disabled={confirming.busy || savingOnly.busy} onClick={onClose}>
          {lines === 0 && !saving ? "Close" : "Cancel"}
        </Button>
        {saving && (
          <Button disabled={confirming.busy} busy={savingOnly.busy} onClick={savingOnly.run}>
            Save only
          </Button>
        )}
        {lines > 0 && (
          <Button
            look="primary"
            disabled={ticked === 0 || savingOnly.busy}
            busy={confirming.busy}
            onClick={confirming.run}
          >
            {`${saving ? "Save and rename" : "Rename"} ${filesWords(ticked)}`}
          </Button>
        )}
      </DialogActions>
    </>
  );
}

/**
 * One round of a plan: what confirming does, the Works to tick, and the
 * confirm. In the save flow the confirm saves and renames, and Save only
 * saves.
 */
export function PlanBody({
  scope,
  plan,
  round,
  onRound,
  onClose,
}: {
  scope: FilesRenameScope;
  plan: FilesRenamePlan;
  round: Round;
  onRound: (round: Round) => void;
  onClose: () => void;
}) {
  const shell = useShell();
  const oneWork = scope.kind === "work";
  const lines = plan.works.filter((line) => line.rename > 0 || line.blocked > 0);
  const review = useReview(round, lines, oneWork);
  const { ticks, rows } = review;
  const { confirming, savingOnly } = useConfirm({
    scope,
    round,
    lines,
    ticks,
    rows,
    onRound,
    onClose,
  });
  const ticked = tickedCounts(lines, ticks, rows);
  const saving = round.store !== undefined;
  return (
    <div className={`flex flex-col gap-3 ${shell === "phone" ? "min-h-0 flex-1" : ""}`}>
      {round.changed > 0 && <Changed count={round.changed} />}
      <Lead
        plan={plan}
        lead={lines.length === 0 ? nothingWords : leadWords({ ...ticked, saving, oneWork })}
      />
      {lines.length > 0 && <PlanList lines={lines} review={review} />}
      <Actions
        lines={lines.length}
        ticked={ticked.files}
        saving={saving}
        confirming={confirming}
        savingOnly={savingOnly}
        onClose={onClose}
      />
    </div>
  );
}
