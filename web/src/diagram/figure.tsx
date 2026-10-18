import "@anatomy/iso.css";
import "./diagram.css";

import { useStillness } from "@anatomy/react/live";
import { type ReactNode, useMemo, useRef } from "react";

import type { Landing } from "../routes/_app/nodes/-landing/view";
import { Art } from "./art";
import { drawingOf } from "./drawing";
import { Ghost } from "./ghost";
import { sayPart } from "./parts-order";
import { type Selecting, useInteraction } from "./use-interaction";
import { useMotion } from "./use-motion";
import { ZoomControls } from "./zoom-controls";

// Only what the art draws: a change to anything else keeps the drawing.
const sceneKeyOf = (landing: Landing): string =>
  JSON.stringify([
    landing.nodes.map(({ id, name, state, plaque, roles }) => [
      id,
      name,
      state,
      plaque,
      roles.map(({ role, label, note, state: held }) => [role, label, note, held]),
    ]),
    landing.links,
  ]);

function useDrawing(landing: Landing) {
  const sceneKey = sceneKeyOf(landing);
  // oxlint-disable-next-line react-hooks/exhaustive-deps -- the key says when what the art draws has changed
  return useMemo(() => drawingOf(landing), [sceneKey]);
}

type FigureProps = Selecting & { readonly children?: ReactNode };

/**
 * The live Diagram: the art, the motion the traffic ring drives, the pointer
 * and keys, and the zoom. Under reduced motion it is one still frame, and a
 * zoom jumps rather than eases.
 */
export function DiagramFigure(props: FigureProps) {
  const { landing, takesDrags, selected, children } = props;
  const drawing = useDrawing(landing);
  const figure = useRef<HTMLElement>(null);
  const svg = useRef<SVGSVGElement>(null);
  const still = useStillness();
  useMotion({ figure, svg, drawing, landing, still });
  const { preview, dragging, zoom } = useInteraction({
    ...props,
    figure,
    svg,
    stage: drawing.stage,
    still,
  });
  const shown = preview ?? selected;
  return (
    <div className="relative">
      <figure
        ref={figure}
        aria-label="Diagram"
        data-takes-drags={takesDrags}
        data-dragging={dragging?.moved === true ? "" : undefined}
        data-zoomed={zoom.fitted ? undefined : ""}
        aria-keyshortcuts="+ - 0"
        // oxlint-disable-next-line jsx-a11y/no-noninteractive-tabindex -- the arrow keys walk its Nodes, Roles and links; the Roster says the same in rows
        tabIndex={0}
        className="diagram iso relative"
      >
        <Art drawing={drawing} view={zoom.view} ref={svg} />
        <p aria-live="polite" className="sr-only">
          {shown === undefined ? "" : sayPart(landing, shown)}
        </p>
        {dragging?.moved === true && <Ghost dragging={dragging} />}
        {children}
      </figure>
      <ZoomControls zoom={zoom} />
    </div>
  );
}
