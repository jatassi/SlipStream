import type { QueryClient } from "@tanstack/react-query";
import { notFound } from "@tanstack/react-router";
import type { ComponentType } from "react";

import {
  getGetAPIKeySuspenseQueryOptions,
  getGetIdentitySettingsSuspenseQueryOptions,
} from "../api/gen/query/identity/identity";
import { getGetMailSettingsSuspenseQueryOptions } from "../api/gen/query/mail/mail";
import { freshQuery } from "../api/query-client";
import { AddressSection } from "./address";
import { ApiKeySection } from "./api-key";
import { EmailSection } from "./email";

export type SettingsSection = {
  readonly id: string;
  readonly label: string;
  readonly load: (queryClient: QueryClient) => Promise<void>;
  readonly component: ComponentType;
};

/** Controller-wide settings, one row per section. A later spec adds its section as a row. */
export const settingsSections: readonly [SettingsSection, ...SettingsSection[]] = [
  {
    id: "address",
    label: "Address",
    load: async (queryClient) => {
      await freshQuery(queryClient, getGetIdentitySettingsSuspenseQueryOptions());
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
    id: "api-key",
    label: "API key",
    load: async (queryClient) => {
      await freshQuery(queryClient, getGetAPIKeySuspenseQueryOptions());
    },
    component: ApiKeySection,
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
