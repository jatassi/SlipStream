import { Suspense } from "react";

import type { GrabsGrab } from "../api/gen/model";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { useShell } from "../ui/shell-context";
import { Skeleton } from "../ui/skeleton";
import { CancelGrab } from "./cancel-grab";
import { GrabFacts } from "./grab-facts";
import { GrabOutcome, GrabRecord, GrabStages } from "./grab-record";
import { SendToSheet } from "./send-to-sheet";

/** Which of a Grab's dialogs is open. One at a time, so a phone never stacks two sheets. */
export type GrabView = "details" | "cancel" | "send-to";

type OnView = (view: GrabView | undefined) => void;

function GrabActions({ grab, onView }: { grab: GrabsGrab; onView: OnView }) {
  const stacked = useShell() === "phone";
  return (
    <div className={`mt-6 flex gap-2 ${stacked ? "flex-col-reverse" : "justify-end"}`}>
      <Button
        onClick={() => {
          onView(undefined);
        }}
      >
        Close
      </Button>
      {grab.actions.sendTo && (
        <Button
          onClick={() => {
            onView("send-to");
          }}
        >
          Send to another client
        </Button>
      )}
      {grab.actions.cancel && (
        <Button
          look="fail"
          onClick={() => {
            onView("cancel");
          }}
        >
          Cancel Grab
        </Button>
      )}
    </div>
  );
}

/**
 * A Grab's record and the actions on it: Details, Cancel and Send to another
 * client. `view` names the open one; the caller opens it from a menu or a card.
 */
export function GrabDialogs({
  grab,
  work,
  view,
  onView,
}: {
  grab: GrabsGrab;
  work: string;
  view: GrabView | undefined;
  onView: OnView;
}) {
  const close = (open: boolean) => {
    if (!open) {
      onView(undefined);
    }
  };
  return (
    <>
      <Dialog title={`Grab of ${work}`} size={680} open={view === "details"} onOpenChange={close}>
        <div className="flex flex-col gap-4 text-ink-2">
          <GrabOutcome grab={grab} />
          <GrabRecord grab={grab} />
          <GrabStages grab={grab} />
          <Suspense fallback={<Skeleton label="Loading the Grab" />}>
            <GrabFacts grab={grab} />
          </Suspense>
        </div>
        <GrabActions grab={grab} onView={onView} />
      </Dialog>
      <CancelGrab grab={grab} work={work} open={view === "cancel"} onOpenChange={close} />
      {grab.actions.sendTo && (
        <SendToSheet grab={grab} open={view === "send-to"} onOpenChange={close} />
      )}
    </>
  );
}
