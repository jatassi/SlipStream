import type { NodesNode } from "../../api/gen/model";
import { Details } from "../../ui/details";
import { Switch } from "../../ui/switch";
import { prioritySchema } from "../edit-values";
import { LimitField } from "../fields/limit-field";
import { Choices } from "./choices";
import { type AddValues, type Source, listsRecent } from "./source";
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

/** The On, Automatic search and RSS switches; RSS stays off for a site with no list of new Releases. */
function Switches({ form, source }: { form: AddForm; source: Source }) {
  return (
    <div className="flex flex-wrap gap-x-6">
      {switches.map(([name, label]) => (
        <form.Field key={name} name={name}>
          {(field) => (
            <Switch
              label={label}
              checked={field.state.value}
              disabled={name === "rss" && !listsRecent(source)}
              onCheckedChange={field.handleChange}
            />
          )}
        </form.Field>
      ))}
    </div>
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
            <Switches form={form} source={source} />
          </div>
        </Details>
      )}
    </form.Subscribe>
  );
}
