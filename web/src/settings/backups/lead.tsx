import type {
  BackupsBackupList,
  BackupsDestination,
  NodesNode,
  SettingField as Field,
} from "../../api/gen/model";
import type { Clock } from "../../format";
import { useAttempt } from "../../targets/attempt";
import { Button } from "../../ui/button";
import { AddDestinationButton } from "./add-destination";
import { downloadSheet } from "./downloads";
import { backupSettingKeys, secondsOf, settingOf } from "./settings";
import { TaskLink } from "./task-link";
import {
  type Names,
  destinationAdvice,
  leadOf,
  passphraseAdvice,
  recommendationsOf,
  sheetAdvice,
} from "./words";

const daySeconds = 86_400;

export function Advice({
  list,
  destinations,
  nodes,
  names,
}: {
  list: BackupsBackupList;
  destinations: readonly BackupsDestination[];
  nodes: readonly NodesNode[];
  names: Names;
}) {
  const download = useAttempt(downloadSheet);
  return recommendationsOf(list, destinations).map((advice) => (
    <div key={advice} className="flex flex-col items-start gap-2">
      {advice === "destination" && (
        <>
          <p className="text-ink-1">{destinationAdvice(nodes.length > 1)}</p>
          <AddDestinationButton nodes={nodes} />
        </>
      )}
      {advice === "passphrase" && <p className="text-ink-1">{passphraseAdvice}</p>}
      {advice === "sheet" && (
        <>
          <p className="text-ink-1">{sheetAdvice(list, destinations, names)}</p>
          {download.failure}
          <Button busy={download.busy} onClick={download.run}>
            Download recovery sheet
          </Button>
        </>
      )}
    </div>
  ));
}

export function Lead({
  list,
  destinations,
  settings,
  names,
  clock,
  start,
}: {
  list: BackupsBackupList;
  destinations: readonly BackupsDestination[];
  settings: readonly Field[];
  names: Names;
  clock: Clock;
  start: string;
}) {
  const lead = leadOf({
    list,
    destinations,
    everySeconds: secondsOf(settingOf(settings, backupSettingKeys.every)),
    keepDays: secondsOf(settingOf(settings, backupSettingKeys.keepScheduled)) / daySeconds,
    start,
    names,
    clock,
  });
  return (
    <p className="text-pretty text-ink-1">
      {lead.text}
      {lead.failed && (
        <>
          {" "}
          <TaskLink /> for the cause.
        </>
      )}
    </p>
  );
}
