import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import type { IndexersIndexer } from "../api/gen/model";
import { updateIndexer } from "../api/gen/query/indexers/indexers";
import { adoptIndexer } from "../live/adopt";
import { useAppForm } from "../ui/form";
import { toast } from "../ui/toast";
import { needsCode } from "./captcha-of";
import type { SettingsForm } from "./definition-form";
import { editOf, editSchema, refusal, valuesOf } from "./edit-values";

/**
 * The edit form. A save sends only what changed, and the server tests the
 * Indexer first when the change is one its Test reads. A Test that meets a
 * sign-in needing a two-factor code opens the code step, and the next save
 * sends the code with the change.
 */
export function useEditForm(
  indexer: IndexersIndexer,
  settings: SettingsForm | undefined,
  onDone: () => void,
) {
  const queryClient = useQueryClient();
  const [codeShown, setCodeShown] = useState(false);
  const form = useAppForm({
    defaultValues: valuesOf(indexer, settings),
    validators: { onSubmit: editSchema },
    onSubmit: async ({ value, formApi }) => {
      const edit = editOf(indexer, value, settings, codeShown);
      if (Object.keys(edit).length > 0) {
        try {
          adoptIndexer(queryClient, await updateIndexer(indexer.id, edit));
        } catch (error) {
          formApi.setFieldValue("code", "");
          setCodeShown(codeShown || needsCode(error));
          formApi.setErrorMap({ onSubmit: refusal(error, settings?.form.fields) });
          return;
        }
        toast(`Saved ${edit.name ?? indexer.name}`);
      }
      onDone();
    },
  });
  return { form, codeShown };
}

export type EditForm = ReturnType<typeof useEditForm>["form"];
