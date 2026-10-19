import { useSelector } from "@tanstack/react-form";

import { useListRootsSuspense } from "../api/gen/query/roots/roots";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { purposeCheck } from "../roots/check";
import { useNode } from "../routes/_app/nodes/-node";
import { useAppForm } from "../ui/form";
import {
  type ControlProps,
  type Draft,
  type Kind,
  SaveButton,
  defaultHint,
  onSubmit,
} from "../ui/setting-kind";
import { FolderField } from "./folder-field";

function parseFolder(path: string): Draft<"folder"> {
  const folder = path.trim();
  return folder === "" ? { refused: "Choose a folder." } : { value: folder };
}

/** A path on the setting's Node, typed or browsed there, with that Node's verdict for its purpose. */
function FolderControl({ field, busy, disabled, submit, onEdit }: ControlProps<"folder">) {
  const node = useNode(field.nodeId);
  const { roots } = useListRootsSuspense().data;
  const { contentTypes } = useListContentTypesSuspense().data;
  const check = purposeCheck(field.purpose, { node: node.name, roots, types: contentTypes });
  const form = useAppForm({ defaultValues: { path: field.value } });
  const path = useSelector(form.store, (state) => state.values.path);
  return (
    <form
      onSubmit={onSubmit(() => {
        submit(parseFolder(path));
      })}
      className="flex flex-col items-start gap-2"
    >
      <div className="self-stretch">
        <form.AppForm>
          <form.AppField name="path" listeners={{ onChange: onEdit }}>
            {() => (
              <FolderField
                label={field.label}
                nodeId={node.id}
                nodeName={node.name}
                check={check}
                confirmVerb="Use"
                allowNewFolder
                disabled={disabled}
              />
            )}
          </form.AppField>
        </form.AppForm>
      </div>
      <SaveButton shown={path.trim() !== field.value} busy={busy} />
    </form>
  );
}

export const folderKind: Kind<"folder"> = {
  words: (field) => field.value,
  hints: (field) => [defaultHint(field, field.default)],
  Control: FolderControl,
};
