import type { DownloadclientsKind as Kind } from "../api/gen/model";
import { Button } from "../ui/button";
import { Refused } from "./refused";
import type { ClientForm } from "./use-client-sheet";

type Pair = { form: ClientForm; kind: Kind; node: string };

function PairRow({
  form,
  index,
  kind,
  node,
  onRemove,
}: Pair & { index: number; onRemove: () => void }) {
  return (
    <div className="flex flex-wrap items-start gap-2">
      <div className="min-w-48 flex-1">
        <form.AppField name={`clientFolders[${index}].client`}>
          {(field) => <field.TextField label={`Folder in ${kind.name}`} data spellCheck={false} />}
        </form.AppField>
      </div>
      <div className="min-w-48 flex-1">
        <form.AppField name={`clientFolders[${index}].node`}>
          {(field) => <field.TextField label={`Folder on ${node}`} data spellCheck={false} />}
        </form.AppField>
      </div>
      <Button
        look="ghost"
        className="mt-7"
        aria-label={`Remove pair ${index + 1}`}
        onClick={onRemove}
      >
        Remove
      </Button>
    </div>
  );
}

/** Pairs of a folder the client reports and the same folder on its Node. */
export function ClientFoldersField({ form, kind, node }: Pair) {
  return (
    <form.Field name="clientFolders" mode="array">
      {(pairs) => (
        <fieldset className="flex flex-col gap-3">
          <legend className="mb-1.5 text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch">
            Client folders
          </legend>
          <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
            {`Pair each folder ${kind.name} reports with the same folder on ${node}, such as /downloads with D:\\Torrents. Leave this empty when both see the same paths.`}
          </p>
          {pairs.state.value.map((_, index) => (
            <PairRow
              // oxlint-disable-next-line react/no-array-index-key -- a pair is its place in the list
              key={index}
              form={form}
              index={index}
              kind={kind}
              node={node}
              onRemove={() => {
                pairs.removeValue(index);
              }}
            />
          ))}
          <Refused errors={pairs.state.meta.errors} />
          <div>
            <Button
              size="small"
              onClick={() => {
                pairs.pushValue({ client: "", node: "" });
              }}
            >
              Add a pair
            </Button>
          </div>
        </fieldset>
      )}
    </form.Field>
  );
}
