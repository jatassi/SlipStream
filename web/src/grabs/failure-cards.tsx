import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { clock } from "../format";
import { openManualSearch } from "../manual/sheet-state";
import { Data } from "../ui/data";
import { FailureCard } from "../ui/failure-card";
import { type GrabView, GrabDialogs } from "./grab-details";
import { useStartSearch } from "./search";
import { type FailedGrab, failureWords, skippedStages, trackStage } from "./stages";

export const failureAnchor = (grab: FailedGrab): string => `failed-${grab.id}`;

function failedAt(grab: FailedGrab): string {
  const entered = grab.history.at(-1);
  if (entered === undefined) {
    throw new Error(`Grab ${grab.id} has no history`);
  }
  return clock(entered.at);
}

function useCardActions(grab: FailedGrab, form: "full" | "compact", onDetails: () => void) {
  const navigate = useNavigate();
  const manual = {
    label: "Manual search",
    onClick: () => {
      openManualSearch({ workId: grab.workId, scope: { itemIds: [grab.itemId] } });
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

/** `subject` heads the card; `work` names the Work in the Grab's Details and Cancel. */
export function GrabFailure({
  grab,
  subject,
  work,
  form,
}: {
  grab: FailedGrab;
  subject: string;
  work: string;
  form: "full" | "compact";
}) {
  const search = useStartSearch();
  const [view, setView] = useState<GrabView>();
  const { cause, recovery } = failureWords[grab.failure.code];
  const actions = useCardActions(grab, form, () => {
    setView("details");
  });
  return (
    <div id={failureAnchor(grab)} className="scroll-mt-20">
      <FailureCard
        form={form}
        stage={trackStage[grab.failure.stage]}
        time={failedAt(grab)}
        work={subject}
        track={{ skipped: skippedStages }}
        facts={[
          { label: "Cause", value: cause },
          ...(recovery === undefined ? [] : [{ label: "Recovery", value: recovery }]),
          { label: "Release", value: <Data>{grab.releaseTitle}</Data> },
          { label: "Detail", value: grab.failure.message },
        ]}
        fix={{
          label: "Search again",
          onClick: () => {
            search.start(grab.workId, { levelIds: [], itemIds: [grab.itemId] });
          },
        }}
        actions={actions}
      />
      <GrabDialogs grab={grab} work={work} view={view} onView={setView} />
    </div>
  );
}
