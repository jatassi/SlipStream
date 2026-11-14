import type { QueryClient } from "@tanstack/react-query";

import type {
  MediaserversAccountView as Account,
  MediaserversKindView as Kind,
  MediaserversServerView as Server,
} from "../api/gen/model";
import { List, ListRow } from "../ui/list";
import { Section } from "../ui/section";
import { SignalLine } from "./lines";
import { SignOut } from "./plex-sign-out";
import { plexAccountQuery, usePlexAccount } from "./use-plex-pin";

export const loadPlexAccount = (queryClient: QueryClient) =>
  queryClient.query({ ...plexAccountQuery, staleTime: "static" });

function AccountRow({ account, plex }: { account: Account; plex: readonly Server[] }) {
  return (
    <List label="Plex account">
      <ListRow
        title={account.name}
        meta={account.plexPass ? "Plex Pass" : "No Plex Pass"}
        trailing={<SignOut plex={plex} />}
        {...(account.refused && {
          detail: (
            <div className="pt-1 pb-1">
              <SignalLine
                level="fail"
                text="plex.tv refused SlipStream's last renewal of the Plex sign-in. Sign in with Plex again from Add."
              />
            </div>
          ),
        })}
      />
    </List>
  );
}

/** The Plex account SlipStream is signed in to, and Sign out, which removes every Plex server. */
export function PlexAccount({
  servers,
  kinds,
}: {
  servers: readonly Server[];
  kinds: readonly Kind[];
}) {
  const account = usePlexAccount();
  if (account === null) {
    return null;
  }
  const plexKinds = new Set(
    kinds.filter((kind) => kind.signIn === "plex_pin").map((kind) => kind.kind),
  );
  return (
    <Section title="Plex account">
      <AccountRow account={account} plex={servers.filter((server) => plexKinds.has(server.kind))} />
    </Section>
  );
}
