import type { QueryClient } from "@tanstack/react-query";

import type { BackupsBackupList, BackupsBackupSettings } from "../api/gen/model";
import {
  getGetBackupPassphraseQueryKey,
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

// The words Settings shows on Show live in the cache, apart from first setup's
// read, so an answer that lands after a PIN prompt still shows when a shell
// switch remounted the page meanwhile.
export const shownPassphraseKey = [...getGetBackupPassphraseQueryKey(), "shown"] as const;

export function hidePassphrase(queryClient: QueryClient) {
  void queryClient.resetQueries({ queryKey: shownPassphraseKey, exact: true });
}
