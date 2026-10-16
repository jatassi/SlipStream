import { beforeEach, expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import type * as Prefs from "./prefs";
import { readPref, usePref, writePref } from "./prefs";

const isPrefs = (module: unknown): module is typeof Prefs =>
  typeof module === "object" && module !== null && "readPref" in module && "writePref" in module;

let loads = 0;

// nextPageLoad imports a new instance of prefs, as a reload would.
async function nextPageLoad(): Promise<typeof Prefs> {
  loads += 1;
  const url = new URL("prefs.ts", import.meta.url);
  url.searchParams.set("load", String(loads));
  // oxlint-disable-next-line no-inline-comments -- Vite reads @vite-ignore only inside import()
  const module: unknown = await import(/* @vite-ignore */ url.href);
  if (!isPrefs(module)) {
    throw new Error("prefs.ts did not load");
  }
  return module;
}

function Collapse() {
  const collapsed = usePref("sidebarCollapsed");
  return (
    <button
      type="button"
      onClick={() => {
        writePref("sidebarCollapsed", !collapsed);
      }}
    >
      {collapsed ? "Expand" : "Collapse"}
    </button>
  );
}

const refuse = () => {
  throw new DOMException("The operation is insecure.", "SecurityError");
};

beforeEach(() => {
  writePref("sidebarCollapsed", false);
  writePref("startupOff", false);
  writePref("lastEmail", "");
});

test("a preference written in one page load reads back in the next", async () => {
  expect(readPref("sidebarCollapsed")).toBe(false);
  writePref("sidebarCollapsed", true);

  const next = await nextPageLoad();
  expect(next.readPref("sidebarCollapsed")).toBe(true);
  expect(next.readPref("startupOff")).toBe(false);
});

test("a string preference sits beside the booleans and reads back as typed", async () => {
  expect(readPref("lastEmail")).toBe("");
  writePref("lastEmail", "ada@example.test");
  writePref("startupOff", true);

  const next = await nextPageLoad();
  expect(next.readPref("lastEmail")).toBe("ada@example.test");
  expect(next.readPref("startupOff")).toBe(true);
});

test("a component re-renders when its preference changes", async () => {
  renderKit(<Collapse />);

  await userEvent.click(page.getByRole("button", { name: "Collapse" }));
  await expect.element(page.getByRole("button", { name: "Expand" })).toBeVisible();
});

test("a browser that refuses storage reads the defaults and keeps writes for the page", async () => {
  writePref("startupOff", true);
  vi.spyOn(Storage.prototype, "getItem").mockImplementation(refuse);
  vi.spyOn(Storage.prototype, "setItem").mockImplementation(refuse);

  const next = await nextPageLoad();
  expect(next.readPref("startupOff")).toBe(false);
  next.writePref("sidebarCollapsed", true);
  expect(next.readPref("sidebarCollapsed")).toBe(true);
  vi.restoreAllMocks();
});
