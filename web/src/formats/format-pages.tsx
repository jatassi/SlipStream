import { useNavigate } from "@tanstack/react-router";

import type { FormatsCustomFormat, FormatsSchema } from "../api/gen/model";
import { createCustomFormat, updateCustomFormat } from "../api/gen/query/formats/formats";
import { useImpactSheet } from "../targets/impact-sheet";
import { toast } from "../ui/toast";
import { bodyOf, draftOf, newDraft } from "./draft";
import { FormatEditor } from "./format-editor";

const toSection = { to: "/settings/$section", params: { section: "custom-formats" } } as const;

export function NewFormatEditor({ schema }: { schema: FormatsSchema }) {
  const navigate = useNavigate();
  return (
    <FormatEditor
      initial={newDraft(schema)}
      schema={schema}
      label="Create Custom Format"
      onSave={async (draft) => {
        const made = await createCustomFormat(bodyOf(draft));
        toast(`Created ${made.name}.`);
        await navigate(toSection);
      }}
    />
  );
}

export function SavedFormatEditor({
  format,
  schema,
  matchLimit,
}: {
  format: FormatsCustomFormat;
  schema: FormatsSchema;
  matchLimit: number;
}) {
  const navigate = useNavigate();
  const impact = useImpactSheet();
  return (
    <>
      <FormatEditor
        initial={draftOf(format, schema)}
        schema={schema}
        slow={{ releases: format.slowReleases, limit: matchLimit }}
        label="Save"
        onSave={(draft) => {
          const body = bodyOf(draft);
          return impact.save({
            title: `Save ${format.name}`,
            action: "Save",
            run: (changing) => updateCustomFormat(format.id, body, changing),
            done: async () => {
              toast(`Saved ${body.name}.`);
              await navigate(toSection);
            },
          });
        }}
      />
      {impact.sheet}
    </>
  );
}
