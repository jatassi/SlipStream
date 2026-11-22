import type { MovesPreviewUnit, MovesPreviewUnitVerdict } from "../api/gen/model";
import type { Signal } from "../ui/signal-chip";
import { SignalLine } from "../ui/signal-line";

export const toneOf: { readonly [V in MovesPreviewUnitVerdict]: Signal } = {
  red: "fail",
  amber: "warn",
  green: "ok",
};

/** One unit of a preview: its name, then why it stays or what to know before it moves. */
export function VerdictLine({ unit }: { unit: MovesPreviewUnit }) {
  return (
    <SignalLine tone={toneOf[unit.verdict]}>
      <span className="font-medium text-ink-1">{unit.name}</span>
      {unit.reasons.map(({ code, words }) => (
        <span key={code}> {words}</span>
      ))}
    </SignalLine>
  );
}
