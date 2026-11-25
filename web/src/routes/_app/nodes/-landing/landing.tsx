import { useState } from "react";

import { diagramQuery, useMedia } from "../../../../shell/media";
import { Page } from "../../../../shell/page";
import { AddNodeButton, useAddNode } from "../-add-node/add-node";
import { type Acting, NodeDialogs } from "../-manage/dialogs";
import { Diagram } from "./diagram";
import { Roster } from "./roster";
import { LandingStatus } from "./status";
import { type View, ViewTabs } from "./tabs";
import { useLanding } from "./use-landing";

/**
 * Nodes: the Diagram and the Roster of the same Nodes. A window 1000px wide
 * or more opens on the Diagram, which takes drags; a narrower one and a
 * phone open on the Roster, and the Diagram is there to look at.
 */
export function NodesLanding({ view }: { view: View | undefined }) {
  const landing = useLanding();
  const wide = useMedia(diagramQuery);
  const shown = view ?? (wide ? "diagram" : "roster");
  const adding = useAddNode();
  const [acting, setActing] = useState<Acting>();
  return (
    <Page beside={<ViewTabs current={shown} />} action={<AddNodeButton adding={adding} />}>
      <LandingStatus sentence={landing.sentence} />
      {shown === "diagram" ? (
        <Diagram landing={landing} takesDrags={wide} adding={adding} act={setActing} />
      ) : (
        <Roster landing={landing} adding={adding} act={setActing} />
      )}
      {adding.dialog}
      <NodeDialogs
        acting={acting}
        onClose={() => {
          setActing(undefined);
        }}
      />
    </Page>
  );
}
