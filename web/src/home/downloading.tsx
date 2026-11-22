import { Link } from "@tanstack/react-router";

import type {
  DownloadclientsNodeTorrents,
  NodesNode,
  NodesTraffic,
  ShellCounts,
} from "../api/gen/model";
import { fillOf } from "../shell/counts";
import { Rates, moving, totalOf, useLatestTraffic } from "../torrents/rates";
import { ProgressBar } from "../ui/progress";
import { Section } from "../ui/section";
import { bodyText } from "../ui/type";
import { nodeCounts } from "./words";

const dot = (
  <span aria-hidden="true" className="px-1.5 text-ink-4">
    ·
  </span>
);

/** A Node's failed or warned torrents, linking to that Node's problems on Torrents. */
function Problems({ node, words, ink }: { node: string; words: string; ink: string }) {
  return (
    <>
      {dot}
      <Link
        to="/activity/torrents"
        search={{ tab: "problems", node }}
        preload={false}
        className={`rounded-item underline-offset-4 hover:underline ${ink}`}
      >
        {words}
      </Link>
    </>
  );
}

function NodeLine({
  counted,
  name,
  traffic,
}: {
  counted: DownloadclientsNodeTorrents;
  name: string;
  traffic: NodesTraffic | undefined;
}) {
  const { plain, failures, warnings } = nodeCounts(counted);
  const pace = traffic?.nodes.find(({ id }) => id === counted.nodeId);
  return (
    <li
      className={`px-4 py-3 text-ink-2 ${bodyText} not-first:shadow-[inset_0_1px_0_var(--color-hairline)]`}
    >
      <span className="font-semibold text-ink-1">{name}</span>
      {pace !== undefined && moving(pace) && (
        <>
          {dot}
          <Rates pace={pace} />
        </>
      )}
      {plain.map((words) => (
        <span key={words}>
          {dot}
          {words}
        </span>
      ))}
      {failures !== undefined && (
        <Problems node={counted.nodeId} words={failures} ink="text-fail-ink" />
      )}
      {warnings !== undefined && (
        <Problems node={counted.nodeId} words={warnings} ink="text-warn" />
      )}
    </li>
  );
}

/**
 * The torrent client side: the rates across every Node while bytes move, every download's progress as one bar, and each Downloading Node's
 * counts. Home leaves it out while no Node downloads.
 */
export function Downloading({
  counted,
  counts,
  nodes,
}: {
  counted: readonly DownloadclientsNodeTorrents[];
  counts: ShellCounts;
  nodes: readonly NodesNode[];
}) {
  const traffic = useLatestTraffic();
  const total = traffic === undefined ? undefined : totalOf(traffic);
  const pace = total !== undefined && moving(total) ? total : undefined;
  const fill = fillOf(counts, "downloads");
  return (
    <Section title="Downloading">
      <div className="flex flex-col gap-3">
        {(pace !== undefined || fill !== undefined) && (
          <div className={`flex flex-col gap-2 ${bodyText}`}>
            {pace !== undefined && (
              <span className="text-ink-1">
                <Rates pace={pace} />
              </span>
            )}
            {fill !== undefined && (
              <ProgressBar
                label="Every download"
                value={fill.percent}
                motion={fill.paused ? "settled" : "live"}
              />
            )}
            {fill?.paused === true && <p className="text-ink-2">All downloads are paused.</p>}
          </div>
        )}
        <ul
          aria-label="Downloading Nodes"
          className="rounded-card bg-glass ring-1 ring-hairline ring-inset"
        >
          {counted.map((each) => (
            <NodeLine
              key={each.nodeId}
              counted={each}
              name={nodes.find(({ id }) => id === each.nodeId)?.name ?? "A removed Node"}
              traffic={traffic}
            />
          ))}
        </ul>
      </div>
    </Section>
  );
}
