import { useState } from "react";

import { DialogActions } from "../../account/dialog-actions";
import type { SettingField, WorksContentType, WorksTree } from "../../api/gen/model";
import { useListRootsSuspense, useListVolumesSuspense } from "../../api/gen/query/roots/roots";
import { useListProfilesSuspense } from "../../api/gen/query/targets/targets";
import { Button } from "../../ui/button";
import { DialogClose } from "../../ui/dialog";
import { useAdding } from "./adding";
import { ChoiceFields, note } from "./choice-fields";
import { type AddChoices, addDefaultsOf, movedDefaults, startChoices } from "./choices";
// oxlint-disable-next-line import/max-dependencies -- the form reads the Profiles and the roots with their free space, and says what Add will do
import { Preview } from "./preview";
import { addSentence } from "./sentence";

function AddButton({
  type,
  work,
  choices,
}: {
  type: WorksContentType;
  work: WorksTree;
  choices: AddChoices;
}) {
  const addWork = useAdding();
  return (
    <Button
      look="primary"
      busy={addWork.isPending}
      onClick={() => {
        addWork.mutate({
          data: {
            contentType: type.id,
            key: work.key,
            profileId: choices.profileId,
            ...(choices.rootId !== undefined && { rootId: choices.rootId }),
            monitor: choices.monitor,
            remember: choices.remember,
          },
        });
      }}
    >
      Add {type.work.one}
    </Button>
  );
}

export function AddForm({
  type,
  work,
  settings,
}: {
  type: WorksContentType;
  work: WorksTree;
  settings: readonly SettingField[];
}) {
  const profiles = useListProfilesSuspense().data.profiles.filter(
    (profile) => profile.contentType === type.id,
  );
  const roots = useListRootsSuspense().data.roots.filter((root) => root.contentType === type.id);
  const volumes = useListVolumesSuspense().data.nodes;
  const defaults = addDefaultsOf({ type, profiles, roots, settings });
  const [choices, setChoices] = useState(() => startChoices(defaults));
  return (
    <div className="flex flex-col gap-4">
      <Preview type={type} work={work} />
      <p className="text-body text-ink-1">
        {addSentence(work, choices, { type, profiles, roots })}
      </p>
      <ChoiceFields
        type={type}
        profiles={profiles}
        roots={roots}
        volumes={volumes}
        choices={choices}
        moves={movedDefaults(choices, defaults).length > 0}
        choose={(next) => {
          setChoices((current) => ({ ...current, ...next }));
        }}
      />
      <p className={note}>
        Adding doesn&apos;t search. To search now, use Search on its Work page.
      </p>
      <DialogActions>
        <DialogClose>
          <Button>Cancel</Button>
        </DialogClose>
        <AddButton type={type} work={work} choices={choices} />
      </DialogActions>
    </div>
  );
}
