import { useId } from "react";

import type {
  MediaserversLibraries as Libraries,
  MediaserversServerView as Server,
} from "../api/gen/model";
import { Empty } from "../shell/empty";
import { Button } from "../ui/button";
import { MappingRow } from "./mapping-row";
import { foldersOf } from "./mappings-form";
import { useMappingsForm } from "./use-mappings-form";

type Props = {
  server: Server;
  libraries: Libraries;
  onSaved?: () => void;
  /** Closes without saving, as the Add sheet's last step offers. */
  onLater?: () => void;
};

function Editor({ server, libraries, onSaved, onLater }: Props) {
  const form = useMappingsForm(server, libraries, onSaved);
  const folders = useId();
  return (
    <form
      noValidate
      aria-label={`Path mappings on ${server.name}`}
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <form.AppForm>
        <datalist id={folders} aria-label={`Library folders on ${server.name}`}>
          {foldersOf(libraries).map((folder) => (
            <option key={folder} value={folder}>
              {folder}
            </option>
          ))}
        </datalist>
        <p className="text-pretty text-ink-1">
          {`Give each library root the folder ${server.name} sees it at. ${server.name} gets changes only for the roots it shows.`}
        </p>
        {libraries.mappings.map((view, index) => (
          <MappingRow
            key={view.rootId}
            form={form}
            index={index}
            view={view}
            server={server.name}
            folders={folders}
          />
        ))}
        <form.FormAlert />
        <div className="flex flex-wrap justify-end gap-2">
          {onLater !== undefined && <Button onClick={onLater}>Later</Button>}
          <form.SubmitButton label="Save" />
        </div>
      </form.AppForm>
    </form>
  );
}

/**
 * Each library root's Path mapping on one server: the folder the server sees
 * it at, picked from the server's library folders or typed, or the root
 * marked as not shown there. Save checks every folder against the server's
 * library folders.
 */
export function MappingsEditor(props: Props) {
  return props.libraries.mappings.length === 0 ? (
    <Empty>
      No library roots yet. Once a Node holds Storage and has a root, its Path mapping shows here.
    </Empty>
  ) : (
    <Editor {...props} />
  );
}
