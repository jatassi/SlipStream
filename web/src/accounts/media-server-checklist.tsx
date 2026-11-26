import { Link } from "@tanstack/react-router";

import type { IdentityServerGrant, MediaserversServerView } from "../api/gen/model";
import { Checkbox } from "../ui/checkbox";
import { Select } from "../ui/select";

type JoinAs = "friend" | "home";

const joinAs: readonly { value: JoinAs; label: string }[] = [
  { value: "friend", label: "Friend" },
  { value: "home", label: "Plex Home" },
];

const grantOf = (server: MediaserversServerView): IdentityServerGrant =>
  server.kind === "plex" ? { server: server.id, plexHome: false } : { server: server.id };

/** A new link offers every server shown to Members, each Plex one as a friend. */
export const shownGrants = (servers: readonly MediaserversServerView[]): IdentityServerGrant[] =>
  servers.filter((server) => server.membersSee).map((server) => grantOf(server));

function Row({
  server,
  grant,
  onGrant,
}: {
  server: MediaserversServerView;
  grant: IdentityServerGrant | undefined;
  onGrant: (grant: IdentityServerGrant | undefined) => void;
}) {
  return (
    <div className="flex flex-col gap-2">
      <Checkbox
        label={server.name}
        checked={grant !== undefined}
        disabled={!server.membersSee}
        onCheckedChange={(ticked) => {
          onGrant(ticked ? grantOf(server) : undefined);
        }}
      />
      {!server.membersSee && (
        <Link
          to="/settings/media-servers/$serverId/settings"
          params={{ serverId: server.id }}
          hash="members"
          className="text-meta text-ink-3 underline"
        >
          Show to Members is off
        </Link>
      )}
      {grant?.plexHome !== undefined && (
        <Select
          label="Add on Plex as"
          options={joinAs}
          value={grant.plexHome ? "home" : "friend"}
          onValueChange={(as) => {
            onGrant({ ...grant, plexHome: as === "home" });
          }}
        />
      )}
    </div>
  );
}

/** The Media Servers a Signup link offers. With no Media Servers it doesn't show. */
export function MediaServerChecklist({
  servers,
  value,
  onChange,
}: {
  servers: readonly MediaserversServerView[];
  value: readonly IdentityServerGrant[];
  onChange: (value: IdentityServerGrant[]) => void;
}) {
  if (servers.length === 0) {
    return null;
  }
  return (
    <fieldset className="flex flex-col gap-3">
      <legend className="mb-2 text-body font-semibold text-ink-2">Media Servers</legend>
      {servers.map((server) => (
        <Row
          key={server.id}
          server={server}
          grant={value.find((grant) => grant.server === server.id)}
          onGrant={(grant) => {
            onChange(
              servers.flatMap((each) => {
                const kept =
                  each.id === server.id ? grant : value.find((other) => other.server === each.id);
                return kept === undefined ? [] : [kept];
              }),
            );
          }}
        />
      ))}
    </fieldset>
  );
}
