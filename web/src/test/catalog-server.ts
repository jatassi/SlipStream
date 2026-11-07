import { HttpResponse, http } from "msw";

import type { IndexersIndexerCatalog, TasksTask } from "../api/gen/model";
import { SetCatalogAddressBody } from "../api/gen/zod/indexers/indexers.zod";
import { catalog } from "./catalog";
import { problem } from "./handlers";
import { housekeeping, taskServer } from "./tasks";

const hours = (count: number) => count * 3600;

const cadenceTask = (id: string, name: string, every: number, words: string): TasksTask => ({
  ...housekeeping,
  id,
  name,
  cadence: {
    words,
    inWindow: false,
    field: {
      ...housekeeping.cadence.field,
      key: `tasks.${id}.cadence`,
      label: "Runs every",
      value: every,
      running: every,
      default: every,
      min: hours(1),
      max: hours(168),
      step: hours(1),
    },
  },
});

export const catalogRefresh = cadenceTask(
  "indexer-catalog-refresh",
  "Indexer catalog refresh",
  hours(24),
  "Every day",
);
export const healthCheck = cadenceTask(
  "indexer-health-check",
  "Indexer health check",
  hours(6),
  "Every 6 hours",
);

const unreachable = () =>
  HttpResponse.json(
    {
      status: 422,
      code: "catalog_address_unreachable",
      title: "SlipStream can't read an Indexer catalog at that address",
      detail: "Check the address and that it serves schema 12.",
    },
    { status: 422, headers: { "Content-Type": "application/problem+json" } },
  );

/** shown without custom definition id, the catalog's namesake offered again. */
const without = (shown: IndexersIndexerCatalog, id: string): IndexersIndexerCatalog => ({
  ...shown,
  entries: shown.entries
    .filter((entry) => entry.origin === "catalog" || entry.id !== id)
    .map((entry) => Object.assign({}, entry, entry.id === id ? { replacedByCustom: false } : {})),
});

export type CatalogWrite = { method: string; path: string; body?: unknown };

/** An address the fake reads a catalog at; any other is refused as unreachable. */
export const readableAddress = "https://catalog.example.test/indexers";

/**
 * The Indexer catalog with its address and custom definitions, and the two
 * tasks Settings > Indexers shows. Deleting `inUse` is refused as an
 * Indexer runs it; each write lands in writes.
 */
export function catalogServer({ inUse }: { inUse?: string } = {}) {
  const state: { catalog: IndexersIndexerCatalog } = { catalog };
  const writes: CatalogWrite[] = [];
  const tasks = taskServer({ tasks: [catalogRefresh, healthCheck] });
  const handlers = [
    http.get("*/api/v1/indexer-catalog", () => HttpResponse.json(state.catalog)),
    http.put("*/api/v1/indexer-catalog/address", async ({ request }) => {
      const body = SetCatalogAddressBody.parse(await request.json());
      writes.push({ method: "PUT", path: "address", body });
      if (body.address !== null && body.address !== readableAddress) {
        return unreachable();
      }
      state.catalog = {
        ...state.catalog,
        address: body.address ?? catalog.address,
        addressIsDefault: body.address === null,
      };
      return HttpResponse.json(state.catalog);
    }),
    http.delete("*/api/v1/indexer-catalog/custom/:id", ({ params }) => {
      const id = String(params["id"]);
      writes.push({ method: "DELETE", path: `custom/${id}` });
      if (id === inUse) {
        return problem(
          409,
          "definition_in_use",
          "An Indexer runs this definition. Delete the Indexer first.",
        );
      }
      state.catalog = without(state.catalog, id);
      return new HttpResponse(null, { status: 204 });
    }),
    ...tasks.handlers,
  ];
  return { handlers, writes, taskWrites: tasks.writes };
}
