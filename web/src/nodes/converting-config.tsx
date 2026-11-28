import type { NodesNode } from "../api/gen/model";
import { useGetNodeConfigSuspense } from "../api/gen/query/system/system";
import { Details } from "../ui/details";
import { SettingGroup } from "../ui/setting-field";
import { NodeSettingField, settingOf } from "./node-setting";

const saved = "Converting settings saved";

/** The scratch folder and, under Details, the ffmpeg path: both kept in the Node's config.yaml. */
export function ConfigFields({ node, free }: { node: NodesNode; free: string | undefined }) {
  const config = useGetNodeConfigSuspense(node.id).data;
  return (
    <>
      <SettingGroup>
        <div className="flex flex-col gap-1">
          <NodeSettingField
            nodeId={node.id}
            field={settingOf(config, "converting.scratch", "folder")}
            saved={saved}
          />
          {free !== undefined && <p className="text-meta text-ink-3">{free}</p>}
        </div>
      </SettingGroup>
      <Details>
        <div className="mt-2 flex flex-col gap-2">
          <p>
            {node.name} reruns its test runs when this changes. SlipStream supports only the ffmpeg
            it comes with.
          </p>
          <SettingGroup>
            <NodeSettingField
              nodeId={node.id}
              field={settingOf(config, "converting.ffmpeg", "file")}
              saved={saved}
            />
          </SettingGroup>
        </div>
      </Details>
    </>
  );
}
