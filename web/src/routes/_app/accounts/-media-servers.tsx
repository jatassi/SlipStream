import { useState } from "react";

import { AccessActions, accessLine, useAccessRun } from "../../../accounts/access-actions";
import type {
  IdentityAccountRow,
  MediaserversAccessServer as Server,
} from "../../../api/gen/model";
import {
  offerMediaServer,
  useListMediaServerAccessSuspense,
} from "../../../api/gen/query/mediaservers/mediaservers";
import { Button } from "../../../ui/button";
import { ConfirmDialog } from "../../../ui/confirm-dialog";
import { Empty } from "../../../ui/empty";
import { List, ListRow } from "../../../ui/list";
import { Section } from "../../../ui/section";
import { Select } from "../../../ui/select";
// oxlint-disable-next-line import/max-dependencies -- each row draws its access actions, and each offerable server its Offer and Plex Home choice
import { toast } from "../../../ui/toast";

type JoinAs = "friend" | "home";

const joinAs: readonly { value: JoinAs; label: string }[] = [
  { value: "friend", label: "Friend" },
  { value: "home", label: "Plex Home" },
];

/** One Member's access, as the Account page and its delete confirmation read it. */
export const useMemberAccess = (account: string) =>
  useListMediaServerAccessSuspense({ account }).data;

function Offer({ account, server }: { account: IdentityAccountRow; server: Server }) {
  const [asking, setAsking] = useState(false);
  const [as, setAs] = useState<JoinAs>("friend");
  const offered = `Offered ${server.name} to ${account.displayName}`;
  const { busy, run } = useAccessRun(() => offerMediaServer(account.id, server.id, {}), offered);
  if (server.access !== "plex_tv") {
    return (
      <Button size="small" busy={busy} onClick={run}>
        Offer
      </Button>
    );
  }
  return (
    <>
      <Button
        size="small"
        onClick={() => {
          setAsking(true);
        }}
      >
        Offer
      </Button>
      <ConfirmDialog
        title={`Offer ${server.name} to ${account.displayName}?`}
        confirmLabel="Offer"
        open={asking}
        onOpenChange={setAsking}
        onConfirm={async () => {
          await offerMediaServer(account.id, server.id, {
            plexHome: as === "home",
          });
          toast(offered);
        }}
      >
        <Select label="Add on Plex as" options={joinAs} value={as} onValueChange={setAs} />
      </ConfirmDialog>
    </>
  );
}

/** Each Media Server the Member has access to or may be offered, with what an Admin may do there. */
export function MediaServers({ account }: { account: IdentityAccountRow }) {
  const { rows, offerable } = useMemberAccess(account.id);
  const offers = account.admin ? [] : (offerable ?? []);
  if (account.admin && rows.length === 0) {
    return null;
  }
  return (
    <Section title="Media Servers">
      {rows.length === 0 && offers.length === 0 ? (
        <Empty look="sheet">
          {"No Media Server is shown to Members. Turn on Show to Members in a server's Settings."}
        </Empty>
      ) : (
        <List label="Media Servers">
          {rows.map((row) => (
            <ListRow
              key={row.id}
              title={row.server.name}
              detail={
                <div className="mt-1 flex flex-col gap-2">
                  <p className="text-pretty text-ink-2">{accessLine(row)}</p>
                  <AccessActions row={row} />
                </div>
              }
            />
          ))}
          {offers.map((server) => (
            <ListRow
              key={server.id}
              title={server.name}
              meta={`${account.displayName} can't watch on it yet.`}
              trailing={<Offer account={account} server={server} />}
            />
          ))}
        </List>
      )}
    </Section>
  );
}
