import { Link } from "@tanstack/react-router";

import type { NodesStorageResource } from "../../../../api/gen/model";
import { movesWithWork } from "../../../../diagram/machines";
import { bytes, countWords } from "../../../../format";
import { Details } from "../../../../ui/details";
import { InspectorSection } from "../../../../ui/inspector";
import { type NodeRow, type RoleChip, shareDone } from "./view";

const roots = { one: "root", many: "roots" } as const;
const volumes = { one: "volume", many: "volumes" } as const;

/** "3 roots on 2 volumes, 1.2 TB free", or what the Node's roots lack. */
function storageWords(storage: NodesStorageResource | undefined): string {
  if (storage === undefined || storage.roots.length === 0) {
    return "No library roots yet";
  }
  const held = countWords(storage.roots.length, roots);
  if (storage.volumes.length === 0) {
    return `${held}, not checked yet`;
  }
  const free = storage.volumes.reduce((sum, volume) => sum + volume.free, 0);
  return `${held} on ${countWords(storage.volumes.length, volumes)}, ${bytes(free)} free`;
}

function StorageFigures({ storage }: { storage: NodesStorageResource | undefined }) {
  return (
    <p className="mt-2">
      {storageWords(storage)}.{" "}
      <Link
        to="/settings/$section"
        params={{ section: "library-roots" }}
        className="text-ink-1 underline"
      >
        Library roots
      </Link>
    </p>
  );
}

/** A Role on a Node in the Inspector: what it is doing, and what its machine shows. */
export function RoleWords({ node, chip }: { node: NodeRow; chip: RoleChip }) {
  const live = chip.role === "downloading";
  return (
    <>
      <InspectorSection>
        <p>
          {chip.note === "" ? `${node.name} holds ${chip.label}.` : `${chip.label}: ${chip.note}.`}
        </p>
        {!movesWithWork(chip.role) && <p className="mt-2 text-ink-3">No live figures yet</p>}
        {chip.role === "storage" && <StorageFigures storage={node.storage} />}
        {live && node.torrents.length === 0 && (
          <p className="mt-2 text-ink-3">No torrents right now.</p>
        )}
      </InspectorSection>
      {live && node.torrents.length > 0 && (
        <InspectorSection title="Torrents">
          <ul className="flex flex-col gap-2">
            {node.torrents.map((torrent) => (
              <li key={torrent.grabId} className="flex justify-between gap-3">
                <span className="min-w-0 truncate">{torrent.title}</span>
                <span className="shrink-0 text-ink-3 tabular-nums">
                  {shareDone(torrent)} of {bytes(torrent.wanted)}
                </span>
              </li>
            ))}
          </ul>
        </InspectorSection>
      )}
      <Details summary={`${node.name}'s Roles are set in its Roles section.`}>
        <Link
          to="/nodes/$nodeId/roles"
          params={{ nodeId: node.id }}
          className="text-ink-1 underline"
        >
          Roles on {node.name}
        </Link>
      </Details>
    </>
  );
}
