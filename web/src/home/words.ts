import type {
  DownloadclientsNodeTorrents,
  HealthCategory,
  HealthReport,
  HomeSummary,
  ShellCounts,
} from "../api/gen/model";
import { failureAnchor } from "../grabs/failure-cards";
import { isFailed } from "../grabs/stages";
import { levels } from "../health/levels";
import type { StatusPart } from "../ui/status-sentence";

export const counted = (count: number, one: string, many: string): string =>
  `${count} ${count === 1 ? one : many}`;

/** The failed count reads "100+" at the cap, since the walk stops counting there. */
export const failedCap = 100;

export const needsAnchor = "needs-you";
export const healthAnchor = "health";

/** The status sentence's parts after the jobs: failures, what waits for the Admin, and Health's worst. */
export function signalParts(
  { needsYou }: HomeSummary,
  counts: ShellCounts,
  report: HealthReport,
): StatusPart[] {
  const parts: StatusPart[] = [];
  if (needsYou.failed > 0) {
    const first = needsYou.rows.map(({ grab }) => grab).find((grab) => isFailed(grab));
    parts.push({
      kind: "failure",
      text:
        needsYou.failed >= failedCap
          ? `${failedCap}+ failures`
          : counted(needsYou.failed, "failure", "failures"),
      href: `#${first === undefined ? needsAnchor : failureAnchor(first)}`,
    });
  }
  const waiting = needsYou.waiting + counts.pendingImports;
  if (waiting > 0) {
    parts.push({ kind: "warning", text: `${waiting} waiting for you`, href: `#${needsAnchor}` });
  }
  const { first, level } = report;
  const category = report.categories.find(({ id }) => id === first?.category);
  if (level !== null && category !== undefined) {
    parts.push({
      kind: levels[level].part,
      text: `${category.label} ${levels[level].chip.toLowerCase()}`,
      href: `#${healthAnchor}`,
    });
  }
  return parts;
}

/** "Today: 5 imported · 1 upgraded", leaving out a total of zero; undefined when both are. */
export function todayWords({ imported, upgraded }: HomeSummary["recent"]["today"]) {
  const parts = [
    ...(imported > 0 ? [`${imported} imported`] : []),
    ...(upgraded > 0 ? [`${upgraded} upgraded`] : []),
  ];
  return parts.length === 0 ? undefined : `Today: ${parts.join(" · ")}`;
}

/** A Downloading Node's counts in the words Torrents uses, each left out at zero. */
export function nodeCounts(node: DownloadclientsNodeTorrents): {
  plain: string[];
  failures: string | undefined;
  warnings: string | undefined;
} {
  if (node.torrents === 0) {
    return { plain: ["no torrents"], failures: undefined, warnings: undefined };
  }
  const plain = [
    ...(node.downloading > 0 ? [`${node.downloading} downloading`] : []),
    ...(node.seeding > 0 ? [`${node.seeding} seeding`] : []),
  ];
  const failures = node.failures > 0 ? counted(node.failures, "failure", "failures") : undefined;
  const warnings = node.warnings > 0 ? counted(node.warnings, "warning", "warnings") : undefined;
  if (plain.length === 0 && failures === undefined && warnings === undefined) {
    plain.push(counted(node.torrents, "torrent", "torrents"));
  }
  return { plain, failures, warnings };
}

/** "1 error · 2 warnings", the levels of a category's items. */
export function categoryWords({ items }: HealthCategory): string {
  const errors = items.filter(({ level }) => level === "fail").length;
  const warnings = items.filter(({ level }) => level === "warn").length;
  return [
    ...(errors > 0 ? [counted(errors, "error", "errors")] : []),
    ...(warnings > 0 ? [counted(warnings, "warning", "warnings")] : []),
  ].join(" · ");
}
