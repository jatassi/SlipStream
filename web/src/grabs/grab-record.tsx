import type { GrabsGrab } from "../api/gen/model";
import { clock } from "../format";
import { failureWords, needsYou, stageWord } from "./stages";

const lead = "text-sentence in-data-[density=touch]:text-sentence-touch";

export function GrabOutcome({ grab }: { grab: GrabsGrab }) {
  if (grab.failure === undefined) {
    return <p className={`${lead} text-ink-1`}>{stageWord[grab.stage]}</p>;
  }
  if (!needsYou(grab)) {
    return <p className={`${lead} text-ink-1`}>{grab.failure.message}</p>;
  }
  const { cause, recovery } = failureWords[grab.failure.code];
  return (
    <div className="flex flex-col gap-1">
      <p className={`${lead} text-fail-ink`}>{cause}</p>
      {recovery !== undefined && <p>{recovery}</p>}
      <p className="text-fail-dim">{grab.failure.message}</p>
    </div>
  );
}

const madeWords = ({ grabbedBy, createdAt }: GrabsGrab) =>
  grabbedBy === null
    ? `Automatic search, ${clock(createdAt)}.`
    : `Manual search by ${grabbedBy}, ${clock(createdAt)}.`;

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
