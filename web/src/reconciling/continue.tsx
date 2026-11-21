import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { ReconcilingNodeRow, ReconcilingRootRow } from "../api/gen/model";
import {
  continueReconcilingNode,
  continueReconcilingRoot,
} from "../api/gen/query/reconciling/reconciling";
import { Button, type ButtonLook } from "../ui/button";
import { toast } from "../ui/toast";

/**
 * A press whose answer needs no words: reconciling.changed brings the new
 * state, and a refusal toasts why.
 */
export function usePress(action: () => Promise<unknown>) {
  const [busy, setBusy] = useState(false);
  const press = async () => {
    setBusy(true);
    try {
      await action();
    } catch (error) {
      toast(sentenceOf(error), { kind: "fail" });
    } finally {
      setBusy(false);
    }
  };
  return { busy, press: () => void press() };
}

export function ContinueWithout({
  node,
  look = "default",
}: {
  node: ReconcilingNodeRow;
  look?: ButtonLook;
}) {
  const { busy, press } = usePress(() => continueReconcilingNode(node.id));
  return (
    <Button look={look} size="small" busy={busy} onClick={press}>
      {`Continue without ${node.name}`}
    </Button>
  );
}

export function ContinueWithoutRoot({ root }: { root: ReconcilingRootRow }) {
  const { busy, press } = usePress(() => continueReconcilingRoot(root.id));
  return (
    <Button size="small" busy={busy} onClick={press}>
      Continue without it
    </Button>
  );
}
