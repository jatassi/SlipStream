import { useNavigate, useRouteContext } from "@tanstack/react-router";
import { useState } from "react";

import { useRestoreFlow } from "../../restore/flow";
import { RestoreSteps } from "../../restore/restore-steps";
import { markRestored } from "../../session/session";
import { forgetSetupCode } from "../../session/setup-code";
import { Button } from "../../ui/button";
import type { FormProps } from "./form-props";
import { AlreadySetUp } from "./refusal";
import { setupCalls } from "./restore-calls";

export function RestoreForm(props: FormProps) {
  const [claimed, setClaimed] = useState(false);
  const flow = useRestoreFlow(
    setupCalls(props, () => {
      setClaimed(true);
    }),
  );
  const { queryClient } = useRouteContext({ from: "/setup" });
  const navigate = useNavigate();
  if (claimed) {
    return <AlreadySetUp />;
  }
  return (
    <RestoreSteps
      flow={flow}
      lead="Pick the newest Backup you have. Its name holds the date it was taken."
      onCancel={flow.reset}
      signIn={() => {
        forgetSetupCode();
        markRestored(queryClient);
        void navigate({ to: "/sign-in" });
      }}
    >
      <div>
        <Button look="ghost" onClick={props.back}>
          Back
        </Button>
      </div>
    </RestoreSteps>
  );
}
