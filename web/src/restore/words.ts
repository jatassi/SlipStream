import type { RestoreSummary } from "../api/gen/model";
import { type Clock, dateOf, sinceWords, takenWords } from "../format";

const nodesWords = (count: number) => (count === 1 ? "1 Node" : `${count} Nodes`);

export const summarySentence = (
  { takenAt, version, controller, nodes }: RestoreSummary,
  clock: Clock,
) =>
  `This Backup was taken ${takenWords(takenAt, clock)} by SlipStream ${version}. It's a Backup of ${controller} and lists ${nodesWords(nodes?.length ?? 0)}.`;

export const pinSentence = ({ takenAt }: RestoreSummary, clock: Clock) =>
  `After the restore, sign in with the PIN you had on ${dateOf(takenAt, clock.zone)}.`;

export const oldControllerSentence = (controller: string) =>
  `Make sure ${controller} stays off. If it starts again with this library, both machines seed the same private torrents. Each Worker you pair with this Controller also pauses ${controller} when it next connects to it.`;

export const oldControllerOffLabel = (controller: string) =>
  `${controller} is off, and I'll wipe it if it starts again`;

export const inPlaceTitle = (takenAt: string, clock: Clock) =>
  `Restore the Backup from ${sinceWords(takenAt, clock)}?`;

export const inPlaceLost = (takenAt: string, clock: Clock) =>
  `SlipStream backs up the library as it is now, replaces it and restarts. Changes since ${dateOf(takenAt, clock.zone)} are lost.`;

export const restartingSentence = "SlipStream is restarting to finish the restore.";

export const restartFailedSentence =
  "SlipStream hasn't responded since the restart. Check its log for the cause, then try again.";
