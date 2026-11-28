import type {
  FilesFailedNeed,
  FilesNeed,
  FilesNotFoundNeed,
  FilesUpgradeNeed,
  FilesWaitingNeed,
  GrabsGrab,
  GrabsStopped,
} from "../../api/gen/model";
import { GrabFailure } from "../../grabs/failure-cards";
import { stoppedWords } from "../../grabs/search-words";
import { type FailedGrab, isFailed, trackStage } from "../../grabs/stages";
import { useClock } from "../../ui/clock";
import { Section } from "../../ui/section";
import { SignalRow } from "../../ui/signal-row";
import { stageLabel } from "../../ui/stage-track";
import { NoRoot, NotFound, Upgrades } from "./need-rows";
import { type FailedEntry, type Naming, failedKey, failedWords } from "./need-words";
// oxlint-disable-next-line import/max-dependencies -- Needs you lists every kind of need, a stopped cell's card included
import { Waiting } from "./waiting";

const cardsShown = 3;

function Failed({
  naming,
  need,
  grab,
  stop,
  card,
}: {
  naming: Naming;
  need: FilesFailedNeed;
  grab: FailedGrab;
  stop: GrabsStopped | undefined;
  card: boolean;
}) {
  const subject = failedWords(naming, need, grab);
  const clock = useClock();
  if (card) {
    return (
      <GrabFailure
        grab={grab}
        subject={subject}
        work={naming.work.label}
        form="compact"
        stop={stop}
      />
    );
  }
  const { cause, recovery } = grab.failure;
  return (
    <SignalRow
      signal="fail"
      chip={`Failed at ${stageLabel[trackStage(grab)]}`}
      subject={subject}
      verdict={
        stop === undefined
          ? recovery === null
            ? cause
            : `${cause} ${recovery}`
          : `${cause} ${stoppedWords(stop, clock)}`
      }
      at={grab.history.at(-1)?.at ?? grab.createdAt}
    >
      <p className="text-fail-dim">{grab.failure.message}</p>
    </SignalRow>
  );
}

type Grouped = {
  readonly failed: FailedEntry[];
  readonly upgrades: FilesUpgradeNeed[];
  readonly notFound: FilesNotFoundNeed[];
  readonly waiting: FilesWaitingNeed[];
  noRoot: boolean;
};

// The server lists failures, then upgrades, then not found, then waiting jobs, then no_root.
function grouped(needs: readonly FilesNeed[], grabs: readonly GrabsGrab[]): Grouped {
  const out: Grouped = { failed: [], upgrades: [], notFound: [], waiting: [], noRoot: false };
  for (const need of needs) {
    switch (need.kind) {
      case "failed": {
        const grab = grabs.find(({ id }) => id === need.grabId);
        if (grab !== undefined && isFailed(grab)) {
          out.failed.push({ need, grab });
        }
        break;
      }
      case "upgrade":
        out.upgrades.push(need);
        break;
      case "not_found":
        out.notFound.push(need);
        break;
      case "waiting":
        out.waiting.push(need);
        break;
      case "no_root":
        out.noRoot = true;
        break;
      case "conversion_failed":
        break;
      default:
        return need satisfies never;
    }
  }
  return out;
}

/** The stop of a failed need's cell, when failed Grabs stopped SlipStream searching it again. */
const stopOf = (stops: readonly GrabsStopped[], need: FilesFailedNeed) =>
  stops.find(
    ({ cell }) =>
      need.itemIds.includes(cell.itemId) &&
      cell.targetId === need.targetId &&
      cell.edition === need.edition,
  );

/**
 * What only a person can resolve on this Work, failures first: the first
 * three as compact failure cards, the rest as Signal rows, then upgrades,
 * then Items searched with nothing found, then waiting jobs. It shows
 * nothing while nothing needs the Admin.
 */
export function NeedsYou({
  naming,
  needs,
  grabs,
  stops,
}: {
  naming: Naming;
  needs: readonly FilesNeed[];
  grabs: readonly GrabsGrab[];
  /** The cells failed Grabs stopped SlipStream searching again. */
  stops: readonly GrabsStopped[];
}) {
  const { failed, upgrades, notFound, waiting, noRoot } = grouped(needs, grabs);
  const count =
    failed.length + upgrades.length + notFound.length + waiting.length + (noRoot ? 1 : 0);
  if (count === 0) {
    return null;
  }
  return (
    <Section title="Needs you" count={count}>
      <div className="flex flex-col gap-3">
        {failed.map((entry, index) => (
          <Failed
            key={failedKey(entry)}
            naming={naming}
            need={entry.need}
            grab={entry.grab}
            stop={stopOf(stops, entry.need)}
            card={index < cardsShown}
          />
        ))}
        {upgrades.length > 0 && <Upgrades naming={naming} needs={upgrades} />}
        {notFound.length > 0 && <NotFound naming={naming} needs={notFound} />}
        {waiting.map((need) => (
          <Waiting
            key={need.grabId}
            type={naming.type}
            grab={grabs.find(({ id }) => id === need.grabId)}
            need={need}
          />
        ))}
        {noRoot && <NoRoot type={naming.type} />}
      </div>
    </Section>
  );
}
