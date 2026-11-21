import { useState } from "react";

import type { SupersededView } from "../api/gen/model";
import { resumeSupersededController } from "../api/gen/query/superseded/superseded";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { Data } from "../ui/data";
import { Details } from "../ui/details";

function ResumeButton() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        onClick={() => {
          setOpen(true);
        }}
      >
        Resume this Controller…
      </Button>
      <ConfirmDialog
        title="Resume this Controller?"
        confirmLabel="Resume"
        open={open}
        onOpenChange={setOpen}
        onConfirm={() => resumeSupersededController()}
      >
        <p>
          Do this only if the restore on the other machine was a mistake. Workers already paired
          with the other Controller won&apos;t connect to this one.
        </p>
      </ConfirmDialog>
    </>
  );
}

/** Who replaced this Controller and who said so, and the way back, under Details. */
export function FenceDetails({ view }: { view: SupersededView }) {
  const { replacement, worker } = view;
  return (
    <Details>
      <div className="flex flex-col items-start gap-3 pt-1">
        {replacement === undefined && worker === undefined ? null : (
          <dl className="grid grid-cols-[max-content_minmax(0,1fr)] gap-x-4 gap-y-1 text-meta">
            {replacement === undefined ? null : (
              <>
                <dt className="text-ink-3">New Controller&apos;s key</dt>
                <dd className="wrap-anywhere">
                  <Data>{replacement.key}</Data>
                </dd>
              </>
            )}
            {worker === undefined ? null : (
              <>
                <dt className="text-ink-3">Reported by</dt>
                <dd className="wrap-anywhere">{worker}</dd>
              </>
            )}
          </dl>
        )}
        <ResumeButton />
      </div>
    </Details>
  );
}
