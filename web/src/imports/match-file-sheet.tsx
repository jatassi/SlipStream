import { Suspense, useState } from "react";

import type { ImportsImportEntry, ImportsImportFile, WorksContentType } from "../api/gen/model";
import { actOnImportFile } from "../api/gen/query/imports/imports";
import { useListCutsSuspense } from "../api/gen/query/targets/targets";
import { getWork, useGetWorkSuspense, useListLibrarySuspense } from "../api/gen/query/works/works";
import { Button } from "../ui/button";
import { Checkbox } from "../ui/checkbox";
import { Dialog } from "../ui/dialog";
import { submitErrors, useAppForm } from "../ui/form";
import { useFormContext } from "../ui/form-context";
// oxlint-disable-next-line import/max-dependencies -- the sheet edits a file's Work, Items, Edition and release name
import { Select } from "../ui/select";
import { Skeleton } from "../ui/skeleton";
import { codeOf } from "../works/fields-by-tier";
import { nameOf } from "../works/paths";

/** The Edition choice that leaves the file's cut to what its name reads. */
const asNamed = "";

type Match = { readonly workId: string; readonly itemIds: string[]; readonly edition: string };

const releaseOf = (entry: ImportsImportEntry, file: ImportsImportFile) =>
  entry.grab?.release ?? nameOf(file.path).replace(/\.[^.]+$/u, "");

function ItemPicks({
  type,
  workId,
  chosen,
  onChosen,
}: {
  type: WorksContentType;
  workId: string;
  chosen: readonly string[];
  onChosen: (chosen: string[]) => void;
}) {
  const { items } = useGetWorkSuspense(workId).data;
  return (
    <fieldset className="flex flex-col">
      <legend className="mb-1 font-semibold text-ink-2">{type.item.many}</legend>
      {items.map((item) => (
        <Checkbox
          key={item.id}
          label={`${codeOf(type, item)} ${item.label}`}
          checked={chosen.includes(item.id)}
          onCheckedChange={(checked) => {
            onChosen(
              items
                .filter(({ id }) => (id === item.id ? checked : chosen.includes(id)))
                .map(({ id }) => id),
            );
          }}
        />
      ))}
    </fieldset>
  );
}

function MatchFields({
  type,
  match,
  onMatch,
}: {
  type: WorksContentType;
  match: Match;
  onMatch: (match: Match) => void;
}) {
  const { rows } = useListLibrarySuspense({ contentType: type.id }).data;
  const { cuts } = useListCutsSuspense().data;
  const items = (itemIds: string[]) => {
    onMatch({ ...match, itemIds });
  };
  return (
    <>
      <Select
        label={type.work.one}
        options={rows.map((row) => ({ value: row.id, label: row.label }))}
        value={match.workId}
        onValueChange={(workId) => {
          onMatch({ ...match, workId, itemIds: [] });
        }}
      />
      <Suspense fallback={<Skeleton label={`Loading the ${type.item.many.toLowerCase()}`} />}>
        {type.items !== "one" && (
          <ItemPicks type={type} workId={match.workId} chosen={match.itemIds} onChosen={items} />
        )}
      </Suspense>
      <Select
        label="Edition"
        options={[
          { value: asNamed, label: "As its name reads" },
          ...cuts.map((cut) => ({ value: cut.key, label: cut.name })),
        ]}
        value={match.edition}
        onValueChange={(edition) => {
          onMatch({ ...match, edition });
        }}
      />
    </>
  );
}

// A one-Item type's file holds its Work's one Item, so the sheet asks for none.
const itemsOf = async (type: WorksContentType, { workId, itemIds }: Match) =>
  type.items === "one" ? (await getWork(workId)).items.map(({ id }) => id) : itemIds;

function useMatchForm(
  entry: ImportsImportEntry,
  file: ImportsImportFile,
  type: WorksContentType,
  match: Match,
  onDone: () => void,
) {
  return useAppForm({
    defaultValues: { releaseName: releaseOf(entry, file) },
    onSubmit: async ({ value, formApi }) => {
      try {
        await actOnImportFile(entry.id, file.position, {
          action: "match",
          workId: match.workId,
          itemIds: await itemsOf(type, match),
          edition: match.edition === asNamed ? null : match.edition,
          releaseName: value.releaseName.trim(),
        });
        onDone();
      } catch (error) {
        formApi.setErrorMap({ onSubmit: submitErrors(error) });
      }
    },
  });
}

function SaveMatch({ disabled }: { disabled: boolean }) {
  const form = useFormContext();
  return (
    <form.Subscribe selector={(state) => state.isSubmitting}>
      {(submitting) => (
        <Button type="submit" look="primary" busy={submitting} disabled={disabled}>
          Save the match
        </Button>
      )}
    </form.Subscribe>
  );
}

function MatchForm({
  entry,
  file,
  type,
  onDone,
}: {
  entry: ImportsImportEntry;
  file: ImportsImportFile;
  type: WorksContentType;
  onDone: () => void;
}) {
  const [match, setMatch] = useState<Match>({
    workId: entry.workId ?? "",
    itemIds: file.itemIds,
    edition: asNamed,
  });
  const form = useMatchForm(entry, file, type, match, onDone);
  return (
    <form
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <form.AppForm>
        <MatchFields type={type} match={match} onMatch={setMatch} />
        <form.AppField name="releaseName">
          {(field) => (
            <field.TextField
              label="Release name"
              data
              description="SlipStream reads the group, the revision and the source from it."
            />
          )}
        </form.AppField>
        <form.FormAlert />
        <div className="flex justify-end">
          <SaveMatch disabled={type.items !== "one" && match.itemIds.length === 0} />
        </div>
      </form.AppForm>
    </form>
  );
}

/**
 * Changes what a download's or a folder's file is: its Work, the Items it
 * holds (several, in order, for a multi-episode file), its Edition and the
 * release name SlipStream reads its group, revision and source from.
 */
export function MatchFileSheet({
  entry,
  file,
  type,
  onClose,
}: {
  entry: ImportsImportEntry;
  file: ImportsImportFile;
  type: WorksContentType;
  onClose: () => void;
}) {
  return (
    <Dialog
      title={`Change the match for ${nameOf(file.path)}`}
      size={680}
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      <MatchForm entry={entry} file={file} type={type} onDone={onClose} />
    </Dialog>
  );
}
