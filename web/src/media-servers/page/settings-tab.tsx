import { useQueryClient } from "@tanstack/react-query";

import type { MediaserversServerView as Server } from "../../api/gen/model";
import {
  updateMediaServer,
  useListMediaServerKindsSuspense,
} from "../../api/gen/query/mediaservers/mediaservers";
import { useListNodesSuspense } from "../../api/gen/query/nodes/nodes";
import { adoptMediaServer } from "../../live/adopt-media-servers";
import { toast } from "../../ui/toast";
import { ServerFields } from "../server-fields";
import { editOf, kindOf, reachingOf, valuesOf } from "../server-form";
import { useServerForm } from "../use-server-form";

/** The server's own settings, as the Add sheet took them. Save tests the routes a change reaches. */
export function SettingsTab({ server }: { server: Server }) {
  const queryClient = useQueryClient();
  const kind = kindOf(useListMediaServerKindsSuspense().data.kinds, server.kind);
  const { nodes } = useListNodesSuspense().data;
  const sheet = useServerForm(valuesOf(server), async (value) => {
    const edit = editOf(server, value, kind);
    if (Object.keys(edit).length === 0) {
      return value;
    }
    const got = await updateMediaServer(server.id, edit);
    adoptMediaServer(queryClient, got);
    toast(`Saved ${got.name}`);
    return valuesOf(got);
  });
  const { form } = sheet;
  return (
    <form
      noValidate
      aria-label={`Settings of ${server.name}`}
      className="flex max-w-140 flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <form.AppForm>
        <ServerFields
          sheet={sheet}
          kind={kind}
          nodes={nodes}
          reaching={reachingOf(nodes)}
          keySaved
        />
        <form.FormAlert />
        <div className="flex justify-end">
          <form.SubmitButton label="Save" />
        </div>
      </form.AppForm>
    </form>
  );
}
