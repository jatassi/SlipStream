import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { sentenceOf } from "../../api/errors";
import type { MediaserversServerView as Server } from "../../api/gen/model";
import { updateMediaServer } from "../../api/gen/query/mediaservers/mediaservers";
import { adoptMediaServer } from "../../live/adopt-media-servers";
import { SettingGroup } from "../../ui/setting-field";
import { Switch } from "../../ui/switch";
import { toast } from "../../ui/toast";

/** Whether the Conversions on the Node the server runs on pause while it transcodes video, saved at once; shown only when it runs on a Node. */
export function PauseConversions({ server }: { server: Server }) {
  const queryClient = useQueryClient();
  const [busy, setBusy] = useState(false);
  if (server.runsOnNode === undefined) {
    return null;
  }
  const node = server.runsOnNode.name;
  return (
    <div className="mt-6">
      <SettingGroup>
        <div className="flex flex-col gap-1">
          <Switch
            label="Pause Conversions while this server transcodes"
            checked={server.pausesConversions}
            disabled={busy}
            onCheckedChange={(pausesConversions) => {
              setBusy(true);
              void updateMediaServer(server.id, { pausesConversions })
                .then((got) => {
                  adoptMediaServer(queryClient, got);
                  toast(`Saved ${got.name}`);
                  return got;
                })
                .catch((error: unknown) => {
                  toast(`${server.name} wasn't saved. ${sentenceOf(error)}`, { kind: "fail" });
                })
                .finally(() => {
                  setBusy(false);
                });
            }}
          />
          <p className="text-meta text-ink-3">
            Conversions on {node} pause while {server.name} re-encodes video, and start again when
            it stops.
          </p>
        </div>
      </SettingGroup>
    </div>
  );
}
