import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { folderEntries, packEntry, takenEntry } from "../test/import-entries";
import { openPending } from "../test/pending-page";
import { asPhone } from "../test/viewport";

const pack = () => page.getByRole("listitem", { name: "Severance.S01.1080p.WEB-DL" });
const episodeOne = () =>
  pack().getByRole("listitem", { name: "Severance.S01E01.1080p.WEB-DL.mkv" });
const brazil = () => page.getByRole("listitem", { name: "Brazil.1985.1080p.BluRay" });

test("Pending imports lists downloads first, then each root, then the folders imported by hand", async () => {
  openPending({ start: { imports: [packEntry, takenEntry, ...folderEntries] } });
  await expect.element(pack()).toBeVisible();
  expect(
    page
      .getByRole("heading", { level: 2 })
      .elements()
      .map((heading) => heading.textContent),
  ).toEqual([
    "From downloads",
    "Films library root on home",
    "Shows library root on home",
    "Imported by hand",
  ]);
});

test("a waiting download names its Release, Work and Node, and each waiting file leads with its sentence", async () => {
  openPending();
  await expect.element(pack().getByText("Severance · Downloaded on home")).toBeVisible();
  await expect.element(pack().getByText("Seeding continues while this waits.")).toBeVisible();
  await expect
    .element(episodeOne().getByText("This isn't better than the file you have for 1080p."))
    .toBeVisible();
  expect(
    pack().getByRole("listitem", { name: "Severance.S01E02.1080p.WEB-DL.mkv" }).elements(),
  ).toHaveLength(0);
  await expect.element(episodeOne().getByRole("button", { name: "Don't import" })).toBeVisible();
  await userEvent.click(episodeOne().getByRole("button", { name: "Details" }));
  await expect.element(episodeOne().getByText("Its copy waiting here uses 2.0 GB.")).toBeVisible();
  await expect.element(episodeOne().getByText("1080p", { exact: true })).toBeVisible();
  await expect.element(brazil().getByText("Brazil · Downloaded on seedbox")).toBeVisible();
});

test("Force names the file it deletes before it forces the file in", async () => {
  const { sent } = openPending();
  await userEvent.click(
    episodeOne().getByRole("button", {
      name: "Other actions for Severance.S01E01.1080p.WEB-DL.mkv",
    }),
  );
  await userEvent.click(page.getByRole("menuitem", { name: "Force into 1080p" }));
  const confirm = page.getByRole("dialog", { name: "Force into 1080p?" });
  await expect
    .element(
      confirm.getByText(
        "This deletes Severance - S01E01 - Good News About Hell.mkv, because it fills no other Target.",
      ),
    )
    .toBeVisible();
  await userEvent.click(confirm.getByRole("button", { name: "Force into 1080p" }));
  await expect.element(confirm).not.toBeInTheDocument();
  expect(sent.acted).toEqual([{ id: packEntry.id, position: 0, action: "force" }]);
  await expect.element(pack()).not.toBeInTheDocument();
});

test("the recommended action is the primary button and the rest sit in the menu", async () => {
  const { sent, app } = openPending();
  await userEvent.click(brazil().getByRole("button", { name: "Retry" }));
  await expect
    .poll(() => sent.acted)
    .toEqual([{ id: takenEntry.id, position: 0, action: "retry" }]);
  await userEvent.click(
    brazil().getByRole("button", { name: "Other actions for Brazil.1985.1080p.BluRay.mkv" }),
  );
  expect(
    page
      .getByRole("menuitem")
      .elements()
      .map((item) => item.textContent),
  ).toEqual(["Turn on renaming", "Don't import", "Not this Release"]);
  await userEvent.click(page.getByRole("menuitem", { name: "Turn on renaming" }));
  await expect.poll(() => app.router.state.location.pathname).toBe("/settings/naming");
});

test("Don't import ends a download's wait and it leaves the list", async () => {
  const { sent } = openPending();
  await userEvent.click(episodeOne().getByRole("button", { name: "Don't import" }));
  await expect.element(pack()).not.toBeInTheDocument();
  expect(sent.acted).toEqual([{ id: packEntry.id, position: 0, action: "decline" }]);
  await expect.element(brazil()).toBeVisible();
});

test("on a phone a download opens as a pushed page that links back to Pending imports", async () => {
  await asPhone();
  const { app, sent } = openPending();
  await userEvent.click(page.getByRole("link", { name: /Severance\.S01\.1080p\.WEB-DL/u }));
  await expect
    .poll(() => app.router.state.location.pathname)
    .toBe(`/library/pending/${packEntry.id}`);
  await expect
    .element(page.getByRole("banner").getByRole("link", { name: "Pending imports" }))
    .toHaveAttribute("href", "/library/pending");
  await userEvent.click(episodeOne().getByRole("button", { name: "Don't import" }));
  await expect.element(page.getByText("Nothing in this download waits for you now.")).toBeVisible();
  expect(sent.acted).toEqual([{ id: packEntry.id, position: 0, action: "decline" }]);
});
