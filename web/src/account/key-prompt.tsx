import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { useGetSessionSuspense, useUpdateOwnAccount } from "../api/gen/query/identity/identity";
import { markSignedIn } from "../session/session";
import { Switch } from "../ui/switch";

export function KeyPromptSwitch() {
  const queryClient = useQueryClient();
  const { data } = useGetSessionSuspense();
  const update = useUpdateOwnAccount();
  return (
    <Switch
      label="Offer to add a passkey after I sign in"
      checked={data.keyPrompt === "ask"}
      disabled={update.isPending}
      onCheckedChange={(ask) => {
        void (async () => {
          markSignedIn(
            queryClient,
            await update.mutateAsync({ data: { keyPrompt: ask ? "ask" : "never" } }),
          );
        })();
      }}
    />
  );
}

/** Sets the Account's prompt to never, from the prompt itself. */
export function useNeverAsk(close: () => void) {
  const queryClient = useQueryClient();
  const update = useUpdateOwnAccount();
  const [failure, setFailure] = useState<string>();
  const neverAsk = async () => {
    setFailure(undefined);
    try {
      markSignedIn(queryClient, await update.mutateAsync({ data: { keyPrompt: "never" } }));
      close();
    } catch (error) {
      setFailure(error instanceof Error ? error.message : String(error));
    }
  };
  return { busy: update.isPending, failure, neverAsk };
}
