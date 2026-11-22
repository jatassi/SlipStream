import { useEffect, useRef, useState } from "react";

import { sentenceOf } from "../../api/errors";
import type { BackupsBackup, BackupsDestination } from "../../api/gen/model";
import { pullBackup } from "../../api/gen/query/backups/backups";
import { stageBackupRestore } from "../../api/gen/query/restore/restore";
import type { RestoreFlow } from "../../restore/flow";

export type Prepare =
  | { readonly at: "staging" }
  | { readonly at: "pull"; readonly from: BackupsDestination }
  | { readonly at: "pulling"; readonly from: BackupsDestination }
  | { readonly at: "refused"; readonly refusal: string }
  | { readonly at: "staged" };

/** The destination a Backup with no local file comes back from: the first with a copy. */
const copyOf = (backup: BackupsBackup, destinations: readonly BackupsDestination[]) => {
  const copied = new Set(
    backup.copies.filter((copy) => copy.state === "copied").map((copy) => copy.destinationId),
  );
  return destinations.find((each) => copied.has(each.id));
};

function startOf(backup: BackupsBackup, destinations: readonly BackupsDestination[]): Prepare {
  if (backup.local) {
    return { at: "staging" };
  }
  const from = copyOf(backup, destinations);
  return from === undefined
    ? {
        at: "refused",
        refusal: "No destination has a copy of this Backup, so it can't be restored.",
      }
    : { at: "pull", from };
}

export function usePrepare(
  backup: BackupsBackup,
  destinations: readonly BackupsDestination[],
  flow: RestoreFlow,
) {
  const [prepare, setPrepare] = useState<Prepare>(() => startOf(backup, destinations));
  const asked = useRef(false);
  const ready = backup.local && (prepare.at === "staging" || prepare.at === "pulling");
  const { staged } = flow;
  useEffect(() => {
    if (!ready || asked.current) {
      return;
    }
    asked.current = true;
    const stage = async () => {
      try {
        const { id, summary } = await stageBackupRestore(backup.id);
        staged(id, summary);
        setPrepare({ at: "staged" });
      } catch (error) {
        setPrepare({ at: "refused", refusal: sentenceOf(error) });
      }
    };
    void stage();
  }, [ready, backup.id, staged]);
  const pull = async (from: BackupsDestination) => {
    setPrepare({ at: "pulling", from });
    try {
      await pullBackup(backup.id, { destinationId: from.id });
    } catch (error) {
      setPrepare({ at: "refused", refusal: sentenceOf(error) });
    }
  };
  return { prepare, pull };
}
