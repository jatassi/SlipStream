import { Chip } from "../ui/chip";
import { LiveChip } from "../ui/live-chip";
import { SignalChip } from "../ui/signal-chip";
import { type Outcome, outcomes } from "./words";

export function OutcomeChip({ outcome }: { outcome: Outcome | null }) {
  if (outcome === null) {
    return <LiveChip>Running</LiveChip>;
  }
  const { word, signal } = outcomes[outcome];
  return signal === undefined ? <Chip>{word}</Chip> : <SignalChip kind={signal}>{word}</SignalChip>;
}
