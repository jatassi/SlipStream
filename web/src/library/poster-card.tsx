import { Link } from "@tanstack/react-router";
import { memo } from "react";

import { Artwork } from "../ui/artwork";
import { Checkbox } from "../ui/checkbox";
import { EpisodeStrip, byCounts, segmentsOf } from "../ui/episode-strip";
import { type CellHandlers, focusTarget } from "./picking";
import type { LibraryWork } from "./rows";
import { WorkMenu } from "./row-menu";
import { leadFact } from "./values";
import { WorkStatus } from "./work-status";

export type PosterCardProps = {
  readonly work: LibraryWork;
  readonly picked: boolean;
  readonly selecting: boolean;
  readonly tabbable: boolean;
  readonly handlers: CellHandlers;
  readonly onPick: (id: string, range: boolean) => void;
};

// A Work of many Items carries its strip on the poster, so every card keeps one height.
const stripOf = (work: LibraryWork) =>
  work.type.items === "many" && work.items > 0 ? (
    <EpisodeStrip
      strip={segmentsOf(byCounts(work.itemStatus), "poster")}
      look="poster"
      named={false}
    />
  ) : undefined;

function PickBox({
  work,
  picked,
  shown,
  onPick,
}: Pick<PosterCardProps, "work" | "picked" | "onPick"> & { shown: boolean }) {
  return (
    <span
      className={`absolute top-2 left-2 rounded-item bg-popover px-1.5 ${shown ? "" : "opacity-0 group-hover/card:opacity-100 group-focus-within/card:opacity-100"}`}
    >
      <Checkbox
        label={`Select ${work.label}`}
        hideLabel
        checked={picked}
        onCheckedChange={(_, by) => {
          onPick(work.id, by.shiftKey);
        }}
      />
    </span>
  );
}

/**
 * One Work in the grid: its poster, quiet when nothing of it is monitored
 * and with an episode strip over its bottom edge for a Work of many Items,
 * its label, its first fact and its status. It draws again only when its
 * row or its pick changes. Its menu, shown on hover or focus and always on
 * a touch screen, holds Search and Manual search.
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
        className={`flex flex-col gap-2 rounded-card outline-offset-4 focus-visible:outline-2 focus-visible:outline-solid ${picked ? "[&>[data-art]]:ring-2 [&>[data-art]]:ring-glow" : ""}`}
      >
        <Artwork
          use="poster"
          size="card"
          url={work.poster}
          label={work.label}
          quiet={work.monitored === "none"}
          overlay={stripOf(work)}
        />
        <span className="truncate text-body font-semibold text-ink-1">{work.label}</span>
      </Link>
      {fact !== undefined && <span className="truncate text-meta text-ink-3">{fact}</span>}
      <WorkStatus work={work} />
      <PickBox work={work} picked={picked} shown={selecting || picked} onPick={onPick} />
      <span className="absolute top-2 right-2 rounded-capsule bg-popover opacity-0 group-hover/card:opacity-100 group-focus-within/card:opacity-100 has-aria-expanded:opacity-100 pointer-coarse:opacity-100">
        <WorkMenu work={work} />
      </span>
    </li>
  );
});
