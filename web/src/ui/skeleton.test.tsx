import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import type { ShellKind } from "./shell-context";
import { Skeleton } from "./skeleton";

test.each(["desktop", "phone"] satisfies ShellKind[])(
  "in the %s shell a skeleton is a status named for what loads",
  async (shell) => {
    renderKit(<Skeleton label="Loading the Library" />, { shell });
    await expect.element(page.getByRole("status", { name: "Loading the Library" })).toBeVisible();
  },
);
