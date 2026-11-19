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

/** One sentence that ends in a period, as a definition reads on hover or tap. */
export type Definition = `${string}.`;

export type StatusMark = {
  readonly word: string;
  readonly icon: IconSvgElement;
  readonly tone: Tone;
  /** What a technical state's word means (WRITING.md rule 25). A mark with one opens it on hover and tap. */
  readonly definition?: Definition;
};

/**
 * Every state of S in the term the rest of the industry uses, with tones
 * limited to T. Each state has a definition except the Plain ones, whose word
 * needs none. Write one beside a screen's model, `as const satisfies
 * TermsFor<WireState>`, and draw a state with `<Mark mark={terms[state]} />`.
 * A wire state added later fails the build until it has a row.
 */
export type TermsFor<S extends string, T extends Tone = Tone, Plain extends S = never> = {
  readonly [K in S]: StatusMark & { readonly tone: T } & (K extends Plain
      ? { readonly definition?: never }
      : { readonly definition: Definition });
};

/** A state's mark with its 16px square and its state bar segment, as DESIGN.md draws them. */
export type StateMark = StatusMark & { readonly square: string; readonly bar: string };

export type MarksFor<S extends string> = { readonly [K in S]: StateMark };

const hatch =
  "bg-[repeating-linear-gradient(135deg,var(--color-ink-2)_0_1.5px,transparent_1.5px_4px)] ring-1 ring-hairline-strong ring-inset";
const hairline = "ring-1 ring-hairline-strong ring-inset";

/** Every state in the wire's order, what needs attention first, which the bar, sort, filter and hero follow. */
export const stateOrder: readonly FilesState[] = wireStates.options;

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
