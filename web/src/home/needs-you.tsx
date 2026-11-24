import { Link } from "@tanstack/react-router";

import type { HomeNeed, HomeSummary, WorksContentType } from "../api/gen/model";
import { GrabFailure, failureAnchor } from "../grabs/failure-cards";
import { type FailedGrab, isFailed, trackStage } from "../grabs/stages";
import { PendingLink } from "../imports/pending-link";
import { useCounts } from "../shell/counts";
import { Section } from "../ui/section";
import { SignalRow } from "../ui/signal-row";
import { stageLabel } from "../ui/stage-track";
import { typeOf } from "../works/fields";
// oxlint-disable-next-line import/max-dependencies -- Needs you draws failure cards, Signal rows, waiting Callouts and pending imports
import { Waiting } from "../works/page/waiting";
import { seeAllLink } from "./jobs";
import { MoveNeeds } from "./move-need";
import { failedCap, needsAnchor } from "./words";

// As the Work page does: the first failures as cards, the rest as Signal rows.
const cardsShown = 3;

function FailedRow({ grab, work }: { grab: FailedGrab; work: string }) {
  const { cause, recovery, message } = grab.failure;
  return (
    <div id={failureAnchor(grab)} className="scroll-mt-20">
      <SignalRow
        signal="fail"
        chip={`Failed at ${stageLabel[trackStage(grab)]}`}
        subject={work}
        verdict={recovery === null ? cause : `${cause} ${recovery}`}
        at={grab.history.at(-1)?.at ?? grab.createdAt}
      >
        <p className="text-fail-dim">{message}</p>
      </SignalRow>
    </div>
  );
}

const failedOf = (needs: readonly HomeNeed[]) =>
  needs.flatMap(({ grab, work, wait }) =>
    wait === null && isFailed(grab) ? [{ grab, work: work.label }] : [],
  );

/**
 * What only the Admin can resolve: failed Grabs, Grabs that wait on the
 * Admin in their Work page's words, Moves that failed or wait long to
 * switch, then the pending imports row. It is left
 * out while nothing needs the Admin.
 */
export function NeedsYou({
  needsYou,
  types,
}: {
  needsYou: HomeSummary["needsYou"];
  types: WorksContentType[];
}) {
  const pending = useCounts()?.pendingImports ?? 0;
  const { rows, moves, failed, waiting } = needsYou;
  if (rows.length === 0 && moves.length === 0 && pending === 0) {
    return null;
  }
  const failures = failedOf(rows);
  const total = failed + waiting + moves.length;
  return (
    <Section id={needsAnchor} title="Needs you" {...(failed < failedCap && { count: total })}>
      <div className="flex flex-col gap-3">
        {failures.length > 0 && (
          <div className="grid grid-cols-[repeat(auto-fit,minmax(min(360px,100%),1fr))] gap-3">
            {failures.slice(0, cardsShown).map(({ grab, work }) => (
              <GrabFailure key={grab.id} grab={grab} subject={work} work={work} form="full" />
            ))}
          </div>
        )}
        {failures.slice(cardsShown).map(({ grab, work }) => (
          <FailedRow key={grab.id} grab={grab} work={work} />
        ))}
        {rows.map(
          ({ grab, work, wait }) =>
            wait !== null && (
              <Waiting
                key={grab.id}
                type={typeOf(types, work.contentType)}
                grab={grab}
                need={wait}
              />
            ),
        )}
        <MoveNeeds needs={moves} />
        <PendingLink preload={false} />
        {total > rows.length + moves.length && (
          <Link to="/activity" preload={false} className={seeAllLink}>
            See all {failed >= failedCap ? `${failedCap}+` : total} in Activity
          </Link>
        )}
      </div>
    </Section>
  );
}
