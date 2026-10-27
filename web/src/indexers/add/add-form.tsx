import { Link } from "@tanstack/react-router";

import type { NodesNode } from "../../api/gen/model";
import { Button } from "../../ui/button";
import { nameSchema } from "../edit-values";
import { searchingNodes } from "../move-indexer";
import { Fields, leadOf } from "./kind-fields";
import { MoreFields } from "./more-fields";
import type { Source } from "./source";
import { Step } from "./step";
import { type Next, nextOf, useAdd } from "./use-add";

const primaryWords: { readonly [N in Next]: string } = {
  test: "Test",
  captcha: "Continue",
  save: "Save",
};

/**
 * The form that adds an Indexer from source: what it reaches and how it
 * signs in first, its Name and settings, then the rest under Details. Its
 * primary button tests a Torznab feed, loads a captcha or saves, as its
 * step needs.
 */
export function AddIndexerForm({ source, nodes }: { source: Source; nodes: readonly NodesNode[] }) {
  const searching = searchingNodes(nodes);
  const home = searching[0];
  const add = useAdd(source, home, searching.length > 1);
  const { form, step, tried } = add;
  return (
    <form
      noValidate
      className="flex max-w-[640px] flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <form.AppForm>
        <p className="text-ink-1">{leadOf(source, home?.name ?? "its home Node")}</p>
        <form.AppField
          name="name"
          validators={{
            onSubmit: ({ value }) => nameSchema.safeParse(value).error?.issues[0]?.message,
          }}
        >
          {(field) => (
            <field.TextField
              label="Name"
              autoComplete="off"
              description="Shown in searches and on the Indexer's page. Rename it to tell two accounts apart."
            />
          )}
        </form.AppField>
        <Fields form={form} source={source} home={home} />
        <Step add={add} source={source} home={home?.name ?? "The home Node"} />
        <MoreFields form={form} source={source} nodes={nodes} />
        <form.FormAlert />
        <div className="flex flex-wrap justify-end gap-2">
          <Button nativeButton={false} render={<Link to="/settings/indexers/new" />}>
            Cancel
          </Button>
          <form.Subscribe selector={(state) => nextOf(source, state.values, step, tried)}>
            {(next) => <form.SubmitButton label={primaryWords[next]} />}
          </form.Subscribe>
        </div>
      </form.AppForm>
    </form>
  );
}
