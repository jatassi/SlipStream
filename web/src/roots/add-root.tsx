import { useState } from "react";

import type { RootsRoot } from "../api/gen/model";
import { createRoot } from "../api/gen/query/roots/roots";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { Button } from "../ui/button";
import { toast } from "../ui/toast";
import { PlaceActions, PlaceForm, useSaving } from "./place-form";
import { placeRefusal } from "./refused";
import { DefaultChoice, TypeSelect } from "./type-fields";
import { usePlace } from "./use-place";

export function AddRootForm({
  contentType,
  onSaved,
  onCancel,
}: {
  contentType: string;
  onSaved: (root: RootsRoot) => void;
  onCancel: () => void;
}) {
  const { contentTypes } = useListContentTypesSuspense().data;
  const [type, setType] = useState(contentType);
  const [makeDefault, setMakeDefault] = useState(false);
  const editing = usePlace({ nodeId: "", path: "" });
  const adding = useSaving(
    async () => {
      const made = await createRoot({ contentType: type, ...editing.place, makeDefault });
      const label = contentTypes.find((candidate) => candidate.id === type)?.work.many ?? type;
      toast(made.default ? `Added ${made.path}. New ${label} go there.` : `Added ${made.path}.`);
      onSaved(made);
    },
    (error) => placeRefusal(error, editing.namer),
  );
  return (
    <PlaceForm
      editing={editing}
      onSubmit={() => void adding.run()}
      lead={<TypeSelect type={type} onTypeChange={setType} />}
    >
      <DefaultChoice type={type} makeDefault={makeDefault} onMakeDefaultChange={setMakeDefault} />
      <PlaceActions busy={adding.busy} failure={adding.failure} onCancel={onCancel}>
        <Button type="submit" look="primary" busy={adding.busy} disabled={!editing.usable}>
          Add
        </Button>
      </PlaceActions>
    </PlaceForm>
  );
}
