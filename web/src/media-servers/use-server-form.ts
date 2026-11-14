import { useState } from "react";

import { useAppForm } from "../ui/form";
import { type ServerValues, refusalOf, underDetails } from "./server-form";

/**
 * The form behind the Add sheet and a server's Settings tab. A refused save
 * marks the fields it names, opening Details when one sits there.
 */
export function useServerForm(
  defaultValues: ServerValues,
  /** Answers the values the form shows once saved, or null when the form closes. */
  save: (value: ServerValues) => Promise<ServerValues | null>,
) {
  const [detailsOpen, setDetailsOpen] = useState(false);
  const form = useAppForm({
    defaultValues,
    onSubmit: async ({ value }) => {
      try {
        const saved = await save(value);
        if (saved !== null) {
          form.reset(saved);
        }
      } catch (error) {
        const errors = refusalOf(error);
        form.setErrorMap({ onSubmit: errors });
        if (Object.keys(errors.fields).some((field) => underDetails.has(field))) {
          setDetailsOpen(true);
        }
      }
    },
  });
  return { form, detailsOpen, setDetailsOpen };
}

export type ServerFormState = ReturnType<typeof useServerForm>;
export type ServerForm = ServerFormState["form"];
