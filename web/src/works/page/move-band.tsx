import type { WorksWorkMove } from "../../api/gen/model";
import { StatusSentence, type StatusPart } from "../../ui/status-sentence";

/** "Moving to den", then the step the Move is on, while it runs; its failure once it fails. */
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
  const lead = `Moving to ${move.toNodeName}`;
  const parts: StatusPart[] = [{ kind: "live", text: lead }];
  if (move.words !== lead) {
    parts.push({ kind: "plain", text: move.words });
  }
  return <StatusSentence scale="section" parts={parts} />;
}
