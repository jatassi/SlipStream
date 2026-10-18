import { Link } from "@tanstack/react-router";

import { Chip } from "../../../../ui/chip";
import type { Adding } from "../-add-node/add-node";
import { Findings } from "../-findings";
import type { Acting } from "../-manage/dialogs";
import { StateDot } from "../-state-dot";
import { AddRow } from "./add-row";
import { RowMenu } from "./row-menu";
import { rosterGroup, rowKeys, rowShell, rowWash } from "./rows";
import { type Landing, type NodeRow, shareDone } from "./view";

const torrentsLine = ({ torrents }: NodeRow): string =>
  `Downloading ${torrents.map((torrent) => `${torrent.title}, ${shareDone(torrent)}`).join(" · ")}`;

function RosterRow({ node, act }: { node: NodeRow; act: (acting: Acting) => void }) {
  return (
    <li className={rowShell}>
      <div className="flex items-start gap-1 pr-2">
        <Link
          to="/nodes/$nodeId"
          params={{ nodeId: node.id }}
          {...rowKeys}
          className={`flex min-w-0 flex-1 items-start gap-3 px-4 py-3 ${rowWash}`}
        >
          <span className="flex h-(--body-line-height) items-center">
            <StateDot state={node.state} worded />
          </span>
          <span className="flex min-w-0 flex-1 flex-col gap-1.5">
            <span className="text-(length:--body-size) leading-(--body-line-height) font-medium text-ink-1 in-data-[density=touch]:text-card-title">
              {node.name}
            </span>
            <span className="text-meta text-ink-3">{node.facts.join(" · ")}</span>
            {node.roles.length === 0 ? (
              <span className="text-meta text-ink-3">No Roles yet</span>
            ) : (
              <span className="flex flex-wrap gap-1.5">
                {node.roles.map(({ role, words }) => (
                  <Chip key={role} look="outline">
                    {words}
                  </Chip>
                ))}
              </span>
            )}
            {node.torrents.length > 0 && (
              <span className="text-meta text-ink-3">{torrentsLine(node)}</span>
            )}
            <Findings findings={node.findings} label={`What ${node.name} found`} />
          </span>
        </Link>
        <span className="pt-2">
          <RowMenu node={node} act={act} />
        </span>
      </div>
    </li>
  );
}

/**
 * Every Node as a row: its state, Roles, when it was last seen and how it is
 * reached. It says everything the Diagram draws, for a screen reader and for
 * a narrow window.
 */
export function Roster({
  landing,
  adding,
  act,
}: {
  landing: Landing;
  adding: Adding;
  act: (acting: Acting) => void;
}) {
  return (
    <ul
      aria-label="Roster"
      {...rosterGroup}
      className="rounded-card bg-glass ring-1 ring-hairline ring-inset"
    >
      {landing.nodes.map((node) => (
        <RosterRow key={node.id} node={node} act={act} />
      ))}
      {landing.nodes.length === 1 && <AddRow adding={adding} />}
    </ul>
  );
}
