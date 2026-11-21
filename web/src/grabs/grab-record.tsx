import type { GrabsGrab } from "../api/gen/model";
import { type Clock, useClock } from "../ui/clock";
import { RelativeTime } from "../ui/relative-time";
import { whenWords } from "../ui/time-words";
import { foundByWords } from "./search-words";
import { needsYou, stageWord } from "./stages";

const lead = "text-sentence in-data-[density=touch]:text-sentence-touch";

export function GrabOutcome({ grab }: { grab: GrabsGrab }) {
  if (grab.failure === undefined) {
    return <p className={`${lead} text-ink-1`}>{stageWord[grab.stage]}</p>;
  }
  if (!needsYou(grab)) {
    return <p className={`${lead} text-ink-1`}>{grab.failure.message}</p>;
  }
  const { cause, recovery } = grab.failure;
  return (
    <div className="flex flex-col gap-1">
      <p className={`${lead} text-fail-ink`}>{cause}</p>
      {recovery !== null && <p>{recovery}</p>}
      <p className="text-fail-dim">{grab.failure.message}</p>
    </div>
  );
}

const madeWords = (grab: GrabsGrab, clock: Clock) =>
  grab.grabbedBy === null
    ? `Found by ${foundByWords[grab.foundBy](grab)} ${whenWords(grab.createdAt, "at", clock)}.`
    : `Found by ${grab.grabbedBy} in Manual search ${whenWords(grab.createdAt, "at", clock)}.`;

export function GrabRecord({ grab }: { grab: GrabsGrab }) {
  const clock = useClock();
  return (
    <div className="flex flex-col gap-1">
      <p>{madeWords(grab, clock)}</p>
      {grab.overrides.map((sentence) => (
        <p key={sentence}>{sentence}</p>
      ))}
    </div>
  );
}

export function GrabStages({ grab }: { grab: GrabsGrab }) {
  return (
    <section aria-label="Stages">
      <ol className="flex flex-col gap-1">
        {grab.history.map((entered) => (
          <li key={entered.stage} className="flex justify-between gap-4">
            <span className="text-ink-1">{stageWord[entered.stage]}</span>
            <RelativeTime at={entered.at} form="clock" />
          </li>
        ))}
      </ol>
    </section>
  );
}
