import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { useDeleteSession } from "../../api/gen/query/identity/identity";
import { markSignedOut } from "../../session/session";
import { Button } from "../../ui/button";
import { ConfirmDialog } from "../../ui/confirm-dialog";

/** A link for someone else can't be used while signed in, so the page offers to sign out and stay. */
export function SignedInVisitor({ name, onSignedOut }: { name: string; onSignedOut: () => void }) {
  const queryClient = useQueryClient();
  const signOut = useDeleteSession();
  const [asking, setAsking] = useState(false);
  return (
    <>
      <p className="text-ink-1">You&apos;re signed in as {name}. Sign out to use this link.</p>
      <Button
        look="primary"
        onClick={() => {
          setAsking(true);
        }}
      >
        Sign out
      </Button>
      <ConfirmDialog
        title="Sign out of this device?"
        confirmLabel="Sign out"
        open={asking}
        onOpenChange={setAsking}
        onConfirm={async () => {
          await signOut.mutateAsync();
          markSignedOut(queryClient);
          onSignedOut();
        }}
      />
    </>
  );
}
