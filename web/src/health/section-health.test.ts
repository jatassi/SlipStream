import { expect, test } from "vitest";

import { awayNode, failedTask, fillingVolume, fullVolume, reportOf } from "../test/health";
import { sectionHealth } from "./section-health";

test("a section whose categories have nothing wrong has no health to show", () => {
  expect(sectionHealth(reportOf({ nodes: [awayNode] }), ["storage"])).toBeUndefined();
});

test("a section counts only its own categories and leads with an error over an older warning", () => {
  const report = reportOf({ storage: [fillingVolume, fullVolume], tasks: [failedTask] });
  expect(sectionHealth(report, ["storage"])).toEqual({
    level: "fail",
    words: "1 error, 1 warning",
    first: { category: "storage", item: fullVolume },
  });
});

test("warnings alone read as warnings, counted in words", () => {
  const second = { ...fillingVolume, key: "v.home.other" };
  expect(sectionHealth(reportOf({ storage: [fillingVolume, second] }), ["storage"])).toMatchObject({
    level: "warn",
    words: "2 warnings",
    first: { item: fillingVolume },
  });
});
