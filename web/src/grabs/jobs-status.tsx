import type { GrabsGrab } from "../api/gen/model";
import { type StatusPart, StatusSentence } from "../ui/status-sentence";
import { failureAnchor } from "./failure-cards";
import type { FailedGrab } from "./stages";

const count = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

function statusParts(
  grabs: readonly GrabsGrab[],
  failed: readonly FailedGrab[],
  moves: number,
): StatusPart[] {
  const moving = grabs.filter((grab) => grab.inFlight).length + moves;
  const imported = grabs.filter((grab) => grab.stage === "imported").length;
  const [firstFailed] = failed;
  return [
    moving === 0
      ? { kind: "plain", text: "Nothing in progress" }
      : { kind: "live", text: `${moving} in progress` },
    ...(firstFailed === undefined
      ? []
      : [
          {
            kind: "failure",
            text: count(failed.length, "failure", "failures"),
            href: `#${failureAnchor(firstFailed)}`,
          } as const,
        ]),
    ...(imported === 0 ? [] : [{ kind: "plain", text: `${imported} imported` } as const]),
  ];
}

/** The Grabs and Moves in progress, the failures and the imports, in one sentence. */
export function JobsStatus({
  grabs,
  failed,
  moves,
}: {
  grabs: readonly GrabsGrab[];
  failed: readonly FailedGrab[];
  /** The Moves still running. */
  moves: number;
}) {
  return <StatusSentence parts={statusParts(grabs, failed, moves)} />;
}
