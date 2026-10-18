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

export function AlertDot({ tone }: { tone: AlertTone }) {
  return (
    <span className="inline-flex items-center">
      {dots[tone].dot}
      <span className="sr-only"> {dots[tone].word}</span>
    </span>
  );
}

export const dotOf = (dot: CountDot): ReactElement => dots[dot].dot;

export const dotWord = (dot: CountDot): string => dots[dot].word;
