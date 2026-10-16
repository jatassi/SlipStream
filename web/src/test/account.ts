import { HttpResponse, type RequestHandler, http } from "msw";
import { expect, onTestFinished, vi } from "vitest";
import { page } from "vitest/browser";

import type { IdentityPasskey, IdentitySessionRow, SystemPasskeys } from "../api/gen/model";
import { getGetStatusMockHandler } from "../api/gen/query/system/system.msw";
import { webauthn } from "../session/webauthn";
import { renderApp } from "./render-app";

export const thisLaptop: IdentitySessionRow = {
  id: "s-laptop",
  device: "Chrome on macOS",
  address: "192.168.1.20",
  createdAt: "2026-10-01T09:00:00Z",
  lastSeenAt: "2026-10-16T08:00:00Z",
  current: true,
};

export const livingRoomTv: IdentitySessionRow = {
  id: "s-tv",
  device: "Safari on Apple TV",
  address: "192.168.1.31",
  createdAt: "2026-09-20T19:00:00Z",
  lastSeenAt: "2026-10-15T21:30:00Z",
  current: false,
};

export const iPhonePasskey: IdentityPasskey = {
  id: "p-iphone",
  name: "iCloud Keychain",
  host: "localhost",
  createdAt: "2026-09-01T10:00:00Z",
  lastUsedAt: "2026-10-15T07:45:00Z",
};

export const passkeysHere: SystemPasskeys = { available: true };

export const passkeysAt = (passkeys: SystemPasskeys) =>
  getGetStatusMockHandler({ claim: "controller", passkeys });

/** Your Account's lists, as the Node answers them. */
export function accountLists({
  passkeys = [iPhonePasskey],
  sessions = [thisLaptop, livingRoomTv],
}: { passkeys?: IdentityPasskey[]; sessions?: IdentitySessionRow[] } = {}) {
  return [
    http.get("*/api/v1/passkeys", () => HttpResponse.json({ passkeys })),
    http.get("*/api/v1/sessions", () => HttpResponse.json({ sessions })),
  ];
}

/** Stands in for the browser's authenticator, which a test browser lacks. */
export function stubPasskeys({ autofill = false }: { autofill?: boolean } = {}) {
  const get = vi.spyOn(webauthn, "get");
  const create = vi.spyOn(webauthn, "create");
  vi.spyOn(webauthn, "conditionalMediation").mockResolvedValue(autofill);
  onTestFinished(() => {
    vi.restoreAllMocks();
  });
  return { get, create };
}

export const ceremony = (id: string) =>
  HttpResponse.json({ ceremony: id, options: { challenge: "Y2hhbGxlbmdl" } });

export const accountMain = () => page.getByRole("main");
export const accountDialog = (name: string) => page.getByRole("dialog", { name });

export async function openAccount(handlers: RequestHandler[] = []) {
  const app = renderApp({ path: "/account", handlers: [...accountLists(), ...handlers] });
  await expect.element(page.getByRole("heading", { name: "Your Account", level: 1 })).toBeVisible();
  return app;
}

export function recorder() {
  const seen: { method: string; url: string; body: unknown }[] = [];
  return {
    seen,
    record: async (request: Request) => {
      const url = new URL(request.url);
      seen.push({
        method: request.method,
        url: url.pathname + url.search,
        body: request.method === "DELETE" ? null : await request.json(),
      });
    },
  };
}
