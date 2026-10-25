import type {
  ActInputBodyAction,
  DownloadclientsActResult,
  DownloadclientsRefusal,
  DownloadclientsRefusalCode,
} from "../api/gen/model";
import { type Clock, sinceWords } from "../format";
import type { Row, State } from "./model";

export type TorrentAction = ActInputBodyAction;

/** A Torrent as an action names it: the Work it is for, else its release. */
export type Subject = { readonly id: string; readonly name: string };

export type Wanted = "started" | "paused";

/** What the Admin last asked of it: a pending ask, else what its Node holds it for. */
export function wantedOf(row: Row): Wanted {
  switch (row.pending) {
    case "pause":
      return "paused";
    case "resume":
      return "started";
    case undefined:
      return row.hold === "wanted" ? "paused" : "started";
    default:
      return row.pending satisfies never;
  }
}

const announcing = new Set<State>(["downloading", "seeding", "stalled", "tracker-error"]);

/**
 * The actions a Torrent offers, in menu order. Announce now needs the
 * Inspector's word that every tracker accepts one now; a row has none.
 */
export function menuFor(row: Row, trackersAccept = false): TorrentAction[] {
  if (row.state === "removing") {
    return [];
  }
  if (row.state === "failed") {
    return ["remove"];
  }
  const online = row.lastKnown === undefined;
  const announces =
    online && row.pending === undefined && announcing.has(row.state) && trackersAccept;
  return [
    wantedOf(row) === "started" ? "pause" : "resume",
    ...(announces ? (["announce"] as const) : []),
    ...(online && row.state !== "checking" ? (["recheck"] as const) : []),
    ...(row.state === "stalled" ? (["mark-failed"] as const) : []),
    "remove",
  ];
}

export const actionLabels: { readonly [A in TorrentAction]: string } = {
  pause: "Pause",
  resume: "Resume",
  announce: "Announce now",
  recheck: "Recheck",
  "mark-failed": "Mark failed",
  remove: "Remove",
};

const doing: { readonly [A in TorrentAction]: (subject: string) => string } = {
  pause: (subject) => `Pausing ${subject}.`,
  resume: (subject) => `Resuming ${subject}.`,
  announce: (subject) => `${subject} is announcing to its trackers.`,
  recheck: (subject) => `Rechecking ${subject}.`,
  "mark-failed": (subject) => `Marked ${subject} failed.`,
  remove: (subject) => `Removing ${subject}.`,
};

const verbs: { readonly [A in TorrentAction]: string } = {
  pause: "pause",
  resume: "resume",
  announce: "announce",
  recheck: "recheck",
  "mark-failed": "be marked failed",
  remove: "be removed",
};

const torrents = (count: number) => `${count} ${count === 1 ? "torrent" : "torrents"}`;

type Refused = { readonly names: readonly string[]; readonly refusal: DownloadclientsRefusal };

type Context = {
  readonly action: TorrentAction;
  readonly work: (id: string) => string | undefined;
  readonly clock: Clock;
};

// Each refusal's sentence, for one Torrent or for several refused alike.
const refusalWords: {
  readonly [C in DownloadclientsRefusalCode]: (refused: Refused, context: Context) => string;
} = {
  not_found: ({ names }) =>
    names.length === 1
      ? `${names[0]} is no longer there.`
      : `${torrents(names.length)} are no longer there.`,
  removed: ({ names }) =>
    names.length === 1
      ? `${names[0]} is already being removed.`
      : `${torrents(names.length)} are already being removed.`,
  importing: ({ names, refusal }, { work }) =>
    `SlipStream is finishing the import of ${(refusal.workId === undefined ? undefined : work(refusal.workId)) ?? names[0]}. Try again once it's done.`,
  not_stalled: ({ names }) =>
    `${names.join(", ")} ${names.length === 1 ? "is" : "are"} moving again, so nothing was marked failed.`,
  offline: ({ names }) =>
    `The Node holding ${names.join(", ")} is offline. Try again once it reconnects.`,
  window: ({ names, refusal }, { clock }) =>
    refusal.acceptsAt === undefined
      ? `A tracker of ${names.join(", ")} would reject an announce now.`
      : `A tracker of ${names.join(", ")} would reject an announce until ${sinceWords(refusal.acceptsAt, clock)}.`,
  not_running: ({ names }) => `${names.join(", ")} isn't running, so it has nothing to announce.`,
  failed: ({ names }, { action }) =>
    `${names.join(", ")} failed in the torrent client and can't ${verbs[action]}. Remove it instead.`,
};

// Refusals that say the same thing for every Torrent they hold group into one sentence.
const groupKey = ({ code, workId, acceptsAt }: DownloadclientsRefusal) =>
  [code, workId ?? "", acceptsAt ?? ""].join("\n");

/** What an action's answer says: the Torrents it acts on, and why it refused any. */
export function outcomeWords(
  results: readonly DownloadclientsActResult[],
  subjects: readonly Subject[],
  context: Context,
): { done: string | undefined; refused: string[] } {
  const nameOf = new Map(subjects.map(({ id, name }) => [id, name]));
  const name = (id: string) => nameOf.get(id) ?? "The torrent";
  const done = results.filter((result) => result.done);
  const groups = new Map<string, { names: string[]; refusal: DownloadclientsRefusal }>();
  for (const { id, refusal } of results) {
    if (refusal !== undefined) {
      const key = groupKey(refusal);
      const group = groups.get(key) ?? { names: [], refusal };
      group.names.push(name(id));
      groups.set(key, group);
    }
  }
  const [only] = done;
  return {
    done:
      only === undefined
        ? undefined
        : doing[context.action](done.length === 1 ? name(only.id) : torrents(done.length)),
    refused: [...groups.values()].map((group) => refusalWords[group.refusal.code](group, context)),
  };
}
