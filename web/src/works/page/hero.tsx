import type { ReactNode } from "react";

import type { FilesSummary, WorksContentType, WorksWorkView } from "../../api/gen/model";
import { Artwork } from "../../ui/artwork";
import { Data } from "../../ui/data";
import { useShell } from "../../ui/shell-context";
import { placed } from "./about";
import { factsLine } from "./facts";
import { aboutText, ratingValue } from "./fields-format";
import { Backdrop, TitleHeading, artOf } from "./hero-art";
import { MoveBand } from "./move-band";
// oxlint-disable-next-line import/max-dependencies -- the hero lays out its art, facts, summary, search line, Move band and actions
import { type HeroControls, Actions } from "./hero-actions";
import { Overview } from "./overview";
import { Summary } from "./summary";

/**
 * The Work page's header over its backdrop: the poster, the title
 * treatment, the facts line and first rating, what its cells add up to,
 * the line about its searches, the overview, its Move while one runs, then
 * the actions.
 */
export function Hero({
  type,
  work,
  summary,
  warning,
  searchLine,
  actions,
}: {
  type: WorksContentType;
  work: WorksWorkView;
  summary: FilesSummary;
  warning: ReactNode;
  /** What SlipStream is doing for the Work: under the title row, and on phone under the actions. */
  searchLine: ReactNode;
  actions: HeroControls;
}) {
  const backdrop = artOf(work, type.art.backdrop);
  const overview = placed(type, work, "overview").at(0);
  const [rating] = work.ratings;
  const phone = useShell() === "phone";
  return (
    <header className="relative isolate">
      {backdrop !== undefined && <Backdrop route={backdrop} />}
      <div className="flex flex-col gap-5 pt-[clamp(24px,14vw,180px)] sm:flex-row sm:items-end">
        <Artwork use="poster" size="hero" url={artOf(work, type.art.poster)} label={work.label} />
        <div className="flex min-w-0 flex-col gap-3">
          <TitleHeading label={work.label} treatment={artOf(work, type.art.title)} />
          <p className="flex flex-wrap items-baseline gap-x-3 gap-y-1 text-ink-2">
            <span>{factsLine(type, work.fields)}</span>
            {rating !== undefined && (
              <span>
                <span className="font-semibold text-ink-1">{rating.label}</span>{" "}
                <Data className="text-ink-1">{ratingValue(rating)}</Data>
              </span>
            )}
          </p>
          <Summary summary={summary} holds={type.items} />
          {!phone && searchLine}
        </div>
      </div>
      <div className="mt-4 flex flex-col gap-4">
        {overview !== undefined && <Overview text={aboutText(overview.value)} />}
        <MoveBand move={work.move} />
        {warning}
        <Actions work={work} actions={actions} />
        {phone && searchLine}
      </div>
    </header>
  );
}
