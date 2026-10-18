import { Link } from "@tanstack/react-router";

import { movesWithWork } from "../../../../diagram/machines";
import { bytes } from "../../../../format";
import { Details } from "../../../../ui/details";
import { InspectorSection } from "../../../../ui/inspector";
import { type NodeRow, type RoleChip, shareDone } from "./view";

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
