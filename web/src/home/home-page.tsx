import { Page } from "../shell/page";
import type { WorkNames } from "../torrents/acting";
import { Downloading } from "./downloading";
import { Health } from "./health";
import { HomeMenu } from "./home-menu";
import { Jobs } from "./jobs";
import { NeedsYou } from "./needs-you";
import { useHomeReads } from "./reads";
import { Recent } from "./recent";
import { HomeStatus } from "./status";
// oxlint-disable-next-line import/max-dependencies -- Home draws every section from its own reads
import { Storage } from "./storage";

/**
 * The Works Home's summary names, for a Torrent's menu. A Torrent shared with
 * a Work Home doesn't show reads as another Work, so Home never reads the
 * Library.
 */
const namesOf =
  (labels: ReadonlyMap<string, string>): WorkNames =>
  (id) =>
    labels.get(id) ?? "Another Work";

export function HomePage() {
  const { summary, counts, report, volumes, nodes, contentTypes, counted } = useHomeReads();
  const labels = new Map(
    [...summary.jobs.rows, ...summary.needsYou.rows, ...summary.recent.rows].map(({ work }) => [
      work.id,
      work.label,
    ]),
  );
  return (
    <Page title="Home" action={<HomeMenu />}>
      <HomeStatus summary={summary} counts={counts} report={report} />
      <NeedsYou needsYou={summary.needsYou} types={contentTypes} />
      <Jobs
        jobs={summary.jobs}
        names={namesOf(labels)}
        nodes={nodes}
        manyNodes={counted.length > 1}
      />
      <div className="@container mt-6 in-data-[density=touch]:mt-7">
        <div className="grid gap-x-6 @[900px]:grid-cols-2">
          <div>
            <Recent recent={summary.recent} />
          </div>
          <div className="mt-6 @[900px]:mt-0 in-data-[density=touch]:mt-7">
            {counted.length > 0 && <Downloading counted={counted} counts={counts} nodes={nodes} />}
            <Health report={report} />
            <Storage volumes={volumes} nodes={nodes} />
          </div>
        </div>
      </div>
    </Page>
  );
}
