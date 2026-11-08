import { HttpResponse, http } from "msw";
import type { z } from "zod";

import type {
  IndexersIndexer,
  IndexersLimitUse,
  IndexersTracker,
  NodesNode,
} from "../api/gen/model";
import {
  MergeTrackerBody,
  RenameTrackerBody,
  UpdateIndexerBody,
  UpdateIndexerSettingsBody,
} from "../api/gen/zod/indexers/indexers.zod";
import { catalogServer } from "./catalog-server";
import { imported } from "./grabs";
import { problem } from "./handlers";
import { type Server, find, missing, testHandlers, tested } from "./indexer-plays";
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

type Edit = z.output<typeof UpdateIndexerBody>;

const testsOn = (body: Edit) =>
  body.apiKey !== undefined ||
  body.address !== undefined ||
  body.homeNodeId !== undefined ||
  body.cookie !== undefined ||
  body.baseAddress !== undefined ||
  body.values !== undefined;

function readHandlers(server: Server) {
  return [
    http.get("*/api/v1/indexers", () => HttpResponse.json({ indexers: server.state.indexers })),
    http.get("*/api/v1/trackers", () => HttpResponse.json({ trackers: server.state.trackers })),
    http.get("*/api/v1/indexer-settings", () =>
      HttpResponse.json({ settings: server.state.settings }),
    ),
    http.get("*/api/v1/indexers/:id/queries", ({ params }) =>
      find(server, params["id"]) === undefined
        ? missing()
        : HttpResponse.json({ queries, nextOffset: null }),
    ),
    http.get("*/api/v1/indexers/:id/grabs", ({ params }) =>
      find(server, params["id"]) === undefined
        ? missing()
        : HttpResponse.json({ grabs: [imported], nextOffset: null }),
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
  const refused = testsOn(body)
    ? tested(server, body.captcha !== undefined, body.code !== undefined)
    : undefined;
  if (refused !== undefined) {
    return refused;
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
  ];
}

const noTracker = () =>
  problem(
    404,
    "tracker_not_found",
    "That Tracker no longer exists. It may have been merged into another Tracker.",
  );

const folded = (name: string) => name.toLowerCase().replaceAll(/[^\p{L}\p{N}]/gu, "");

const findTracker = ({ state }: Server, id: unknown) =>
  state.trackers.find((tracker) => tracker.id === id);

function merge(server: Server, from: IndexersTracker, into: IndexersTracker) {
  const kept: IndexersTracker = {
    ...into,
    keys: [...into.keys, ...from.keys],
    hosts: [...new Set([...into.hosts, ...from.hosts])],
    indexerIds: [...new Set([...into.indexerIds, ...from.indexerIds])],
  };
  server.state.trackers = server.state.trackers
    .filter((tracker) => tracker.id !== from.id)
    .map((tracker) => (tracker.id === into.id ? kept : tracker));
  server.state.indexers = server.state.indexers.map((indexer) => ({
    ...indexer,
    trackerIds: [...new Set(indexer.trackerIds.map((id) => (id === from.id ? into.id : id)))],
  }));
  return kept;
}

function trackerHandlers(server: Server) {
  return [
    http.patch("*/api/v1/trackers/:id", async ({ request, params }) => {
      const tracker = findTracker(server, params["id"]);
      if (tracker === undefined) {
        return noTracker();
      }
      const body = RenameTrackerBody.parse(await request.json());
      server.writes.push({ method: "PATCH", id: tracker.id, body });
      const taken = server.state.trackers.some(
        (other) => other.id !== tracker.id && folded(other.name) === folded(body.name),
      );
      if (taken) {
        return problem(409, "tracker_name_taken", "Another Tracker has that name");
      }
      const renamed = { ...tracker, name: body.name };
      server.state.trackers = server.state.trackers.map((other) =>
        other.id === tracker.id ? renamed : other,
      );
      return HttpResponse.json(renamed);
    }),
    http.post("*/api/v1/trackers/:id/merge", async ({ request, params }) => {
      const from = findTracker(server, params["id"]);
      const body = MergeTrackerBody.parse(await request.json());
      const into = findTracker(server, body.into);
      if (from === undefined || into === undefined) {
        return noTracker();
      }
      server.writes.push({ method: "MERGE", id: from.id, body });
      if (from.id === into.id) {
        return problem(422, "tracker_merge_self", "A Tracker can't be merged into itself");
      }
      return HttpResponse.json(merge(server, from, into));
    }),
  ];
}

/**
 * An Indexers server that keeps its list: a PATCH merges what it sends, a
 * DELETE drops the Indexer, and each is recorded in writes. A Test, and a
 * PATCH that tests, passes unless captchaTests, codeNeeded or failTest give
 * indexer_test_failed's sentence; each of captchaTests hands back a
 * captcha, and codeNeeded holds until a code is sent. A rename refuses a
 * name another Tracker has, and a merge moves the Tracker's keys, hosts
 * and Indexers to the one kept.
 */
export function indexersServer({
  indexers = [torrentLeech, prowlarr, larkspur],
  trackerList = trackers,
  nodes = [home, searchingSeedbox],
  failTest,
  captchaTests = [],
  expireFirst = false,
  codeNeeded,
}: {
  indexers?: IndexersIndexer[];
  trackerList?: IndexersTracker[];
  nodes?: NodesNode[];
  failTest?: string;
  captchaTests?: string[];
  expireFirst?: boolean;
  codeNeeded?: string;
} = {}) {
  const server: Server = {
    state: {
      indexers: [...indexers],
      trackers: [...trackerList],
      settings: [longBackoff],
    },
    writes: [],
    failTest,
    captchaTests: [...captchaTests],
    expireFirst,
    captchasHanded: 0,
    codeNeeded,
  };
  const handlers = [
    nodesAre(...nodes),
    ...readHandlers(server),
    ...writeHandlers(server),
    ...testHandlers(server),
    ...trackerHandlers(server),
    ...catalogServer().handlers,
  ];
  return { handlers, state: server.state, writes: server.writes };
}
