import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { openManualSearch } from "../manual/sheet-state";
import { Data } from "../ui/data";
import { FailureCard } from "../ui/failure-card";
import { type GrabView, GrabDialogs } from "./grab-details";
import { useStartSearch } from "./search";
import { type FailedGrab, skippedStages, trackStage } from "./stages";
import { type Stop, useRecoveryFacts } from "./stopped-facts";

export const failureAnchor = (grab: FailedGrab): string => `failed-${grab.id}`;

function failedAt(grab: FailedGrab): string {
  const entered = grab.history.at(-1);
  if (entered === undefined) {
    throw new Error(`Grab ${grab.id} has no history`);
  }
  return entered.at;
}

function useCardActions(grab: FailedGrab, form: "full" | "compact", onDetails: () => void) {
  const navigate = useNavigate();
  const manual = {
    label: "Manual search",
    onClick: () => {
      openManualSearch({ workId: grab.workId, scope: { itemIds: grab.items } });
    },
  };
  const details = { label: "Details", onClick: onDetails };
  const openWork = {
    label: "Open Work",
    onClick: () => {
      void navigate({ to: "/works/$workId", params: { workId: grab.workId } });
    },
  };
  return form === "full" ? [manual, details, openWork] : [manual, details];
}

/**
 * `subject` heads the card; `work` names the Work in the Grab's Details and
 * Cancel. `stop` is set when failed Grabs stopped SlipStream searching the
 * cell again, and the card then says when Backlog search tries next.
 */
export function GrabFailure({
  grab,
  subject,
  work,
  form,
  stop,
}: {
  grab: FailedGrab;
  subject: string;
  work: string;
  form: "full" | "compact";
  stop?: Stop | undefined;
}) {
  const search = useStartSearch();
  const recovery = useRecoveryFacts(grab, stop);
  const [view, setView] = useState<GrabView>();
  const { cause } = grab.failure;
  const actions = useCardActions(grab, form, () => {
    setView("details");
  });
  return (
    <div id={failureAnchor(grab)} className="scroll-mt-20">
      <FailureCard
        form={form}
        stage={trackStage(grab)}
        at={failedAt(grab)}
        work={subject}
        track={{ skipped: skippedStages }}
        facts={[
          { label: "Cause", value: cause },
          ...recovery,
          { label: "Release", value: <Data>{grab.releaseTitle}</Data> },
          ...(grab.failure.message === cause
            ? []
            : [{ label: "Detail", value: grab.failure.message }]),
        ]}
        fix={{
          label: "Search again",
          onClick: () => {
            search.start(grab.workId, { name: work }, { levelIds: [], itemIds: grab.items });
          },
        }}
        actions={actions}
      />
      <GrabDialogs grab={grab} work={work} view={view} onView={setView} />
    </div>
  );
}
