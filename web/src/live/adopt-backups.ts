import type { QueryClient } from "@tanstack/react-query";

import type { BackupsBackupList, BackupsBackupSettings } from "../api/gen/model";
import {
  getGetBackupSettingsQueryKey,
  getListBackupsQueryKey,
} from "../api/gen/query/backups/backups";

// A save or a delete answers with what it wrote, so the section shows it at
// once rather than after backups.changed refetches it.

export function adoptBackupSettings(queryClient: QueryClient, saved: BackupsBackupSettings) {
  queryClient.setQueryData(getGetBackupSettingsQueryKey(), saved);
}

export function adoptBackupList(queryClient: QueryClient, list: BackupsBackupList) {
  queryClient.setQueryData(getListBackupsQueryKey(), list);
}
