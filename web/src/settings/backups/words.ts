import type {
  BackupsBackup,
  BackupsBackupCopyState,
  BackupsBackupList,
  BackupsBackupTrigger,
  BackupsDestination,
} from "../../api/gen/model";
import { bytes } from "../../format";
import type { Clock } from "../../ui/clock";
import { whenWords } from "../../ui/time-words";

const daySeconds = 86_400;

// Kinds this build can't take still name a reason a screen can show.
export const triggerWords: { readonly [T in BackupsBackupTrigger]: string } = {
  scheduled: "Scheduled",
  manual: "Manual",
  update: "Before an update",
  "before-restore": "Before a restore",
  "before-import": "Automatic",
  "before-handover": "Automatic",
  superseded: "Before a wipe",
  restored: "Restored from a file",
};

export const copyWords: { readonly [S in BackupsBackupCopyState]: string } = {
  waiting: "waiting",
  copying: "copying",
  copied: "copied",
  failed: "failed",
};

/** "and" between the last two, commas before: "vps-relay, nas and den". */
export function andList(words: readonly string[]): string {
  if (words.length <= 1) {
    return words.join("");
  }
  return `${words.slice(0, -1).join(", ")} and ${words.at(-1) ?? ""}`;
}

const scheduleWords = (everySeconds: number, start: string) => {
  const days = Math.round(everySeconds / daySeconds);
  return days === 1 ? `every night at ${start}` : `every ${days} days at ${start}`;
};

export type Names = (nodeId: string) => string;

export const newestGood = (list: BackupsBackupList): BackupsBackup | undefined =>
  list.backups.find((backup) => backup.state === "verified");

function holders(backup: BackupsBackup, destinations: readonly BackupsDestination[], names: Names) {
  const copied = new Set(
    backup.copies.filter((copy) => copy.state === "copied").map((copy) => copy.destinationId),
  );
  return [
    ...new Set(
      destinations.filter((each) => copied.has(each.id)).map((each) => names(each.nodeId)),
    ),
  ];
}

function newestSentence(
  backup: BackupsBackup | undefined,
  destinations: readonly BackupsDestination[],
  names: Names,
  clock: Clock,
) {
  if (backup === undefined) {
    return "SlipStream has no good Backup yet.";
  }
  const taken = `The newest good Backup was taken ${whenWords(backup.takenAt, "at", clock)}`;
  const copies = holders(backup, destinations, names);
  if (copies.length === 0) {
    return `${taken}.`;
  }
  return `${taken}, and ${andList(copies)} ${copies.length === 1 ? "has a copy" : "have copies"}.`;
}

/** A failed lead ends with "Open the Backup task for the cause.", its link on the task's name. */
export type Lead = { readonly text: string; readonly failed: boolean };

export function leadOf({
  list,
  destinations,
  everySeconds,
  keepDays,
  start,
  names,
  clock,
}: {
  list: BackupsBackupList;
  destinations: readonly BackupsDestination[];
  everySeconds: number;
  keepDays: number;
  start: string;
  names: Names;
  clock: Clock;
}): Lead {
  const good = newestGood(list);
  const newest = newestSentence(good, destinations, names, clock);
  if (list.failure !== null) {
    return {
      text: `The last Backup failed ${whenWords(list.failure.at, "at", clock)}.${good === undefined ? "" : ` ${newest}`}`,
      failed: true,
    };
  }
  return {
    text: `SlipStream backs up your library ${scheduleWords(everySeconds, start)} and keeps scheduled Backups for ${keepDays === 1 ? "1 day" : `${keepDays} days`}. ${newest}`,
    failed: false,
  };
}

export type Recommendation = "destination" | "passphrase" | "sheet";

export function recommendationsOf(
  list: BackupsBackupList,
  destinations: readonly BackupsDestination[],
): readonly Recommendation[] {
  const out: Recommendation[] = [];
  if (destinations.length === 0) {
    out.push("destination");
  }
  if (list.passphrase.savedAt === null) {
    out.push("passphrase");
  } else if (list.sheet.downloadedAt !== null && list.sheet.unlisted.length > 0) {
    out.push("sheet");
  }
  return out;
}

export function destinationAdvice(otherNodes: boolean): string {
  return otherNodes
    ? "Add a Backup destination on another Node. If this machine's disk fails, Backups kept only on it are lost too."
    : "Add a Backup destination on another disk, or download a Backup and keep it on another device. If this disk fails, Backups kept only on it are lost too.";
}

export const passphraseAdvice =
  "Save your Backup passphrase somewhere other than this machine. You need it to restore a Backup if this machine is lost.";

export function sheetAdvice(
  list: BackupsBackupList,
  destinations: readonly BackupsDestination[],
  names: Names,
): string {
  const unlisted = new Set(list.sheet.unlisted);
  const missing = [
    ...new Set(
      destinations.filter((each) => unlisted.has(each.id)).map((each) => names(each.nodeId)),
    ),
  ];
  return `Your recovery sheet doesn't list ${andList(missing)} yet. Download it again and replace your saved copy.`;
}

export const deletedToast = (backup: BackupsBackup, clock: Clock) =>
  `Deleted the Backup ${whenWords(backup.takenAt, "from", clock)} and its copies.`;

export function roomToast(need: number, free: number, node: string) {
  return `Keeping these Backups needs about ${bytes(need)} on ${node}, which has ${bytes(free)} free. Free some space on ${node}, or add a Backup destination with more room.`;
}
