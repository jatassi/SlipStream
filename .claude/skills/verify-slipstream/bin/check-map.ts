#!/usr/bin/env bun
// check-map checks the structure of verify-slipstream's feature map and
// the parity rows it cites. It reads files and writes nothing.
import { readFileSync, readdirSync } from "node:fs";
import { join, relative, resolve } from "node:path";
import { exit, stdout } from "node:process";

const skill = resolve(import.meta.dirname, "..");
const repo = resolve(skill, "../../..");
const features = join(skill, "features");
const matrix = join(repo, "docs/parity-matrix.md");

const sections = [
  "Sub-features",
  "How to get to it (user POV)",
  "Driving it with ",
  "Gotchas",
] as const;

const shape = "Sub-features, How to get to it (user POV), Driving it with <harness>, Gotchas";

type Row = { disposition: string; status: string };

const problems: string[] = [];
const problem = (path: string, line: number, message: string) => {
  problems.push(`${relative(repo, path)}:${line}: ${message}`);
};

const linesOf = (path: string) => readFileSync(path, "utf8").split("\n");

function parityRows(): Map<string, Row> {
  const rows = new Map<string, Row>();
  for (const line of linesOf(matrix)) {
    const cells = line.split(" | ");
    const id = /^\| (P-\d+)$/u.exec(cells[0] ?? "")?.[1];
    if (id !== undefined) {
      rows.set(id, {
        disposition: cells[4] ?? "",
        status: (cells[6] ?? "").replace(/ \|$/u, ""),
      });
    }
  }
  return rows;
}

function checkIndex(files: readonly string[]): void {
  const readme = join(features, "README.md");
  const lines = linesOf(readme);
  const start = lines.indexOf("## Features");
  if (start === -1) {
    problem(readme, 1, "has no ## Features section");
    return;
  }
  const linked = new Set<string>();
  lines.slice(start + 1).forEach((line, i) => {
    for (const [, target] of line.matchAll(/\]\(\.\/([^)]+)\)/gu)) {
      if (target === undefined) {
        continue;
      }
      if (linked.has(target)) {
        problem(readme, start + 2 + i, `links ${target} twice`);
      }
      linked.add(target);
      if (!files.includes(target)) {
        problem(readme, start + 2 + i, `links ${target}, which does not exist`);
      }
    }
  });
  for (const file of files) {
    if (!linked.has(file)) {
      problem(readme, start + 1, `does not link ${file}`);
    }
  }
}

function checkFeature(path: string): { id: string; cites: string[]; line: number }[] {
  const lines = linesOf(path);
  if (lines[0]?.startsWith("# ") !== true) {
    problem(path, 1, "does not start with an H1");
  }
  const paragraph = lines.slice(1).find((line) => line.trim() !== "");
  if (paragraph === undefined || /^(#|-|\||```)/u.test(paragraph)) {
    problem(path, 2, "has no paragraph after its H1");
  }
  const h2s = lines.flatMap((line, i) =>
    line.startsWith("## ") ? [{ title: line.slice(3), i }] : [],
  );
  const titles = h2s.map((h) => h.title);
  const shaped =
    titles.length === sections.length &&
    sections.every((section, i) =>
      section.endsWith(" ") ? titles[i]?.startsWith(section) === true : titles[i] === section,
    );
  if (!shaped) {
    problem(path, (h2s[0]?.i ?? 0) + 1, `has H2s ${titles.join(", ")}; want ${shape}`);
  }
  const subStart = h2s.find((h) => h.title === "Sub-features")?.i;
  if (subStart === undefined) {
    return [];
  }
  const subEnd = h2s.find((h) => h.i > subStart)?.i ?? lines.length;
  const subs = lines.slice(subStart + 1, subEnd).flatMap((line, i) => {
    if (!line.startsWith("- ")) {
      return [];
    }
    const id = /^- `([a-z0-9][a-z0-9-]*)` /u.exec(line)?.[1];
    if (id === undefined) {
      problem(path, subStart + 2 + i, "is a sub-feature bullet without a `kebab-case-id`");
      return [];
    }
    return [
      { id, cites: [...line.matchAll(/P-\d{3}/gu)].map(([cite]) => cite), line: subStart + 2 + i },
    ];
  });
  if (subs.length === 0) {
    problem(path, subStart + 1, "lists no sub-features");
  }
  return subs;
}

const files = readdirSync(features)
  .filter((name) => name.endsWith(".md") && name !== "README.md")
  .toSorted();
checkIndex(files);

const rows = parityRows();
const seen = new Map<string, string>();
const cited = new Set<string>();
let count = 0;
for (const file of files) {
  const path = join(features, file);
  for (const sub of checkFeature(path)) {
    count++;
    const first = seen.get(sub.id);
    if (first !== undefined) {
      problem(path, sub.line, `repeats sub-feature id ${sub.id} from ${first}`);
    }
    seen.set(sub.id, file);
    for (const id of sub.cites) {
      const row = rows.get(id);
      if (row === undefined) {
        problem(path, sub.line, `cites ${id}, which docs/parity-matrix.md does not list`);
      } else if (/^(out of scope|excepted)/u.test(row.disposition)) {
        problem(path, sub.line, `cites ${id}, which is ${row.disposition}`);
      }
      cited.add(id);
    }
  }
}

const uncited = [...rows].filter(([id, row]) => row.status === "shipped" && !cited.has(id));
for (const [id] of uncited) {
  stdout.write(`shipped and cited by no sub-feature: ${id}\n`);
}
if (problems.length > 0) {
  stdout.write(`${problems.join("\n")}\n`);
  exit(1);
}
stdout.write(
  `check-map: ${files.length} feature files, ${count} sub-features, ${cited.size} parity rows cited\n`,
);
