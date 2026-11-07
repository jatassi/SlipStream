import type { QueryClient } from "@tanstack/react-query";

import { getListTorrentsSuspenseQueryOptions } from "../api/gen/query/downloadclients/downloadclients";
import {
  getListGrabsSuspenseQueryOptions,
  getListSearchesSuspenseQueryOptions,
  useListGrabsSuspense,
} from "../api/gen/query/grabs/grabs";
import { getListTransfersSuspenseQueryOptions } from "../api/gen/query/transfers/transfers";
import { ensureQuery } from "../api/query-client";
import { Section } from "../ui/section";
import { GrabTable } from "./grab-table";
import { isFailed, withoutRetriedFailures } from "./stages";

/** What a Work's searches and Grabs sections read. */
export async function loadWorkGrabs(queryClient: QueryClient, workId: string): Promise<void> {
  await Promise.all([
    ensureQuery(queryClient, getListSearchesSuspenseQueryOptions(workId)),
    ensureQuery(queryClient, getListGrabsSuspenseQueryOptions()),
    ensureQuery(queryClient, getListTorrentsSuspenseQueryOptions()),
    ensureQuery(queryClient, getListTransfersSuspenseQueryOptions()),
  ]);
}

function useWorkGrabs(workId: string) {
  return withoutRetriedFailures(
    useListGrabsSuspense().data.grabs.filter((grab) => grab.workId === workId),
  );
}

export function Grabs({ workId, work }: { workId: string; work: string }) {
  const grabs = useWorkGrabs(workId).filter((grab) => !isFailed(grab));
  return (
    <Section title="Grabs" count={grabs.length}>
      {grabs.length === 0 ? (
        <p className="text-ink-3">No Grabs yet. A search that finds a Release grabs it.</p>
      ) : (
        <GrabTable layout="work" rows={grabs.map((grab) => ({ grab, work }))} />
      )}
    </Section>
  );
}
