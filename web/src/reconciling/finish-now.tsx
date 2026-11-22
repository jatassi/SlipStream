import { useState } from "react";

import type { ReconcilingPage } from "../api/gen/model";
import { finishReconciling } from "../api/gen/query/reconciling/reconciling";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { finishWords } from "./words";

export function FinishNow({ page }: { page: ReconcilingPage }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        onClick={() => {
          setOpen(true);
        }}
      >
        Finish now
      </Button>
      <ConfirmDialog
        title="Finish Reconciling now"
        confirmLabel="Finish now"
        open={open}
        onOpenChange={setOpen}
        onConfirm={() => finishReconciling()}
      >
        <p>{finishWords(page)}</p>
      </ConfirmDialog>
    </>
  );
}
