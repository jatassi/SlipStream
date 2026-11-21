import { useId } from "react";

import type {
  NotificationsConnection as Connection,
  NotificationsEvent as Event,
  NotificationsProvider as Provider,
} from "../api/gen/model";
import type { Clock } from "../ui/clock";
import { SignalLine } from "../ui/signal-line";
import { EditConnectionButton } from "./connection-sheet";
import { DeleteConnectionButton } from "./delete-connection";
import type { NodeChoice } from "./provider-field";
import { givenUpNote, labelOf, providerOf, statusLine } from "./words";

type Choices = {
  providers: readonly Provider[];
  events: readonly Event[];
  nodes: readonly NodeChoice[];
};

const rowLook =
  "relative flex flex-col gap-2 px-4 py-3 not-first:before:absolute not-first:before:inset-x-0 not-first:before:top-0 not-first:before:h-px not-first:before:bg-hairline in-data-[density=touch]:not-first:before:left-4";

function StatusWords({ connection, clock }: { connection: Connection; clock: Clock }) {
  const { level, text } = statusLine(connection, clock);
  return level === "off" ? (
    <p className="text-ink-3">{text}</p>
  ) : (
    <SignalLine tone={level}>{text}</SignalLine>
  );
}

/** A connection's row: its name and provider, whether it is sending, then Edit and Delete. */
function ConnectionRow({
  connection,
  clock,
  ...choices
}: Choices & { connection: Connection; clock: Clock }) {
  const nameId = useId();
  const provider = providerOf(choices.providers, connection.provider);
  const note = givenUpNote(connection, provider);
  return (
    <li aria-labelledby={nameId} className={rowLook}>
      <div className="flex flex-wrap items-baseline gap-x-2">
        <h3 id={nameId} className="font-semibold text-ink-1">
          {connection.name}
        </h3>
        <p className="text-meta text-ink-3">{provider.label}</p>
      </div>
      <StatusWords connection={connection} clock={clock} />
      {note !== undefined && <p className="text-ink-2">{note}</p>}
      <div className="flex flex-wrap items-center gap-2">
        <EditConnectionButton
          connection={connection}
          provider={provider}
          events={choices.events}
          nodes={choices.nodes}
        />
        <DeleteConnectionButton label={labelOf(provider, connection)} id={connection.id} />
      </div>
    </li>
  );
}

/** Every connection, by name. */
export function ConnectionList({
  connections,
  clock,
  ...choices
}: Choices & { connections: readonly Connection[]; clock: Clock }) {
  return (
    <ul
      aria-label="Notification connections"
      className="rounded-card bg-glass ring-1 ring-hairline ring-inset"
    >
      {connections.map((connection) => (
        <ConnectionRow key={connection.id} connection={connection} clock={clock} {...choices} />
      ))}
    </ul>
  );
}
