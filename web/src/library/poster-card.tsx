import { Link } from "@tanstack/react-router";
import { memo, useState } from "react";

import { Checkbox } from "../ui/checkbox";
import { type CellHandlers, focusTarget } from "./picking";
import type { LibraryWork } from "./rows";
import { leadFact } from "./values";
import { WorkStatus } from "./work-status";

function Art({ url, quiet }: { url: string | undefined; quiet: boolean }) {
  const [failed, setFailed] = useState<string>();
  return (
    <span
      data-poster={url === undefined ? "none" : "art"}
      className={`relative block aspect-[2/3] overflow-hidden rounded-card bg-glass-raised shadow-poster ring-1 ring-hairline ring-inset transition-opacity duration-(--duration-control) ${quiet ? "opacity-55" : ""}`}
    >
      {url !== undefined && failed !== url && (
        <img
          src={`${url}?size=medium`}
          alt=""
          loading="lazy"
          decoding="async"
          className="size-full object-cover"
          onError={() => {
            setFailed(url);
          }}
        />
      )}
    </span>
  );
}

export type PosterCardProps = {
  readonly work: LibraryWork;
  readonly picked: boolean;
  readonly selecting: boolean;
  readonly tabbable: boolean;
  readonly handlers: CellHandlers;
  readonly onPick: (id: string, range: boolean) => void;
};

/**
 * One Work in the grid: its poster, quiet when nothing of it is monitored,
 * its label, its first fact and its status. It draws again only when its
 * row or its pick changes.
 */
export const PosterCard = memo(function PosterCard({
  work,
  picked,
  selecting,
  tabbable,
  handlers,
  onPick,
}: PosterCardProps) {
  const fact = leadFact(work);
  return (
    <li data-work-id={work.id} className="group/card relative flex min-w-0 flex-col gap-1">
      <Link
        to="/works/$workId"
        params={{ workId: work.id }}
        {...{ [focusTarget]: "" }}
        tabIndex={tabbable ? 0 : -1}
        onClick={(event) => {
          handlers.onClick(event, work.id);
        }}
        onKeyDown={(event) => {
          handlers.onKeyDown(event, work.id);
        }}
        onFocus={() => {
          handlers.onFocus(work.id);
        }}
        className={`flex flex-col gap-2 rounded-card outline-offset-4 focus-visible:outline-2 focus-visible:outline-solid ${picked ? "[&>[data-poster]]:ring-2 [&>[data-poster]]:ring-glow" : ""}`}
      >
        <Art url={work.poster} quiet={work.monitored === "none"} />
        <span className="truncate text-body font-semibold text-ink-1">{work.label}</span>
      </Link>
      {fact !== undefined && <span className="truncate text-meta text-ink-3">{fact}</span>}
      <WorkStatus work={work} />
      <span
        className={`absolute top-2 left-2 rounded-item bg-popover px-1.5 ${selecting || picked ? "" : "opacity-0 group-hover/card:opacity-100 group-focus-within/card:opacity-100"}`}
      >
        <Checkbox
          label={`Pick ${work.label}`}
          hideLabel
          checked={picked}
          onCheckedChange={(_, by) => {
            onPick(work.id, by.shiftKey);
          }}
        />
      </span>
    </li>
  );
});
