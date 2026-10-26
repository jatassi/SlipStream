import type { PairingProblem, PairingWorkerClaim } from "../../api/gen/model";
import { useNow } from "../../tasks/use-now";
import { AlertLine } from "../../ui/alert-line";
import { Button } from "../../ui/button";
import { ReadOnlyField } from "../../ui/read-only-field";
import { Spinner } from "../../ui/spinner";
import { countdown } from "../_app/nodes/-add-node/state";
import { joinSentences } from "./join-sentences";

function validity(until: string, now: Date): string {
  const left = Date.parse(until) - now.getTime();
  return left > 0 ? `Expires in ${countdown(left)}.` : "The Node address has expired.";
}

/** A Join that waits for its Controller to dial the Node address this Node printed. */
export function JoinWait({ claim }: { claim: PairingWorkerClaim }) {
  const now = useNow(1000);
  return (
    <div className="flex flex-col gap-4">
      <p>This Node can&apos;t reach your SlipStream.</p>
      <ReadOnlyField
        label="Node address"
        value={claim.nodeAddress ?? ""}
        {...(claim.until === undefined ? {} : { description: validity(claim.until, now) })}
      />
      <p>Paste this Node address into Add a Node on your SlipStream.</p>
      <p className="flex items-center gap-2 text-ink-3">
        <Spinner />
        Waiting for your SlipStream to dial this Node…
      </p>
    </div>
  );
}

/** A Join that failed while it waited, and the way to try again. */
export function JoinFailed({
  problem,
  onAnother,
}: {
  problem: PairingProblem | undefined;
  onAnother: () => void;
}) {
  return (
    <div className="flex flex-col items-start gap-3">
      <AlertLine text={joinSentences[problem?.code ?? "internal"].text} />
      {problem?.code === "internal" ? (
        <p className="text-meta text-ink-3">{problem.detail}</p>
      ) : null}
      <Button look="primary" onClick={onAnother}>
        Try another Join Code
      </Button>
    </div>
  );
}
