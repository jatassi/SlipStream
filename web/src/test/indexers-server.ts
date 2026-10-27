import { HttpResponse, type JsonBodyType, http } from "msw";
import type { z } from "zod";

import type {
  IndexersIndexer,
  IndexersLimitUse,
  IndexersTracker,
  NodesNode,
  SettingDurationField,
} from "../api/gen/model";
import { UpdateIndexerBody, UpdateIndexerSettingsBody } from "../api/gen/zod/indexers/indexers.zod";
import { problem } from "./handlers";
import {
  larkspur,
  longBackoff,
  prowlarr,
  queries,
  searchingSeedbox,
  torrentLeech,
  trackers,
} from "./indexers";
import { home, nodesAre } from "./nodes";

const testFailed = (detail: string) =>
  HttpResponse.json(
    { status: 422, code: "indexer_test_failed", title: "The Indexer's Test failed", detail },
    { status: 422, headers: { "Content-Type": "application/problem+json" } },
  );

export type IndexerWrite = { method: string; id: string; body?: JsonBodyType };

const missing = () => problem(404, "indexer_not_found", "No Indexer has that id");

type Server = {
  state: { indexers: IndexersIndexer[]; settings: SettingDurationField[] };
  writes: IndexerWrite[];
  failTest: string | undefined;
};

const find = ({ state }: Server, id: unknown) => state.indexers.find((row) => row.id === id);

type Edit = z.output<typeof UpdateIndexerBody>;

const testsOn = (body: Edit) =>
  body.apiKey !== undefined || body.address !== undefined || body.homeNodeId !== undefined;

function readHandlers(server: Server, trackerList: IndexersTracker[]) {
  return [
    http.get("*/api/v1/indexers", () => HttpResponse.json({ indexers: server.state.indexers })),
    http.get("*/api/v1/trackers", () => HttpResponse.json({ trackers: trackerList })),
    http.get("*/api/v1/indexer-settings", () =>
      HttpResponse.json({ settings: server.state.settings }),
    ),
    http.get("*/api/v1/indexers/:id/queries", ({ params }) =>
      find(server, params["id"]) === undefined
        ? missing()
        : HttpResponse.json({ queries, nextOffset: null }),
    ),
    http.get("*/api/v1/indexers/:id", ({ params }) => {
      const row = find(server, params["id"]);
      return row === undefined ? missing() : HttpResponse.json(row);
    }),
  ];
}

const limitOf = (was: IndexersLimitUse, sent: Edit["searches"]): IndexersLimitUse =>
  sent === undefined ? was : { ...was, count: sent?.count ?? 0, per: sent?.per ?? "hour" };

/** row after edit, as the server saves it: null resets a field to its default. */
const switched = (was: boolean, sent: boolean | null | undefined, unset: boolean) =>
  sent === undefined ? was : (sent ?? unset);

/** row after edit, as the server saves it: null resets a field to its default. */
function edited(row: IndexersIndexer, edit: Edit): IndexersIndexer {
  const on = switched(row.on, edit.on, true);
  return {
    ...row,
    name: edit.name ?? row.name,
    homeNodeId: edit.homeNodeId ?? row.homeNodeId,
    on,
    automaticSearch: switched(row.automaticSearch, edit.automaticSearch, true),
    rss: switched(row.rss, edit.rss, true),
    allowFailover: switched(row.allowFailover, edit.allowFailover, false),
    priority: edit.priority === undefined ? row.priority : (edit.priority ?? 25),
    searches: limitOf(row.searches, edit.searches),
    grabs: limitOf(row.grabs, edit.grabs),
    sentence: on ? row.sentence : `${row.name} is off.`,
    revision: row.revision + 1,
  };
}

function patchIndexer(server: Server, row: IndexersIndexer, body: Edit) {
  server.writes.push({ method: "PATCH", id: row.id, body });
  if (server.failTest !== undefined && testsOn(body)) {
    return testFailed(server.failTest);
  }
  const next = edited(row, body);
  server.state.indexers = server.state.indexers.map((other) =>
    other.id === row.id ? next : other,
  );
  return HttpResponse.json(next);
}

function writeHandlers(server: Server) {
  return [
    http.patch("*/api/v1/indexer-settings", async ({ request }) => {
      const body = UpdateIndexerSettingsBody.parse(await request.json());
      server.writes.push({ method: "PATCH", id: "settings", body });
      const value = Number(body.values["indexers.longBackoff"] ?? longBackoff.default);
      server.state.settings = [
        { ...longBackoff, value, running: value, isDefault: value === longBackoff.default },
      ];
      return HttpResponse.json({ settings: server.state.settings });
    }),
    http.patch("*/api/v1/indexers/:id", async ({ request, params }) => {
      const row = find(server, params["id"]);
      return row === undefined
        ? missing()
        : patchIndexer(server, row, UpdateIndexerBody.parse(await request.json()));
    }),
    http.delete("*/api/v1/indexers/:id", ({ params }) => {
      const row = find(server, params["id"]);
      if (row === undefined) {
        return missing();
      }
      server.writes.push({ method: "DELETE", id: row.id });
      server.state.indexers = server.state.indexers.filter((other) => other.id !== row.id);
      return new HttpResponse(null, { status: 204 });
    }),
    http.post("*/api/v1/indexers/:id/test", ({ params }) => {
      const row = find(server, params["id"]);
      if (row === undefined) {
        return missing();
      }
      server.writes.push({ method: "TEST", id: row.id });
      return server.failTest === undefined
        ? HttpResponse.json({ searchModes: row.searchModes, items: 40, sources: [] })
        : testFailed(server.failTest);
    }),
  ];
}

/**
 * An Indexers server that keeps its list: a PATCH merges what it sends, a
 * DELETE drops the Indexer, and each is recorded in writes. A Test, and a
 * PATCH that tests, passes unless failTest gives indexer_test_failed's
 * sentence.
 */
export function indexersServer({
  indexers = [torrentLeech, prowlarr, larkspur],
  trackerList = trackers,
  nodes = [home, searchingSeedbox],
  failTest,
}: {
  indexers?: IndexersIndexer[];
  trackerList?: IndexersTracker[];
  nodes?: NodesNode[];
  failTest?: string;
} = {}) {
  const server: Server = {
    state: { indexers: [...indexers], settings: [longBackoff] },
    writes: [],
    failTest,
  };
  const handlers = [
    nodesAre(...nodes),
    ...readHandlers(server, trackerList),
    ...writeHandlers(server),
  ];
  return { handlers, state: server.state, writes: server.writes };
}
