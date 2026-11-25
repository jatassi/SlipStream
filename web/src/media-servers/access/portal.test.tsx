import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { MediaserversMyAccessRow as MyRow } from "../../api/gen/model";
import { accountLists } from "../../test/account";
import { signedInAsGrace } from "../../test/handlers";
import { mediaServerAccessHandlers, myAccess, myRows } from "../../test/media-server-access";
import { renderApp } from "../../test/render-app";

const { plexWaiting, jellyfinOffered, jellyfinActive, plexActive } = myRows;

const handlersFor = (rows: MyRow[]) => [
  signedInAsGrace,
  ...accountLists(),
  ...mediaServerAccessHandlers({ ...myAccess, rows }),
];

const regionNames = () =>
  page
    .getByRole("region", { name: "Where to watch" })
    .getByRole("heading", { level: 3 })
    .all()
    .map((heading) => heading.element().textContent);

test.each<[string, MyRow[], string[]]>([
  ["nothing ready", [plexWaiting, jellyfinOffered], []],
  ["one ready, one pending", [plexWaiting, jellyfinActive], [jellyfinActive.server.name]],
  ["all ready", [plexActive, jellyfinActive], [plexActive.server.name, jellyfinActive.server.name]],
])(
  "with %s, Home shows the Watch card while a row is pending, and Your Account lists where to watch",
  async (_, rows, ready) => {
    const { router } = renderApp({ path: "/portal", handlers: handlersFor(rows) });
    await expect.element(page.getByRole("heading", { name: "Home", level: 1 })).toBeVisible();
    const card = page.getByRole("region", { name: "Watch" });
    if (rows.some((row) => row.pending)) {
      await expect.element(card).toBeVisible();
    } else {
      await expect.element(card).not.toBeInTheDocument();
    }

    await router.navigate({ to: "/portal/account" });
    await expect
      .element(page.getByRole("heading", { name: "Your Account", level: 1 }))
      .toBeVisible();
    if (ready.length === 0) {
      await expect
        .element(page.getByRole("region", { name: "Where to watch" }))
        .not.toBeInTheDocument();
    } else {
      await expect.element(page.getByRole("region", { name: "Where to watch" })).toBeVisible();
      expect(regionNames()).toEqual(ready);
    }
  },
);

test("the Watch card opens the Watch step, which goes back to Home", async () => {
  const { router } = renderApp({ path: "/portal", handlers: handlersFor([plexWaiting]) });
  await userEvent.click(
    page.getByRole("region", { name: "Watch" }).getByRole("button", { name: "See what's next" }),
  );
  await expect.poll(() => router.state.location.pathname).toBe("/welcome/watch");
  await expect.element(page.getByRole("region", { name: plexWaiting.server.name })).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Skip for now" })).not.toBeInTheDocument();
  await userEvent.click(page.getByRole("button", { name: "Back to Home" }));
  await expect.poll(() => router.state.location.pathname).toBe("/portal");
});
