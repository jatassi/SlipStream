import { HttpResponse, http } from "msw";

import type {
  ImportsImportEntry,
  ImportsLead,
  ImportsOutcome,
  WorksCandidate,
  WorksLibraryRow,
  WorksWorkView,
} from "../api/gen/model";
import {
  ActOnRootImportBody,
  AddRootImportsBody,
  IgnoreImportsBody,
} from "../api/gen/zod/imports/imports.zod";
import type { RootEntry } from "../imports/root-entry";
import { filesOf, refused } from "./import-actions";
import { brazilRow, brazilWork, severance, severanceRow, vaultRow } from "./import-entries";
import type { liveServer } from "./live-server";
import {
  brazil,
  filmsLead,
  halloween,
  halloween1978,
  halloween2018,
  heat,
  orchardLeft,
  severanceEntry,
  showsLead,
} from "./pending-imports";
import { worker } from "./worker";

export type ImportsWorld = {
  entries: RootEntry[];
  /** The download and folder Imports, beside the root entries. */
  imports: ImportsImportEntry[];
  /** What importFolder opens. */
  folder: ImportsImportEntry[];
  leads: ImportsLead[];
  candidates: WorksCandidate[];
  rows: WorksLibraryRow[];
  works: WorksWorkView[];
  /** Publishes imports.changed, as the Controller does after each change. */
  changed: (rootIds: string[]) => void;
};

export type Sent = {
  added: string[][];
  matched: { id: string; key: string }[];
  imported: { id: string; workId: string; files: { position: number; itemIds: string[] }[] }[];
  ignored: string[][];
  restored: string[][];
  searched: string[];
  acted: { id: string; position: number; action: string }[];
  confirmed: string[][];
  moved: { importIds: string[]; on: boolean }[];
  opened: { nodeId: string; folder: string }[];
};

const needsYou = (entry: RootEntry) => entry.ignoredAt === null;

const failed = (id: string, code: string, title: string): ImportsOutcome => ({
  id,
  outcome: "failed",
  problem: { code, title },
});

function outcomeOf(world: ImportsWorld, entry: RootEntry): ImportsOutcome {
  const { id, suggestion } = entry;
  if (suggestion === null) {
    return failed(id, "import_no_suggestion", "The Import has no suggested Work");
  }
  if (entry.files.some(({ reason }) => reason.code === "elsewhere")) {
    return failed(id, "import_elsewhere", "The Work lives in another library root");
  }
  world.entries = world.entries.filter((candidate) => candidate.id !== id);
  return {
    id,
    outcome: "added",
    workId: suggestion.workId ?? `0199f0c3-9999-7000-8000-${id.slice(-12)}`,
  };
}

function matchedOf(entry: RootEntry, candidate: WorksCandidate): RootEntry {
  const year = candidate.fields.find((field) => field.key === "year");
  return {
    ...entry,
    suggestion: {
      key: candidate.key,
      label: candidate.label,
      year: year?.kind === "number" ? year.number : null,
      posterUrl: candidate.posterUrl,
      workId: candidate.workId,
      confident: false,
    },
  };
}

const readsOf = (world: ImportsWorld, sent: Sent) => [
  http.get("*/api/v1/imports", ({ request }) => {
    const query = new URL(request.url).searchParams;
    const ignored = query.get("ignored") === "true";
    const source = query.get("source");
    const roots = world.entries.filter((entry) => needsYou(entry) !== ignored);
    const others = ignored ? [] : world.imports;
    return HttpResponse.json({
      leads: source === null || source === "root" ? world.leads : [],
      imports: [...roots, ...others].filter((entry) => source === null || entry.source === source),
      next: null,
    });
  }),
  http.get("*/api/v1/imports/:id", ({ params }) => {
    const entry = [...world.entries, ...world.imports].find(({ id }) => id === params["id"]);
    return entry === undefined
      ? refused(404, "import_not_found", "No Import has that id")
      : HttpResponse.json(entry);
  }),
  http.get("*/api/v1/content-types/:id/catalog", ({ request }) => {
    sent.searched.push(new URL(request.url).searchParams.get("query") ?? "");
    return HttpResponse.json({ candidates: world.candidates });
  }),
  http.get("*/api/v1/library", () => HttpResponse.json({ rows: world.rows })),
  http.get("*/api/v1/works/:workId", ({ params }) => {
    const work = world.works.find(({ id }) => id === params["workId"]);
    return work === undefined
      ? refused(404, "work_not_found", "No Work has that id")
      : HttpResponse.json(work);
  }),
];

