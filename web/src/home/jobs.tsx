import { Link } from "@tanstack/react-router";

import type { HomeJob, HomeSummary, NodesNode } from "../api/gen/model";
import { StageCell } from "../grabs/grab-table";
import { JobMenu } from "../grabs/job-menu";
import { ProgressCell } from "../grabs/progress-cell";
import type { WorkNames } from "../torrents/acting";
import { Artwork } from "../ui/artwork";
import { Data } from "../ui/data";
import { Empty } from "../ui/empty";
import { Section } from "../ui/section";
// oxlint-disable-next-line import/max-dependencies -- a Job's row draws its Work, stage, progress and menu
import { type Column, Table } from "../ui/table";
import { bodyText } from "../ui/type";
import { jobAnchor, useHomeTint } from "./tint";

export const seeAllLink = `w-fit rounded-item text-ink-2 underline-offset-4 hover:text-ink-1 hover:underline ${bodyText}`;

function WorkCell({ job: { grab, work } }: { job: HomeJob }) {
  return (
    <span className="flex min-w-0 items-center gap-3">
      <Artwork use="poster" size="row" url={work.posterUrl ?? undefined} />
      <span className="flex min-w-0 flex-col gap-0.5">
        <Link
          to="/works/$workId"
          params={{ workId: work.id }}
          preload={false}
          className="truncate rounded-item hover:underline"
        >
          {work.label}
        </Link>
        <span className="truncate text-meta text-ink-3">
          <Data>{grab.releaseTitle}</Data>
        </span>
      </span>
    </span>
  );
}

type Row = { readonly job: HomeJob; readonly names: WorkNames; readonly node: string | undefined };

const columns = (nodes: boolean): readonly [Column<Row>, ...Column<Row>[]] => [
  { id: "work", header: "Work", cell: ({ job }) => <WorkCell job={job} /> },
  {
    id: "stage",
    header: "Stage",
    cell: ({ job }) => <StageCell grab={job.grab} work={job.work.label} />,
  },
  {
    id: "progress",
    header: "Progress",
    cell: ({ job }) => <ProgressCell grab={job.grab} torrent={job.torrent ?? undefined} />,
  },
  ...(nodes ? [{ id: "node", header: "Node", hideBelow: 760, cell: ({ node }: Row) => node }] : []),
  {
    id: "menu",
    header: "",
    align: "end",
    cell: ({ job, names }) => (
      <JobMenu
        grab={job.grab}
        work={job.work.label}
        torrent={job.torrent ?? undefined}
        names={names}
      />
    ),
  },
];

/**
 * The moving Grabs nearest Available first, as the server orders them, the
 * top one tinting the page. A Node's name shows only when more than one
 * Node downloads.
 */
export function Jobs({
  jobs,
  names,
  nodes,
  manyNodes,
}: {
  jobs: HomeSummary["jobs"];
  names: WorkNames;
  nodes: readonly NodesNode[];
  manyNodes: boolean;
}) {
  const tint = useHomeTint(jobs.rows);
  const nodeName = (id: string) => nodes.find((node) => node.id === id)?.name;
  return (
    <Section title="Jobs" {...(jobs.moving > 0 && { count: jobs.moving })}>
      {jobs.rows.length === 0 ? (
        <Empty look="sheet">
          No jobs are moving. Open a Work and choose Search to find a release for it.
        </Empty>
      ) : (
        <div className="flex flex-col gap-3">
          <div {...tint}>
            <Table
              label="Jobs"
              lines={2}
              stackBelow={620}
              rows={jobs.rows.map((job) => ({ job, names, node: nodeName(job.grab.nodeId) }))}
              rowKey={({ job }) => job.grab.id}
              anchor={({ job }) => jobAnchor(job)}
              columns={columns(manyNodes)}
            />
          </div>
          {jobs.moving > jobs.rows.length && (
            <Link to="/activity" preload={false} className={seeAllLink}>
              See all {jobs.moving} jobs
            </Link>
          )}
        </div>
      )}
    </Section>
  );
}
