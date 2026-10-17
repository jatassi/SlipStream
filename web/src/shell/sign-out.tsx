import { useRouteContext } from "@tanstack/react-router";
import { useEffect } from "react";

import { useDeleteSession } from "../api/gen/query/identity/identity";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { createFlag } from "./flag";

const asked = createFlag();

export function askSignOut(): void {
  asked.set(true);
}

// Each signed-in frame mounts the dialog, so a question left open closes with
// the session that asked it.
export function SignOutDialog() {
  useEffect(
    () => () => {
      asked.set(false);
    },
    [],
  );
  const endSession = useRouteContext({ from: "__root__", select: (context) => context.endSession });
  const signOut = useDeleteSession();
  return (
    <ConfirmDialog
      title="Sign out of this device?"
      confirmLabel="Sign out"
      open={asked.useValue()}
      onOpenChange={asked.set}
      onConfirm={async () => {
        await signOut.mutateAsync();
        endSession("signed-out");
      }}
    />
  );
}
