import type { QueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";

import type { WorksContentType } from "../api/gen/model";
import {
  getListHistorySuspenseQueryOptions,
  useListHistorySuspense,
} from "../api/gen/query/history/history";
import { ensureQuery } from "../api/query-client";
import { List } from "../ui/list";
import { Section } from "../ui/section";
import { HistoryRow } from "./history-row";

const workHistoryParams = (workId: string) => ({ work: workId, limit: 10 });

export const loadWorkHistory = (queryClient: QueryClient, workId: string) =>
  ensureQuery(queryClient, getListHistorySuspenseQueryOptions(workHistoryParams(workId)));

/** A Work's ten newest History entries, and a link to all of them. */
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
        <List label="History">
          {entries.map((entry) => (
            <HistoryRow key={entry.id} entry={entry} />
          ))}
        </List>
      )}
    </Section>
  );
}
