import type { NodesNode, SystemReport } from "../api/gen/model";
import { getListNodesMockHandler } from "../api/gen/query/nodes/nodes.msw";

export const homeId = "4a2c6e8f-1b3d-4f5a-9c7e-2d4f6a8b0c1e";
export const seedboxId = "9e7c5a3f-1d2b-4c6e-8f0a-3b5d7f9a1c2e";

const build = { version: "0.4.0", os: "linux", arch: "amd64", installKind: "deb" };

export const home: NodesNode = {
  id: homeId,
  name: "home",
  self: true,
  addresses: ["127.0.0.2:8485"],
  intentVersion: "0123456789abcdef0123456789abcdef",
  key: "SHA256:home",
  link: { state: "local", since: "2026-10-12T09:00:00Z", build },
  dial: { controllerDials: [] },
  pairedAt: "2026-10-12T09:00:00Z",
  roles: [
    { role: "searching", state: "holding" },
    { role: "downloading", state: "holding" },
  ],
};

export const seedbox: NodesNode = {
  id: seedboxId,
  name: "seedbox",
  self: false,
  addresses: ["127.0.0.3:8485"],
  intentVersion: "fedcba9876543210fedcba9876543210",
  key: "SHA256:seedbox",
  link: { state: "down", since: "2026-10-12T10:00:00Z" },
  dial: { controllerDials: [] },
  pairedAt: "2026-10-12T09:30:00Z",
  roles: [{ role: "downloading", state: "away" }],
};

export const nodesAre = (...nodes: NodesNode[]) => getListNodesMockHandler({ nodes });

export const homeSystem: SystemReport = {
  build: { stamped: true, version: "0.0.0-dev+3f2a1c9" },
  dataDir: "/data/home",
  http: { configured: "127.0.0.2:8080" },
  listeners: {
    http: "127.0.0.2:8080",
    torrent: "127.0.0.2:6881",
    link: "127.0.0.2:8485",
    control: "/data/home/control.sock",
  },
  logsDir: "/data/home/logs",
  pid: 4242,
  seed: "home",
  startedAt: "2026-10-12T09:00:00Z",
};
