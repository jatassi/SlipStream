import { useNavigate, useRouteContext } from "@tanstack/react-router";

import type { RestoreFlow } from "../../restore/flow";
import { RestoreSteps } from "../../restore/restore-steps";
import { markRestored } from "../../session/session";

/** The restore flow in Settings. The restore's Write ends every session, so it ends at sign-in. */
export function FlowStep({ flow, onCancel }: { flow: RestoreFlow; onCancel: () => void }) {
  const { queryClient } = useRouteContext({ from: "/_app" });
  const navigate = useNavigate();
  return (
    <RestoreSteps
      flow={flow}
      lead="Pick a Backup file from this device. You need the passphrase it was taken with."
      onCancel={onCancel}
      signIn={() => {
        markRestored(queryClient);
        void navigate({ to: "/sign-in" });
      }}
    />
  );
}
