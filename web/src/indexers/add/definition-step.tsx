import type { IndexersDefinitionForm, NodesNode } from "../../api/gen/model";
import { Details } from "../../ui/details";
import { AddIndexerForm } from "./add-form";
import { unrunnable } from "./picker-parts";

/** A definition's form, or why SlipStream can't run it, with the reason under Details. */
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
  return <AddIndexerForm source={{ kind: "catalog", form: definition }} nodes={nodes} />;
}
