import { Link } from "@tanstack/react-router";

import type { GrabsGrab } from "../api/gen/model";
import { Data } from "../ui/data";
import { StageTrack, StageWord } from "../ui/stage-track";
import { Mark } from "../ui/state-mark";
import { type Column, Table } from "../ui/table";
import { ForCell, JobMenu } from "./job-cells";
import { ProgressCell } from "./progress-cell";
import { importedMark, progressOf, skippedStages, stageWord } from "./stages";

export type GrabRow = { readonly grab: GrabsGrab; readonly work: string };

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
  if (grab.failure !== undefined) {
    return <span>{grab.failure.message}</span>;
  }
  if (grab.stage === "imported") {
    return <Mark mark={importedMark} />;
  }
  const progress = progressOf(grab);
  return (
    <span className="flex max-w-[220px] flex-col items-start gap-1.5">
      <StageWord
        word={stageWord[grab.stage]}
        title={work}
        progress={progress}
        skipped={skippedStages}
      />
      <StageTrack title={work} progress={progress} skipped={skippedStages} />
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

const menu: Column<GrabRow> = {
  id: "menu",
  header: "",
  cell: (row) => <JobMenu {...row} />,
  align: "end",
};

const layouts = {
  activity: [
    { id: "work", header: "Work", cell: (row) => <WorkCell {...row} /> },
    { id: "for", header: "For", cell: (row) => <ForCell {...row} /> },
    stage,
    progress,
    menu,
  ],
  work: [
    {
      id: "release",
      header: "Release",
      cell: ({ grab }) => <Data className="wrap-anywhere">{grab.releaseTitle}</Data>,
    },
    stage,
    progress,
    menu,
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
