import { notFound } from "@tanstack/react-router";

import {
  getGetAPIKeySuspenseQueryOptions,
  getGetAddressSuspenseQueryOptions,
  getGetIdentitySettingsSuspenseQueryOptions,
} from "../api/gen/query/identity/identity";
import { getGetMailSettingsSuspenseQueryOptions } from "../api/gen/query/mail/mail";
import {
  getListCutsSuspenseQueryOptions,
  getListProfilesSuspenseQueryOptions,
  getListQualitiesSuspenseQueryOptions,
  getListTargetsSuspenseQueryOptions,
} from "../api/gen/query/targets/targets";
import { getListContentTypesSuspenseQueryOptions } from "../api/gen/query/works/works";
import { ensureQuery, freshQuery } from "../api/query-client";
import { downloadClientsSection } from "../download-clients/section";
import { indexersSection } from "../indexers/section";
import { libraryRootsSection } from "../roots/section";
import { EditionsSection } from "../targets/editions-section";
// oxlint-disable-next-line import/max-dependencies -- the table names every section of Settings
import { ProfilesSection } from "../targets/profiles-section";
import { TargetsSection } from "../targets/targets-section";
import { AddressSection } from "./address";
import { ApiKeySection } from "./api-key";
import { EmailSection } from "./email";
import { metadataSection } from "./metadata";
import { nodesSection } from "./nodes";
import type { SettingsSection } from "./section";
import { SignInSection } from "./sign-in";

export const settingsSections: readonly [SettingsSection, ...SettingsSection[]] = [
  {
    id: "address",
    label: "Address",
    load: async (queryClient) => {
      await freshQuery(queryClient, getGetAddressSuspenseQueryOptions());
    },
    component: AddressSection,
  },
  {
    id: "email",
    label: "Email",
    load: async (queryClient) => {
      await freshQuery(queryClient, getGetMailSettingsSuspenseQueryOptions());
    },
    component: EmailSection,
  },
  {
    id: "sign-in",
    label: "Sign-in",
    load: async (queryClient) => {
      await freshQuery(queryClient, getGetIdentitySettingsSuspenseQueryOptions());
    },
    component: SignInSection,
  },
  nodesSection,
  downloadClientsSection,
  indexersSection,
  {
    id: "api-key",
    label: "API key",
    load: async (queryClient) => {
      await freshQuery(queryClient, getGetAPIKeySuspenseQueryOptions());
    },
    component: ApiKeySection,
  },
  libraryRootsSection,
  metadataSection,
  {
    id: "targets",
    label: "Targets",
    load: async (queryClient) => {
      await Promise.all([
        ensureQuery(queryClient, getListTargetsSuspenseQueryOptions()),
        ensureQuery(queryClient, getListQualitiesSuspenseQueryOptions()),
      ]);
    },
    component: TargetsSection,
    wide: true,
  },
  {
    id: "profiles",
    label: "Profiles",
    load: async (queryClient) => {
      await Promise.all([
        ensureQuery(queryClient, getListProfilesSuspenseQueryOptions()),
        ensureQuery(queryClient, getListTargetsSuspenseQueryOptions()),
        ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
      ]);
    },
    component: ProfilesSection,
    wide: true,
  },
  {
    id: "editions",
    label: "Editions",
    load: async (queryClient) => {
      await Promise.all([
        ensureQuery(queryClient, getListCutsSuspenseQueryOptions()),
        ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
      ]);
    },
    component: EditionsSection,
    wide: true,
  },
];

export function sectionOf(id: string): SettingsSection {
  const section = settingsSections.find((row) => row.id === id);
  if (section === undefined) {
    // oxlint-disable-next-line typescript/only-throw-error -- the router renders its own notFound() as the not-found page
    throw notFound();
  }
  return section;
}
