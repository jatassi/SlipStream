import type { ReactElement } from "react";

import { LiveDot } from "./live-chip";

export type AlertTone = "warn" | "fail";

export type CountDot = "live" | AlertTone;

const dots: { readonly [D in CountDot]: { word: string; dot: ReactElement } } = {
  live: { word: "live", dot: <LiveDot /> },
  warn: {
    word: "needs attention",
    dot: <span aria-hidden="true" className="size-1.5 shrink-0 rounded-capsule bg-warn" />,
  },
  fail: {
    word: "failed",
    dot: <span aria-hidden="true" className="size-1.5 shrink-0 rounded-capsule bg-fail" />,
  },
};

export function Count({ value, dot }: { value: number; dot?: CountDot }) {
  return (
    <span className="inline-flex items-center gap-1.5 text-label text-ink-3 tabular-nums">
      {dot === undefined ? null : dots[dot].dot}
      {value}
      {dot === undefined ? null : <span className="sr-only"> {dots[dot].word}</span>}
    </span>
  );
}

/** A warning or error dot; `words` replaces its spoken word, such as "1 error". */
export function AlertDot({ tone, words }: { tone: AlertTone; words?: string }) {
  return (
    <span className="inline-flex items-center">
      {dots[tone].dot}
      <span className="sr-only"> {words ?? dots[tone].word}</span>
    </span>
  );
}

export const dotOf = (dot: CountDot): ReactElement => dots[dot].dot;

export const dotWord = (dot: CountDot): string => dots[dot].word;
