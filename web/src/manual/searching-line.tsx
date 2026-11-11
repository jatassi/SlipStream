import { Link } from "@tanstack/react-router";
import { use, useState } from "react";

import type { GrabsIndexerLine, GrabsManualSearch } from "../api/gen/model";
import { agoWords } from "../format";
import { Button } from "../ui/button";
import { Details } from "../ui/details";
import { SearchField } from "../ui/search-field";
import { SignalChip } from "../ui/signal-chip";
import { Spinner } from "../ui/spinner";
import { ReadingContext } from "./reading";

/** Starts the Manual search again for the sheet's scope, with a typed query or the one it had. */
export type Restart = (query: string | null) => void;

function IndexerLine({
  line,
  busy,
  onRetry,
  onLeave,
}: {
  line: GrabsIndexerLine;
  busy: boolean;
  onRetry: () => void;
  onLeave: () => void;
}) {
  return (
    <li className="col-span-3 grid grid-cols-subgrid items-center gap-y-1 py-2">
      <span className="flex items-center gap-2">
        <span className="font-semibold text-ink-1">{line.name}</span>
        {line.failed && <SignalChip kind="fail">Failed</SignalChip>}
      </span>
      <span className={`min-w-0 ${line.failed ? "text-fail-ink" : ""}`}>{line.sentence}</span>
      <span className="flex items-center justify-end gap-2">
        {line.indexerId !== null && (
          <Button
            look="ghost"
            size="small"
            nativeButton={false}
            render={
              <Link
                to="/settings/indexers/$indexerId"
                params={{ indexerId: line.indexerId }}
                onClick={onLeave}
              />
            }
          >
            Open Indexer
          </Button>
        )}
        {line.done && (
          <Button size="small" busy={busy} onClick={onRetry}>
            Retry
          </Button>
        )}
      </span>
    </li>
  );
}

function IndexerLines({
  search,
  busy,
  onRetry,
  onLeave,
}: {
  search: GrabsManualSearch;
  busy: boolean;
  onRetry: () => void;
  onLeave: () => void;
}) {
  return (
    <ul
      aria-label="Indexers"
      className="grid grid-cols-[auto_minmax(0,1fr)_auto] gap-x-4 divide-y divide-hairline"
    >
      {search.indexers.map((line) => (
        <IndexerLine
          key={line.indexerId ?? line.name}
          line={line}
          busy={busy}
          onRetry={onRetry}
          onLeave={onLeave}
        />
      ))}
    </ul>
  );
}

/**
 * How many Indexers responded, who started the search and when, and Search
 * again. Details lists each Indexer's outcome with Open Indexer, and Retry
 * once it has one, which is a new Manual search.
 */
export function SearchingLine({
  search,
  busy,
  onRestart,
  onLeave,
}: {
  search: GrabsManualSearch;
  busy: boolean;
  onRestart: Restart;
  onLeave: () => void;
}) {
  const { now } = use(ReadingContext);
  const again = () => {
    onRestart(search.query);
  };
  return (
    <Details
      summary={
        <span className="flex flex-col gap-0.5">
          <span className="flex items-center gap-2">
            {search.searching && <Spinner />}
            {search.line}
          </span>
          <span className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
            {`Started ${agoWords(search.startedAt, now)} by ${search.startedBy}`}
          </span>
        </span>
      }
      action={
        <Button size="small" busy={busy} onClick={again}>
          Search again
        </Button>
      }
    >
      <IndexerLines search={search} busy={busy} onRetry={again} onLeave={onLeave} />
    </Details>
  );
}

/** A typed query for a scope of one Item, searched in place of its usual terms. */
export function QueryField({
  query,
  busy,
  onRestart,
}: {
  query: string | null;
  busy: boolean;
  onRestart: Restart;
}) {
  const [typed, setTyped] = useState(query ?? "");
  return (
    <form
      className="flex items-center gap-2"
      onSubmit={(event) => {
        event.preventDefault();
        const trimmed = typed.trim();
        onRestart(trimmed === "" ? null : trimmed);
      }}
    >
      <SearchField
        label="Query"
        placeholder="Search by another name"
        value={typed}
        onValueChange={setTyped}
        className="min-w-0 flex-1"
      />
      <Button type="submit" busy={busy}>
        Search
      </Button>
    </form>
  );
}
