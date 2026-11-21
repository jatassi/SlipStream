import type { RestoreSummary } from "../api/gen/model";
import type { Clock } from "../ui/clock";
import { whenWords } from "../ui/time-words";

const nodesWords = (count: number) => (count === 1 ? "1 Node" : `${count} Nodes`);

export const summarySentence = (
  { takenAt, version, controller, nodes }: RestoreSummary,
  clock: Clock,
) =>
  `This Backup was taken ${whenWords(takenAt, "at", clock)} by SlipStream ${version}. It's a Backup of ${controller} and lists ${nodesWords(nodes?.length ?? 0)}.`;

export const pinSentence = ({ takenAt }: RestoreSummary, clock: Clock) =>
  `After the restore, sign in with the PIN you had ${whenWords(takenAt, "on", clock)}.`;

export const oldControllerSentence = (controller: string) =>
  `Make sure ${controller} stays off. If it starts again with this library, both machines seed the same private torrents. Each Worker you pair with this Controller also pauses ${controller} when it next connects to it.`;

export const oldControllerOffLabel = (controller: string) =>
  `${controller} is off, and I'll wipe it if it starts again`;

export const inPlaceTitle = (takenAt: string, clock: Clock) =>
  `Restore the Backup ${whenWords(takenAt, "from", clock)}?`;

export const inPlaceLost = (takenAt: string, clock: Clock) =>
  `SlipStream backs up the library as it is now, replaces it and restarts. Changes ${whenWords(takenAt, "since", clock)} are lost.`;

export const restartingSentence = "SlipStream is restarting to finish the restore.";

export const restartFailedSentence =
  "SlipStream hasn't responded since the restart. Check its log for the cause, then try again.";
