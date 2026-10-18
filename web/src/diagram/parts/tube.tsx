import type { LineTone, Point3, Projection } from "@anatomy/iso-kit";
import { type TubeOptions, type TubePiece, tubePieces } from "@anatomy/tube";

import { type Drawn, type Pipe, keyed } from "./part";

function Piece({ piece, tone }: { piece: TubePiece; tone: LineTone }) {
  return (
    <g className="tb">
      <path className="tb-body" d={piece.body} />
      {piece.wide && (
        <>
          <path className="tb-shine" d={piece.shine} />
          <path className="tb-shade" d={piece.shade} />
        </>
      )}
      <path className="iso-line tb-edge" data-tone={tone} d={piece.edges} />
    </g>
  );
}

/** A tube along a route, cut into pieces; the pipe is its true shape for the audit. */
export function tube(
  P: Projection,
  name: string,
  points: readonly Point3[],
  r: number,
  {
    tone = "mid",
    className,
    ...options
  }: TubeOptions & { readonly tone?: LineTone; readonly className?: string } = {},
): Drawn & { readonly pipe: Pipe } {
  const pieces = tubePieces(points, r, P, { maxLength: 400, ...options });
  const drawn = pieces.map((piece): Drawn => ({
    el: <Piece piece={piece} tone={tone} />,
    shapes: [],
  }));
  return {
    el: className === undefined ? keyed(drawn) : <g className={className}>{keyed(drawn)}</g>,
    shapes: [],
    pipe: { name, points, r },
  };
}
