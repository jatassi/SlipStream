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
import { needsYou, withoutReplacedFailures } from "./stages";

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
  return withoutReplacedFailures(
    useListGrabsSuspense().data.grabs.filter((grab) => grab.workId === workId),
  );
}

/** What the Grabs section says while it lists none, pointing at the failed ones under Needs you. */
function emptyGrabsWords(failed: number): string {
  if (failed === 0) {
    return "No Grabs yet. A search that finds a Release grabs it.";
  }
  return failed === 1
    ? "No other Grabs. The failed one is under Needs you."
    : `No other Grabs. The ${String(failed)} failed ones are under Needs you.`;
}

export function Grabs({ workId, work }: { workId: string; work: string }) {
  const all = useWorkGrabs(workId);
  const grabs = all.filter((grab) => !needsYou(grab));
  return (
    <Section title="Grabs" count={grabs.length}>
      {grabs.length === 0 ? (
        <p className="text-ink-3">{emptyGrabsWords(all.length - grabs.length)}</p>
      ) : (
        <GrabTable layout="work" rows={grabs.map((grab) => ({ grab, work }))} />
      )}
    </Section>
  );
}
