import { useQueryClient } from "@tanstack/react-query";

import type {
  SettingField as Field,
  SystemConfigSectionId,
  SystemNodeConfig,
} from "../../../api/gen/model";
import {
  useGetNodeConfigSuspense,
  useUpdateNodeConfig,
} from "../../../api/gen/query/system/system";
import { adoptNodeConfig } from "../../../live/adopt";
import { SettingField, SettingGroup } from "../../../ui/setting-field";
import { toast } from "../../../ui/toast";

const savedToast: { readonly [S in SystemConfigSectionId]: string } = {
  web: "Web and API settings saved",
  log: "Log settings saved",
  torrent: "Torrent client settings saved",
};

function fieldsOf(config: SystemNodeConfig, id: SystemConfigSectionId): readonly Field[] {
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

/** One section of a Node's settings, each field saved on its own. */
export function NodeSettings({
  nodeId,
  section,
}: {
  nodeId: string;
  section: SystemConfigSectionId;
}) {
  const config = useGetNodeConfigSuspense(nodeId).data;
  const queryClient = useQueryClient();
  const update = useUpdateNodeConfig();
  return (
    <SettingGroup>
      {fieldsOf(config, section).map((field) => (
        <SettingField
          key={field.key}
          field={field}
          onSave={async (value) => {
            const saved = await update.mutateAsync({ id: nodeId, data: { [field.key]: value } });
            adoptNodeConfig(queryClient, nodeId, saved);
            toast(savedToast[section]);
          }}
        />
      ))}
    </SettingGroup>
  );
}
