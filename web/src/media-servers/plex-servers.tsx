import { useQueryClient } from "@tanstack/react-query";

import type {
  MediaserversAccountView as Account,
  MediaserversKindView as Kind,
  MediaserversPlexServerView as PlexServer,
  MediaserversServerView as Server,
  NodesNode,
} from "../api/gen/model";
import {
  addMediaServer,
  useListPlexServersSuspense,
} from "../api/gen/query/mediaservers/mediaservers";
import { adoptMediaServer } from "../live/adopt-media-servers";
import { Button } from "../ui/button";
import { toast } from "../ui/toast";
import { PlexDetails, ServerChoice, namer, pickedValues } from "./plex-choice";
import { NodeFields } from "./server-fields";
import { draftOf, newValues, reachingOf } from "./server-form";
import { useServerForm } from "./use-server-form";

const plexDetails = new Set<string>(["address"]);

type Props = {
  kind: Kind;
  nodes: readonly NodesNode[];
  onBack: () => void;
  onSaved: (server: Server) => void;
};

type Servers = readonly [PlexServer, ...PlexServer[]];

function usePlexForm({ kind, nodes, onSaved }: Props, first: PlexServer) {
  const queryClient = useQueryClient();
  const reaching = reachingOf(nodes);
  const sheet = useServerForm(
    { ...newValues(kind, reaching), ...pickedValues(first) },
    async (value) => {
      const made = await addMediaServer(draftOf(value, kind));
      adoptMediaServer(queryClient, made);
      toast(`Added ${made.name}`);
      onSaved(made);
      return null;
    },
    plexDetails,
  );
  return { sheet, reaching };
}

function PlexForm({ account, servers, ...props }: Props & { account: Account; servers: Servers }) {
  const { sheet, reaching } = usePlexForm(props, servers[0]);
  const { form } = sheet;
  const nameOf = namer(props.nodes);
  return (
    <form
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <form.AppForm>
        <ServerChoice sheet={sheet} servers={servers} nameOf={nameOf} />
        <form.AppField name="name">
          {(field) => (
            <field.TextField
              label="Name"
              autoComplete="off"
              description="Shown in Health and on each Work's Open in button. Rename it to tell two servers apart."
            />
          )}
        </form.AppField>
        <NodeFields sheet={sheet} nodes={props.nodes} reaching={reaching} />
        <form.Subscribe selector={(state) => state.values.plexServer}>
          {(id) => (
            <PlexDetails
              sheet={sheet}
              account={account}
              server={servers.find((each) => each.machineIdentifier === id)}
              nameOf={nameOf}
            />
          )}
        </form.Subscribe>
        <form.FormAlert />
        <div className="flex flex-wrap justify-end gap-2">
          <Button onClick={props.onBack}>Back</Button>
          <form.SubmitButton label="Save" />
        </div>
      </form.AppForm>
    </form>
  );
}

/** The signed-in account's Plex servers to pick from, led by whether the account has Plex Pass. */
export function ServersPart({ account, ...props }: Props & { account: Account }) {
  const { servers } = useListPlexServersSuspense().data;
  const [first, ...rest] = servers;
  return (
    <div className="flex flex-col gap-4">
      <p className="text-pretty text-ink-1">
        {`Signed in to Plex as ${account.name}. ${account.plexPass ? "The account has Plex Pass." : "The account doesn't have Plex Pass."}`}
      </p>
      {first === undefined ? (
        <>
          <p className="text-pretty text-ink-1">
            {`${account.name}'s Plex account has no servers. Sign in to Plex on the server with this account, then add it again.`}
          </p>
          <div className="flex justify-end">
            <Button onClick={props.onBack}>Back</Button>
          </div>
        </>
      ) : (
        <>
          <p className="text-pretty text-ink-1">
            Pick the server SlipStream keeps up to date. SlipStream finds an address for it from the
            ones plex.tv lists.
          </p>
          <PlexForm {...props} account={account} servers={[first, ...rest]} />
        </>
      )}
    </div>
  );
}
