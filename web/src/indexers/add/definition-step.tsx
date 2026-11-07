import type { IndexersDefinitionForm, NodesNode } from "../../api/gen/model";
import { Details } from "../../ui/details";
import { portFormOf } from "../port-shape";
import { AddIndexerForm } from "./add-form";
import { unrunnable } from "./picker-parts";
import type { Source } from "./source";

/** A definition's or a ported site's form, or why SlipStream can't run it, with the reason under Details. */
export function DefinitionStep({
  definition,
  nodes,
}: {
  definition: IndexersDefinitionForm;
  nodes: readonly NodesNode[];
}) {
  const { reason } = definition.entry;
  if (reason !== null) {
    return (
      <div className="max-w-[640px]">
        <Details summary={`${unrunnable}.`}>
          <p className="mt-1 max-w-[64ch]">{reason}</p>
        </Details>
      </div>
    );
  }
  const source: Source =
    definition.entry.origin === "native"
      ? { kind: "native", form: portFormOf(definition) }
      : { kind: "catalog", form: definition };
  return <AddIndexerForm source={source} nodes={nodes} />;
}
