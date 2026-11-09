import type { QueryClient } from "@tanstack/react-query";

import { getListRootsQueryKey, type listRoots } from "../api/gen/query/roots/roots";
import { getListOpenTaskRunsQueryKey, type listOpenTaskRuns } from "../api/gen/query/tasks/tasks";
import { RootsRunning } from "../api/gen/zod/slipStream.schemas.zod";
import type { PayloadOf } from "./types";

type Progressed = PayloadOf<"tasks.progress">;

const libraryScan = "library-scan";

function patchOpenRun(queryClient: QueryClient, data: Progressed) {
  queryClient.setQueryData<Awaited<ReturnType<typeof listOpenTaskRuns>>>(
    getListOpenTaskRunsQueryKey(),
    (open) =>
      open && {
        ...open,
        runs: open.runs.map((run) =>
          run.runId === data.runId ? { ...run, done: data.done, total: data.total } : run,
        ),
      },
  );
}

// A Library scan unit is keyed root:<id>.
function patchScanLine(queryClient: QueryClient, { runId, unit }: Progressed) {
  if (unit === undefined) {
    return;
  }
  const running = RootsRunning.parse({ runId, ...unit });
  queryClient.setQueryData<Awaited<ReturnType<typeof listRoots>>>(
    getListRootsQueryKey(),
    (list) =>
      list && {
        ...list,
        roots: list.roots.map((root) =>
          `root:${root.id}` === unit.key ? { ...root, scan: { ...root.scan, running } } : root,
        ),
      },
  );
}

export function takeProgress(queryClient: QueryClient, data: Progressed) {
  patchOpenRun(queryClient, data);
  if (data.taskId === libraryScan) {
    patchScanLine(queryClient, data);
  }
}
