import type { NodesNode } from "../../../api/gen/model";
import { useGetNodeConfigSuspense } from "../../../api/gen/query/system/system";
import { askRestart } from "../../../shell/restart/restart-state";
import { WarningLine } from "../../../ui/warning-line";
import { settingOf } from "./-config";

// A Docker container publishes only the ports its mapping names, so a new
// port needs the mapping changed before the restart.
export function PendingPort({ node }: { node: NodesNode }) {
  const port = settingOf(useGetNodeConfigSuspense(node.id).data, "http.listen", "port");
  if (!port.pendingRestart) {
    return null;
  }
  return (
    <WarningLine
      signal="warn"
      chip="Restart needed"
      subject={`Restart to use port ${port.value}`}
      detail={
        node.link.build?.installKind === "docker"
          ? `Change the container's -p mapping to publish ${port.value} too, then restart.`
          : `SlipStream listens on ${port.running} until it restarts.`
      }
      action={{
        label: "Restart now",
        onClick: () => {
          askRestart(node);
        },
      }}
    />
  );
}
