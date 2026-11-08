import type { DownloadclientsDaemon as Daemon } from "../api/gen/model";
import { Select } from "../ui/select";
import { daemonKey } from "./client-form";
import { Refused } from "./refused";
import type { ClientForm } from "./use-client-sheet";

export const onlyDaemon = "";

/** Deluge's daemon: its only one, the saved one, or one a passing Test listed. */
export function DaemonField({ form, daemons }: { form: ClientForm; daemons: readonly Daemon[] }) {
  return (
    <form.Field name="daemon">
      {(field) => {
        const keys = new Set([...daemons.map((daemon) => daemonKey(daemon)), field.state.value]);
        keys.delete(onlyDaemon);
        return (
          <div className="flex flex-col gap-1.5">
            <Select
              label="Daemon"
              options={[
                { value: onlyDaemon, label: "The one its web UI connects to" },
                ...[...keys].map((key) => ({ value: key, label: key })),
              ]}
              value={field.state.value}
              onValueChange={field.handleChange}
            />
            <Refused errors={field.state.meta.errors} />
          </div>
        );
      }}
    </form.Field>
  );
}
