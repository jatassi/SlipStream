import { Meter as BaseMeter } from "@base-ui/react/meter";

import { Data } from "./data";

/**
 * One lane: a value against the rule that must be met and the goal past
 * it, each absent where it doesn't count the lane.
 */
export type Lane = {
  readonly label: string;
  readonly value: number;
  readonly rule?: number | undefined;
  readonly goal?: number | undefined;
  readonly words: (value: number) => string;
};

export type LaneSize = "small" | "large";

const sizes: { readonly [S in LaneSize]: { lane: string; mark: string } } = {
  small: { lane: "h-1", mark: "h-2" },
  large: { lane: "h-2", mark: "h-3.5" },
};

// The lane runs to the goal, or to the rule when the rule asks more, and
// is 0 where neither counts it.
const endOf = (lane: Lane) => Math.max(lane.rule ?? 0, lane.goal ?? 0);

const percent = (part: number, whole: number) => `${(Math.min(part, whole) / whole) * 100}%`;

const markLooks = { rule: "bg-ink-1", risk: "bg-warn", goal: "bg-ink-3" } as const;

function Mark({
  kind,
  at,
  end,
  size,
}: {
  kind: keyof typeof markLooks;
  at: number;
  end: number;
  size: LaneSize;
}) {
  return (
    <span
      aria-hidden="true"
      data-mark={kind}
      className={`absolute top-1/2 w-0.5 -translate-x-1/2 -translate-y-1/2 rounded-full ${sizes[size].mark} ${markLooks[kind]}`}
      style={{ left: percent(at, end) }}
    />
  );
}

function valueText(lane: Lane, risk: boolean) {
  const marks = [
    ...(lane.rule === undefined ? [] : [`rule ${lane.words(lane.rule)}${risk ? ", at risk" : ""}`]),
    ...(lane.goal === undefined ? [] : [`goal ${lane.words(lane.goal)}`]),
  ];
  return [lane.words(lane.value), ...(marks.length === 0 ? ["not counted"] : marks)].join(", ");
}

function LaneBar({ lane, risk, size }: { lane: Lane; risk: boolean; size: LaneSize }) {
  const end = endOf(lane);
  const track =
    end === 0 ? (
      <BaseMeter.Track
        className={`${sizes[size].lane} rounded-capsule ring-1 ring-hairline-strong ring-inset`}
      />
    ) : (
      <BaseMeter.Track className={`${sizes[size].lane} overflow-hidden rounded-capsule bg-fill`}>
        <BaseMeter.Indicator className="rounded-capsule bg-ink-2" />
      </BaseMeter.Track>
    );
  return (
    <BaseMeter.Root
      value={Math.min(lane.value, end)}
      max={Math.max(end, 1)}
      aria-label={lane.label}
      getAriaValueText={() => valueText(lane, risk)}
      className="relative w-full"
      render={(props) => (
        <div {...props}>
          {track}
          {lane.goal !== undefined && lane.goal < end && (
            <Mark kind="goal" at={lane.goal} end={end} size={size} />
          )}
          {lane.rule !== undefined && (
            <Mark kind={risk ? "risk" : "rule"} at={lane.rule} end={end} size={size} />
          )}
        </div>
      )}
    />
  );
}

/**
 * Lanes of progress toward a rule and a goal, such as a seed's ratio and
 * seeding time: small in a table row and large, with each lane's words, in
 * the Inspector. The rule's tick turns amber while `risk` holds.
 */
export function LaneMeter({
  lanes,
  risk,
  size = "small",
}: {
  lanes: readonly Lane[];
  risk: boolean;
  size?: LaneSize;
}) {
  return (
    <div className={`flex w-full flex-col ${size === "small" ? "gap-1.5" : "gap-3"}`}>
      {lanes.map((lane) =>
        size === "small" ? (
          <LaneBar key={lane.label} lane={lane} risk={risk} size={size} />
        ) : (
          <div key={lane.label} className="flex flex-col gap-1.5">
            <span className="flex items-baseline justify-between gap-3 text-meta">
              <span className="text-ink-2">{lane.label}</span>
              <Data className="text-ink-1">{laneWords(lane)}</Data>
            </span>
            <LaneBar lane={lane} risk={risk} size={size} />
          </div>
        ),
      )}
    </div>
  );
}

function laneWords(lane: Lane) {
  const end = endOf(lane);
  return end === 0 ? lane.words(lane.value) : `${lane.words(lane.value)} of ${lane.words(end)}`;
}
