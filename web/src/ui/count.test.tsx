import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Count } from "./count";

test("a count reads its number and the word its dot stands for", async () => {
  renderKit(
    <ul>
      <li>
        Activity <Count value={2} dot="live" />
      </li>
      <li>
        Health <Count value={1} dot="warn" />
      </li>
      <li>
        Imports <Count value={3} dot="fail" />
      </li>
      <li>
        Library <Count value={240} />
      </li>
    </ul>,
  );
  const rows = page.getByRole("listitem");
  await expect.element(rows.nth(0)).toHaveTextContent("Activity 2 live");
  await expect.element(rows.nth(1)).toHaveTextContent("Health 1 needs attention");
  await expect.element(rows.nth(2)).toHaveTextContent("Imports 3 failed");
  await expect.element(rows.nth(3)).toHaveTextContent("Library 240");
});
