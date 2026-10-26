import type { WorksDeletePlan, WorksDeletingFiles } from "../../api/gen/model";
import { bytes, countWords } from "../../format";
import type { Named } from "./actions";

const file = { one: "file", many: "files" } as const;
const keptFile = { one: "Kept file", many: "Kept files" } as const;
const download = { one: "download", many: "downloads" } as const;

function heldWords(plan: WorksDeletePlan, its: string): string {
  const size = `${its} ${countWords(plan.files, file)} (${bytes(plan.bytes)})`;
  if (plan.kept === 0) {
    return size;
  }
  return plan.kept === plan.files
    ? `${size}, all of them Kept files,`
    : `${size}, including ${countWords(plan.kept, keptFile)},`;
}

function filesWords(plan: WorksDeletePlan, files: WorksDeletingFiles, named: Named): string[] {
  const its = named.one ? "Its" : "Their";
  const one = plan.files === 1;
  if (plan.files === 0) {
    return [`${named.words} ${named.one ? "has" : "have"} no files.`];
  }
  return files === "delete"
    ? [`${heldWords(plan, its)} ${one ? "is" : "are"} deleted from disk.`]
    : [
        `${heldWords(plan, its)} ${one ? "stays" : "stay"} on disk.`,
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
