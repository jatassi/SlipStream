import { type Ref, memo } from "react";

import type { Drawing, DrawnTile } from "./drawing";
import { type View, viewBoxOf } from "./view";

const Tile = memo(function Tile({ tile }: { tile: DrawnTile }) {
  const { node } = tile;
  return (
    <g
      data-part={`node:${node.id}`}
      data-node={node.id}
      data-state={node.state}
      data-controller={node.controller ? "" : undefined}
    >
      {tile.items.map((item) =>
        item.role === undefined ? (
          <g key={item.name}>{item.el}</g>
        ) : (
          <g
            key={item.name}
            data-part={`role:${node.id}:${item.role}`}
            data-role={item.role}
            data-role-state={node.roles.find(({ role }) => role === item.role)?.state}
            data-chip={item.name.startsWith("chip.") ? item.role : undefined}
          >
            {item.el}
          </g>
        ),
      )}
    </g>
  );
});

/**
 * The Diagram's still art: the links on the floor, then every tile back to
 * front, each tile's parts in depth-key order, seen through the view. The
 * live loop only writes attributes on what this draws.
 */
export const Art = memo(function Art({
  drawing,
  view,
  ref,
}: {
  drawing: Drawing;
  view: View;
  ref: Ref<SVGSVGElement>;
}) {
  const { links, tiles } = drawing;
  return (
    <svg
      ref={ref}
      className="iso-svg"
      viewBox={viewBoxOf(view)}
      aria-hidden="true"
      focusable="false"
    >
      <g data-links="">
        {links.map(({ route, item }) => (
          <g key={route.worker} data-part={`link:${route.worker}`} data-link={route.worker}>
            {item.el}
          </g>
        ))}
      </g>
      {tiles.map((tile) => (
        <Tile key={tile.node.id} tile={tile} />
      ))}
    </svg>
  );
});
