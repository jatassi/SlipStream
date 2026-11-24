import { useState } from "react";

import type { ReachabilityRemoteAccess } from "../api/gen/model";
import { Section } from "../ui/section";
import { ClassLines } from "./class-lines";
import { HopInspector, type SelectedHop } from "./hop-inspector";
import { Recommendation } from "./recommendation";
import { RouteCard } from "./route";
import { StatusHead, routesAnchor } from "./status";
import { Suggestions } from "./suggestions";

type Selection = { readonly exposureId: string; readonly hop: number };

function selectedOf(
  page: ReachabilityRemoteAccess,
  selection: Selection | undefined,
): SelectedHop | undefined {
  const route = page.routes.find((candidate) => candidate.exposure.id === selection?.exposureId);
  const hop = selection === undefined ? undefined : route?.hops[selection.hop];
  return route === undefined || hop === undefined
    ? undefined
    : { subject: route.exposure.subject, hop, judgedAt: route.judgedAt };
}

/** The page once something is set up: the sentence, the card, the routes, the suggestions and each Node's line. */
export function Wanted({ page, pair }: { page: ReachabilityRemoteAccess; pair: () => void }) {
  const [selection, setSelection] = useState<Selection>();
  return (
    <>
      <StatusHead page={page} />
      {page.card !== null && (
        <div className="mb-6">
          <Recommendation card={page.card} pair={pair} />
        </div>
      )}
      <Section id={routesAnchor} title="Routes">
        <div className="flex flex-col gap-3">
          {page.routes.map((route) => (
            <RouteCard
              key={route.exposure.id}
              route={route}
              recommended={route.exposure.id === page.card?.exposureId}
              selectedHop={selection?.exposureId === route.exposure.id ? selection.hop : undefined}
              onSelectHop={(hop) => {
                setSelection({ exposureId: route.exposure.id, hop });
              }}
            />
          ))}
        </div>
      </Section>
      {page.suggestions.length > 0 && (
        <Section title="Suggested">
          <Suggestions suggestions={page.suggestions} />
        </Section>
      )}
      <Section title="Each Node's network">
        <ClassLines nodes={page.nodes} />
      </Section>
      <HopInspector
        selected={selectedOf(page, selection)}
        onClose={() => {
          setSelection(undefined);
        }}
      />
    </>
  );
}
