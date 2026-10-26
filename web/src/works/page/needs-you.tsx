import type {
  FilesFailedNeed,
  FilesNeed,
  FilesUpgradeNeed,
  FilesWaitingNeed,
  GrabsGrab,
} from "../../api/gen/model";
import { clock } from "../../format";
import { GrabFailure } from "../../grabs/failure-cards";
import { type FailedGrab, failureWords, isFailed, trackStage } from "../../grabs/stages";
import { Section } from "../../ui/section";
import { SignalRow } from "../../ui/signal-row";
import { stageLabel } from "../../ui/stage-track";
import { NoRoot, Upgrades, Waiting } from "./need-rows";
import { type Naming, failedWords } from "./need-words";

const cardsShown = 3;

function Failed({
  naming,
  need,
  grab,
  card,
}: {
  naming: Naming;
  need: FilesFailedNeed;
  grab: FailedGrab;
  card: boolean;
}) {
  const subject = failedWords(naming, need, grab);
  if (card) {
    return <GrabFailure grab={grab} work={subject} form="compact" />;
  }
  const { cause, recovery } = failureWords[grab.failure.code];
  return (
    <SignalRow
      signal="fail"
      chip={`Failed at ${stageLabel[trackStage[grab.failure.stage]]}`}
      subject={subject}
      verdict={recovery === undefined ? cause : `${cause} ${recovery}`}
      time={clock(grab.history.at(-1)?.at ?? grab.createdAt)}
    >
      <p className="text-fail-dim">{grab.failure.message}</p>
    </SignalRow>
  );
}

type Grouped = {
  readonly failed: readonly { need: FilesFailedNeed; grab: FailedGrab }[];
  readonly upgrades: readonly FilesUpgradeNeed[];
  readonly waiting: readonly FilesWaitingNeed[];
  readonly noRoot: boolean;
};

// The server lists failures, then upgrades, then waiting jobs, then no_root.
function grouped(needs: readonly FilesNeed[], grabs: readonly GrabsGrab[]): Grouped {
  const grabOf = (id: string) => grabs.find((grab) => grab.id === id);
  return {
    failed: needs.flatMap((need) => {
      if (need.kind !== "failed") {
        return [];
      }
      const grab = grabOf(need.grabId);
      return grab !== undefined && isFailed(grab) ? [{ need, grab }] : [];
    }),
    upgrades: needs.filter((need) => need.kind === "upgrade"),
    waiting: needs.filter((need) => need.kind === "waiting"),
    noRoot: needs.some((need) => need.kind === "no_root"),
  };
}

/**
 * What only a person can resolve on this Work, failures first: the first
 * three as compact failure cards, the rest as Signal rows, then upgrades,
 * then waiting jobs. It shows nothing while nothing needs the Admin.
 */
export function NeedsYou({
  naming,
  needs,
  grabs,
}: {
  naming: Naming;
  needs: readonly FilesNeed[];
  grabs: readonly GrabsGrab[];
}) {
  const { failed, upgrades, waiting, noRoot } = grouped(needs, grabs);
  const count = failed.length + upgrades.length + waiting.length + (noRoot ? 1 : 0);
  if (count === 0) {
    return null;
  }
  return (
    <Section title="Needs you" count={count}>
      <div className="flex flex-col gap-3">
        {failed.map(({ need, grab }, index) => (
          <Failed
            key={need.grabId}
            naming={naming}
            need={need}
            grab={grab}
            card={index < cardsShown}
          />
        ))}
        {upgrades.length > 0 && <Upgrades naming={naming} needs={upgrades} />}
        {waiting.map((need) => (
          <Waiting
            key={need.grabId}
            need={need}
            grab={grabs.find(({ id }) => id === need.grabId)}
          />
        ))}
        {noRoot && <NoRoot />}
      </div>
    </Section>
  );
}
