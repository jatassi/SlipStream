import type { ReactNode } from "react";

import type { IndexersIndexer, IndexersQuery, NodesNode } from "../api/gen/model";
import { clock } from "../format";
import { KeyValues } from "../routes/_app/nodes/-facts";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { Section } from "../ui/section";
import { nodeName, outcomeWords, useWords } from "./words";

/** The newest searches, fetches and Tests, newest first. */
export function RecentQueries({ queries }: { queries: readonly IndexersQuery[] }) {
  return (
    <Section title="Recent queries" count={queries.length}>
      {queries.length === 0 ? (
        <p className="text-ink-3">No searches, fetches or Tests yet.</p>
      ) : (
        <ul
          aria-label="Recent queries"
          className="rounded-card bg-glass ring-1 ring-hairline ring-inset"
        >
          {queries.map((query) => (
            <li
              key={query.id}
              className="flex min-h-(--row-height) flex-wrap items-center gap-x-3 px-4 py-2 not-first:shadow-[inset_0_1px_0_var(--color-hairline)]"
            >
              <span className="w-28 shrink-0 text-ink-3">{useWords[query.use]}</span>
              <span className="min-w-0 flex-1 basis-40 text-ink-1">{outcomeWords(query)}</span>
              <Data className="shrink-0 text-ink-3">{clock(query.reservedAt)}</Data>
            </li>
          ))}
        </ul>
      )}
    </Section>
  );
}

function sessionWords(indexer: IndexersIndexer, nodes: readonly NodesNode[]): string {
  const { session } = indexer;
  if (session === null) {
    return indexer.signsIn ? "Not signed in on any Node" : "Doesn't sign in";
  }
  const how = session.pasted ? "from a pasted browser cookie" : "signed in";
  return `On ${nodeName(nodes, session.nodeId)}, ${how} at ${clock(session.earnedAt)}`;
}

const words = (text: string): ReactNode => <span>{text}</span>;

type Row = readonly [string, ReactNode];

function machineryOf(indexer: IndexersIndexer, nodes: readonly NodesNode[]): Row[] {
  const { status, feedHost } = indexer;
  const failure = status.lastFailure;
  return [
    ...(feedHost === null ? [] : [["Feed host", feedHost] as const]),
    ["Session", words(sessionWords(indexer, nodes))],
    ["State", status.state],
    ["Backoff step", String(status.rung)],
    ...(failure === null
      ? []
      : [["Last failure", words(`${failure.text} at ${clock(failure.at)}`)] as const]),
    ["Search modes", indexer.searchModes.map((mode) => mode.mode).join(", ") || "None"],
    ["Revision", String(indexer.revision)],
    ["Added", clock(indexer.createdAt)],
  ];
}

/** When it last responded, and under Details its session, backoff and wire facts. */
export function IndexerMachinery({
  indexer,
  nodes,
}: {
  indexer: IndexersIndexer;
  nodes: readonly NodesNode[];
}) {
  const success = indexer.status.successAt;
  return (
    <Details
      summary={
        success === null ? "It hasn't responded yet." : `It last responded at ${clock(success)}.`
      }
    >
      <div className="mt-2 w-full">
        <KeyValues rows={machineryOf(indexer, nodes)} />
      </div>
    </Details>
  );
}
