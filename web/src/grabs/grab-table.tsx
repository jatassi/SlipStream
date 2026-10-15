import { Link } from "@tanstack/react-router";

import type { GrabsGrab } from "../api/gen/model";
import { bytes } from "../format";
import { Data } from "../ui/data";
import { LiveBar } from "../ui/live-bar";
import { StageTrack } from "../ui/stage-track";
import { type Column, Table } from "../ui/table";
import { movesBytes, progressOf, skippedStages, stageWord } from "./stages";

export type GrabRow = { readonly grab: GrabsGrab; readonly work: string };

const percent = ({ done, total }: { done: number; total: number }) =>
  total === 0 ? 0 : Math.floor((done / total) * 100);

function WorkCell({ grab, work }: GrabRow) {
  return (
    <span className="flex min-w-0 flex-col gap-0.5">
      <Link
        to="/works/$workId"
        params={{ workId: grab.workId }}
        className="truncate rounded-item hover:underline"
      >
        {work}
      </Link>
      <span className="truncate text-meta text-ink-3">
        <Data>{grab.releaseTitle}</Data>
      </span>
    </span>
  );
}

function StageCell({ grab, work }: GrabRow) {
  return (
    <span className="flex max-w-[220px] flex-col gap-1.5">
      <span>{stageWord[grab.stage]}</span>
      <StageTrack title={work} progress={progressOf(grab)} skipped={skippedStages} />
    </span>
  );
}

function ProgressCell({ grab }: GrabRow) {
  if (!movesBytes(grab)) {
    return null;
  }
  const progress = grab.progress ?? { done: 0, total: grab.releaseSize };
  const shown = percent(progress);
  return (
    <span className="flex max-w-[240px] flex-col gap-1.5">
      <span className="flex items-baseline justify-between gap-3 text-meta text-ink-3">
        <Data className="text-ink-1">{shown}%</Data>
        <span>
          <Data>{bytes(progress.done)}</Data> of <Data>{bytes(progress.total)}</Data>
        </span>
      </span>
      <LiveBar label={`${stageWord[grab.stage]} ${grab.releaseTitle}`} value={shown} />
    </span>
  );
}

const stage: Column<GrabRow> = {
  id: "stage",
  header: "Stage",
  cell: (row) => <StageCell {...row} />,
};

const progress: Column<GrabRow> = {
  id: "progress",
  header: "Progress",
  cell: (row) => <ProgressCell {...row} />,
};

const layouts = {
  activity: [{ id: "work", header: "Work", cell: (row) => <WorkCell {...row} /> }, stage, progress],
  work: [
    {
      id: "release",
      header: "Release",
      cell: ({ grab }) => <Data className="wrap-anywhere">{grab.releaseTitle}</Data>,
    },
    stage,
    progress,
  ],
} as const satisfies Record<string, readonly [Column<GrabRow>, ...Column<GrabRow>[]]>;

export function GrabTable({
  rows,
  layout,
}: {
  rows: readonly GrabRow[];
  layout: keyof typeof layouts;
}) {
  return (
    <Table
      label="Grabs"
      lines={2}
      stackBelow={620}
      rows={rows}
      rowKey={({ grab }) => grab.id}
      columns={layouts[layout]}
    />
  );
}
