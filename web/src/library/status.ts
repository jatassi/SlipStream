import {
  Alert02Icon,
  Calendar03Icon,
  CircleArrowUp01Icon,
  CloudOffIcon,
  DashedLineCircleIcon,
  Download04Icon,
  NotificationOff01Icon,
  Tick02Icon,
} from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";

import type { FilesState } from "../api/gen/model";
import { FilesState as wireStates } from "../api/gen/zod/slipStream.schemas.zod";
import type { LibraryWork } from "./rows";

export type Tone = "quiet" | "neutral" | "live" | "fail" | "warn" | "met";

/** A mark always carries its word beside its icon, so it never relies on colour. */
export type StatusMark = {
  readonly word: string;
  readonly icon: IconSvgElement;
  readonly tone: Tone;
};

/** One mark for every state, so a state the API adds fails the build until it has one. */
export type MarksFor<S extends string> = { readonly [K in S]: StatusMark };

/** Every state in the wire's order, what needs attention first, which the bar, sort, filter and hero follow. */
export const stateOrder: readonly FilesState[] = wireStates.options;

export const statusMarks = {
  failed: { word: "Failed", icon: Alert02Icon, tone: "fail" },
  missing: { word: "Missing", icon: DashedLineCircleIcon, tone: "neutral" },
  downloading: { word: "Downloading", icon: Download04Icon, tone: "live" },
  upgradable: { word: "Upgrade available", icon: CircleArrowUp01Icon, tone: "neutral" },
  unreleased: { word: "Unreleased", icon: Calendar03Icon, tone: "quiet" },
  met: { word: "Met", icon: Tick02Icon, tone: "met" },
} as const satisfies MarksFor<FilesState>;

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
