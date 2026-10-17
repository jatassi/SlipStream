import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { changeOp, noImpact } from "../test/impact";
import { orchardId } from "../test/library";
import { films, ultra } from "../test/profiles";
import { renderApp } from "../test/render-app";
import { fromFilms, workServer } from "../test/work-server";

const setting = () =>
  changeOp(
    "put",
    `*/api/v1/works/${orchardId}/wants`,
    () => ({ ...noImpact, works: 1, unwanted: { files: 1, bytes: 734_003_200 }, newlyWanted: 2 }),
    { wants: { ...fromFilms, profileId: ultra.id, profileName: ultra.name } },
  );

const sheet = () => page.getByRole("dialog", { name: "Profile and Editions" });
const editions = () => sheet().getByRole("heading", { level: 3 });

async function openSheet(op: ReturnType<typeof setting>) {
  renderApp({ path: `/works/${orchardId}`, handlers: [op.handler, ...workServer().handlers] });
  await expect.element(page.getByText("Profile Films. Editions from the Profile.")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Profile and Editions" }));
  await expect.element(sheet()).toBeVisible();
}

test("the Work's cells sit under their Edition, each Item with its status", async () => {
  renderApp({ path: `/works/${orchardId}`, handlers: workServer().handlers });
  await expect.element(page.getByRole("heading", { name: "Standard", level: 3 })).toBeVisible();
  await expect
    .element(
      page
        .getByRole("table", { name: "Standard cells" })
        .getByRole("row", { name: /^The Quiet Orchard /u }),
    )
    .toHaveTextContent("The Quiet OrchardMissingMissingMissing");
});

test("a Work moves to another Profile through the Impact, its Editions following the new Profile", async () => {
  const op = setting();
  await openSheet(op);
  const picker = sheet().getByRole("combobox", { name: "Profile" });
  await expect.element(picker).toHaveTextContent(`${films.name} (default)`);
  await expect.element(editions()).toHaveTextContent("Editions from Profile Films");
  await userEvent.click(picker);
  await userEvent.click(page.getByRole("option", { name: "Ultra" }));
  await expect.element(editions()).toHaveTextContent("Editions from Profile Ultra");

  await userEvent.click(sheet().getByRole("radio", { name: "Choose cuts" }));
  await expect.element(editions()).toHaveTextContent("Editions set for this Work");
  await userEvent.click(sheet().getByRole("button", { name: "Reset to Profile" }));
  await expect.element(editions()).toHaveTextContent("Editions from Profile Ultra");
  await expect.element(sheet().getByRole("radio", { name: "Any cut" })).toBeChecked();

  await userEvent.click(sheet().getByRole("button", { name: "Save", exact: true }));
  const impact = page.getByRole("dialog", { name: "Save The Quiet Orchard" });
  await expect
    .element(impact.getByRole("list", { name: "What changes" }))
    .toHaveTextContent("1 Work changes.1 file would fill nothing (700 MB).2 cells become wanted.");
  await userEvent.click(impact.getByRole("button", { name: "Save and delete 1 file" }));

  await expect.element(page.getByText("Saved what The Quiet Orchard wants.")).toBeVisible();
  await expect.element(sheet()).not.toBeInTheDocument();
  expect(op.sent).toEqual([
    { query: { preview: "true" }, body: { profileId: ultra.id, editions: null } },
    { query: { unwanted: "delete", confirm: "1" }, body: { profileId: ultra.id, editions: null } },
  ]);
});

test("Any cut set for this Work is sent as its own list", async () => {
  const op = setting();
  await openSheet(op);
  await userEvent.click(sheet().getByRole("radio", { name: "Choose cuts" }));
  await userEvent.click(sheet().getByRole("radio", { name: "Any cut" }));
  await expect.element(editions()).toHaveTextContent("Editions set for this Work");
  await userEvent.click(sheet().getByRole("button", { name: "Save", exact: true }));
  await expect.element(page.getByRole("dialog", { name: "Save The Quiet Orchard" })).toBeVisible();
  expect(op.sent).toEqual([
    { query: { preview: "true" }, body: { profileId: films.id, editions: { kind: "any" } } },
  ]);
});

test("chosen cuts for a Work are held back with the reason, and send nothing", async () => {
  const op = setting();
  await openSheet(op);
  await userEvent.click(sheet().getByRole("radio", { name: "Choose cuts" }));
  await userEvent.click(sheet().getByRole("button", { name: "Save", exact: true }));
  await expect
    .element(sheet().getByRole("alert"))
    .toHaveTextContent(
      "Chosen cuts can't be saved yet. Choose Any cut to save this Work's Editions.",
    );
  expect(op.sent).toEqual([]);
});
