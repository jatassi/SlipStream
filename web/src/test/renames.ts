import { HttpResponse, http, sse } from "msw";

import type {
  FilesRenamePlan,
  FilesRenameRow,
  FilesWorkRenamePlan,
  PlanInputBody,
} from "../api/gen/model";
import { PlanRenamesBody, RenameFilesBody } from "../api/gen/zod/files/files.zod";
import type { LiveMap } from "./handlers";

type Confirmed = ReturnType<typeof RenameFilesBody.parse>["works"];

/** One Work's plan as the server holds it: its rows, and the digest a confirm must send back. */
export type PlannedWork = Omit<FilesWorkRenamePlan, "moving"> & { readonly moving?: number };

export const row = (
  fileId: string,
  from: string,
  to: string,
  part: Partial<FilesRenameRow> = {},
): FilesRenameRow => ({ fileId, from, to, group: 0, companions: [], ...part });

const summaryOf = (planned: readonly PlannedWork[]): FilesRenamePlan => {
  const works = planned.map(({ workId, contentType, title, digest, rows, moving = 0 }) => ({
    workId,
    contentType,
    title,
    digest,
    rename: rows.filter((each) => each.blocked === undefined).length,
    blocked: rows.filter((each) => each.blocked !== undefined).length,
    moving,
  }));
  return {
    works,
    files: works.reduce((sum, work) => sum + work.rename, 0),
    blocked: works.reduce((sum, work) => sum + work.blocked, 0),
  };
};

/** POST /renames: queues each confirmed Work's rows but those left, and answers the changed Works it was sent. */
const confirmHandler = (
  plan: readonly PlannedWork[],
  changed: readonly string[],
  confirmed: Confirmed[],
  onConfirm: (() => void) | undefined,
) =>
  http.post("*/api/v1/renames", async ({ request }) => {
    onConfirm?.();
    const { works } = RenameFilesBody.parse(await request.json());
    confirmed.push(works);
    const queued = works
      .filter(({ work }) => !changed.includes(work))
      .flatMap(({ work, leave }) =>
        (plan.find(({ workId }) => workId === work)?.rows ?? []).filter(
          (each) => each.blocked === undefined && !(leave ?? []).includes(each.fileId),
        ),
      ).length;
    return HttpResponse.json({
      queued,
      changed: changed.filter((id) => works.some(({ work }) => work === id)),
    });
  });

type Unmatched = readonly { contentType: string; files: number }[];

/**
 * The /renames endpoints over one plan that answers every scope and draft
 * alike, keeping each plan's body and each confirm's Works. A confirm
 * answers the Works `changed` names as changed, and calls `onConfirm`
 * first. `counts` changes the counts and sends files.moved, as the disk
 * queue does.
 */
export function renamesServer({
  plan,
  unmatched = [],
  moving = 0,
  changed = [],
  onConfirm,
}: {
  plan: readonly PlannedWork[];
  unmatched?: Unmatched;
  moving?: number;
  changed?: readonly string[];
  onConfirm?: () => void;
}) {
  const planned: PlanInputBody[] = [];
  const confirmed: Confirmed[] = [];
  const counts = { unmatched, moving };
  const clients: Parameters<Parameters<typeof sse<LiveMap>>[1]>[0]["client"][] = [];
  const handlers = [
    http.post("*/api/v1/renames/plan", async ({ request }) => {
      planned.push(PlanRenamesBody.parse(await request.json()));
      return HttpResponse.json(summaryOf(plan));
    }),
    http.post<{ workId: string }>("*/api/v1/renames/plan/:workId", ({ params }) => {
      const work = plan.find(({ workId }) => workId === params.workId);
      if (work === undefined) {
        throw new Error(`no plan for ${params.workId}`);
      }
      return HttpResponse.json({ moving: 0, ...work });
    }),
    confirmHandler(plan, changed, confirmed, onConfirm),
    http.get("*/api/v1/renames/unmatched", () =>
      HttpResponse.json({ contentTypes: counts.unmatched }),
    ),
    http.get("*/api/v1/renames/moving", () => HttpResponse.json({ files: counts.moving })),
    sse<LiveMap>("*/api/v1/events", ({ client }) => {
      clients.push(client);
      client.send({ event: "live.resync", data: {} });
    }),
  ];
  return {
    handlers,
    planned,
    confirmed,
    counts: (next: { unmatched: Unmatched; moving: number }, workIds: string[]) => {
      Object.assign(counts, next);
      clients[0]?.send({ event: "files.moved", data: { workIds } });
    },
  };
}
