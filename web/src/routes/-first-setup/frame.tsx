import type { ReactNode } from "react";

import { reconfirmWithPasskey } from "../../session/passkey";
import { ReconfirmDialog } from "../../shell/reconfirm-dialog";
import { Restart } from "../../shell/restart/restart-overlay";
import { useAttempt } from "../../targets/attempt";
import { listed } from "../../targets/words";
import { Button } from "../../ui/button";

export type StepNav = {
  /** Stores the step done, unless it already reads done, and moves to the next place. */
  readonly complete: () => Promise<void>;
  readonly skip: () => Promise<void>;
  readonly back: (() => void) | undefined;
};

/**
 * Back, Skip and the step's way on. Continue stores the step done unless the
 * screen brings its own primary, such as Add a passkey.
 */
export function StepFooter({ nav, primary }: { nav: StepNav; primary?: ReactNode }) {
  const skip = useAttempt(nav.skip);
  const go = useAttempt(nav.complete);
  const moving = skip.busy || go.busy;
  return (
    <>
      {skip.failure}
      {go.failure}
      <div className="flex flex-wrap items-center gap-2">
        {nav.back !== undefined && (
          <Button look="ghost" disabled={moving} onClick={nav.back}>
            Back
          </Button>
        )}
        <span className="flex-1" />
        <Button busy={skip.busy} disabled={go.busy} onClick={skip.run}>
          Skip
        </Button>
        {primary ?? (
          <Button look="primary" busy={go.busy} disabled={skip.busy} onClick={go.run}>
            Continue
          </Button>
        )}
      </div>
    </>
  );
}

export function Progress({
  number,
  count,
  ahead,
}: {
  number: number;
  count: number;
  ahead: readonly string[];
}) {
  return (
    <div className="flex flex-col gap-0.5 text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
      <p>
        Step {number} of {count}
      </p>
      {ahead.length > 0 && <p>After this: {listed(ahead)}</p>}
    </div>
  );
}

export function FinishSetup({ finish }: { finish: () => Promise<void> }) {
  const attempt = useAttempt(finish);
  return (
    <div className="flex flex-col items-start gap-2 border-t border-hairline pt-3">
      {attempt.failure}
      <Button look="ghost" size="small" busy={attempt.busy} onClick={attempt.run}>
        Finish setup
      </Button>
    </div>
  );
}

// The page sits outside the app's layout, so it mounts what a step's saves
// may ask for: the PIN for a re-confirmed save, and a Node's restart.
export function FrameOverlays({ passkeys }: { passkeys: boolean }) {
  return (
    <>
      <ReconfirmDialog passkey={passkeys ? reconfirmWithPasskey : undefined} />
      <Restart />
    </>
  );
}
