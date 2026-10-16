import { Download01Icon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import { UnreachableError, isApiError } from "../../../api/errors";
import { getNodeSystem } from "../../../api/gen/query/system/system";
import { nodeLogsArchiveUrl } from "../../../api/urls";
import { Button } from "../../../ui/button";
import { toast } from "../../../ui/toast";

const sentence = (text: string) => (text.endsWith(".") ? text : `${text}.`);

function reasonOf(error: unknown): string {
  if (isApiError(error)) {
    return sentence(error.detail ?? error.title);
  }
  if (error instanceof UnreachableError) {
    return sentence(error.message);
  }
  throw error;
}

// The browser streams the zip straight to disk through a link, so a Node that
// can't answer is caught first and the page never navigates to a problem.
async function downloadLogs(nodeId: string) {
  try {
    await getNodeSystem(nodeId);
  } catch (error: unknown) {
    toast(`Couldn't download the logs. ${reasonOf(error)}`, { kind: "fail" });
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
