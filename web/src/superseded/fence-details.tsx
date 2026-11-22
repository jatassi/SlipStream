import { useState } from "react";

import type { SupersededView } from "../api/gen/model";
import { resumeSupersededController } from "../api/gen/query/superseded/superseded";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { Fact, Facts } from "../ui/facts";

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
          <Facts look="well">
            {replacement === undefined ? null : (
              <Fact label="New Controller's key">
                <Data>{replacement.key}</Data>
              </Fact>
            )}
            {worker === undefined ? null : <Fact label="Reported by">{worker}</Fact>}
          </Facts>
        )}
        <ResumeButton />
      </div>
    </Details>
  );
}
