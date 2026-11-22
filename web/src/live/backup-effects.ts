import { getGetBackupPassphraseQueryKey } from "../api/gen/query/backups/backups";
import type { LiveEffects } from "./types";

// The passphrase changes only by Replace, whose own tab drops it, and a
// refetch of it could open the PIN prompt over whatever page is open.
const everyBackupsQuery = {
  predicate: ({ queryKey: [path] }: { queryKey: readonly unknown[] }) =>
    typeof path === "string" &&
    /^\/api\/v1\/backup(?:s|-settings|-destinations)(?:\/|$)/u.test(path) &&
    path !== getGetBackupPassphraseQueryKey()[0],
};

export const backupEffects = {
  "backups.changed": (_data, { refresh }) => {
    refresh(everyBackupsQuery);
  },
} satisfies Pick<LiveEffects, "backups.changed">;
