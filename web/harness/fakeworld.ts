import { execFile } from "node:child_process";
import { join } from "node:path";
import { platform } from "node:process";

import { z } from "zod";

import { type ManifestNode, manifestNode } from "./manifest.ts";

const eventRecord = z.strictObject({
  seq: z.int(),
  at: z.string(),
  kind: z.string(),
  node: z.optional(z.string()),
  data: z.optional(z.unknown()),
  files: z.optional(z.record(z.string(), z.string())),
});

export type EventRecord = z.infer<typeof eventRecord>;

const doctorReport = z.strictObject({
  ok: z.boolean(),
  checks: z.array(
    z.strictObject({
      n: z.int(),
      name: z.string(),
      ok: z.boolean(),
      failures: z.optional(z.array(z.string())),
      notes: z.optional(z.array(z.string())),
    }),
  ),
  activeFaults: z.array(
    z.strictObject({
      seq: z.int(),
      id: z.string(),
      target: z.string(),
      origin: z.optional(z.string()),
      fault: z.string(),
    }),
  ),
});

export type DoctorReport = z.infer<typeof doctorReport>;

export type FaultRule = {
  id: string;
  target: string;
  node?: string;
  fault: string;
  route?: string;
  params?: Record<string, string>;
  skip?: number;
  times?: number;
  after?: { kind: string; match?: Record<string, string> };
};

export type CaptureQuery = {
  node?: string;
  where?: Record<string, string>;
  after?: number;
};

export type NodeAction = "stop" | "kill" | "start" | "wipe";

export type Fakeworld = ReturnType<typeof fw>;

const maxOutput = 64 << 20;

function run(
  runDir: string,
  args: readonly string[],
  exits: readonly number[] = [0],
): Promise<string> {
  const exe = join(runDir, "bin", platform === "win32" ? "fakeworld.exe" : "fakeworld");
  return new Promise((resolve, reject) => {
    execFile(exe, args, { maxBuffer: maxOutput }, (error, stdout, stderr) => {
      if (error === null || (typeof error.code === "number" && exits.includes(error.code))) {
        resolve(stdout);
        return;
      }
      reject(new Error(`fakeworld ${args.join(" ")}: ${stderr.trim() || error.message}`));
    });
  });
}

function lines(stdout: string): string[] {
  return stdout.split("\n").filter((line) => line !== "");
}

function records(stdout: string): EventRecord[] {
  return lines(stdout).map((line) => eventRecord.parse(JSON.parse(line)));
}

function one(stdout: string): EventRecord {
  const [record, ...rest] = records(stdout);
  if (record === undefined || rest.length > 0) {
    throw new Error(`fakeworld printed ${stdout}, want one record`);
  }
  return record;
}

const pairs = (flag: string, values: Record<string, string> = {}) =>
  Object.entries(values).flatMap(([k, v]) => [flag, `${k}=${v}`]);

const optional = (flag: string, value: string | number | undefined) =>
  value === undefined ? [] : [flag, String(value)];

const timeoutFlag = (ms: number | undefined) =>
  optional("--timeout", ms === undefined ? undefined : `${ms}ms`);

function ruleArgs(rule: FaultRule): string[] {
  const after =
    rule.after === undefined
      ? undefined
      : [
          rule.after.kind,
          ...Object.entries(rule.after.match ?? {}).map(([k, v]) => `${k}=${v}`),
        ].join(",");
  return [
    rule.id,
    rule.node === undefined ? rule.target : `${rule.target}@${rule.node}`,
    rule.fault,
    ...optional("--route", rule.route),
    ...pairs("--param", rule.params),
    ...optional("--skip", rule.skip),
    ...optional("--times", rule.times),
    ...optional("--after", after),
  ];
}

function queryArgs(q: CaptureQuery): string[] {
  return [
    ...optional("--node", q.node),
    ...pairs("--where", q.where),
    ...optional("--after", q.after),
  ];
}

function reader(runDir: string, verb: string, fixed: readonly string[] = []) {
  return {
    list: async (q: CaptureQuery = {}) =>
      records(await run(runDir, [verb, "list", runDir, ...fixed, ...queryArgs(q)])),
    wait: async (q: CaptureQuery & { timeout?: number } = {}) =>
      one(
        await run(runDir, [
          verb,
          "wait",
          runDir,
          ...fixed,
          ...queryArgs(q),
          ...timeoutFlag(q.timeout),
        ]),
      ),
    open: async (seq: number, proof?: string) =>
      lines(await run(runDir, [verb, "open", runDir, String(seq), ...optional("--proof", proof)])),
  };
}

// fw drives the run in runDir through the run's own fakeworld copy, so it
// needs no client for the run's control socket. Each call is a journaled
// step of the run.
export function fw(runDir: string) {
  return {
    fault: {
      set: async (rule: FaultRule) =>
        one(await run(runDir, ["fault", "set", runDir, ...ruleArgs(rule)])),
      clear: async (id?: string) =>
        one(await run(runDir, ["fault", "clear", runDir, ...(id === undefined ? [] : [id])])),
      list: async () => records(await run(runDir, ["fault", "list", runDir])),
      wait: async (q: { rule?: string; node?: string; after?: number; timeout?: number } = {}) =>
        one(
          await run(runDir, [
            "fault",
            "wait",
            runDir,
            ...optional("--rule", q.rule),
            ...optional("--node", q.node),
            ...optional("--after", q.after),
            ...timeoutFlag(q.timeout),
          ]),
        ),
    },
    cue: async (name: string, args?: unknown) =>
      one(
        await run(runDir, [
          "cue",
          runDir,
          name,
          ...(args === undefined ? [] : [JSON.stringify(args)]),
        ]),
      ),
    node: async (action: NodeAction, name: string): Promise<ManifestNode> =>
      manifestNode.parse(JSON.parse(await run(runDir, ["node", action, runDir, name]))),
    capture: (name: string) => reader(runDir, name),
    sse: (as: string) => reader(runDir, "sse", ["--as", as]),
    // doctor resolves with the report of a run that fails a check too.
    doctor: async (): Promise<DoctorReport> =>
      doctorReport.parse(JSON.parse(await run(runDir, ["doctor", runDir, "--json"], [0, 1]))),
  };
}
