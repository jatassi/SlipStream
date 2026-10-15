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

export function StageTrack({
  title,
  progress,
  skipped = noSkips,
  size = "default",
}: {
  title: string;
  progress: TrackProgress;
  skipped?: readonly Stage[];
  size?: keyof typeof sizeLook;
}) {
  const shown = useShell() === "phone" ? "large" : size;
  return (
    <div
      {...trackRole(title, progress, skipped)}
      className={`flex w-full gap-[3px] ${sizeLook[shown]}`}
    >
      {stages.map((stage) => (
        <span key={stage} className={segmentLook[segmentOf(stage, progress, skipped)]} />
      ))}
    </div>
  );
}
