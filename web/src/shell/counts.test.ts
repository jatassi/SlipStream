import { expect, test } from "vitest";

import { activityStages } from "./counts";

test("Activity counts every stage but imported and failed", () => {
  expect(activityStages).toStrictEqual({
    fetching: true,
    downloading: true,
    probing: true,
    transferring: true,
    importing: true,
    imported: false,
    failed: false,
  });
});
