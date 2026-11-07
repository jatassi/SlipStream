import { HttpResponse, http } from "msw";

import type {
  IndexersCaptchaImage,
  IndexersDefinitionForm,
  IndexersIndexerCatalog,
  IndexersReport,
  WorksContentType,
} from "../api/gen/model";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { captchas, catalog as fullCatalog, entry, forms } from "./catalog";
import { indexer } from "./indexers";
import { indexersServer } from "./indexers-server";
import { cinema } from "./library";

export const addedId = "5e4d3c2b-1a0f-4e9d-8c7b-6a5f4e3d2c1b";

const problemOf = (status: number, code: string, title: string, more: object = {}) =>
  HttpResponse.json(
    { status, code, title, ...more },
    { status, headers: { "Content-Type": "application/problem+json" } },
  );

const testFailed = (detail: string, captcha?: IndexersCaptchaImage) =>
  problemOf(422, "indexer_test_failed", "The Indexer's Test failed", {
    detail,
    ...(captcha !== undefined && { captcha }),
  });

const passed: IndexersReport = {
  items: 50,
  searchModes: [{ mode: "search", params: ["q"] }],
  sources: [],
};

type Options = {
  catalog?: IndexersIndexerCatalog;
  definitions?: readonly IndexersDefinitionForm[];
  report?: IndexersReport;
  /** The sentence a draft's Test fails with. */
  failTest?: string;
  /** The Test sentences the saves fail with, one a save, before one passes. */
  failSaves?: string[];
  /** The first save with a captcha finds it expired. */
  expireFirst?: boolean;
  contentTypes?: WorksContentType[];
};

type Server = {
  writes: { method: string; path: string; body: Record<string, unknown> }[];
  failSaves: string[];
  expired: boolean;
  captchasHanded: number;
  report: IndexersReport;
  failTest: string | undefined;
};

const handOut = (server: Server) => {
  const captcha = captchas[server.captchasHanded % captchas.length];
  server.captchasHanded += 1;
  return captcha;
};

const bodyOf = async (request: Request): Promise<Record<string, unknown>> => {
  const body: unknown = await request.json();
  return typeof body === "object" && body !== null ? Object.fromEntries(Object.entries(body)) : {};
};

function draftHandlers(server: Server) {
  return [
    http.post("*/api/v1/indexers/captcha", async ({ request }) => {
      server.writes.push({
        method: "CAPTCHA",
        path: "/indexers/captcha",
        body: await bodyOf(request),
      });
      return HttpResponse.json(handOut(server));
    }),
    http.post("*/api/v1/indexers/test", async ({ request }) => {
      server.writes.push({ method: "TEST", path: "/indexers/test", body: await bodyOf(request) });
      return server.failTest === undefined
        ? HttpResponse.json(server.report)
        : testFailed(server.failTest);
    }),
  ];
}

function createHandler(server: Server, added: ReturnType<typeof indexersServer>["state"]) {
  return http.post("*/api/v1/indexers", async ({ request }) => {
    const body = await bodyOf(request);
    server.writes.push({ method: "POST", path: "/indexers", body });
    if (server.expired && "captcha" in body) {
      server.expired = false;
      return problemOf(
        410,
        "captcha_expired",
        "The captcha expired. Load a new one and type its code.",
      );
    }
    const failed = server.failSaves.shift();
    if (failed !== undefined) {
      return testFailed(failed, "captcha" in body ? handOut(server) : undefined);
    }
    const name = typeof body["name"] === "string" ? body["name"] : "";
    const made = indexer({ id: addedId, name, kind: "catalog" });
    added.indexers.push(made);
    return HttpResponse.json(made, { status: 201 });
  });
}

const takenId = /^id:\s*marsh$/mu;
const unrunnable = /^\s*method:\s*oauth$/mu;

function customHandler(server: Server) {
  return http.post("*/api/v1/indexer-catalog/custom", async ({ request }) => {
    const body = await bodyOf(request);
    const yaml = typeof body["yaml"] === "string" ? body["yaml"] : "";
    server.writes.push({ method: "POST", path: "/indexer-catalog/custom", body: { yaml } });
    if (unrunnable.test(yaml)) {
      return problemOf(422, "definition_unloadable", "SlipStream can't run this definition yet", {
        detail: "The definition signs in with the method 'oauth', which SlipStream doesn't run.",
      });
    }
    if (takenId.test(yaml)) {
      return problemOf(
        409,
        "definition_id_taken",
        "The Indexer catalog already has a definition with this id",
      );
    }
    const id = /^id:\s*(\S+)$/mu.exec(yaml)?.[1] ?? "custom";
    return HttpResponse.json(entry({ id, name: id, origin: "custom" }), { status: 201 });
  });
}

/**
 * The Add Indexer flow's server over indexersServer's list: the catalog
 * and its definition forms, captchas handed out in turn, a draft Test that
 * reports report or fails with failTest, and a save that fails with each of failSaves' sentences
 * first, handing back a new captcha when the save carried one. A saved
 * Indexer joins the list, so its page opens.
 */
export function addIndexerServer({
  catalog = fullCatalog,
  definitions = forms,
  report = passed,
  failTest,
  failSaves = [],
  expireFirst = false,
  contentTypes = [cinema],
}: Options = {}) {
  const indexers = indexersServer();
  const server: Server = {
    writes: [],
    failSaves: [...failSaves],
    expired: expireFirst,
    captchasHanded: 0,
    report,
    failTest,
  };
  const handlers = [
    http.get("*/api/v1/indexer-catalog", () => HttpResponse.json(catalog)),
    http.get("*/api/v1/indexer-catalog/:id", ({ params }) => {
      const found = definitions.find((form) => form.entry.id === params["id"]);
      return found === undefined
        ? problemOf(404, "definition_not_found", "No definition has that id")
        : HttpResponse.json(found);
    }),
    ...draftHandlers(server),
    createHandler(server, indexers.state),
    customHandler(server),
    getListContentTypesMockHandler({ contentTypes }),
    ...indexers.handlers,
  ];
  return { handlers, writes: server.writes };
}
