import type { WorksDeletePlan, WorksDeletingFiles } from "../../api/gen/model";
import { type Noun, bytes, countWords } from "../../format";
import type { Named } from "./actions";

const file = { one: "file", many: "files" } as const;
const keptFile = { one: "Kept file", many: "Kept files" } as const;
const download = { one: "download", many: "downloads" } as const;

/** The Works' files as a possessive, "its file" or "their 4 files", with the count only when asked. */
export function filesOf(named: Named, count: number, counted: boolean, noun: Noun = file): string {
  if (count === 1) {
    return `${named.one ? "its" : "their one"} ${noun.one}`;
  }
  const its = named.one ? "its" : "their";
  return counted ? `${its} ${countWords(count, noun)}` : `${its} ${noun.many}`;
}

export const capital = (words: string) => `${words.charAt(0).toUpperCase()}${words.slice(1)}`;

function heldWords(plan: WorksDeletePlan, named: Named): string {
  const allKept = plan.kept === plan.files;
  const held = `${capital(filesOf(named, plan.files, true, allKept ? keptFile : file))} (${bytes(plan.bytes)})`;
  return plan.kept === 0 || allKept
    ? held
    : `${held}, including ${countWords(plan.kept, keptFile)},`;
}

function filesWords(plan: WorksDeletePlan, files: WorksDeletingFiles, named: Named): string[] {
  const one = plan.files === 1;
  if (plan.files === 0) {
    return [`${named.words} ${named.one ? "has" : "have"} no files.`];
  }
  return files === "delete"
    ? [`${heldWords(plan, named)} ${one ? "is" : "are"} deleted from disk.`]
    : [
        `${heldWords(plan, named)} ${one ? "stays" : "stay"} on disk.`,
        `The next library scan finds ${one ? "it" : "them"} again.`,
      ];
}

/** What the delete takes with it, from its plan, as the dialog says it. */
export function planWords(
  plan: WorksDeletePlan,
  files: WorksDeletingFiles,
  named: Named,
): string[] {
  const seeding =
    plan.downloading === 0
      ? "Torrents keep seeding in the torrent client."
      : `${countWords(plan.downloading, download)} in progress ${plan.downloading === 1 ? "keeps" : "keep"} downloading, and torrents keep seeding in the torrent client.`;
  return [
    ...filesWords(plan, files, named),
    `${named.one ? "Its" : "Their"} Grabs and searches are deleted too.`,
    seeding,
  ];
}
