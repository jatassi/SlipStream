import type { GrabsGrab } from "../api/gen/model";
import { clock } from "../format";
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

const madeWords = (grab: GrabsGrab) =>
  grab.grabbedBy === null
    ? `Found by ${foundByWords[grab.foundBy](grab)}, ${clock(grab.createdAt)}.`
    : `Found by ${grab.grabbedBy} in Manual search, ${clock(grab.createdAt)}.`;

export function GrabRecord({ grab }: { grab: GrabsGrab }) {
  return (
    <div className="flex flex-col gap-1">
      <p>{madeWords(grab)}</p>
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
            <span className="tabular-nums">{clock(entered.at)}</span>
          </li>
        ))}
      </ol>
    </section>
  );
}
