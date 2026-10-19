import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import {
  browse,
  current,
  dialog,
  expectActive,
  field,
  option,
  place,
  useButton,
} from "../test/folder-browsing";

test("it opens on Places: the roots on the Node, then its drives with their free space", async () => {
  await browse();
  await expect
    .element(place("/srv/media/Movies"))
    .toHaveTextContent("/srv/media/MoviesLibrary root · 1.2 TB free");
  await expect.element(place("/mnt/nas")).toHaveTextContent("/mnt/nas3.0 TB free of 16 TBNetwork");
  await expect.element(page.getByText("Media server, 3 places")).toBeInTheDocument();
  await expect.element(useButton("Use")).toBeDisabled();
});

test("one click opens a folder, and the main button names the folder on show and uses it", async () => {
  await browse();
  await userEvent.click(page.getByRole("option", { name: /^\/ /u }));
  await userEvent.click(option("srv"));
  await userEvent.click(option("media"));
  await expect.element(current()).toHaveTextContent("media");
  await expect.element(page.getByText("media, 4 folders")).toBeInTheDocument();
  await userEvent.click(option("Anime"));
  await expect.element(page.getByText("Anime is on /, 1.2 TB free.")).toBeVisible();
  await expect.element(page.getByText("This folder has no folders inside.")).toBeVisible();
  await userEvent.click(useButton("Use Anime"));
  await expect
    .element(page.getByRole("status", { name: "Picked" }))
    .toHaveTextContent("/srv/media/Anime");
  await expect.element(dialog()).not.toBeInTheDocument();
});

test("a folder the check refuses keeps the main button off and says why", async () => {
  await browse({ startPath: "/srv/media/Movies" });
  await expect
    .element(page.getByText("Can't use this folder because it's inside the Movies library root."))
    .toBeVisible();
  await expect.element(useButton("Use Movies")).toBeDisabled();
  const described = useButton("Use Movies").element().getAttribute("aria-describedby") ?? "";
  expect(document.querySelector(`#${CSS.escape(described)}`)?.textContent).toBe(
    "Can't use this folder because it's inside the Movies library root.",
  );
});

test("the keys open, go up onto the folder left, use the folder and close", async () => {
  await browse({ startPath: "/srv/media" });
  await expect.element(field()).toHaveFocus();
  await expectActive("Anime");
  await userEvent.keyboard("{ArrowDown}{ArrowDown}");
  await expectActive("TV");
  await userEvent.keyboard("{ArrowUp}{Enter}");
  await expect.element(current()).toHaveTextContent("Movies");
  await userEvent.keyboard("{Backspace}");
  await expect.element(current()).toHaveTextContent("media");
  await expectActive("Movies");
  await userEvent.keyboard("{Alt>}{ArrowUp}{/Alt}");
  await expect.element(current()).toHaveTextContent("srv");
  await expectActive("media");
  await userEvent.keyboard("{ArrowRight}");
  await expectActive("Anime");
  await userEvent.keyboard("{ArrowRight}");
  await expect.element(current()).toHaveTextContent("Anime");
  await userEvent.keyboard("{Control>}{Enter}{/Control}");
  await expect
    .element(page.getByRole("status", { name: "Picked" }))
    .toHaveTextContent("/srv/media/Anime");
  await expect.element(dialog()).not.toBeInTheDocument();
});

test("Esc clears the field first, then closes the browser", async () => {
  await browse({ startPath: "/srv/media" });
  await userEvent.keyboard("an");
  await expect.element(field()).toHaveValue("an");
  await userEvent.keyboard("{Escape}");
  await expect.element(field()).toHaveValue("");
  await expect.element(dialog()).toBeVisible();
  await userEvent.keyboard("{Escape}");
  await expect.element(dialog()).not.toBeInTheDocument();
});

test("End and Home carry the active row through 5,000 folders", async () => {
  await browse({ startPath: "/srv/big" });
  await expectActive("Folder 1");
  await userEvent.keyboard("{End}");
  await expectActive("Folder 5000");
  await expect.element(option("Folder 5000")).toBeInViewport();
  await expect.element(option("Folder 5000")).toHaveAttribute("aria-setsize", "5000");
  await userEvent.keyboard("{Home}");
  await expectActive("Folder 1");
});

test("a name filters the folder, names starting with it first", async () => {
  await browse({ startPath: "/srv/media" });
  await userEvent.keyboard("m");
  await expect
    .poll(() =>
      page
        .getByRole("option")
        .elements()
        .map((row) => row.textContent),
    )
    .toEqual(["Movies", "Anime"]);
  await userEvent.keyboard("{Enter}");
  await expect.element(current()).toHaveTextContent("Movies");
});

test("a typed Unix path lists the Node's suggestions without touching the text, and Tab opens one", async () => {
  await browse({ startPath: "/srv" });
  await userEvent.keyboard("/srv/me");
  await expect.element(option("/srv/media")).toBeVisible();
  await expect.element(field()).toHaveValue("/srv/me");
  await userEvent.keyboard("{Tab}");
  await expect.element(current()).toHaveTextContent("media");
});

test("a typed Windows path keeps its separators, and Enter goes to the folder the Node cleans it to", async () => {
  await browse({ nodeId: "windows" });
  await expect
    .element(
      page.getByText(
        "Mapped drive letters aren't visible to SlipStream when it runs as a service.",
        { exact: false },
      ),
    )
    .toBeVisible();
  await userEvent.keyboard("D:\\Me");
  await expect.element(option("D:\\Media")).toBeVisible();
  await userEvent.clear(field());
  await userEvent.keyboard("d:/me");
  await expect.element(option("d:/Media")).toBeVisible();
  await userEvent.clear(field());
  await userEvent.keyboard("\\\\nas");
  await expect
    .element(page.getByText("Type a share name after the server", { exact: false }))
    .toBeVisible();
  await userEvent.clear(field());
  await userEvent.keyboard("d:/media{Enter}");
  await expect.element(current()).toHaveTextContent("Media");
  await expect
    .element(page.getByRole("navigation", { name: "Folders above" }))
    .toHaveTextContent("Media serverD:\\Media");
  await expect.element(useButton("Use Media")).toBeEnabled();
});

test("a long path folds its middle breadcrumbs into a menu", async () => {
  await browse({ startPath: "/srv/media/TV/Frieren" });
  const crumbs = page.getByRole("navigation", { name: "Folders above" });
  await expect.element(crumbs).toHaveTextContent("Media server/…TVFrieren");
  await userEvent.click(crumbs.getByRole("button", { name: "More folders above" }));
  await userEvent.click(page.getByRole("menuitem", { name: "media" }));
  await expect.element(current()).toHaveTextContent("media");
});
