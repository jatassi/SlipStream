import type { TasksRun, TasksRunTrigger, TasksRunUnit } from "../api/gen/model";
import { lengthWords, sinceWords } from "../format";
import { Data } from "../ui/data";
import { ExpandList, ExpandRow } from "../ui/expand-list";
import { Section } from "../ui/section";
import { OutcomeChip } from "./outcome-chip";
import { runsAnchor } from "./task-head";

const triggers: { readonly [T in TasksRunTrigger]: string } = {
  schedule: "Scheduled",
  "catch-up": "Catch-up",
  "run-now": "Run now",
  returned: "Node returned",
};

function RunHead({ run, now }: { run: TasksRun; now: Date }) {
  const ended = run.endedAt === null ? now.getTime() : Date.parse(run.endedAt);
  return (
    <span className="grid min-w-0 flex-1 grid-cols-[auto_minmax(0,1fr)] items-center gap-x-3 gap-y-0.5 @[720px]:grid-cols-[6.5rem_minmax(0,1fr)_7rem_8rem_6rem_minmax(0,9rem)]">
      <span className="flex">
        <OutcomeChip outcome={run.result} />
      </span>
      <span className="truncate">{run.summary === "" ? "Running now" : run.summary}</span>
      <span className="col-span-2 flex flex-wrap gap-x-2 text-meta text-ink-3 @[720px]:contents @[720px]:text-(length:--body-size) @[720px]:text-ink-2">
        <span>{triggers[run.trigger]}</span>
        <Data>{sinceWords(run.startedAt, now)}</Data>
        <Data>{lengthWords(ended - Date.parse(run.startedAt))}</Data>
        <span className="truncate">{run.nodes.join(", ")}</span>
      </span>
    </span>
  );
}

function UnitLine({ unit }: { unit: TasksRunUnit }) {
  return (
    <li className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1 px-3 py-2 not-first:shadow-[inset_0_1px_0_var(--color-hairline)] @[720px]:grid-cols-[minmax(0,14rem)_8rem_7rem_minmax(0,1fr)]">
      <span className="truncate">{unit.label}</span>
      <span className="order-last col-span-2 text-meta text-ink-3 @[720px]:order-none @[720px]:col-span-1 @[720px]:text-(length:--body-size)">
        {unit.node === "" ? "No longer paired" : unit.node}
      </span>
      <span className="flex justify-end @[720px]:justify-start">
        <OutcomeChip outcome={unit.outcome} />
      </span>
      <span className="col-span-2 text-meta text-ink-3 @[720px]:col-span-1">{unit.cause}</span>
    </li>
  );
}

function RunUnits({ run }: { run: TasksRun }) {
  if (run.units.length === 0) {
    return <p className="text-ink-3">Its units report when the run ends.</p>;
  }
  return (
    <ul aria-label="Units in this run" className="rounded-item bg-well">
      {run.units.map((unit) => (
        <UnitLine key={unit.key} unit={unit} />
      ))}
    </ul>
  );
}

export function TaskRuns({ runs, now }: { runs: readonly TasksRun[]; now: Date }) {
  return (
    <Section id={runsAnchor} title="Runs" count={runs.length}>
      {runs.length === 0 ? (
        <p className="text-ink-3">No runs yet.</p>
      ) : (
        <ExpandList label="Runs">
          {runs.map((run) => (
            <ExpandRow key={run.id} head={<RunHead run={run} now={now} />}>
              {run.cause !== null && <p className="mb-2 text-fail-ink">{run.cause}</p>}
              <RunUnits run={run} />
            </ExpandRow>
          ))}
        </ExpandList>
      )}
    </Section>
  );
}
