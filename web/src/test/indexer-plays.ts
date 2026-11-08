import { HttpResponse, type JsonBodyType, http } from "msw";

import type {
  IndexersCaptchaImage,
  IndexersIndexer,
  TrackersTracker,
  SettingDurationField,
} from "../api/gen/model";
import { TestIndexerBody } from "../api/gen/zod/indexers/indexers.zod";
import { MergeTrackerBody, RenameTrackerBody } from "../api/gen/zod/trackers/trackers.zod";
import { captchas } from "./catalog";
import { problem } from "./handlers";

export type IndexerWrite = { method: string; id: string; body?: JsonBodyType };

export type Server = {
  state: {
    indexers: IndexersIndexer[];
    trackers: TrackersTracker[];
    settings: SettingDurationField[];
  };
  writes: IndexerWrite[];
  failTest: string | undefined;
  /** Sentences of the Tests that fail at a sign-in showing a captcha, in turn. */
  captchaTests: string[];
  /** The first answered captcha finds its attempt expired. */
  expireFirst: boolean;
  captchasHanded: number;
};

export const missing = () => problem(404, "indexer_not_found", "No Indexer has that id");

export const find = ({ state }: Server, id: unknown) => state.indexers.find((row) => row.id === id);

const testFailed = (detail: string, captcha?: IndexersCaptchaImage) =>
  HttpResponse.json(
    {
      status: 422,
      code: "indexer_test_failed",
      title: "The Indexer's Test failed",
      detail,
      ...(captcha !== undefined && { captcha }),
    },
    { status: 422, headers: { "Content-Type": "application/problem+json" } },
  );

const captchaExpired = () =>
  problem(410, "captcha_expired", "The captcha expired. Load a new one and type its code.");

function handOut(server: Server) {
  const captcha = captchas[server.captchasHanded % captchas.length];
  server.captchasHanded += 1;
  return captcha;
}

/** A Test as the server runs it: expired, failed with a new captcha, failed, or passed. */
export function tested(server: Server, answered: boolean) {
  if (answered && server.expireFirst) {
    server.expireFirst = false;
    return captchaExpired();
  }
  const sentence = server.captchaTests.shift();
  if (sentence !== undefined) {
    return testFailed(sentence, handOut(server));
  }
  return server.failTest === undefined ? undefined : testFailed(server.failTest);
}

/** A saved Indexer's Test, and the new captcha its home fetches on request. */
export function testHandlers(server: Server) {
  return [
    http.post("*/api/v1/indexers/:id/test", async ({ request, params }) => {
      const row = find(server, params["id"]);
      if (row === undefined) {
        return missing();
      }
      const text = await request.text();
      const body = TestIndexerBody.parse(text === "" ? {} : JSON.parse(text));
      server.writes.push({ method: "TEST", id: row.id, body });
      return (
        tested(server, body.captcha !== undefined) ??
        HttpResponse.json({ searchModes: row.searchModes, items: 40, sources: [] })
      );
    }),
    http.post("*/api/v1/indexers/:id/captcha", ({ params }) => {
      const row = find(server, params["id"]);
      if (row === undefined) {
        return missing();
      }
      server.writes.push({ method: "CAPTCHA", id: row.id });
      return HttpResponse.json(handOut(server));
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

function merge(server: Server, from: TrackersTracker, into: TrackersTracker) {
  const kept: TrackersTracker = {
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

export function trackerHandlers(server: Server) {
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
