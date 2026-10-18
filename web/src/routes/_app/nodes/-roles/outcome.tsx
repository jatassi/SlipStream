import type { NodesCheck, NodesOutcome } from "../../../../api/gen/model";
import { type Signal, SignalChip } from "../../../../ui/signal-chip";

const staleWords = "Roles changed since you opened this. Here is the change as it stands now.";

const checkSignals: { readonly [V in NodesCheck["verdict"]]: { signal: Signal; chip: string } } = {
  block: { signal: "fail", chip: "Blocked" },
  warn: { signal: "warn", chip: "Warning" },
  allow: { signal: "ok", chip: "Checked" },
};

export function Planned({ outcome, stale }: { outcome: NodesOutcome; stale: boolean }) {
  return (
    <>
      {stale && <p className="font-semibold text-warn-ink">{staleWords}</p>}
      {outcome.effects.length === 0 ? (
        <p className="text-ink-2">Nothing changes.</p>
      ) : (
        <ul aria-label="What changes" className="flex flex-col gap-2 text-ink-1">
          {outcome.effects.map((effect) => (
            <li key={effect.words}>{effect.words}</li>
          ))}
        </ul>
      )}
    </>
  );
}

export function Checks({ checks }: { checks: readonly NodesCheck[] }) {
  if (checks.length === 0) {
    return null;
  }
  return (
    <ul aria-label="Checks" className="flex flex-col gap-2">
      {checks.map((check) => {
        const { signal, chip } = checkSignals[check.verdict];
        return (
          <li
            key={`${check.node.id}:${check.role ?? ""}:${check.reason}`}
            className="flex items-start gap-2"
          >
            <SignalChip kind={signal}>{chip}</SignalChip>
            <span className="text-ink-2">{check.words}</span>
          </li>
        );
      })}
    </ul>
  );
}
