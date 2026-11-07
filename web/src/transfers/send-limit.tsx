import { useQueryClient } from "@tanstack/react-query";

import {
  setTransferSettings,
  useGetTransferSettingsSuspense,
} from "../api/gen/query/transfers/transfers";
import { adoptTransferSettings } from "../live/adopt";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";
import { toast } from "../ui/toast";

/** A Node's send limit, edited as the torrent client's upload limit is and applied live. */
export function SendLimit({ nodeId }: { nodeId: string }) {
  const { sendLimit } = useGetTransferSettingsSuspense(nodeId).data;
  const queryClient = useQueryClient();
  return (
    <Section title="Settings">
      <SettingGroup>
        <SettingField
          field={sendLimit}
          onSave={async (value) => {
            adoptTransferSettings(
              queryClient,
              nodeId,
              await setTransferSettings(nodeId, { sendLimit: value }),
            );
            toast("Transfer settings saved");
          }}
        />
      </SettingGroup>
    </Section>
  );
}
