import {
  Alert02Icon,
  Calendar03Icon,
  CircleArrowUp01Icon,
  DashedLineCircleIcon,
  Download04Icon,
  Tick02Icon,
} from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";

import type { FilesState } from "../api/gen/model";
import { FilesState as wireStates } from "../api/gen/zod/slipStream.schemas.zod";

export type Tone = "quiet" | "neutral" | "live" | "fail" | "warn" | "met";

/** A mark always carries its word beside its icon, so it never relies on colour. */
export type StatusMark = {
  readonly word: string;
  readonly icon: IconSvgElement;
  readonly tone: Tone;
};

/** A state's mark with its 16px square and its state bar segment, as DESIGN.md draws them. */
export type StateMark = StatusMark & { readonly square: string; readonly bar: string };

/** One mark for every state, so a state the API adds fails the build until it has one. */
export type MarksFor<S extends string> = { readonly [K in S]: StateMark };

const hatch =
  "bg-[repeating-linear-gradient(135deg,var(--color-ink-2)_0_1.5px,transparent_1.5px_4px)] ring-1 ring-hairline-strong ring-inset";
const hairline = "ring-1 ring-hairline-strong ring-inset";

/** Every state in the wire's order, what needs attention first, which the bar, sort, filter and hero follow. */
export const stateOrder: readonly FilesState[] = wireStates.options;

/** Each state's word, icon, tone, square and bar: the one record every view of a state reads. */
export const statusMarks = {
  failed: { word: "Failed", icon: Alert02Icon, tone: "fail", square: "bg-fail", bar: "bg-fail" },
  missing: {
    word: "Missing",
    icon: DashedLineCircleIcon,
    tone: "neutral",
    square: hatch,
    bar: hatch,
  },
  downloading: {
    word: "Downloading",
    icon: Download04Icon,
    tone: "live",
    square:
      "overflow-hidden shadow-live-glow ring-1 ring-[hsl(var(--h)_90%_65%/.32)] ring-inset after:absolute after:inset-x-0 after:bottom-0 after:h-1/2 after:bg-linear-to-t after:from-glow after:to-solid",
    bar: "bg-linear-to-r from-glow to-solid shadow-live-glow",
  },
  upgradable: {
    word: "Upgrade available",
    icon: CircleArrowUp01Icon,
    tone: "neutral",
    square:
      "ring-2 ring-solid ring-inset after:absolute after:inset-1.25 after:rounded-capsule after:bg-solid",
    bar: "ring-1 ring-solid ring-inset",
  },
  unreleased: {
    word: "Unreleased",
    icon: Calendar03Icon,
    tone: "quiet",
    square: hairline,
    bar: hairline,
  },
  met: { word: "Met", icon: Tick02Icon, tone: "met", square: "bg-solid", bar: "bg-solid" },
} as const satisfies MarksFor<FilesState>;
