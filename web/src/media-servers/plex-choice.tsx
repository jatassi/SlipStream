import type {
  MediaserversAccountView as Account,
  MediaserversPlexConnection as Connection,
  MediaserversPlexServerView as PlexServer,
  NodesNode,
} from "../api/gen/model";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { RadioGroup } from "../ui/radio";
import { Refused } from "./server-fields";
import { noNode } from "./server-form";
import type { ServerFormState } from "./use-server-form";

const list = new Intl.ListFormat("en", { type: "conjunction" });

export type NameOf = (nodeId: string) => string;

export function namer(nodes: readonly NodesNode[]): NameOf {
  return (nodeId) => {
    const node = nodes.find((each) => each.id === nodeId);
    if (node === undefined) {
      throw new Error(`the Nodes list has no Node ${nodeId}`);
    }
    return node.name;
  };
}

const reachedBy = (connections: readonly Connection[], nameOf: NameOf): string[] => [
  ...new Set(
    connections.flatMap((connection) =>
      connection.reachedFrom.filter((each) => each.ok).map((each) => nameOf(each.nodeId)),
    ),
  ),
];

const reachedWords = (names: readonly string[]): string =>
  names.length === 0 ? "No Node reached it." : `Reached from ${list.format(names)}.`;

function choiceWords(server: PlexServer, nameOf: NameOf): string {
  const names = reachedBy(server.connections, nameOf);
  const typed = names.length === 0 ? " Type an address it responds at under Details." : "";
  return `Plex ${server.version}. ${reachedWords(names)}${typed}`;
}

/** A server's choices as the form takes them: its own name and the Nodes its addresses suggest. */
export const pickedValues = (server: PlexServer) => ({
  plexServer: server.machineIdentifier,
  name: server.name,
  runsOnNodeId: server.suggestedRunsOn?.id ?? noNode,
  reachingNodeId: server.suggestedReaching.id,
});

const passWords = {
  has: "With Plex Pass, Plex shows a folder's edition tags as editions of one title, can transcode on the server's graphics hardware and can use IPv6 addresses.",
  lacks:
    "Without Plex Pass, Plex ignores edition tags, transcodes on the processor only and can't use IPv6 addresses.",
} as const;

function ConnectionRows({ server, nameOf }: { server: PlexServer; nameOf: NameOf }) {
  return (
    <ul aria-label={`${server.name}'s addresses`} className="flex flex-col gap-2">
      {server.connections.map((connection) => (
        <li key={connection.uri} className="flex flex-col">
          <Data className="text-ink-1 [overflow-wrap:anywhere]">{connection.uri}</Data>
          <span className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
            {`${connection.local ? "Local" : "Remote"}. ${reachedWords(reachedBy([connection], nameOf))}`}
          </span>
        </li>
      ))}
    </ul>
  );
}

/** The address an Admin types when plex.tv doesn't list it, the listed addresses, and what Plex Pass changes. */
export function PlexDetails({
  sheet,
  account,
  server,
  nameOf,
}: {
  sheet: ServerFormState;
  account: Account;
  server: PlexServer | undefined;
  nameOf: NameOf;
}) {
  const { form } = sheet;
  return (
    <Details
      fields
      open={sheet.detailsOpen}
      onOpenChange={sheet.setDetailsOpen}
      summary="SlipStream uses the first of the server's addresses that responds from the Node it's reached from, local before remote. It never uses a Plex relay."
    >
      <div className="mt-2 flex flex-col gap-4">
        <form.AppField name="address">
          {(field) => (
            <field.TextField
              label="Address"
              type="url"
              data
              autoComplete="off"
              spellCheck={false}
              placeholder="http://192.168.1.20:32400"
              description="Leave it empty to use an address plex.tv lists. Type one when the server responds at an address plex.tv doesn't list."
            />
          )}
        </form.AppField>
        {server !== undefined && <ConnectionRows server={server} nameOf={nameOf} />}
        <p className="text-pretty">{passWords[account.plexPass ? "has" : "lacks"]}</p>
      </div>
    </Details>
  );
}

/** One radio per server. Picking one fills the name and Nodes it suggests, unless the Admin changed them. */
export function ServerChoice({
  sheet: { form },
  servers,
  nameOf,
}: {
  sheet: ServerFormState;
  servers: readonly PlexServer[];
  nameOf: NameOf;
}) {
  const choose = (id: string) => {
    const server = servers.find((each) => each.machineIdentifier === id);
    if (server === undefined) {
      throw new Error(`the Plex servers list has no server ${id}`);
    }
    const picked = pickedValues(server);
    for (const name of ["name", "runsOnNodeId", "reachingNodeId"] as const) {
      if (form.getFieldMeta(name)?.isTouched !== true) {
        form.setFieldValue(name, picked[name], { dontUpdateMeta: true });
      }
    }
  };
  return (
    <form.Field name="plexServer">
      {(field) => (
        <div className="flex flex-col gap-1.5">
          <RadioGroup
            label="Plex server"
            options={servers.map((server) => ({
              value: server.machineIdentifier,
              label: server.name,
              description: choiceWords(server, nameOf),
            }))}
            value={field.state.value}
            onValueChange={(next) => {
              field.handleChange(next);
              choose(next);
            }}
          />
          <Refused errors={field.state.meta.errors} />
        </div>
      )}
    </form.Field>
  );
}
