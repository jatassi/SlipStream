import type { ReachabilityRoute } from "../api/gen/model";
import { Chip } from "../ui/chip";
import { Defined } from "../ui/defined";
import { SignalChip } from "../ui/signal-chip";
import { notTested, verdicts } from "./words";

/**
 * The verdict's chip, in signal colour only when a failing check decides it
 * (DESIGN.md Fixed Signal Rule), and a "Not tested" chip beside it until a
 * Node outside the home tested every family.
 */
export function VerdictChips({ route }: { route: ReachabilityRoute }) {
  if (route.verdict === undefined) {
    return null;
  }
  const term = verdicts[route.verdict];
  return (
    <span className="inline-flex flex-wrap items-center gap-1.5">
      <Defined term={term}>
        {(word) =>
          route.card === "fail" || route.card === "warn" ? (
            <SignalChip kind={route.card}>{word}</SignalChip>
          ) : route.tested ? (
            <SignalChip kind="ok">{word}</SignalChip>
          ) : (
            <Chip>{word}</Chip>
          )
        }
      </Defined>
      {!route.tested && (
        <Defined term={notTested}>{(word) => <Chip look="outline">{word}</Chip>}</Defined>
      )}
    </span>
  );
}
