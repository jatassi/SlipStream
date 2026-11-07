import type { WorksContentType } from "../../api/gen/model";
import { countWords } from "../../format";
import { Checkbox } from "../../ui/checkbox";
import { SignalChip } from "../../ui/signal-chip";
import { ChoiceSet } from "./families-field";
import type { AddForm, Tested } from "./use-add";

const needed = (message: string) => ({
  onSubmit: ({ value }: { value: string }) => (value.trim() === "" ? message : undefined),
});

const feedAddress = /^https?:\/\/[^/?#]+(?:\/[^?#]*)?(?:\?[^#]*)?$/u;

const addressCheck = {
  onSubmit: ({ value }: { value: string }) => {
    if (value.trim() === "") {
      return "Enter the feed's address.";
    }
    return feedAddress.test(value.trim())
      ? undefined
      : "Enter the whole address, starting with http:// or https://.";
  },
};

export function TorznabFields({ form }: { form: AddForm }) {
  return (
    <>
      <form.AppField name="address" validators={addressCheck}>
        {(field) => (
          <field.TextField
            label="Feed address"
            type="url"
            data
            autoComplete="off"
            spellCheck={false}
            description="The feed's base, which SlipStream adds /api to, such as Prowlarr's or Jackett's Torznab address."
          />
        )}
      </form.AppField>
      <form.AppField name="apiKey">
        {(field) => <field.SecretField label="API key" saved={false} />}
      </form.AppField>
    </>
  );
}

const items = { one: "item", many: "items" } as const;
const sources = { one: "source", many: "sources" } as const;

/** A passing Torznab Test: what it listed, then the warning or field its sources call for. */
export function TestReport({ form, tried }: { form: AddForm; tried: Tested }) {
  const { report } = tried;
  return (
    <div className="flex flex-col gap-3">
      <output className="text-ink-2">
        The feed passed the Test and listed {countWords(report.items, items)}.
      </output>
      {report.sources.length > 1 && (
        <output className="flex flex-wrap items-center gap-x-3 gap-y-2 rounded-card bg-warn-card px-4 py-3 text-warn-text ring-1 ring-warn-line ring-inset">
          <SignalChip kind="warn">Many sources</SignalChip>
          <span className="min-w-0 flex-[1_1_32ch]">
            This feed returned results from {countWords(report.sources.length, sources)}. Each
            source becomes its own Tracker, and all of them share this Indexer&apos;s limits. Add
            one feed per site instead.
          </span>
        </output>
      )}
      {report.sources.length === 0 && (
        <form.AppField name="trackerName" validators={needed("Name the Tracker the feed reaches.")}>
          {(field) => (
            <field.TextField
              label="Tracker"
              autoComplete="off"
              description="The feed's items don't name their source. Name the Tracker its torrents belong to."
            />
          )}
        </form.AppField>
      )}
    </div>
  );
}

function MediaField({
  form,
  contentTypes,
}: {
  form: AddForm;
  contentTypes: readonly WorksContentType[];
}) {
  return (
    <form.Field
      name="media"
      validators={{
        onSubmit: ({ value }) =>
          value.length === 0 ? "Choose the kind of media the feed carries." : undefined,
      }}
    >
      {(field) => (
        <ChoiceSet
          legend="Media"
          options={contentTypes.map((type) => ({ value: type.id, label: type.name }))}
          chosen={field.state.value}
          errors={field.state.meta.errors}
          description="Its items are judged for Works of the kinds chosen here."
          onChange={field.handleChange}
        />
      )}
    </form.Field>
  );
}

function CookieField({ form }: { form: AddForm }) {
  return (
    <form.AppField name="cookie">
      {(field) => (
        <field.SecretField
          label="Cookie"
          saved={false}
          description="Only a feed that asks for one needs it. Paste the Cookie header of a browser signed in to the Tracker."
        />
      )}
    </form.AppField>
  );
}

export function RssFields({
  form,
  contentTypes,
}: {
  form: AddForm;
  contentTypes: readonly WorksContentType[];
}) {
  return (
    <>
      <form.AppField name="address" validators={addressCheck}>
        {(field) => (
          <field.TextField
            label="Feed address"
            type="url"
            data
            autoComplete="off"
            spellCheck={false}
            description="The whole address, passkey included. Only its host is shown once saved."
          />
        )}
      </form.AppField>
      <form.AppField name="trackerName" validators={needed("Name the Tracker the feed reaches.")}>
        {(field) => (
          <field.TextField
            label="Tracker"
            autoComplete="off"
            description="The Tracker the feed's torrents belong to."
          />
        )}
      </form.AppField>
      <MediaField form={form} contentTypes={contentTypes.filter((type) => type.enabled)} />
      <form.Field name="allowNoSize">
        {(field) => (
          <Checkbox
            label="Allow items with no size"
            checked={field.state.value}
            onCheckedChange={field.handleChange}
          />
        )}
      </form.Field>
      <CookieField form={form} />
    </>
  );
}
