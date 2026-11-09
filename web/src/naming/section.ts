import {
  getListNamingSuspenseQueryOptions,
  getListNamingTokensSuspenseQueryOptions,
} from "../api/gen/query/naming/naming";
import { getListContentTypesSuspenseQueryOptions } from "../api/gen/query/works/works";
import { ensureQuery, freshQuery } from "../api/query-client";
import type { SettingsSection } from "../settings/section";
import { NamingSection, namedTypes } from "./naming-section";

export const namingSection: SettingsSection = {
  id: "naming",
  label: "Naming",
  load: async (queryClient) => {
    const [{ contentTypes }, naming] = await Promise.all([
      ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
      freshQuery(queryClient, getListNamingSuspenseQueryOptions()),
    ]);
    await Promise.all(
      namedTypes(contentTypes, naming).map(({ type }) =>
        ensureQuery(queryClient, getListNamingTokensSuspenseQueryOptions(type.id)),
      ),
    );
  },
  component: NamingSection,
};
