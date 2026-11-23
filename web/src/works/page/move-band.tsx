import type { WorksWorkMove } from "../../api/gen/model";
import { StatusSentence, type StatusPart } from "../../ui/status-sentence";

/**
 * The step the Move is on while it runs, led by "Moving to den" unless the
 * step already names den; its failure once it fails.
 */
export function MoveBand({ move }: { move: WorksWorkMove | null }) {
  if (move === null || move.phase === "canceled") {
    return null;
  }
  if (move.phase === "failed") {
    return (
      <StatusSentence
        scale="section"
        parts={[
          { kind: "plain", text: `Couldn't move to ${move.toNodeName}` },
          { kind: "failure", text: move.words, href: "/activity" },
        ]}
      />
    );
  }
  const parts: StatusPart[] = move.words.includes(` to ${move.toNodeName}`)
    ? [{ kind: "live", text: move.words }]
    : [
        { kind: "live", text: `Moving to ${move.toNodeName}` },
        { kind: "plain", text: move.words },
      ];
  return <StatusSentence scale="section" parts={parts} />;
}
