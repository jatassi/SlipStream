import type { NodesNode } from "../../api/gen/model";
import { Select } from "../../ui/select";
import { AddressField } from "../fields/address-field";
import { searchingNodes } from "../move-edit";
import { Families } from "./families-field";
import { type Source, formOf } from "./source";
import type { AddForm } from "./use-add";

/** Where it searches from, which of a catalog site's addresses, and the kinds of media it asks for. */
export function Choices({
  form,
  source,
  nodes,
}: {
  form: AddForm;
  source: Source;
  nodes: readonly NodesNode[];
}) {
  const searching = searchingNodes(nodes);
  const definition = formOf(source);
  const links = source.kind === "catalog" ? source.form.links : [];
  return (
    <>
      {searching.length > 1 && (
        <form.Field name="homeNodeId">
          {(field) => (
            <Select
              label="Search from"
              options={searching.map((node) => ({ value: node.id, label: node.name }))}
              value={field.state.value}
              onValueChange={field.handleChange}
            />
          )}
        </form.Field>
      )}
      {links.length > 1 && (
        <AddressField form={form} fields="baseAddress" name="" links={links} typeable={false} />
      )}
      {definition !== undefined && (
        <Families form={form} categories={definition.entry.categories} />
      )}
    </>
  );
}
