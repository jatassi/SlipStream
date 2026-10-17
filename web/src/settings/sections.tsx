import type { QueryClient } from "@tanstack/react-query";
import { notFound } from "@tanstack/react-router";
import type { ComponentType } from "react";

import {
  getGetAPIKeySuspenseQueryOptions,
  getGetAddressSuspenseQueryOptions,
  getGetIdentitySettingsSuspenseQueryOptions,
} from "../api/gen/query/identity/identity";
import { getGetMailSettingsSuspenseQueryOptions } from "../api/gen/query/mail/mail";
import { freshQuery } from "../api/query-client";
import { AddressSection } from "./address";
import { ApiKeySection } from "./api-key";
import { EmailSection } from "./email";
import { SignInSection } from "./sign-in";

export type SettingsSection = {
  readonly id: string;
  readonly label: string;
  readonly load: (queryClient: QueryClient) => Promise<void>;
  readonly component: ComponentType;
  /** Drops the 560px column, for an editor that needs the page's width. */
  readonly wide?: boolean;
};

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
