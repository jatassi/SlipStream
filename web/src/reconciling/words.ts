import type {
  ReconcilingGroup,
  ReconcilingNodeRow,
  ReconcilingPage,
  ReconcilingRootRow,
  ReconcilingUnknown,
} from "../api/gen/model";
import type { Clock } from "../ui/clock";
import { whenWords } from "../ui/time-words";

/** Parts joined as the Controller's own words join them: "a, b and c". */
export function joinWords(parts: readonly string[]): string {
  const last = parts.at(-1);
  return parts.length < 2 || last === undefined
    ? (last ?? "")
    : `${parts.slice(0, -1).join(", ")} and ${last}`;
}

const list = { format: joinWords };
const grouped = new Intl.NumberFormat("en");

export const counted = (count: number, one: string, many: string): string =>
  `${grouped.format(count)} ${count === 1 ? one : many}`;

export const torrentsWords = (count: number): string => counted(count, "torrent", "torrents");

/** A group's object after its verb: "this torrent" for one, "these 3" for more. */
export const theseWords = (count: number): string =>
  count === 1 ? "this torrent" : `these ${grouped.format(count)}`;

const rootsWords = (count: number): string => counted(count, "library root", "library roots");

const capital = (words: string): string => words.charAt(0).toUpperCase() + words.slice(1);

/**
 * Where a Node stands in Reconciling. A Node that hasn't reached this
 * Controller after a restore on a new machine is still paired with the old
 * one, so it needs pairing again. Elsewhere it is only offline.
 */
export type NodeState =
  | "granted"
  | "matched"
  | "continued"
  | "refused"
  | "matching"
  | "pair"
  | "lapsed"
  | "offline"
  | "waiting";

const reached = new Set<ReconcilingNodeRow["link"]>(["in-step", "mismatched"]);

export function nodeState(row: ReconcilingNodeRow, newMachine: boolean): NodeState {
  if (row.adopted) {
    return row.granted === null ? "matched" : "granted";
  }
  if (row.continued) {
    return "continued";
  }
  if (row.refused !== null) {
    return "refused";
  }
  if (row.reported) {
    return "matching";
  }
  if (reached.has(row.link)) {
    return "waiting";
  }
  if (newMachine) {
    return "pair";
  }
  return row.link === "lapsed" ? "lapsed" : "offline";
}

export const nodeWaits = (row: ReconcilingNodeRow): boolean => !row.adopted && !row.continued;

export const rootWaits = (row: ReconcilingRootRow): boolean =>
  !row.continued && !(row.saved && row.scanned);

export function nodeLine(row: ReconcilingNodeRow, state: NodeState, clock: Clock): string {
  const { name } = row;
  switch (state) {
    case "granted":
      return row.granted ?? "";
    case "matched":
      return row.held === 0
        ? `${name} reported no torrents.`
        : `SlipStream matched the ${torrentsWords(row.held)} ${name} reported with the library.`;
    case "continued":
      return `SlipStream stopped waiting for ${name}. It keeps its torrents until it reports them.`;
    case "refused":
      return `${name} sent a list SlipStream can't use, so it keeps all its torrents.`;
    case "matching":
      return `SlipStream is matching the ${torrentsWords(row.held)} ${name} reported with the library.`;
    case "pair":
      return `${name} is still paired with the old Controller. Pair it again.`;
    case "lapsed":
    case "offline":
      return `${name} has been offline ${whenWords(row.linkSince, "since", clock)}.`;
    case "waiting":
      return `Waiting for ${name} to report its torrents.`;
    default:
      return state satisfies never;
  }
}

export function rootLine(row: ReconcilingRootRow): string {
  if (row.continued) {
    return "SlipStream stopped waiting for its scan.";
  }
  if (!row.saved) {
    return "Waiting for its folder.";
  }
  return row.scanned ? "Scanned." : "Scanning…";
}

const osNames: Readonly<Record<string, string>> = {
  linux: "Linux",
  windows: "Windows",
  darwin: "macOS",
  freebsd: "FreeBSD",
};

export function unknownLine(node: ReconcilingUnknown, clock: Clock): string {
  const os = osNames[node.os] ?? node.os;
  return `A Node at ${node.address} (${os}, ${node.arch}) tried to connect ${whenWords(node.at, "at", clock)}. It isn't in this Backup. Give it a Join Code.`;
}

export const bannerLead =
  "SlipStream is matching the restored library with the torrents and files on your Nodes. Searches and removals wait until it finishes.";

/**
 * What the banner says after its lead: every step left in the order the
 * Admin takes them, pairing, then folders, then scans. When only Nodes
 * offline past the offline grace are left, it recommends continuing
 * without them instead.
 */
