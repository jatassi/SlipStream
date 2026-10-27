import type { NodesNode } from "../../api/gen/model";
import { Details } from "../../ui/details";
import { Select } from "../../ui/select";
import { Switch } from "../../ui/switch";
import { prioritySchema } from "../edit-values";
import { LimitField } from "../fields/limit-field";
import { searchingNodes } from "../move-indexer";
import { Families } from "./families-field";
import type { AddValues, Source } from "./source";
import type { AddForm } from "./use-add";

const limitWords = ({ count, per }: AddValues["searches"], noun: string) =>
  count.trim() === "0"
    ? undefined
    : `${count.trim()} ${noun} ${per === "hour" ? "an hour" : "a day"}`;

/** The settings under Details, as one sentence. */
function moreSummary(values: AddValues, nodes: readonly NodesNode[]): string {
  const home = nodes.find((node) => node.id === values.homeNodeId)?.name;
  const limits = [limitWords(values.searches, "searches"), limitWords(values.grabs, "grabs")]
    .filter((words) => words !== undefined)
    .join(" and ");
  const from = home === undefined ? "" : ` from ${home}`;
  return `It searches${from} at priority ${values.priority.trim()}, with ${limits === "" ? "no limits" : `at most ${limits}`}.`;
}

const switches = [
  ["on", "On"],
  ["automaticSearch", "Automatic search"],
  ["rss", "RSS"],
] as const;

function Choices({
  form,
  source,
  nodes,
}: {
  form: AddForm;
  source: Source;
  nodes: readonly NodesNode[];
}) {
  const searching = searchingNodes(nodes);
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
        <form.Field name="baseAddress">
          {(field) => (
            <Select
              label="Address"
              options={links.map((link) => ({ value: link, label: link }))}
              value={field.state.value}
              onValueChange={field.handleChange}
            />
          )}
        </form.Field>
      )}
      {source.kind === "catalog" && (
        <Families form={form} categories={source.form.entry.categories} />
      )}
    </>
  );
}

/** Where it searches from, its priority, limits and switches, under one sentence of them. */
export function MoreFields({
  form,
  source,
  nodes,
}: {
  form: AddForm;
  source: Source;
  nodes: readonly NodesNode[];
}) {
  return (
    <form.Subscribe selector={(state) => moreSummary(state.values, nodes)}>
      {(summary) => (
        <Details summary={summary}>
          <div className="mt-2 flex max-w-[560px] flex-col gap-4">
            <Choices form={form} source={source} nodes={nodes} />
            <form.AppField
              name="priority"
              validators={{
                onSubmit: ({ value }) => prioritySchema.safeParse(value).error?.issues[0]?.message,
              }}
            >
              {(field) => (
                <field.TextField
                  label="Priority"
                  inputMode="numeric"
                  description="From 1 to 50. When two Releases are equal, the one from the lower number wins."
                />
              )}
            </form.AppField>
            <LimitField form={form} fields="searches" label="Searches" />
            <LimitField form={form} fields="grabs" label="Grabs" />
            <div className="flex flex-wrap gap-x-6">
              {switches.map(([name, label]) => (
                <form.Field key={name} name={name}>
                  {(field) => (
                    <Switch
                      label={label}
                      checked={field.state.value}
                      onCheckedChange={field.handleChange}
                    />
                  )}
                </form.Field>
              ))}
            </div>
          </div>
        </Details>
      )}
    </form.Subscribe>
  );
}
