import { prepareBackupDownload, prepareRecoverySheet } from "../../api/gen/query/backups/backups";
import { backupFileUrl, recoverySheetUrl } from "../../api/urls";

// A link the browser follows can't show the PIN prompt, so a JSON pre-flight
// at the same level asks for it first.
function follow(href: string) {
  const link = document.createElement("a");
  link.href = href;
  link.download = "";
  link.click();
}

export async function downloadSheet() {
  await prepareRecoverySheet();
  follow(recoverySheetUrl);
}

export async function downloadBackup(backupId: string) {
  await prepareBackupDownload(backupId);
  follow(backupFileUrl(backupId));
}
