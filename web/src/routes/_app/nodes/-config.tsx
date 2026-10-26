import type { SystemConfigSectionId } from "../../../api/gen/model";
import { useGetNodeConfigSuspense } from "../../../api/gen/query/system/system";
import { NodeSettingField, fieldsOf } from "../../../nodes/node-setting";
import { SettingGroup } from "../../../ui/setting-field";

const savedToast: { readonly [S in SystemConfigSectionId]: string } = {
  web: "Web and API settings saved",
  log: "Log settings saved",
  torrent: "Torrent client settings saved",
};

/** One section of a Node's settings, each field saved on its own. */
export function NodeSettings({
  nodeId,
  section,
}: {
  nodeId: string;
  section: SystemConfigSectionId;
}) {
  const config = useGetNodeConfigSuspense(nodeId).data;
  return (
    <SettingGroup>
      {fieldsOf(config, section).map((field) => (
        <NodeSettingField
          key={field.key}
          nodeId={nodeId}
          field={field}
          saved={savedToast[section]}
        />
      ))}
    </SettingGroup>
  );
}
