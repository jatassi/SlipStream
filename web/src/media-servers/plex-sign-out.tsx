import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import type { MediaserversServerView as Server } from "../api/gen/model";
import { signOutOfPlex } from "../api/gen/query/mediaservers/mediaservers";
import { adoptPlexSignOut } from "../live/adopt-media-servers";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { toast } from "../ui/toast";

const list = new Intl.ListFormat("en", { type: "conjunction" });

function signOutWords(plex: readonly Server[]): string {
  if (plex.length === 0) {
    return "SlipStream forgets its Plex sign-in. Nothing changes on plex.tv or on your Plex servers.";
  }
  const names = list.format(plex.map((server) => server.name));
  return `Signing out also removes ${names} from SlipStream, with ${plex.length === 1 ? "its" : "their"} waiting changes and play history. Nothing changes on the servers themselves.`;
}

/** Signs out of Plex, which removes every Plex server. */
export function SignOut({ plex }: { plex: readonly Server[] }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        size="small"
        onClick={() => {
          setOpen(true);
        }}
      >
        Sign out
      </Button>
      <ConfirmDialog
        title="Sign out of Plex?"
        confirmLabel="Sign out"
        look="fail"
        open={open}
        onOpenChange={setOpen}
        onConfirm={async () => {
          await signOutOfPlex();
          await adoptPlexSignOut(
            queryClient,
            plex.map((server) => server.id),
          );
          toast("Signed out of Plex");
        }}
      >
        <p>{signOutWords(plex)}</p>
      </ConfirmDialog>
    </>
  );
}
