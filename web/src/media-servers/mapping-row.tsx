import { useId } from "react";

import type { MediaserversMappingView as MappingView } from "../api/gen/model";
import { Chip } from "../ui/chip";
import { Switch } from "../ui/switch";
import { WarnLine } from "./lines";
import { folderNoteOf } from "./mappings-form";
import type { MappingsForm } from "./use-mappings-form";

type Row = {
  form: MappingsForm;
  index: number;
  view: MappingView;
  server: string;
};

function RowFields({ form, index, view, server, folders }: Row & { folders: string }) {
  return (
    <>
      <form.Field name={`mappings[${index}].shown`}>
        {(field) => (
          <div>
            <Switch
              label={`Shown on ${server}`}
              checked={field.state.value}
              onCheckedChange={field.handleChange}
            />
          </div>
        )}
      </form.Field>
      <form.Subscribe selector={(state) => state.values.mappings[index]}>
        {(value) =>
          value?.shown === true && (
            <form.AppField name={`mappings[${index}].serverFolder`}>
              {(field) => (
                <field.TextField
                  label={`Folder on ${server}`}
                  data
                  spellCheck={false}
                  autoComplete="off"
                  list={folders}
                  description={folderNoteOf(view, value.serverFolder, server) || undefined}
                />
              )}
            </form.AppField>
          )
        }
      </form.Subscribe>
    </>
  );
}

/**
 * One library root on the server: where it lives, whether the server shows
 * it, and the folder the server sees it at. The guess and what to do describe
 * the folder SlipStream filled in, so an edited folder drops them.
 */
export function MappingRow({ folders, ...row }: Row & { folders: string }) {
  const { form, index, view } = row;
  const titleId = useId();
  return (
    <fieldset
      aria-labelledby={titleId}
      className="flex flex-col gap-3 rounded-card bg-glass px-4 py-3 ring-1 ring-hairline ring-inset"
    >
      <form.Subscribe selector={(state) => state.values.mappings[index]}>
        {(value) => {
          const prefilled =
            value !== undefined && value.serverFolder.trim() === (view.serverFolder ?? "");
          return (
            <>
              <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                <h4 id={titleId} className="font-semibold text-ink-1">
                  {view.rootName}
                </h4>
                <span className="text-meta text-ink-3">On {view.rootNode.name}</span>
                {value?.shown === true && view.guess && prefilled && <Chip>Guess</Chip>}
              </div>
              {view.words !== undefined && prefilled && <WarnLine text={view.words} />}
            </>
          );
        }}
      </form.Subscribe>
      <RowFields {...row} folders={folders} />
    </fieldset>
  );
}
