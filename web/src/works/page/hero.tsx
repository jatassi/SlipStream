import type { ReactNode } from "react";

import type { FilesSummary, WorksContentType, WorksWorkView } from "../../api/gen/model";
import { Data } from "../../ui/data";
import { placed } from "./about";
import { factsLine } from "./facts";
import { aboutText, ratingValue } from "./fields-format";
import { Backdrop, PosterArt, TitleHeading, artOf } from "./hero-art";
import { type HeroActions, Actions } from "./hero-actions";
import { Overview } from "./overview";
import { Summary } from "./summary";

/**
 * The Work page's header over its backdrop: the poster, the title
 * treatment, the facts line and first rating, what its cells add up to,
 * the overview, then the actions.
 */
export function Hero({
  type,
  work,
  summary,
  warning,
  actions,
}: {
  type: WorksContentType;
  work: WorksWorkView;
  summary: FilesSummary;
  warning: ReactNode;
  actions: HeroActions;
}) {
  const backdrop = artOf(work, type.art.backdrop);
  const overview = placed(type, work, "overview").at(0);
  const [rating] = work.ratings;
  return (
    <header className="relative isolate">
      {backdrop !== undefined && <Backdrop url={`${backdrop}?size=large`} />}
      <div className="flex flex-col gap-5 pt-[clamp(24px,14vw,180px)] sm:flex-row sm:items-end">
        <PosterArt url={artOf(work, type.art.poster)} />
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
          <Summary summary={summary} />
        </div>
      </div>
      <div className="mt-4 flex flex-col gap-4">
        {overview !== undefined && <Overview text={aboutText(overview.value)} />}
        {warning}
        <Actions work={work} actions={actions} />
      </div>
    </header>
  );
}
