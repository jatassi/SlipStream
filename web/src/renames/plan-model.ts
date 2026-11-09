import type { FilesConfirmedWork, FilesRenameRow, FilesWorkLine } from "../api/gen/model";

/**
 * What the Admin left out of a rename plan, by Work: the whole Work, or
 * some of its files. A Work with no entry renames every file it can.
 */
export type Ticks = ReadonlyMap<string, "all" | ReadonlySet<string>>;

export const everyTick: Ticks = new Map();

/** A Work's rows as one Work's plan lists them, by its id. */
export type RowsOf = ReadonlyMap<string, readonly FilesRenameRow[]>;

/** A row renames unless it is blocked, or it trades names with a blocked row and so stays with it. */
export type RowState = "renames" | "blocked" | "held";

export function rowStates(rows: readonly FilesRenameRow[]): ReadonlyMap<string, RowState> {
  const stuck = new Set(
    rows.filter((row) => row.blocked !== undefined && row.group !== 0).map((row) => row.group),
  );
  return new Map(
    rows.map((row) => {
      if (row.blocked !== undefined) {
        return [row.fileId, "blocked"];
      }
      return [row.fileId, stuck.has(row.group) ? "held" : "renames"];
    }),
  );
}

const renaming = (rows: readonly FilesRenameRow[]) => {
  const states = rowStates(rows);
  return rows.filter((row) => states.get(row.fileId) === "renames").map((row) => row.fileId);
};

function leftFiles(ticks: Ticks, workId: string, rows: readonly FilesRenameRow[]) {
  const left = ticks.get(workId);
  if (left === "all") {
    return new Set(renaming(rows));
  }
  return new Set(left);
}

export function isTicked(ticks: Ticks, workId: string, fileId: string): boolean {
  const left = ticks.get(workId);
  return left === undefined || (left !== "all" && !left.has(fileId));
}

export function tickWork(ticks: Ticks, workId: string, on: boolean): Ticks {
  const next = new Map(ticks);
  if (on) {
    next.delete(workId);
  } else {
    next.set(workId, "all");
  }
  return next;
}

/** Ticks or leaves a file, and with it every file it trades names with. */
export function tickFile(
  ticks: Ticks,
  workId: string,
  rows: readonly FilesRenameRow[],
  fileId: string,
  on: boolean,
): Ticks {
  const row = rows.find((each) => each.fileId === fileId);
  if (row === undefined) {
    throw new Error(`the plan of ${workId} has no row for ${fileId}`);
  }
  const together =
    row.group === 0
      ? [fileId]
      : rows.filter((each) => each.group === row.group).map((each) => each.fileId);
  const left = leftFiles(ticks, workId, rows);
  for (const id of together) {
    if (on) {
      left.delete(id);
    } else {
      left.add(id);
    }
  }
  const next = new Map(ticks);
  if (left.size === 0) {
    next.delete(workId);
  } else {
    next.set(workId, left);
  }
  return next;
}

/** How many of a Work's files are ticked to rename, read from its rows once they are known. */
export function tickedIn(ticks: Ticks, line: FilesWorkLine, rows: RowsOf): number {
  const left = ticks.get(line.workId);
  if (left === "all") {
    return 0;
  }
  const known = rows.get(line.workId);
  if (known === undefined) {
    return line.rename;
  }
  return renaming(known).filter((id) => left === undefined || !left.has(id)).length;
}

export type WorkTick = "on" | "some" | "off";

export function workTick(ticks: Ticks, line: FilesWorkLine, rows: RowsOf): WorkTick {
  const ticked = tickedIn(ticks, line, rows);
  if (ticked === 0) {
    return "off";
  }
  return ticks.has(line.workId) ? "some" : "on";
}

/** The files and Works a confirm renames, as ticked. */
export function tickedCounts(
  lines: readonly FilesWorkLine[],
  ticks: Ticks,
  rows: RowsOf,
): { readonly files: number; readonly works: number } {
  let files = 0;
  let works = 0;
  for (const line of lines) {
    const ticked = tickedIn(ticks, line, rows);
    files += ticked;
    works += ticked > 0 ? 1 : 0;
  }
  return { files, works };
}

/** The Works whose rows a confirm needs and the sheet hasn't read: those left out whole, since each file is named. */
export const rowsNeeded = (lines: readonly FilesWorkLine[], ticks: Ticks, rows: RowsOf) =>
  lines
    .filter((line) => line.rename > 0 && ticks.get(line.workId) === "all" && !rows.has(line.workId))
    .map((line) => line.workId);

/**
 * The confirm's Works. Every Work with a file to rename goes in, those left
 * out whole with each of their files left, because the server judges each
 * Work's plan with the new paths of the others in the way, as the plan did.
 */
export function confirmBody(
  lines: readonly FilesWorkLine[],
  ticks: Ticks,
  rows: RowsOf,
): FilesConfirmedWork[] {
  return lines
    .filter((line) => line.rename > 0)
    .map(({ workId, digest }) => {
      const left = ticks.get(workId);
      if (left === undefined) {
        return { work: workId, digest };
      }
      if (left !== "all") {
        return { work: workId, digest, leave: [...left] };
      }
      const known = rows.get(workId);
      if (known === undefined) {
        throw new Error(`a confirm leaves ${workId} out without its rows`);
      }
      return { work: workId, digest, leave: known.map((row) => row.fileId) };
    });
}

/** After a confirm, only the Works that changed since the review are ticked, so a second confirm renames only them. */
export const onlyTicked = (lines: readonly FilesWorkLine[], changed: ReadonlySet<string>): Ticks =>
  new Map(lines.filter((line) => !changed.has(line.workId)).map((line) => [line.workId, "all"]));
