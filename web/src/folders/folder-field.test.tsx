import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { useAppForm } from "../ui/form";
import { type FakeNode, renderQueried, serveFolders, unixFolder } from "../test/folders";
import { FolderField } from "./folder-field";
import type { FolderCheck } from "./verdict";

const media: FakeNode = {
  places: [],
  folders: {
    "/srv": unixFolder("/srv", ["media"]),
    "/srv/media": unixFolder("/srv/media", ["Anime"]),
    "/srv/media/Anime": unixFolder("/srv/media/Anime"),
  },
};

function judge() {
  const asked: string[] = [];
  const check: FolderCheck = ({ path }) => {
    asked.push(path);
    return Promise.resolve(
      path === "/srv"
        ? {
            usable: false,
            sentence: "Can't use this folder because it contains SlipStream's data folder.",
          }
        : { usable: true, sentence: `${path} is on /, 1.2 TB free.` },
    );
  };
  return { asked, check };
}

function RootForm({ check }: { check: FolderCheck }) {
  const form = useAppForm({ defaultValues: { path: "" } });
  return (
    <form.AppForm>
      <form.AppField name="path">
        {() => (
          <FolderField
            label="Folder"
            nodeId="media"
            nodeName="Media server"
            check={check}
            confirmVerb="Use"
            allowNewFolder
          />
        )}
      </form.AppField>
    </form.AppForm>
  );
}

const folder = () => page.getByRole("textbox", { name: "Folder" });

test("a typed path is checked on its Node once the typing pauses", async () => {
  serveFolders({ media });
  const { asked, check } = judge();
  renderQueried(<RootForm check={check} />);
  await userEvent.type(folder(), "/srv/media");
  await expect.element(page.getByText("/srv/media is on /, 1.2 TB free.")).toBeVisible();
  expect(asked).toEqual(["/srv/media"]);
  await userEvent.clear(folder());
  await userEvent.type(folder(), "/srv");
  await expect
    .element(page.getByText("Can't use this folder because it contains SlipStream's data folder."))
    .toBeVisible();
});

test("Browse opens on the field's value each time, and a pick fills the field and checks it", async () => {
  serveFolders({ media });
  const { check } = judge();
  renderQueried(<RootForm check={check} />);
  await userEvent.type(folder(), "/srv/media");
  await userEvent.click(page.getByRole("button", { name: "Browse" }));
  const dialog = page.getByRole("dialog", { name: "Choose a folder on Media server" });
  const here = dialog
    .getByRole("navigation", { name: "Folders above" })
    .getByRole("listitem")
    .last();
  await expect.element(here).toHaveTextContent("media");
  await userEvent.click(dialog.getByRole("option", { name: "Anime" }));
  await userEvent.click(dialog.getByRole("button", { name: "Use Anime" }));
  await expect.element(dialog).not.toBeInTheDocument();
  await expect.element(folder()).toHaveValue("/srv/media/Anime");
  await expect.element(page.getByText("/srv/media/Anime is on /, 1.2 TB free.")).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Browse" })).toHaveFocus();

  await userEvent.clear(folder());
  await userEvent.type(folder(), "/srv");
  await userEvent.click(page.getByRole("button", { name: "Browse" }));
  await expect.element(here).toHaveTextContent("srv");
});
