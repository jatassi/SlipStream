import type { TasksOpenRun } from "../api/gen/model";
import { Data } from "../ui/data";
import { LiveDot } from "../ui/live-chip";
import { Meter } from "../ui/meter";
import { Popover } from "../ui/popover";
import { RunningPanel, runMeterLabel, useVisibleRuns } from "./running";
import { percentOf } from "./words";

const capsuleLabel = (top: TasksOpenRun, count: number): string =>
  count === 1
    ? `1 task running: ${top.name}`
    : `${count} tasks running: ${top.name} and ${count - 1} more`;

export function TasksCapsule() {
  const runs = useVisibleRuns();
  const [top] = runs;
  if (top === undefined) {
    return null;
  }
  return (
    <Popover
      label="Running tasks"
      trigger={
        <button
          type="button"
          aria-label={capsuleLabel(top, runs.length)}
          className="flex h-8 max-w-72 min-w-0 items-center gap-2.5 rounded-capsule bg-fill pr-3.5 pl-3 text-body text-ink-2 ring-1 ring-hairline ring-inset transition-colors duration-(--duration-control) hover:bg-fill-hover hover:text-ink-1 data-popup-open:bg-fill-hover data-popup-open:text-ink-1"
        >
          <LiveDot />
          <Data className="shrink-0 text-ink-1">{runs.length}</Data>
          <span className="min-w-0 truncate">{top.name}</span>
          <span className="w-12 shrink-0">
            <Meter label={runMeterLabel(top)} value={percentOf(top)} />
          </span>
        </button>
      }
    >
      <RunningPanel runs={runs} />
    </Popover>
  );
}
