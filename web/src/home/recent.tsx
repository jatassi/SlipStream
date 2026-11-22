import { useSuspenseQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Suspense } from "react";

import type { HomeSummary } from "../api/gen/model";
import { getGetTaskSuspenseQueryOptions } from "../api/gen/query/tasks/tasks";
import { durationWords } from "../format";
import { Artwork } from "../ui/artwork";
import { Empty } from "../ui/empty";
import { List, ListRow } from "../ui/list";
import { PlainTime } from "../ui/relative-time";
// oxlint-disable-next-line import/max-dependencies -- Recent draws its rows, their artwork and times, and Housekeeping's window when empty
import { Section } from "../ui/section";
import { seeAllLink } from "./jobs";

/** Housekeeping prunes import records older than this, so Recent's empty state names it. */
const keepKey = "history.keepFor";

function NothingImported() {
  const { tuning } = useSuspenseQuery(getGetTaskSuspenseQueryOptions("housekeeping")).data;
  const keep = tuning.find((field) => field.key === keepKey);
  if (keep?.kind !== "duration") {
    throw new Error("Housekeeping has no Keep history for");
  }
  return (
    <Empty look="sheet">
      Nothing was imported in the last {durationWords(keep.value)}. Each download or folder Import
      shows here once its files are in the library.
    </Empty>
  );
}

/** The newest finished Imports, one row each, in words. */
export function Recent({ recent }: { recent: HomeSummary["recent"] }) {
  return (
    <Section title="Recent">
      {recent.rows.length === 0 ? (
        <Suspense>
          <NothingImported />
        </Suspense>
      ) : (
        <div className="flex flex-col gap-3">
          <List label="Recent">
            {recent.rows.map(({ importId, work, words, at }) => (
              <ListRow
                key={importId}
                leading={<Artwork use="poster" size="option" url={work.posterUrl ?? undefined} />}
                title={work.label}
                meta={
                  <>
                    {words} · <PlainTime at={at} form="age" />
                  </>
                }
                render={<Link to="/works/$workId" params={{ workId: work.id }} preload={false} />}
              />
            ))}
          </List>
          <Link
            to="/activity/history"
            search={{ groups: ["imports"] }}
            preload={false}
            className={seeAllLink}
          >
            See all in History
          </Link>
        </div>
      )}
    </Section>
  );
}
