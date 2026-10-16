import { useState } from "react";

import { sentenceOf } from "../../api/errors";
import { getNodeSystem, restartNode } from "../../api/gen/query/system/system";
import { Button } from "../../ui/button";
import { Dialog } from "../../ui/dialog";
import { Refusal } from "../../ui/setting-parts";
import { type RestartTarget, moveFor, restartStore, useRestartState } from "./restart-state";

const titleOf = (target: RestartTarget) =>
  target.self ? "Restart SlipStream?" : `Restart ${target.name}?`;

// The log folder is read before the Node stops, since the failure card names
// it once the Node no longer answers.
async function restart(target: RestartTarget) {
  const { logsDir } = await getNodeSystem(target.nodeId);
  const restarting = await restartNode(target.nodeId);
  const move = moveFor(restarting.nextPort, restarting.boundPort, new URL(location.href));
  restartStore.dispatch({
    type: "accepted",
    target,
    startedAt: restarting.startedAt,
    at: Date.now(),
    logsDir,
    ...(move && { move }),
  });
}

function Confirm({ target }: { target: RestartTarget }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();
  return (
    <div className="flex flex-col gap-5">
      <p className="text-(length:--body-size) leading-(--body-line-height) text-ink-2">
        The web app reconnects by itself. Downloads and imports pause and pick up where they left
        off.
      </p>
      {error !== undefined && <Refusal error={error} />}
      <div className="flex flex-wrap justify-end gap-2">
        <Button
          onClick={() => {
            restartStore.dispatch({ type: "dismiss" });
          }}
        >
          Cancel
        </Button>
        <Button
          look="primary"
          busy={busy}
          onClick={() => {
            setBusy(true);
            setError(undefined);
            restart(target).catch((failure: unknown) => {
              setError(sentenceOf(failure));
              setBusy(false);
            });
          }}
        >
          Restart
        </Button>
      </div>
    </div>
  );
}

/** The confirm before any Node restarts, opened from the account menu, More, the palette or Nodes. */
export function RestartDialog() {
  const { asking } = useRestartState();
  const [shown, setShown] = useState(asking);
  if (asking !== null && asking !== shown) {
    setShown(asking);
  }
  return (
    <Dialog
      title={shown === null ? "" : titleOf(shown)}
      open={asking !== null}
      onOpenChange={(open) => {
        if (!open) {
          restartStore.dispatch({ type: "dismiss" });
        }
      }}
    >
      {shown !== null && <Confirm key={shown.nodeId} target={shown} />}
    </Dialog>
  );
}
