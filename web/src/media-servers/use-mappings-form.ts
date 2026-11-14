import { useQueryClient } from "@tanstack/react-query";

import type {
  MediaserversLibraries as Libraries,
  MediaserversServerView as Server,
} from "../api/gen/model";
import { saveMediaServerMappings } from "../api/gen/query/mediaservers/mediaservers";
import { adoptMappings } from "../live/adopt-media-servers";
import { useAppForm } from "../ui/form";
import { toast } from "../ui/toast";
import { mappingValuesOf, mappingsBodyOf } from "./mappings-form";
import { refusalOf } from "./server-form";

/** Saves every mapping through the server's check. A refused folder shows on its row. */
export function useMappingsForm(server: Server, libraries: Libraries, onSaved?: () => void) {
  const queryClient = useQueryClient();
  const form = useAppForm({
    defaultValues: mappingValuesOf(libraries),
    onSubmit: async ({ value }) => {
      try {
        const { mappings } = await saveMediaServerMappings(server.id, {
          mappings: mappingsBodyOf(value),
        });
        adoptMappings(queryClient, server.id, mappings);
      } catch (error) {
        form.setErrorMap({ onSubmit: refusalOf(error) });
        return;
      }
      toast(`Saved the Path mappings of ${server.name}`);
      onSaved?.();
    },
  });
  return form;
}

export type MappingsForm = ReturnType<typeof useMappingsForm>;
