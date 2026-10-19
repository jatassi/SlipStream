import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { healthServer, reportOf } from "../test/health";
import { gib, globalLevels, mediaLevels } from "../test/health-levels";
import { renderApp } from "../test/render-app";
import { refused, serveRoots } from "../test/roots";

const storage = () => page.getByRole("region", { name: "Storage" });

const summary = (warn: string, error: string, every: string) =>
  `A volume shows a warning below ${warn} free and an error below ${error}. Storage Nodes check each volume every ${every}, and again after each import.`;

function openHealth() {
  const roots = serveRoots();
  const health = healthServer(reportOf());
  const app = renderApp({ path: "/health", handlers: health.handlers });
  return { app, health, roots };
}

test("the Storage section ends with the levels and the interval in one sentence, the fields under Details", async () => {
  openHealth();
  await expect.element(storage().getByText(summary("100 GB", "20 GB", "5 min"))).toBeVisible();
  expect(storage().getByRole("textbox").query()).toBeNull();
  await userEvent.click(storage().getByRole("button", { name: "Details" }));
  await expect
    .element(storage().getByRole("textbox", { name: "Warn when free space is below" }))
    .toBeVisible();
  await expect
    .element(storage().getByRole("textbox", { name: "Show an error when free space is below" }))
    .toBeVisible();
  await expect
    .element(storage().getByRole("textbox", { name: "Check storage every" }))
    .toBeVisible();
});

test("a new warning level is saved and the sentence says it", async () => {
  const { health } = openHealth();
  await userEvent.click(storage().getByRole("button", { name: "Details" }));
  await userEvent.fill(
    storage().getByRole("textbox", { name: "Warn when free space is below" }),
    "250",
  );
  await userEvent.click(storage().getByRole("button", { name: "Save" }));
  await expect.element(storage().getByText(summary("250 GB", "20 GB", "5 min"))).toBeVisible();
  expect(health.sent.settings).toEqual([{ warnBelow: { amount: 250 * gib, unit: "bytes" } }]);
});

test("an error level the server refuses shows its sentence beside the field", async () => {
  const { health } = openHealth();
  health.state.refusal = refused(422, "setting_out_of_range", "Out of range", {
    detail: "Set the error level below the warning level, 100 GB.",
  });
  await userEvent.click(storage().getByRole("button", { name: "Details" }));
  await userEvent.fill(
    storage().getByRole("textbox", { name: "Show an error when free space is below" }),
    "150",
  );
  await userEvent.click(storage().getByRole("button", { name: "Save" }));
  await expect
    .element(storage().getByRole("alert"))
    .toHaveTextContent("Set the error level below the warning level, 100 GB.");
  await expect.element(storage().getByText(summary("100 GB", "20 GB", "5 min"))).toBeVisible();
});

test("Check storage every saves the one stored interval from the Health page too", async () => {
  const { roots } = openHealth();
  await userEvent.click(storage().getByRole("button", { name: "Details" }));
  const every = storage().getByRole("textbox", { name: "Check storage every" });
  await userEvent.clear(every);
  await userEvent.type(every, "15");
  await userEvent.click(storage().getByRole("button", { name: "Save" }));
  await expect.element(storage().getByText(summary("100 GB", "20 GB", "15 min"))).toBeVisible();
  expect(roots.sent.sampleEvery).toEqual([900]);
});

test("Details name the volumes whose levels send them straight to an error, and lead to Library roots", async () => {
  serveRoots();
  const health = healthServer(reportOf());
  health.state.settings = {
    ...globalLevels,
    warnBelow: { ...globalLevels.warnBelow, value: { amount: 5, unit: "percent" } },
  };
  health.state.volumes = [{ ...mediaLevels, skipsWarning: true }];
  const { router } = renderApp({ path: "/health", handlers: health.handlers });
  await expect.element(storage().getByText(summary("5%", "20 GB", "5 min"))).toBeVisible();
  await userEvent.click(storage().getByRole("button", { name: "Details" }));
  await expect.element(storage().getByRole("listitem")).toHaveTextContent("/srv/media on home");
  await userEvent.click(
    storage().getByRole("link", { name: "Set a volume's own levels in Library roots" }),
  );
  await expect.poll(() => router.state.location.pathname).toBe("/settings/library-roots");
});
