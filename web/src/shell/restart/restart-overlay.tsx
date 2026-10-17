import { useEffect } from "react";

import { Button } from "../../ui/button";
import { Data } from "../../ui/data";
import { Mark } from "../../ui/mark";
import { Spinner } from "../../ui/spinner";
import { ProblemCard } from "../errors";
import { thisPage } from "../reload";
import { RestartDialog } from "./restart-dialog";
import { WaitForSelf } from "./restart-poll";
import { type Move, type SelfRestart, restartStore, useRestartState } from "./restart-state";

const sentence = "text-sentence text-ink-1 in-data-[density=touch]:text-sentence-touch";

function Moving({ move }: { move: Move }) {
  return (
    <>
      <output className={sentence}>SlipStream is moving to port {move.port}</output>
      {move.href === undefined ? (
        <p className="text-body text-ink-2 in-data-[density=touch]:text-body-touch">
          Open SlipStream at its new address once it has restarted.
        </p>
      ) : (
        <Button
          look="primary"
          nativeButton={false}
          // oxlint-disable-next-line jsx-a11y/control-has-associated-label -- the Button fills the link with its label
          render={<a href={move.href} />}
        >
          Open SlipStream on port {move.port}
        </Button>
      )}
    </>
  );
}

function Restarting() {
  return (
    <>
      <output className={sentence}>Restarting SlipStream…</output>
      <Spinner className="size-5 text-ink-2" />
    </>
  );
}

function Failed({ logsDir }: { logsDir: string }) {
  return (
    <ProblemCard
      said="SlipStream hasn't come back."
      action={
        <Button
          onClick={() => {
            restartStore.dispatch({ type: "retry", at: Date.now() });
          }}
        >
          Try again
        </Button>
      }
      facts={[{ label: "Log folder", value: <Data>{logsDir}</Data> }]}
    />
  );
}

function Shown({ self }: { self: SelfRestart }) {
  if (self.phase === "failed") {
    return <Failed logsDir={self.logsDir} />;
  }
  return (
    <div className="flex flex-col items-center gap-4 text-center">
      <Mark size={48} decorative />
      {self.phase === "waiting" && self.move !== undefined ? (
        <Moving move={self.move} />
      ) : (
        <Restarting />
      )}
    </div>
  );
}

/**
 * The restarting state over the page while the browser's own Node restarts.
 * A port move can't be polled, since the page may only fetch its own origin.
 */
function RestartOverlay() {
  const { self } = useRestartState();
  const back = self?.phase === "back";
  useEffect(() => {
    if (back) {
      thisPage.reload();
    }
  }, [back]);
  if (self === null) {
    return null;
  }
  return (
    <div className="fixed inset-0 z-50 grid place-items-center bg-page/95 p-4">
      {self.phase === "waiting" && self.move === undefined && <WaitForSelf />}
      <Shown self={self} />
    </div>
  );
}

/** The confirm and the restarting state, over whichever shell the Admin uses. */
export function Restart() {
  return (
    <>
      <RestartDialog />
      <RestartOverlay />
    </>
  );
}
