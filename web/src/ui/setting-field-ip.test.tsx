import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SettingIPField } from "../api/gen/model";
import { renderKit } from "../test/render-kit";
import { SettingField } from "./setting-field";

const stableIPv6: SettingIPField = {
  source: "database",
  applies: "live",
  pendingRestart: false,
  key: "reachability.stableIPv6",
  label: "Stable IPv6 address",
  kind: "ip",
  isDefault: false,
  value: "2001:db8::20",
  running: "2001:db8::20",
  default: null,
  family: "ipv6",
};

test.each([
  [" 2001:db8::21 ", "2001:db8::21"],
  ["", ""],
])("an address typed as %j is saved as %j", async (typed, saved) => {
  const onSave = vi.fn<(value: unknown) => Promise<unknown>>(() => Promise.resolve());
  renderKit(<SettingField field={stableIPv6} onSave={onSave} />);
  await userEvent.fill(page.getByRole("textbox", { name: "Stable IPv6 address" }), typed);
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenLastCalledWith(saved);
});
