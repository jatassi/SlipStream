import type { BackupsDestination, NodesNode } from "../../api/gen/model";
import type { Clock } from "../../ui/clock";
import { Data } from "../../ui/data";
import { Section } from "../../ui/section";
import { AddDestinationButton } from "./add-destination";
import { destinationWords } from "./destination-words";
import { RemoveButton } from "./remove-destination";
import type { Names } from "./words";

function DestinationRow({
  destination,
  node,
  clock,
}: {
  destination: BackupsDestination;
  node: string;
  clock: Clock;
}) {
  return (
    <li
      aria-label={`${node}: ${destination.folder}`}
      className="relative flex flex-wrap items-center gap-x-3 gap-y-2 px-4 py-3 not-first:before:absolute not-first:before:inset-x-0 not-first:before:top-0 not-first:before:h-px not-first:before:bg-hairline"
    >
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <p className="font-semibold text-ink-1">
          {node}: <Data className="break-all">{destination.folder}</Data>
        </p>
        <p className="text-meta text-ink-3">{destinationWords(destination, node, clock)}</p>
      </div>
      <RemoveButton destination={destination} node={node} />
    </li>
  );
}

export function DestinationsSection({
  destinations,
  nodes,
  names,
  clock,
}: {
  destinations: readonly BackupsDestination[];
  nodes: readonly NodesNode[];
  names: Names;
  clock: Clock;
}) {
  return (
    <Section
      title="Backup destinations"
      count={destinations.length}
      action={<AddDestinationButton nodes={nodes} />}
    >
      {destinations.length === 0 ? (
        <p className="text-ink-2">
          Every Backup is only on this machine. Add a folder on another Node or disk to keep a copy
          there.
        </p>
      ) : (
        <ul
          aria-label="Backup destinations"
          className="rounded-card bg-glass ring-1 ring-hairline ring-inset"
        >
          {destinations.map((destination) => (
            <DestinationRow
              key={destination.id}
              destination={destination}
              node={names(destination.nodeId)}
              clock={clock}
            />
          ))}
        </ul>
      )}
    </Section>
  );
}
