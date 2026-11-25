import { useState } from "react";

import { useGetRemoteAccessSuspense } from "../api/gen/query/reachability/reachability";
import { useAddNode } from "../routes/_app/nodes/-add-node/add-node";
import { Page } from "../shell/page";
import { Section } from "../ui/section";
import { ClassLines } from "./class-lines";
import { NothingWanted } from "./empty";
import { Suggestions } from "./suggestions";
import { Wanted } from "./wanted";

/**
 * Remote access: what is set up for use away from home and whether players
 * outside can connect to it, one recommendation first and each route under
 * it, then the suggestions and each Node's line. A pushed page of Nodes,
 * with no nav row of its own. The suggestions keep one place in the tree
 * whether or not anything is set up, so adding one keeps what the Admin
 * typed into another.
 */
export function RemoteAccessPage() {
  const page = useGetRemoteAccessSuspense().data;
  const adding = useAddNode();
  const [choosing, setChoosing] = useState(false);
  const nothing = page.nothingWanted;
  return (
    <Page>
      {nothing ? (
        <NothingWanted
          suggested={page.suggestions.length > 0}
          choosing={choosing}
          onChoose={() => {
            setChoosing(true);
          }}
        />
      ) : (
        <Wanted page={page} pair={adding.start} />
      )}
      {page.suggestions.length > 0 && (!nothing || choosing) && (
        <Section title="Suggested">
          <Suggestions suggestions={page.suggestions} />
        </Section>
      )}
      {!nothing && (
        <Section title="Each Node's network">
          <ClassLines nodes={page.nodes} />
        </Section>
      )}
      {adding.dialog}
    </Page>
  );
}
