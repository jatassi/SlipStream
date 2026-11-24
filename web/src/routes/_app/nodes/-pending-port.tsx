import type { NodesNode, SettingListenAt } from "../../../api/gen/model";
import { useGetNodeConfigSuspense } from "../../../api/gen/query/system/system";
import { askRestart } from "../../../shell/restart/restart-state";
import { settingOf } from "../../../nodes/node-setting";
import { Callout } from "../../../ui/callout";

const where = (at: SettingListenAt): string =>
  at.host === "all" ? "all networks" : at.host === "local" ? "this machine only" : at.address;

// A Docker container publishes only the ports its mapping names, so a new
// port needs the mapping changed before the restart.
export function PendingPort({ node }: { node: NodesNode }) {
  const listen = settingOf(useGetNodeConfigSuspense(node.id).data, "http.listen", "listen");
  if (!listen.pendingRestart) {
    return null;
  }
  const { value, running } = listen;
  const moves = value.port !== running.port;
  return (
    <Callout
      tone="warn"
      chip="Restart needed"
      subject={
        moves ? `Restart to use port ${String(value.port)}` : `Restart to listen on ${where(value)}`
      }
      action={{
        label: "Restart now",
        onClick: () => {
          askRestart(node);
        },
      }}
    >
      {moves && node.link.build?.installKind === "docker"
        ? `Change the container's -p mapping to publish ${String(value.port)} too, then restart.`
        : `SlipStream listens on ${moves ? String(running.port) : where(running)} until it restarts.`}
    </Callout>
  );
}