const marksOf = (world: ImportsWorld, sent: Sent) => [
  http.post("*/api/v1/imports/ignore", async ({ request }) => {
    const { importIds } = IgnoreImportsBody.parse(await request.json());
    sent.ignored.push(importIds);
    return marked(world, importIds, "2026-11-08T10:00:00Z");
  }),
  http.post("*/api/v1/imports/unignore", async ({ request }) => {
    const { importIds } = IgnoreImportsBody.parse(await request.json());
    sent.restored.push(importIds);
    return marked(world, importIds, null);
  }),
];

const writesOf = (world: ImportsWorld, sent: Sent) => [
  http.post("*/api/v1/imports/add", async ({ request }) => {
    const { importIds } = AddRootImportsBody.parse(await request.json());
    sent.added.push(importIds);
    const named = world.entries.filter(({ id }) => importIds.includes(id));
    const outcomes = named.map((entry) => outcomeOf(world, entry));
    world.changed(named.map(({ root }) => root.id));
    return HttpResponse.json({ outcomes });
  }),
  http.post("*/api/v1/imports/:id", async ({ params, request }) => {
    const id = String(params["id"]);
    const body = ActOnRootImportBody.parse(await request.json());
    const entry = world.entries.find((found) => found.id === id);
    if (body.action === "match") {
      sent.matched.push({ id, key: body.key });
      const candidate = world.candidates.find((found) => found.key === body.key);
      if (candidate === undefined || entry === undefined) {
        return refused(
          404,
          "metadata_not_found",
          "The metadata source has no record with that key",
        );
      }
      const matched = matchedOf(entry, candidate);
      world.entries = world.entries.map((found) => (found.id === id ? matched : found));
      world.changed([entry.root.id]);
      return HttpResponse.json(matched);
    }
    sent.imported.push({ id, workId: body.workId, files: body.files });
    if (entry === undefined) {
      return refused(404, "import_not_found", "No Import has that id");
    }
    const chosen = new Set(body.files.map(({ position }) => position));
    const files = entry.files.filter(({ position }) => !chosen.has(position));
    const left: RootEntry = { ...entry, stage: files.length === 0 ? "done" : "waiting", files };
    world.entries =
      files.length === 0
        ? world.entries.filter((found) => found.id !== id)
        : world.entries.map((found) => (found.id === id ? left : found));
    world.changed([entry.root.id]);
    return HttpResponse.json(left);
  }),
];

function marked(world: ImportsWorld, ids: readonly string[], ignoredAt: string | null) {
  const roots = world.entries.filter(({ id }) => ids.includes(id)).map(({ root }) => root.id);
  world.entries = world.entries.map((entry) =>
    ids.includes(entry.id) ? { ...entry, ignoredAt } : entry,
  );
  world.changed(roots);
  return new HttpResponse(null, { status: 204 });
}

/** Serves Pending imports from one mutable world, recording what the page sent. */
export function serveImports(start: Partial<ImportsWorld> = {}) {
  const world: ImportsWorld = {
    entries: [halloween, brazil, heat, severanceEntry, orchardLeft],
    imports: [],
    folder: [],
    leads: [filmsLead, showsLead],
    candidates: [halloween1978, halloween2018],
    rows: [brazilRow, vaultRow, severanceRow],
    works: [brazilWork, severance],
    changed: () => {},
    ...start,
  };
  const sent: Sent = {
    added: [],
    matched: [],
    imported: [],
    ignored: [],
    restored: [],
    searched: [],
    acted: [],
    confirmed: [],
    moved: [],
    opened: [],
  };
  worker.use(
    ...readsOf(world, sent),
    ...marksOf(world, sent),
    ...filesOf(world, sent),
    ...writesOf(world, sent),
  );
  return { world, sent };
}

/** Sends imports.changed to the page over a test's live stream. */
export const changedOver =
  (live: ReturnType<typeof liveServer>) =>
  (rootIds: string[]): void => {
    live.clients[0]?.send({ event: "imports.changed", data: { rootIds } });
  };
