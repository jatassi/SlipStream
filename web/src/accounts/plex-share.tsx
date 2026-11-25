import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { MediaserversAccessView as Access } from "../api/gen/model";
import {
  checkMediaServerAccess,
  getListMediaServerAccessQueryKey,
  listMediaServerAccess,
  markMediaServerAccessAdded,
  shareMediaServerAccess,
} from "../api/gen/query/mediaservers/mediaservers";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { Details } from "../ui/details";
import { ReadOnlyField } from "../ui/read-only-field";
import { toast } from "../ui/toast";
// oxlint-disable-next-line import/max-dependencies -- each Plex action draws its own button, dialog or disclosure
import { toastActed, useAccessRun } from "./access-actions";

const list = new Intl.ListFormat("en", { type: "conjunction" });

export function ShareOnPlex({ row }: { row: Access }) {
  const { busy, run } = useAccessRun(
    () => shareMediaServerAccess(row.id, { as: "friend" }),
    `Shared ${row.server.name} with ${row.member.name}`,
  );
  return (
    <Button size="small" busy={busy} onClick={run}>
      Share on Plex
    </Button>
  );
}

/** Adds the Member to the Admin's Plex Home, saying first what Home membership gives when nobody is in it yet. */
export function AddToPlexHome({ row }: { row: Access }) {
  const queryClient = useQueryClient();
  const [warning, setWarning] = useState(false);
  const share = () => shareMediaServerAccess(row.id, { as: "home" });
  const words = `Added ${row.member.name} to your Plex Home`;
  const { busy, run } = useAccessRun(share, words);
  const press = () => {
    void queryClient
      .query({
        queryKey: getListMediaServerAccessQueryKey(),
        queryFn: ({ signal }) => listMediaServerAccess(undefined, { signal }),
        staleTime: "static",
      })
      .then(({ plexHomeUsed }) => {
        if (plexHomeUsed) {
          run();
        } else {
          setWarning(true);
        }
        return null;
      })
      .catch((error: unknown) => {
        toast(sentenceOf(error), { kind: "fail" });
      });
  };
  return (
    <>
      <Button size="small" busy={busy} onClick={press}>
        Add to Plex Home
      </Button>
      <ConfirmDialog
        title={`Add ${row.member.name} to your Plex Home?`}
        confirmLabel="Add to Plex Home"
        open={warning}
        onOpenChange={setWarning}
        onConfirm={async () => {
          toastActed(await share(), words);
        }}
      >
        <p>
          People in your Plex Home can switch to other Home profiles that have no PIN. Give your own
          profile a Plex Home PIN in Plex first.
        </p>
      </ConfirmDialog>
    </>
  );
}

/** The fallback when Plex refuses the share: the Admin shares in Plex, then records it here. */
export function ShareByHand({ row }: { row: Access }) {
  const username = row.plex?.username;
  const { busy, run } = useAccessRun(
    () => markMediaServerAccessAdded(row.id),
    `Marked ${row.member.name}'s access to ${row.server.name} as added`,
  );
  return (
    <div className="basis-full">
      <Details label="Share by hand instead">
        <div className="flex flex-col items-start gap-3 pt-2">
          {username !== undefined && <ReadOnlyField label="Plex username" value={username} />}
          <p className="text-pretty">
            {`In Plex, open Settings, then Manage Library Access, and choose Grant Library Access. Give ${username ?? row.member.name} access to ${list.format(row.libraries)}. Then press Mark as added.`}
          </p>
          <Button size="small" busy={busy} onClick={run}>
            Mark as added
          </Button>
        </div>
      </Details>
    </div>
  );
}

export function CheckAgain({ row }: { row: Access }) {
  const { busy, run } = useAccessRun(
    () => checkMediaServerAccess(row.id),
    ({ row: checked }) =>
      checked?.state === "active"
        ? `${row.member.name} can watch ${row.server.name}`
        : `${row.member.name} hasn't accepted the share of ${row.server.name} yet`,
  );
  return (
    <Button size="small" busy={busy} onClick={run}>
      Check again
    </Button>
  );
}