export type Next =
  | { readonly kind: "steps"; readonly sentence: string }
  | {
      readonly kind: "offline";
      readonly sentence: string;
      readonly nodes: readonly ReconcilingNodeRow[];
    };

export function nextOf(page: ReconcilingPage, clock: Clock): Next {
  const newMachine = page.restore?.newMachine ?? false;
  const waiting = page.nodes.filter(nodeWaits);
  const by = (states: readonly NodeState[]) =>
    waiting.filter((row) => states.includes(nodeState(row, newMachine))).map(({ name }) => name);
  const pair = by(["pair"]);
  const report = by(["waiting", "matching"]);
  const away = by(["offline"]);
  const without = by(["lapsed", "refused"]);
  const roots = page.roots.filter(rootWaits);
  const unsaved = roots.filter(({ saved }) => !saved).length;
  const scanning = roots.length - unsaved;
  const rehome = page.restore?.rehome ?? false;
  const lapsed = waiting.filter((row) => nodeState(row, newMachine) === "lapsed");

  if (lapsed.length > 0 && lapsed.length === waiting.length && roots.length === 0 && !rehome) {
    const [only] = lapsed;
    const sentence =
      lapsed.length === 1 && only !== undefined
        ? `SlipStream is waiting only for ${only.name}, which has been offline ${whenWords(only.linkSince, "since", clock)}. Continue without it so searches start again.`
        : `SlipStream is waiting only for ${list.format(lapsed.map(({ name }) => name))}, which are offline. Continue without them so searches start again.`;
    return { kind: "offline", sentence, nodes: lapsed };
  }

  const steps = [
    pair.length > 0 && `pair ${list.format(pair)} again`,
    rehome && "find the old Controller's folders",
    !rehome &&
      unsaved > 0 &&
      `find the ${unsaved === 1 ? "folder" : "folders"} of ${rootsWords(unsaved)}`,
    report.length > 0 &&
      `wait for ${list.format(report)} to report ${report.length === 1 ? "its" : "their"} torrents`,
    away.length > 0 && `wait for ${list.format(away)} to come back online`,
    without.length > 0 && `continue without ${list.format(without)}`,
    scanning > 0 && `wait for ${rootsWords(scanning)} to finish scanning`,
  ].filter((step) => step !== false);
  return {
    kind: "steps",
    sentence:
      steps.length === 0 ? "SlipStream finishes in a moment." : `${capital(list.format(steps))}.`,
  };
}

/** The review list group's lead sentence, and whether Re-add of the whole group is the step. */
export function groupWords({ reason, found, count }: ReconcilingGroup): {
  sentence: string;
  readdAll: boolean;
} {
  const many = count === 1 ? "has" : "have";
  if (reason === "conflict") {
    return {
      sentence: `${capital(torrentsWords(count))} ${count === 1 ? "is" : "are"} on two Nodes, and both keep seeding ${count === 1 ? "it" : "them"}. Re-add each one on the Node that should keep it.`,
      readdAll: false,
    };
  }
  switch (found) {
    case "files":
      return {
        sentence: `${capital(torrentsWords(count))} ${many} ${count === 1 ? "its" : "their"} files on disk. Re-add ${count === 1 ? "it" : "them"} to keep seeding from those files.`,
        readdAll: true,
      };
    case "no_files":
      return {
        sentence: `${capital(torrentsWords(count))} ${many} no files on any Node. Forget ${count === 1 ? "it" : "them"}, or re-add one to download it again.`,
        readdAll: false,
      };
    case "not_looked":
      return {
        sentence: `SlipStream hasn't looked for the files of ${torrentsWords(count)} yet. Their Node looks once their download folder is found.`,
        readdAll: false,
      };
    default:
      return found satisfies never;
  }
}

/** The Finish now dialog's words: everything Reconciling stops waiting for. */
export function finishWords(page: ReconcilingPage): string {
  const nodes = page.nodes.filter(nodeWaits).map(({ name }) => name);
  const waits = [
    ...(page.restore?.rehome === true ? ["the old Controller's folders"] : []),
    ...nodes,
    ...page.roots.filter(rootWaits).map(({ label }) => `the ${label}`),
  ];
  const lead =
    waits.length === 0
      ? "Finish Reconciling now?"
      : `Finish Reconciling now? SlipStream stops waiting for ${list.format(waits)}.`;
  if (nodes.length === 0) {
    return lead;
  }
  return `${lead} ${list.format(nodes)} ${nodes.length === 1 ? "keeps its" : "keep their"} torrents until ${nodes.length === 1 ? "it reports them" : "they report them"}.`;
}
