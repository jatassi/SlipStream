import { useState } from "react";

import type { TargetsRemakeImpact } from "../api/gen/model";
import { Dialog } from "../ui/dialog";
import { AttemptActions, useAttempt } from "./attempt";
import { olderWords } from "./conversion-words";
import { MadeWorksTable } from "./made-works-table";

export type Remaking = {
  readonly name: string;
  readonly fromSave: boolean;
  readonly remake: TargetsRemakeImpact;
  readonly save: (works: readonly string[], files: number) => Promise<void>;
};

const grouped = new Intl.NumberFormat("en");

const filesWords = (count: number) => (count === 1 ? "1 file" : `${grouped.format(count)} files`);

function actionOf(fromSave: boolean, count: number): string {
  if (count === 0) {
    return "Save";
  }
  return fromSave ? `Save and remake ${filesWords(count)}` : `Remake ${filesWords(count)}`;
}

function RemakeBody({
  remaking: { fromSave, remake, save },
  onClose,
}: {
  remaking: Remaking;
  onClose: () => void;
}) {
  const [picked, setPicked] = useState<ReadonlySet<string>>(new Set());
  const chosen = remake.rows.filter((row) => picked.has(row.workId));
  const count = chosen.reduce((sum, row) => sum + row.files, 0);
  const saving = useAttempt(async () => {
    await save(
      chosen.map((row) => row.workId),
      count,
    );
    onClose();
  });
  return (
    <div className="flex flex-col gap-4">
      <p className="text-ink-1">
        {olderWords(remake.older)} They stay until you remake them. Pick the Works to make again
        with these settings.
      </p>
      <MadeWorksTable
        label="Works with files made with older settings"
        rows={remake.rows}
        picked={picked}
        onPicked={setPicked}
      />
      <AttemptActions attempt={saving} onCancel={onClose}>
        {actionOf(fromSave, count)}
      </AttemptActions>
    </div>
  );
}

export function RemakeDialog({
  remaking,
  onClose,
}: {
  remaking: Remaking | undefined;
  onClose: () => void;
}) {
  return (
    <Dialog
      title={remaking === undefined ? "Remake" : `Remake ${remaking.name} files`}
      size={940}
      open={remaking !== undefined}
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      {remaking !== undefined && <RemakeBody remaking={remaking} onClose={onClose} />}
    </Dialog>
  );
}
