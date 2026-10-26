import { useQueryClient } from "@tanstack/react-query";

import type {
  NodesNode,
  SettingField as Field,
  SystemConfigSectionId,
  SystemNodeConfig,
} from "../api/gen/model";
import { useGetNodeConfigSuspense, useUpdateNodeConfig } from "../api/gen/query/system/system";
import { adoptNodeConfig } from "../live/adopt";
import { askRestart } from "../shell/restart/restart-state";
import { SettingField } from "../ui/setting-field";
import { toast } from "../ui/toast";
import { WarningLine } from "../ui/warning-line";

export function fieldsOf(config: SystemNodeConfig, id: SystemConfigSectionId): readonly Field[] {
  const section = config.sections.find((candidate) => candidate.id === id);
  if (section === undefined) {
    throw new Error(`getNodeConfig has no ${id} section`);
  }
  return section.fields;
}

/** The Node's setting at `key`, which must be of `kind`. */
export function settingOf<K extends Field["kind"]>(
  config: SystemNodeConfig,
  key: string,
  kind: K,
): Extract<Field, { kind: K }> {
  const field = config.sections
    .flatMap((section) => section.fields)
    .find((one): one is Extract<Field, { kind: K }> => one.key === key && one.kind === kind);
  if (field === undefined) {
    throw new Error(`getNodeConfig has no ${kind} setting ${key}`);
  }
  return field;
}

/** One of a Node's own settings, saved on that Node. */
export function NodeSettingField({
  nodeId,
  field,
  saved,
}: {
  nodeId: string;
  field: Field;
  saved: string;
}) {
  const queryClient = useQueryClient();
  const update = useUpdateNodeConfig();
  return (
    <SettingField
      field={field}
      onSave={async (value) => {
        const config = await update.mutateAsync({ id: nodeId, data: { [field.key]: value } });
        adoptNodeConfig(queryClient, nodeId, config);
        toast(saved);
      }}
    />
  );
}

/** The torrent settings a Node saved but reads only at its next start, with Restart now. */
export function PendingRestart({ node }: { node: NodesNode }) {
  const pending = fieldsOf(useGetNodeConfigSuspense(node.id).data, "torrent").filter(
    (field) => field.pendingRestart,
  );
  if (pending.length === 0) {
    return null;
  }
  return (
    <WarningLine
      signal="warn"
      chip="Restart needed"
      subject={`Restart ${node.name} to use the new ${pending.map((field) => field.label.toLowerCase()).join(" and ")}`}
      detail={pending
        .map((field) =>
          field.kind === "address"
            ? `It listens on ${field.running} until then.`
            : "Torrents already added stay where they are.",
        )
        .join(" ")}
      action={{
        label: "Restart now",
        onClick: () => {
          askRestart(node);
        },
      }}
    />
  );
}
