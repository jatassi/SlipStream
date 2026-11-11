import { CloudOffIcon, NotificationOff01Icon } from "@hugeicons/core-free-icons";

import { type StatusMark, statusMarks } from "../ui/state-marks";
import type { LibraryWork } from "./rows";

export const notMonitored: StatusMark = {
  word: "Not monitored",
  icon: NotificationOff01Icon,
  tone: "quiet",
};

/** Its metadata source no longer lists the Work, such as "Gone from TMDB". */
export const goneFrom = (source: string): StatusMark => ({
  word: `Gone from ${source}`,
  icon: CloudOffIcon,
  tone: "warn",
});

/** A Work's marks in the order they show: its status, then Not monitored, then gone upstream. */
export function marksOf(work: LibraryWork): readonly StatusMark[] {
  return [
    ...(work.status === undefined ? [] : [statusMarks[work.status]]),
    ...(work.monitored === "none" ? [notMonitored] : []),
    ...(work.gone ? [goneFrom(work.type.source)] : []),
  ];
}
