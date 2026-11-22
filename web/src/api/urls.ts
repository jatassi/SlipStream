// The operations the generated client leaves out because the browser
// itself reads them: a download it saves, and an event stream it opens.

const nodePath = (nodeId: string) => `/api/v1/nodes/${encodeURIComponent(nodeId)}`;

export const nodeLogsArchiveUrl = (nodeId: string) => `${nodePath(nodeId)}/logs/archive`;

export const nodeLogsStreamUrl = (nodeId: string, after?: number) =>
  after === undefined
    ? `${nodePath(nodeId)}/logs/stream`
    : `${nodePath(nodeId)}/logs/stream?after=${after}`;

export const recoverySheetUrl = "/api/v1/backups/recovery-sheet";

export const backupFileUrl = (backupId: string) =>
  `/api/v1/backups/${encodeURIComponent(backupId)}/file`;
