import { useState } from "react";

import { DiagramFigure } from "../../../../diagram/figure";
import { MOST_NODES } from "../../../../diagram/layout";
import type { Part } from "../../../../diagram/parts-order";
import type { Adding } from "../-add-node/add-node";
import type { Acting } from "../-manage/dialogs";
import { AddOnDiagram } from "./add-row";
import { useDrops } from "./drops";
import { DiagramInspector } from "./inspect";
import type { Landing } from "./view";

/**
 * The Diagram of the landing's Nodes, with the Inspector for what is
 * selected. Where it takes drags, a dropped Role chip opens the Review of
 * that change; below 1000px it is there to look at.
 */
export function Diagram({
  landing,
  takesDrags,
  adding,
  act,
}: {
  landing: Landing;
  takesDrags: boolean;
  adding: Adding;
  act: (acting: Acting) => void;
}) {
  const [selected, setSelected] = useState<Part>();
  const { gestureOf, onDrop, review } = useDrops();
  return (
    <>
      <DiagramFigure
        landing={landing}
        takesDrags={takesDrags}
        selected={selected}
        onSelect={setSelected}
        gestureOf={gestureOf}
        onDrop={onDrop}
      >
        {landing.nodes.length === 1 && <AddOnDiagram adding={adding} />}
      </DiagramFigure>
      {landing.nodes.length > MOST_NODES && (
        <p className="mt-3 text-meta text-ink-3">
          The Diagram draws {MOST_NODES} Nodes. The Roster lists all {landing.nodes.length}.
        </p>
      )}
      <DiagramInspector
        landing={landing}
        part={selected}
        onClose={() => {
          setSelected(undefined);
        }}
        act={act}
      />
      {review}
    </>
  );
}
