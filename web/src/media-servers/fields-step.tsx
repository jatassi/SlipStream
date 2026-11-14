import { useQueryClient } from "@tanstack/react-query";

import type {
  MediaserversKindView as Kind,
  MediaserversServerView as Server,
  NodesNode,
} from "../api/gen/model";
import { addMediaServer } from "../api/gen/query/mediaservers/mediaservers";
import { adoptMediaServer } from "../live/adopt-media-servers";
import { Button } from "../ui/button";
import { toast } from "../ui/toast";
import { AdminSignIn } from "./admin-sign-in";
import { ServerFields } from "./server-fields";
import { draftOf, newValues, reachingOf } from "./server-form";
import { useServerForm } from "./use-server-form";

/** A Jellyfin or Emby server's fields. Save tests it from its Node and from the Controller. */
export function FieldsStep({
  kind,
  nodes,
  onBack,
  onSaved,
}: {
  kind: Kind;
  nodes: readonly NodesNode[];
  onBack: () => void;
  onSaved: (server: Server) => void;
}) {
  const queryClient = useQueryClient();
  const reaching = reachingOf(nodes);
  const sheet = useServerForm(newValues(kind, reaching), async (value) => {
    const made = await addMediaServer(draftOf(value, kind));
    adoptMediaServer(queryClient, made);
    toast(`Added ${made.name}`);
    onSaved(made);
    return null;
  });
  const { form } = sheet;
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
        <ServerFields
          sheet={sheet}
          kind={kind}
          nodes={nodes}
          reaching={reaching}
          keySaved={false}
          signIn={<AdminSignIn form={form} kind={kind} />}
        />
        <form.FormAlert />
        <div className="flex flex-wrap justify-end gap-2">
          <Button onClick={onBack}>Back</Button>
          <form.SubmitButton label="Save" />
        </div>
      </form.AppForm>
    </form>
  );
}
