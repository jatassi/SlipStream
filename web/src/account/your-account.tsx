import type { QueryClient } from "@tanstack/react-query";
import type { ReactNode } from "react";

import {
  getListPasskeysSuspenseQueryOptions,
  getListSessionsSuspenseQueryOptions,
} from "../api/gen/query/identity/identity";
import { ensureQuery } from "../api/query-client";
import { statusQuery } from "../session/session";
import { Page } from "../shell/page";
import { Devices } from "./devices";
import { Passkeys } from "./passkeys";
import { Pin } from "./pin";
import { Profile } from "./profile";

export const loadYourAccount = (queryClient: QueryClient): Promise<unknown> =>
  Promise.all([
    queryClient.query(statusQuery),
    ensureQuery(queryClient, getListPasskeysSuspenseQueryOptions()),
    ensureQuery(queryClient, getListSessionsSuspenseQueryOptions()),
  ]);

/**
 * Your Account, the same in the app and the portal. `adding` is whether Add a
 * passkey is open, which the page's address holds so a link can open it.
 */
export function YourAccount({
  adding,
  onAddingChange,
  action,
}: {
  adding: boolean;
  onAddingChange: (adding: boolean) => void;
  action?: ReactNode;
}) {
  return (
    <Page action={action}>
      <Profile />
      <Pin />
      <Passkeys adding={adding} onAddingChange={onAddingChange} />
      <Devices />
    </Page>
  );
}
