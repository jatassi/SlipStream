import { HttpResponse, type JsonBodyType, http } from "msw";

import type { IndexersIndexer, IndexersTracker, SettingDurationField } from "../api/gen/model";
import { TestIndexerBody } from "../api/gen/zod/indexers/indexers.zod";
import { captchas } from "./catalog";
import { problem } from "./handlers";

export type IndexerWrite = { method: string; id: string; body?: JsonBodyType };

export type Server = {
  state: {
    indexers: IndexersIndexer[];
    trackers: IndexersTracker[];
    settings: SettingDurationField[];
  };
  writes: IndexerWrite[];
  failTest: string | undefined;
  /** Sentences of the Tests that fail at a sign-in showing a captcha, in turn. */
  captchaTests: string[];
  /** The first answered captcha finds its attempt expired. */
  expireFirst: boolean;
  captchasHanded: number;
  /** The sentence of a Test whose sign-in needs a two-factor code, until one is sent. */
  codeNeeded: string | undefined;
};

export const missing = () => problem(404, "indexer_not_found", "No Indexer has that id");

export const find = ({ state }: Server, id: unknown) => state.indexers.find((row) => row.id === id);

export const testFailed = (detail: string, more: object = {}) =>
  HttpResponse.json(
    {
      status: 422,
      code: "indexer_test_failed",
      title: "The Indexer's Test failed",
      detail,
      ...more,
    },
    { status: 422, headers: { "Content-Type": "application/problem+json" } },
  );

/** A Test that met a sign-in needing a two-factor code. */
export const codeNeededFailure = (detail: string) => testFailed(detail, { cause: "code_needed" });

const captchaExpired = () =>
  problem(410, "captcha_expired", "The captcha expired. Load a new one and type its code.");

function handOut(server: Server) {
  const captcha = captchas[server.captchasHanded % captchas.length];
  server.captchasHanded += 1;
  return captcha;
}

/**
 * A Test as the server runs it: expired, failed with a new captcha, failed
 * for want of a two-factor code, failed, or passed.
 */
export function tested(server: Server, answered: boolean, coded = false) {
  if (answered && server.expireFirst) {
    server.expireFirst = false;
    return captchaExpired();
  }
  const sentence = server.captchaTests.shift();
  if (sentence !== undefined) {
    return testFailed(sentence, { captcha: handOut(server) });
  }
  if (server.codeNeeded !== undefined && !coded) {
    return codeNeededFailure(server.codeNeeded);
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
        tested(server, body.captcha !== undefined, body.code !== undefined) ??
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
