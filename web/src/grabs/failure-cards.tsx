import { useNavigate } from "@tanstack/react-router";

import { clock } from "../format";
import { Data } from "../ui/data";
import { FailureCard } from "../ui/failure-card";
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

export function GrabFailure({
  grab,
  work,
  form,
}: {
  grab: FailedGrab;
  work: string;
  form: "full" | "compact";
}) {
  const navigate = useNavigate();
  const search = useStartSearch();
  const { cause, recovery } = failureWords[grab.failure.code];
  return (
    <div id={failureAnchor(grab)} className="scroll-mt-20">
      <FailureCard
        form={form}
        stage={trackStage[grab.failure.stage]}
        time={failedAt(grab)}
        work={work}
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
            search.mutate({ workId: grab.workId });
          },
        }}
        {...(form === "full" && {
          actions: [
            {
              label: "Open Work",
              onClick: () => {
                void navigate({ to: "/works/$workId", params: { workId: grab.workId } });
              },
            },
          ],
        })}
      />
    </div>
  );
}
