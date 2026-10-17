import { Download01Icon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import { sentenceOf } from "../../../api/errors";
import { getNodeSystem } from "../../../api/gen/query/system/system";
import { nodeLogsArchiveUrl } from "../../../api/urls";
import { Button } from "../../../ui/button";
import { toast } from "../../../ui/toast";

const sentence = (text: string) => (text.endsWith(".") ? text : `${text}.`);

// The browser streams the zip straight to disk through a link, so a Node that
// can't answer is caught first and the page never navigates to a problem.
async function downloadLogs(nodeId: string) {
  try {
    await getNodeSystem(nodeId);
  } catch (error: unknown) {
    toast(`Couldn't download the logs. ${sentence(sentenceOf(error))}`, { kind: "fail" });
    return;
  }
  const link = document.createElement("a");
  link.href = nodeLogsArchiveUrl(nodeId);
  link.download = "";
  link.click();
}

export function DownloadLogs({ nodeId }: { nodeId: string }) {
  const [busy, setBusy] = useState(false);
  return (
    <Button
      icon={Download01Icon}
      busy={busy}
      onClick={() => {
        setBusy(true);
        void downloadLogs(nodeId).finally(() => {
          setBusy(false);
        });
      }}
    >
      Download logs
    </Button>
  );
}
