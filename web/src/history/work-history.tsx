import type { QueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";

import type { WorksContentType } from "../api/gen/model";
import {
  getListHistorySuspenseQueryOptions,
  useListHistorySuspense,
} from "../api/gen/query/history/history";
import { ensureQuery } from "../api/query-client";
import { Section } from "../ui/section";
import { DayLists } from "./history-days";

const workHistoryParams = (workId: string) => ({ work: workId, limit: 10 });

export const loadWorkHistory = (queryClient: QueryClient, workId: string) =>
  ensureQuery(queryClient, getListHistorySuspenseQueryOptions(workHistoryParams(workId)));

/** A Work's ten newest History entries under their day's heading, and a link to all of them. */
export function WorkHistory({ type, workId }: { type: WorksContentType; workId: string }) {
  const { entries } = useListHistorySuspense(workHistoryParams(workId)).data;
  return (
    <Section
      title="History"
      action={
        <Link
          to="/activity/history"
          search={{ work: workId }}
          className="text-meta font-semibold text-ink-1 underline"
        >
          See all in History
        </Link>
      }
    >
      {entries.length === 0 ? (
        <p className="text-ink-3">This {type.work.one.toLowerCase()} has no History entries yet.</p>
      ) : (
        <DayLists entries={entries} poster={false} />
      )}
    </Section>
  );
}
