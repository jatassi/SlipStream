import type { QueryClient } from "@tanstack/react-query";
import type { ComponentType } from "react";

import type { FirstrunStepID, FirstrunStepState } from "../../api/gen/model";
import { getListDownloadClientsSuspenseQueryOptions } from "../../api/gen/query/downloadclients/downloadclients";
import { getGetAddressSuspenseQueryOptions } from "../../api/gen/query/identity/identity";
import { getGetMetadataSettingsSuspenseQueryOptions } from "../../api/gen/query/metadata/metadata";
import { getListNodesSuspenseQueryOptions } from "../../api/gen/query/nodes/nodes";
import {
  getListRootsSuspenseQueryOptions,
  getListVolumesSuspenseQueryOptions,
} from "../../api/gen/query/roots/roots";
import {
  getListProfilesSuspenseQueryOptions,
  getListQualitiesSuspenseQueryOptions,
  getListTargetsSuspenseQueryOptions,
} from "../../api/gen/query/targets/targets";
import { getGetNodeConfigSuspenseQueryOptions } from "../../api/gen/query/system/system";
// oxlint-disable-next-line import/max-dependencies -- the table names every step's screen and what it reads
import { getListContentTypesSuspenseQueryOptions } from "../../api/gen/query/works/works";
import { ensureQuery, freshQuery } from "../../api/query-client";
import { AddressStep } from "./address-step";
import { DownloadsStep } from "./downloads-step";
import type { StepNav } from "./frame";
import { LanguageStep } from "./language-step";
import { MediaStep } from "./media-step";
import { PasskeyStep } from "./passkey-step";
import { RootsStep } from "./roots-step";
import { TargetsStep } from "./targets-step";

export type StepBodyProps = { readonly step: FirstrunStepState; readonly nav: StepNav };

export type StepScreen = {
  readonly name: string;
  readonly title: string;
  readonly why: string;
  /** Where Settings keeps what the step sets, for the done screen's link from a skipped step. */
  readonly settings: string;
  readonly load: (queryClient: QueryClient) => Promise<void>;
  readonly Body: ComponentType<StepBodyProps>;
};

const nothing = () => Promise.resolve();

// Each in-step Node's settings hold its downloads folder, read live from the Node.
async function loadDownloadClients(queryClient: QueryClient) {
  const { cards } = await freshQuery(queryClient, getListDownloadClientsSuspenseQueryOptions());
  await Promise.all(
    cards
      .filter((card) => card.inStep)
      .map((card) => freshQuery(queryClient, getGetNodeConfigSuspenseQueryOptions(card.nodeId))),
  );
}

/** A screen for every step the server can list, so a step without one fails the build. */
export const screens = {
  address: {
    name: "Address",
    title: "Where people reach SlipStream",
    why: "Pick an address everyone in your household can reach, because Signup and Reset links use it.",
    settings: "/settings/address",
    load: async (queryClient) => {
      await freshQuery(queryClient, getGetAddressSuspenseQueryOptions());
    },
    Body: AddressStep,
  },
  passkey: {
    name: "Passkey",
    title: "Add a passkey",
    why: "Sign in on this device with Face ID, Touch ID or its screen lock instead of your PIN.",
    settings: "/account?add=passkey",
    load: nothing,
    Body: PasskeyStep,
  },
  media: {
    name: "Kinds of media",
    title: "Kinds of media",
    why: "SlipStream shows only the kinds of media you turn on.",
    settings: "/settings/library-roots",
    load: async (queryClient) => {
      await freshQuery(queryClient, getListContentTypesSuspenseQueryOptions());
    },
    Body: MediaStep,
  },
  language: {
    name: "Language and region",
    title: "Language and region",
    why: "Titles and overviews follow the language, and the region picks the certification and decides when a movie counts as out.",
    settings: "/settings/metadata",
    load: async (queryClient) => {
      await freshQuery(queryClient, getGetMetadataSettingsSuspenseQueryOptions());
    },
    Body: LanguageStep,
  },
  roots: {
    name: "Library roots",
    title: "Library roots",
    why: "A library root is the folder where SlipStream keeps one kind of media's files.",
    settings: "/settings/library-roots",
    load: async (queryClient) => {
      await Promise.all([
        ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
        ensureQuery(queryClient, getListRootsSuspenseQueryOptions()),
        ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
      ]);
    },
    Body: RootsStep,
  },
  downloads: {
    name: "Downloads",
    title: "Downloads",
    why: "Check where each Node puts its downloads before the first Grab fills a drive you didn't expect.",
    settings: "/settings/download-clients",
    load: async (queryClient) => {
      await Promise.all([
        loadDownloadClients(queryClient),
        ensureQuery(queryClient, getListVolumesSuspenseQueryOptions()),
        ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
        ensureQuery(queryClient, getListRootsSuspenseQueryOptions()),
        ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
      ]);
    },
    Body: DownloadsStep,
  },
  targets: {
    name: "What SlipStream grabs",
    title: "What SlipStream grabs",
    why: "Each kind of media grabs what its default Profile's Targets accept until you change them.",
    settings: "/settings/targets",
    load: async (queryClient) => {
      await Promise.all([
        ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
        ensureQuery(queryClient, getListProfilesSuspenseQueryOptions()),
        ensureQuery(queryClient, getListTargetsSuspenseQueryOptions()),
        ensureQuery(queryClient, getListQualitiesSuspenseQueryOptions()),
      ]);
    },
    Body: TargetsStep,
  },
} satisfies { readonly [S in FirstrunStepID]: StepScreen };
