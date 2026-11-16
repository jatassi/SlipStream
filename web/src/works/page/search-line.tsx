import { Link } from "@tanstack/react-router";

import type { GrabsWorkSearching } from "../../api/gen/model";
import { ManualSearchButton } from "../../grabs/search";
import { type Lead, leadWords } from "../../grabs/search-words";
import { useNow } from "../../tasks/use-now";
import { Button } from "../../ui/button";
import { Popover } from "../../ui/popover";
import { Spinner } from "../../ui/spinner";
import { GrabNowButton } from "./grab-now";

const wordsTick = 30_000;

function Action({ lead, workId }: { lead: Lead; workId: string }) {
  if (lead.kind === "waiting") {
    return <GrabNowButton workId={workId} size="small" />;
  }
  if (lead.kind !== "result") {
    return null;
  }
  switch (lead.result) {
    case "nothing_acceptable":
      return <ManualSearchButton workId={workId} size="small" />;
    case "no_indexer":
      return (
        <Link to="/health" className="text-meta font-semibold text-ink-1 underline">
          Open Health
        </Link>
      );
    case "grabbed":
    case "waiting":
    case "nothing_found":
    case "no_root":
      return null;
    default:
      return lead.result satisfies never;
  }
}

const popoverLabels = { queued: "Why it's queued", searching: "Why it waits" } as const;

function Words({ lead }: { lead: Lead }) {
  const spinner = lead.kind === "searching" && <Spinner />;
  if ((lead.kind === "queued" || lead.kind === "searching") && lead.behind !== undefined) {
    return (
      <Popover
        label={popoverLabels[lead.kind]}
        trigger={
          <Button size="small" look="ghost">
            {spinner}
            {lead.words}
          </Button>
        }
      >
        <p className="text-ink-1">{lead.behind}</p>
      </Popover>
    );
  }
  return (
    <span className="flex items-center gap-2 text-ink-2">
      {spinner}
      {lead.words}
    </span>
  );
}

/**
 * The hero's one line about what SlipStream is doing for the Work:
 * searching or queued, with what holds it, waiting for a better Release, or
 * how its latest search ended. It shows nothing before the first search.
 */
export function SearchLine({
  state,
  workId,
  many,
}: {
  state: GrabsWorkSearching;
  workId: string;
  many: boolean;
}) {
  const now = useNow(wordsTick);
  const lead = leadWords(state, { now, zone: undefined }, many);
  if (lead === undefined) {
    return null;
  }
  return (
    <output aria-label="Search" className="flex flex-wrap items-center gap-x-3 gap-y-1">
      <Words lead={lead} />
      <Action lead={lead} workId={workId} />
    </output>
  );
}
