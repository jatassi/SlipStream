import type { EditForm } from "../edit-form";

/** How many background searches go to the Indexer at a time, 1 to 10. */
export function SearchesAtOnceField({ form }: { form: EditForm }) {
  return (
    <form.AppField name="searchesAtOnce">
      {(field) => (
        <field.TextField
          label="Searches at once"
          inputMode="numeric"
          description="Backlog search and Search all send at most this many searches to this Indexer at a time. Your own searches get one more."
        />
      )}
    </form.AppField>
  );
}
