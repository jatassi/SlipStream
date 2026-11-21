import { definedWord } from "./defined";
import { Popover } from "./popover";
import { useShell } from "./shell-context";

export const stages = [
  "searching",
  "downloading",
  "transfer",
  "import",
  "conversion",
  "available",
] as const;

export type Stage = (typeof stages)[number];

export const stageLabel: { readonly [S in Stage]: string } = {
  searching: "Searching",
  downloading: "Downloading",
  transfer: "Transfer",
  import: "Import",
  conversion: "Conversion",
  available: "Available",
};

export type TrackProgress =
  | { kind: "running"; stage: Stage }
  | { kind: "failed"; stage: Stage }
  | { kind: "available" };

type Segment = "done" | "current" | "ahead" | "not-needed" | "failed";

const segmentLook: { readonly [S in Segment]: string } = {
  done: "flex-1 bg-ink-2",
  current: "flex-1 bg-ink-1 shadow-live-glow",
  ahead: "flex-1 bg-fill",
  "not-needed": "flex-[0.45] ring-1 ring-hairline-strong ring-inset",
  failed: "flex-1 bg-fail",
};

const segmentWord: { readonly [S in Segment]: string } = {
  done: "Done",
  current: "In progress",
  ahead: "Not started",
  "not-needed": "Not needed",
  failed: "Failed",
};

const sizeLook = {
  default: "h-1.5 *:rounded-[1.5px]",
  large: "h-2 *:rounded-[2px]",
} as const;

function segmentOf(stage: Stage, progress: TrackProgress, skipped: readonly Stage[]): Segment {
  if (skipped.includes(stage)) {
    return "not-needed";
  }
  if (progress.kind === "available") {
    return "done";
  }
  const at = stages.indexOf(progress.stage);
  const index = stages.indexOf(stage);
  if (index < at) {
    return "done";
  }
  if (index > at) {
    return "ahead";
  }
  return progress.kind === "running" ? "current" : "failed";
}

function trackRole(title: string, progress: TrackProgress, skipped: readonly Stage[]) {
  if (progress.kind === "available") {
    return { role: "img", "aria-label": `${title} is Available` } as const;
  }
  if (progress.kind === "failed") {
    return {
      role: "img",
      "aria-label": `${title} failed at ${stageLabel[progress.stage]}`,
    } as const;
  }
  const needed = stages.filter((stage) => !skipped.includes(stage));
  return {
    role: "progressbar",
    "aria-label": `${stageLabel[progress.stage]} ${title}`,
    "aria-valuemin": 0,
    "aria-valuemax": needed.length,
    "aria-valuenow": needed.indexOf(progress.stage),
    "aria-valuetext": stageLabel[progress.stage],
  } as const;
}

const noSkips: readonly Stage[] = [];

/**
 * A job's track, one segment per stage. A phone draws it large with its
 * ends named, so a skipped stage's outline sits inside a journey that reads
 * Searching to Available.
 */
export function StageTrack({
  title,
  progress,
  skipped = noSkips,
  size = "default",
  labels = "none",
}: {
  title: string;
  progress: TrackProgress;
  skipped?: readonly Stage[];
  size?: keyof typeof sizeLook;
  labels?: "none" | "ends";
}) {
  const phone = useShell() === "phone";
  const track = (
    <div
      {...trackRole(title, progress, skipped)}
      className={`flex w-full gap-[3px] ${sizeLook[phone ? "large" : size]}`}
    >
      {stages.map((stage) => (
        <span key={stage} className={segmentLook[segmentOf(stage, progress, skipped)]} />
      ))}
    </div>
  );
  if (!phone && labels === "none") {
    return track;
  }
  return (
    <div className="flex w-full flex-col gap-1">
      {track}
      <span aria-hidden="true" className="flex justify-between text-meta text-ink-3">
        <span>{stageLabel.searching}</span>
        <span>{stageLabel.available}</span>
      </span>
    </div>
  );
}

/** The six stops, each with its segment's look and its word for this track. */
function StageLegend({
  progress,
  skipped,
}: {
  progress: TrackProgress;
  skipped: readonly Stage[];
}) {
  return (
    <ul className="flex flex-col gap-2">
      {stages.map((stage) => {
        const segment = segmentOf(stage, progress, skipped);
        return (
          <li key={stage} className="flex items-center gap-3 text-meta">
            <span className="flex h-1.5 w-4 shrink-0 *:rounded-[1.5px]">
              <span className={segmentLook[segment]} />
            </span>
            <span className="flex-1 text-ink-1">{stageLabel[stage]}</span>
            <span className="text-ink-3">{segmentWord[segment]}</span>
          </li>
        );
      })}
    </ul>
  );
}

/** A job's stage word over its track, which opens the track's legend on hover, tap and Enter. */
export function StageWord({
  word,
  title,
  progress,
  skipped = noSkips,
}: {
  word: string;
  title: string;
  progress: TrackProgress;
  skipped?: readonly Stage[];
}) {
  return (
    <Popover
      label={`Stages of ${title}`}
      opensOn="hover"
      trigger={
        <button type="button" className={`rounded-chip text-left ${definedWord}`}>
          {word}
        </button>
      }
    >
      <StageLegend progress={progress} skipped={skipped} />
    </Popover>
  );
}
