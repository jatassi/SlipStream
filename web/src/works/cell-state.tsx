import type { ReactNode } from "react";

import type { FilesState } from "../api/gen/model";
import { statusMarks } from "../ui/state-marks";
import { Chip } from "../ui/chip";
import { LiveChip } from "../ui/live-chip";
import { MetPill } from "../ui/met-pill";
import { SignalChip } from "../ui/signal-chip";

type StateTone = (typeof statusMarks)[FilesState]["tone"];

const chips: { readonly [T in StateTone]: (word: string) => ReactNode } = {
  quiet: (word) => <Chip look="outline">{word}</Chip>,
  neutral: (word) => <Chip>{word}</Chip>,
  live: (word) => <LiveChip>{word}</LiveChip>,
  fail: (word) => <SignalChip kind="fail">{word}</SignalChip>,
  met: (word) => <MetPill>{word}</MetPill>,
};

/** A cell state's chip, its word in its tone's look, never colour alone. */
export function StateChip({ state }: { state: FilesState }) {
  const { word, tone } = statusMarks[state];
  return chips[tone](word);
}
