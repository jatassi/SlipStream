import { Link } from "@tanstack/react-router";

import type { GrabsGrab, IndexersQuery } from "../api/gen/model";
import { stageWord } from "../grabs/stages";
import { Data } from "../ui/data";
import { Empty } from "../ui/empty";
import { RelativeTime } from "../ui/relative-time";
import { Section } from "../ui/section";
import { outcomeWords, useWords } from "./words";

const listClass = "rounded-card bg-glass ring-1 ring-hairline ring-inset";

const rowClass =
  "flex min-h-(--row-height) flex-wrap items-center gap-x-3 px-4 py-2 not-first:shadow-[inset_0_1px_0_var(--color-hairline)]";

/** The newest searches, fetches and Tests, newest first. */
export function RecentQueries({ queries }: { queries: readonly IndexersQuery[] }) {
  return (
    <Section title="Recent queries" count={queries.length}>
      {queries.length === 0 ? (
        <Empty look="sheet">No searches, fetches or Tests yet.</Empty>
      ) : (
        <ul aria-label="Recent queries" className={listClass}>
          {queries.map((query) => (
            <li key={query.id} className={rowClass}>
              <span className="w-28 shrink-0 text-ink-3">{useWords[query.use]}</span>
              <span className="min-w-0 flex-1 basis-40 text-ink-1">{outcomeWords(query)}</span>
              <span className="shrink-0 text-ink-3">
                <RelativeTime at={query.reservedAt} form="clock" data />
              </span>
            </li>
          ))}
        </ul>
      )}
    </Section>
  );
}

/** The newest Grabs taken from the Indexer, newest first, each linking to its Work. */
export function RecentGrabs({ grabs }: { grabs: readonly GrabsGrab[] }) {
  return (
    <Section title="Recent grabs" count={grabs.length}>
      {grabs.length === 0 ? (
        <Empty look="sheet">No Grabs yet.</Empty>
      ) : (
        <ul aria-label="Recent grabs" className={listClass}>
          {grabs.map((grab) => (
            <li key={grab.id} className={rowClass}>
              <Link
                to="/works/$workId"
                params={{ workId: grab.workId }}
                className="min-w-0 flex-1 basis-40 truncate rounded-item text-ink-1 hover:underline"
              >
                <Data>{grab.releaseTitle}</Data>
              </Link>
              <span className="w-28 shrink-0 text-ink-3">{stageWord[grab.stage]}</span>
              <span className="shrink-0 text-ink-3">
                <RelativeTime at={grab.createdAt} form="clock" data />
              </span>
            </li>
          ))}
        </ul>
      )}
    </Section>
  );
}
