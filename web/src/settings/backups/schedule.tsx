import { Link } from "@tanstack/react-router";

import type { NodesNode, SettingField as Field } from "../../api/gen/model";
import { bytes } from "../../format";
import { Details } from "../../ui/details";
import { Section } from "../../ui/section";
import { CheckSavedCopy } from "./passphrase-check";
import { RestoreFileButton } from "./restore-file";
import { RetireButton } from "./retire";
import { BackupSettings, backupSettingKeys, detailKeys } from "./settings";

export function Schedule({
  settings,
  used,
  controller,
}: {
  settings: readonly Field[];
  used: number;
  controller: NodesNode | undefined;
}) {
  return (
    <Section title="Schedule">
      <div className="flex flex-col gap-3">
        <BackupSettings
          keys={[backupSettingKeys.every, backupSettingKeys.keepScheduled]}
          settings={settings}
        />
        <p className="text-meta text-ink-3">
          The Backups on this machine take {bytes(used)}.
          {controller !== undefined && (
            <>
              {" "}
              Change where they&apos;re kept in{" "}
              <Link
                to="/nodes/$nodeId/backups"
                params={{ nodeId: controller.id }}
                className="text-ink-2 underline decoration-ink-3 underline-offset-4 hover:text-ink-1"
              >
                {controller.name}&apos;s Backups settings
              </Link>
              .
            </>
          )}
        </p>
      </div>
    </Section>
  );
}

export function RestorePanel() {
  return (
    <Section title="Restore">
      <div className="flex flex-col items-start gap-3">
        <p className="text-ink-2">
          Restore a listed Backup from its menu, or upload a Backup file from this device.
        </p>
        <RestoreFileButton />
      </div>
    </Section>
  );
}

export function BackupDetails({ settings }: { settings: readonly Field[] }) {
  return (
    <Details fields>
      <div className="flex flex-col gap-4 pt-2">
        <BackupSettings keys={detailKeys} settings={settings} />
        <div className="flex flex-wrap gap-2">
          <CheckSavedCopy />
          <RetireButton />
        </div>
      </div>
    </Details>
  );
}
