import { Suspense } from "react";

import type { GrabsGrab } from "../api/gen/model";
import { useGetWorkSuspense, useListContentTypesSuspense } from "../api/gen/query/works/works";
import { countWords } from "../format";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { useShell } from "../ui/shell-context";
import { Skeleton } from "../ui/skeleton";
import { lowered } from "../works/monitoring/labels";
import { CancelGrab } from "./cancel-grab";
// oxlint-disable-next-line import/max-dependencies -- Details shows the Grab, its download's files and each sheet acting on it
import { GrabFacts } from "./grab-facts";
import { GrabOutcome, GrabRecord, GrabStages } from "./grab-record";
import { ImportByHand, ImportLines } from "./import-lines";
import { SendToSheet } from "./send-to-sheet";
import { byHand } from "./stages";

/** "Chosen because it fills 24 episodes with one Release." when the decision chose it to fill a group. */
function ChosenBecause({ grab }: { grab: GrabsGrab }) {
  const work = useGetWorkSuspense(grab.workId).data;
  const type = useListContentTypesSuspense().data.contentTypes.find(
    ({ id }) => id === work.contentType,
  );
  if (grab.chosenBecause === null || type === undefined) {
    return null;
  }
  return (
    <p>
      Chosen because it fills {countWords(grab.chosenBecause.fills, lowered(type.item))} with one
      Release.
    </p>
  );
}

/** Which of a Grab's dialogs is open. One at a time, so a phone never stacks two sheets. */
export type GrabView = "details" | "cancel" | "send-to" | "import";

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
      {byHand(grab) && (
        <Button
          onClick={() => {
            onView("import");
          }}
        >
          Import by hand
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
            {grab.chosenBecause !== null && <ChosenBecause grab={grab} />}
            <GrabFacts grab={grab} />
            {grab.import !== null && <ImportLines importId={grab.import.id} />}
          </Suspense>
        </div>
        <GrabActions grab={grab} onView={onView} />
      </Dialog>
      <CancelGrab grab={grab} work={work} open={view === "cancel"} onOpenChange={close} />
      {grab.actions.sendTo && (
        <SendToSheet grab={grab} open={view === "send-to"} onOpenChange={close} />
      )}
      {view === "import" && grab.import !== null && (
        <ImportByHand
          importId={grab.import.id}
          onClose={() => {
            onView(undefined);
          }}
        />
      )}
    </>
  );
}
