import { getGetLibraryScanSettingsSuspenseQueryOptions } from "../api/gen/query/imports/imports";
import { getListTasksSuspenseQueryOptions } from "../api/gen/query/tasks/tasks";
import { ensureQuery, freshQuery } from "../api/query-client";
import type { SettingsSection } from "../settings/section";
import { LibraryScanSection } from "./library-scan-section";

export const libraryScanSection: SettingsSection = {
  id: "library-scan",
  label: "Library scan",
  load: async (queryClient) => {
    await Promise.all([
      freshQuery(queryClient, getGetLibraryScanSettingsSuspenseQueryOptions()),
      ensureQuery(queryClient, getListTasksSuspenseQueryOptions()),
    ]);
  },
  component: LibraryScanSection,
  health: ["library"],
};
