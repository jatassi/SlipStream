import { expect, test } from "vitest";

import type { WorksOutcome, WorksSkippedOutcomeReason } from "../../api/gen/model";
import { film } from "../../test/actions";
import { cinema } from "../../test/library";
import type { Subject, WorkAction } from "./actions";
import { outcomeWords } from "./outcomes";

const [orchard, second, third, fourth, fifth] = [film(1), film(2), film(3), film(4), film(5)];

const done = ({ id }: Subject): WorksOutcome => ({ workId: id, outcome: "done" });
const skipped = ({ id }: Subject, reason: WorksSkippedOutcomeReason): WorksOutcome => ({
  workId: id,
  outcome: "skipped",
  reason,
});
const failed = ({ id }: Subject, code: string, title: string, detail?: string): WorksOutcome => ({
  workId: id,
  outcome: "failed",
  problem: { code, title, ...(detail !== undefined && { detail }) },
});

const say = (action: WorkAction, outcomes: WorksOutcome[], subjects: Subject[], files = 1) =>
  outcomeWords(
    action,
    { outcomes, plan: { works: subjects.length, files, kept: 0, bytes: 0, downloading: 0 } },
    subjects,
  );

const actionCases: { action: WorkAction; done: string; already: string }[] = [
  {
    action: { kind: "monitor", on: true },
    done: "Monitored The Quiet Orchard.",
    already: "The Quiet Orchard was already monitored.",
  },
  {
    action: { kind: "monitor", on: false },
    done: "Stopped monitoring The Quiet Orchard.",
    already: "The Quiet Orchard wasn't monitored.",
  },
  {
    action: { kind: "profile", profileId: "p", unwanted: "delete", confirm: 0 },
    done: "Changed the Profile of The Quiet Orchard.",
    already: "The Quiet Orchard already has that Profile.",
  },
  {
    action: { kind: "root", rootId: "r" },
    done: "Set the library root of The Quiet Orchard.",
    already: "The Quiet Orchard is already in that library root.",
  },
  {
    action: { kind: "refresh" },
    done: "Refreshed the metadata of The Quiet Orchard.",
    already: "The Quiet Orchard was already being refreshed.",
  },
  {
    action: { kind: "delete", files: "delete" },
    done: "Deleted The Quiet Orchard, with its file.",
    already: "The Quiet Orchard was already deleted.",
  },
  {
    action: { kind: "delete", files: "leave" },
    done: "Deleted The Quiet Orchard. Its file stays on disk.",
    already: "The Quiet Orchard was already deleted.",
  },
];

test.each(actionCases)("$action.kind says what it did and what already read that way", (each) => {
  expect(say(each.action, [done(orchard)], [orchard])).toEqual({ done: each.done, failed: [] });
  expect(say(each.action, [skipped(orchard, "already")], [orchard])).toEqual({
    done: each.already,
    failed: [],
  });
});

test("deleting Works that have no files names no files", () => {
  for (const files of ["delete", "leave"] as const) {
    expect(
      say({ kind: "delete", files }, [done(orchard), done(second)], [orchard, second], 0),
    ).toEqual({
      done: `Deleted ${orchard.label} and ${second.label}.`,
      failed: [],
    });
  }
});

const both = [orchard, second];

const deleteCases: [string, WorkAction, Subject[], number][] = [
  ["Deleted The Quiet Orchard, with its files.", { kind: "delete", files: "delete" }, [orchard], 3],
  [
    `Deleted ${orchard.label} and Film 2, with their one file.`,
    { kind: "delete", files: "delete" },
    both,
    1,
  ],
  [
    `Deleted ${orchard.label} and Film 2, with their files.`,
    { kind: "delete", files: "delete" },
    both,
    4,
  ],
  [
    "Deleted The Quiet Orchard. Its files stay on disk.",
    { kind: "delete", files: "leave" },
    [orchard],
    3,
  ],
  [
    `Deleted ${orchard.label} and Film 2. Their one file stays on disk.`,
    { kind: "delete", files: "leave" },
    both,
    1,
  ],
  [
    `Deleted ${orchard.label} and Film 2. Their files stay on disk.`,
    { kind: "delete", files: "leave" },
    both,
    4,
  ],
];

test.each(deleteCases)("%s", (words, action, subjects, files) => {
  expect(
    say(
      action,
      subjects.map((subject) => done(subject)),
      subjects,
      files,
    ).done,
  ).toBe(words);
});

test("every other skip reason has its words", () => {
  const refresh: WorkAction = { kind: "refresh" };
  const skips: [WorksSkippedOutcomeReason, string][] = [
    [
      "work_has_files",
      "The Quiet Orchard has files, so it stays in its library root. Moving a Work that has files isn't supported yet.",
    ],
    ["work_gone_upstream", "TMDB no longer lists The Quiet Orchard, so it wasn't refreshed."],
    ["work_not_found", "The Quiet Orchard is no longer in the library."],
  ];
  for (const [reason, words] of skips) {
    expect(say(refresh, [skipped(orchard, reason)], [orchard]).done).toBe(words);
  }
});

test("a bulk action counts what it did, then names what it skipped", () => {
  const picked = [orchard, second, third, fourth, fifth];
  expect(
    say(
      { kind: "monitor", on: true },
      [skipped(orchard, "already"), done(second), done(third), done(fourth), done(fifth)],
      picked,
    ),
  ).toEqual({ done: "Monitored 4 Films. The Quiet Orchard was already monitored.", failed: [] });
});

test("a few Works alike are named together, and Works with files stay where they are", () => {
  expect(
    say(
      { kind: "root", rootId: "r" },
      [done(orchard), skipped(second, "work_has_files"), skipped(third, "work_has_files")],
      [orchard, second, third],
    ).done,
  ).toBe(
    "Set the library root of The Quiet Orchard. Film 2 and Film 3 have files, so they stay in their library roots. Moving a Work that has files isn't supported yet.",
  );
});

test("a failure says what happened and what to do, once for Works that failed alike", () => {
  const offline =
    "seedbox is offline, so its files can't be deleted. Delete it and leave its files, or try again once seedbox is back.";
  expect(
    say(
      { kind: "delete", files: "delete" },
      [
        failed(orchard, "storage_unreachable", "A Node holding its files is offline", offline),
        failed(second, "storage_unreachable", "A Node holding its files is offline", offline),
        failed(third, "metadata_unavailable", "TMDB isn't responding"),
      ],
      [orchard, second, third],
    ),
  ).toEqual({
    done: undefined,
    failed: [
      `Couldn't delete The Quiet Orchard and Film 2. ${offline}`,
      "Couldn't delete Film 3. TMDB isn't responding.",
    ],
  });
});

test("Works of several types are counted as Works", () => {
  const zine: Subject = {
    ...fifth,
    type: { ...cinema, id: "zines", work: { one: "Zine", many: "Zines" } },
  };
  const picked = [orchard, second, third, zine];
  expect(
    say(
      { kind: "refresh" },
      picked.map((subject) => done(subject)),
      picked,
    ).done,
  ).toBe("Refreshed the metadata of 4 Works.");
});
