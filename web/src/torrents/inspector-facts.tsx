import type { ReactNode } from "react";

import { seedLength } from "./model";

export type Facts = readonly (readonly [term: string, value: ReactNode])[];

export function FactList({ facts }: { facts: Facts }) {
  return (
    <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-1.5">
      {facts.map(([term, value]) => (
        <div key={term} className="contents">
          <dt className="text-ink-3">{term}</dt>
          <dd className="wrap-anywhere text-ink-1">{value}</dd>
        </div>
      ))}
    </dl>
  );
}

export const lengthOf = (seconds: number) =>
  seconds < 60 ? "Under a minute" : seedLength(seconds);
