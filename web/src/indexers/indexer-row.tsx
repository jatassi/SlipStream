import { Link } from "@tanstack/react-router";
import { useId } from "react";

import type { IndexersIndexer, NodesNode } from "../api/gen/model";
import { StatusLine } from "./status-line";
import { IndexerSwitch } from "./switches";
import { kindWords, nodeName } from "./words";

/** An Indexer in its Tracker's list: what it is, where it searches from, its status and its switches. */
export function IndexerRow({
  indexer,
  nodes,
}: {
  indexer: IndexersIndexer;
  nodes: readonly NodesNode[];
}) {
  const nameId = useId();
  return (
    <li
      aria-labelledby={nameId}
      className="@container relative first:rounded-t-card last:rounded-b-card not-first:before:absolute not-first:before:inset-x-0 not-first:before:top-0 not-first:before:h-px not-first:before:bg-hairline in-data-[density=touch]:not-first:before:left-4"
    >
      <div className="flex flex-col gap-2 px-4 py-3 @[680px]:flex-row @[680px]:items-center @[680px]:gap-6">
        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
          <Link
            id={nameId}
            to="/settings/indexers/$indexerId"
            params={{ indexerId: indexer.id }}
            className="self-start rounded-item font-semibold text-ink-1 underline-offset-4 hover:underline in-data-[density=touch]:text-card-title"
          >
            {indexer.name}
          </Link>
          <p className="text-meta text-ink-3">
            {kindWords[indexer.kind]} on {nodeName(nodes, indexer.homeNodeId)}
          </p>
          <StatusLine indexer={indexer} />
        </div>
        <div className="flex flex-wrap gap-x-5">
          <IndexerSwitch indexer={indexer} name="on" />
          <IndexerSwitch indexer={indexer} name="automaticSearch" />
          <IndexerSwitch indexer={indexer} name="rss" />
        </div>
      </div>
    </li>
  );
}
