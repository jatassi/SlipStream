import { HttpResponse, http } from "msw";

import type {
  HistoryKindsAnswer,
  HistoryListEntry,
  HistoryListed,
  HistoryOneEntry,
  HistorySentence,
} from "../api/gen/model";
import { orchard, orchardId } from "./library";

/** The catalog's groups and kinds, as GET /history/kinds answers. */
export const historyKinds: HistoryKindsAnswer = {
  groups: [
    {
      group: "grabs",
      label: "Grabs",
      kinds: [
        { kind: "grabbed", label: "Grabbed" },
        { kind: "grab_failed", label: "Grab failed" },
      ],
    },
    { group: "imports", label: "Imports", kinds: [{ kind: "imported", label: "Imported" }] },
    { group: "library", label: "Library", kinds: [{ kind: "works_added", label: "Works added" }] },
    {
      group: "health",
      label: "Health",
      kinds: [{ kind: "health_new", label: "New Health item" }],
    },
  ],
};

/** An entry about the Orchard, its words "Grabbed The Quiet Orchard {id}", at at. */
export const anEntry = (id: number, at: string, part: Partial<HistoryListEntry> = {}) =>
  ({
    id,
    kind: "grabbed",
    group: "grabs",
    at,
    words: `Grabbed ${orchard.label} ${id}`,
    tone: "info",
    work: { id: orchardId, label: orchard.label, contentType: "films", posterUrl: null },
    links: { grab: null, import: null, health: null },
    ...part,
  }) satisfies HistoryListEntry;

/** count entries a minute apart, newest first, the newest at newest with id top. */
export function entriesFrom(top: number, count: number, newest: Date): HistoryListEntry[] {
  return Array.from({ length: count }, (_, n) =>
    anEntry(top - n, new Date(newest.getTime() - n * 60_000).toISOString()),
  );
}

/** pages of entries by before, size a page, as GET /history answers them. */
export function paged(entries: readonly HistoryListEntry[], before: number | null, size: number) {
  const below = entries.filter(({ id }) => before === null || id < before);
  const page = below.slice(0, size);
  const answer: HistoryListed = {
    entries: page,
    next: below.length > size ? (page.at(-1)?.id ?? null) : null,
  };
  return answer;
}

const gone = () =>
  HttpResponse.json(
    {
      status: 404,
      code: "history_entry_gone",
      title: "This History entry was deleted",
      detail: "It was cleared, or it was older than Keep history for.",
    },
    { status: 404 },
  );

/** GET /history's answer over entries for a read's query, narrowed to its Work and groups. */
function listed(entries: readonly HistoryListEntry[], params: URLSearchParams) {
  const work = params.get("work");
  const groups = params.getAll("groups").flatMap((each) => each.split(","));
  const shown = entries.filter(
    (entry) =>
      (work === null || entry.work?.id === work) &&
      (groups.length === 0 || groups.includes(entry.group)),
  );
  const before = params.get("before");
  return paged(shown, before === null ? null : Number(before), Number(params.get("limit") ?? 50));
}

/** GET /history over entries, recording each read's query, with the summary, kinds and one entry's long form. */
export function historyServer(
  initial: readonly HistoryListEntry[],
  {
    sentence = { words: "There were 3 Grabs in the last 7 days.", needsYou: 0 },
    one = (entry: HistoryListEntry): HistoryOneEntry => ({
      entry,
      lines: ["It replaced WEBDL-1080p."],
      files: [],
      stages: [],
      health: [],
    }),
  }: { sentence?: HistorySentence | null; one?: (entry: HistoryListEntry) => HistoryOneEntry } = {},
) {
  let entries = [...initial];
  const asked: URLSearchParams[] = [];
  const summaries: URLSearchParams[] = [];
  let cleared = 0;
  return {
    asked,
    summaries,
    get cleared() {
      return cleared;
    },
    record(entry: HistoryListEntry) {
      entries = [entry, ...entries];
    },
    handlers: [
      http.get("*/api/v1/history/summary", ({ request }) => {
        summaries.push(new URL(request.url).searchParams);
        return HttpResponse.json(sentence);
      }),
      http.get("*/api/v1/history/kinds", () => HttpResponse.json(historyKinds)),
      http.get("*/api/v1/history/:id", ({ params }) => {
        const entry = entries.find(({ id }) => String(id) === params["id"]);
        return entry === undefined ? gone() : HttpResponse.json(one(entry));
      }),
      http.get("*/api/v1/history", ({ request }) => {
        const params = new URL(request.url).searchParams;
        asked.push(params);
        return HttpResponse.json(listed(entries, params));
      }),
      http.delete("*/api/v1/history", () => {
        cleared++;
        entries = [];
        return new HttpResponse(null, { status: 202 });
      }),
    ],
  };
}
