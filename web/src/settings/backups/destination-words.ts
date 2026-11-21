import type { BackupsCheck, BackupsDestination } from "../../api/gen/model";
import { type Clock, sinceWords } from "../../format";

const checkWords: {
  readonly [R in BackupsCheck["result"]]: (
    check: BackupsCheck,
    node: string,
    clock: Clock,
  ) => string;
} = {
  ok: (check, _node, clock) =>
    check.at === null ? "checked" : `checked ${sinceWords(check.at, clock)}`,
  asked: (_check, node) => `its check waits for ${node}`,
  none: () => "not checked yet",
  no_local_file: () => "it can't be checked, because this machine no longer has that Backup's file",
  mismatch: (check) => check.cause ?? "its check failed",
  unreadable: (check) => check.cause ?? "its check failed",
};

const counted = (count: number, one: string, many: string) =>
  `${count} ${count === 1 ? one : many}`;

export function destinationWords(destination: BackupsDestination, node: string, clock: Clock) {
  if (destination.removing) {
    return "Removed. The copy already running into this folder finishes first.";
  }
  const { newest, copies } = destination;
  const lead =
    newest === null
      ? "No copy yet."
      : `Newest copy ${sinceWords(newest.copiedAt, clock)}, ${checkWords[newest.check.result](newest.check, node, clock)}.`;
  const waiting = copies.waiting + copies.copying;
  const more = [
    ...(waiting > 0 ? [`${counted(waiting, "Backup waits", "Backups wait")} to copy`] : []),
    ...(copies.failed > 0 ? [`${counted(copies.failed, "copy", "copies")} failed`] : []),
  ];
  return more.length === 0 ? lead : `${lead} ${more.join(", ")}.`;
}
