import type { ReactNode } from "react";

import type { FilesSummary } from "../../api/gen/model";
import { Chip } from "../../ui/chip";
import { LiveChip } from "../../ui/live-chip";
import { MetPill } from "../../ui/met-pill";
import { SignalChip } from "../../ui/signal-chip";
import { type PillTone, pillsOf } from "./pills";

const pillLooks: { readonly [T in PillTone]: (words: string) => ReactNode } = {
  met: (words) => <MetPill>{words}</MetPill>,
  plain: (words) => <Chip>{words}</Chip>,
  fail: (words) => <SignalChip kind="fail">{words}</SignalChip>,
  live: (words) => <LiveChip>{words}</LiveChip>,
};

/** What the Work's cells add up to, as pills, then the rest in tertiary text. */
export function Summary({ summary }: { summary: FilesSummary }) {
  const { pills, rest } = pillsOf(summary);
  if (pills.length === 0) {
    return null;
  }
  return (
    <ul aria-label="Cells" className="flex flex-wrap items-center gap-2">
      {pills.map((pill) => (
        <li key={pill.words}>{pillLooks[pill.tone](pill.words)}</li>
      ))}
      {rest.length > 0 && <li className="text-meta text-ink-3">{rest.join(" · ")}</li>}
    </ul>
  );
}
