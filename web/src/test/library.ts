import type { WorksContentType, WorksTree } from "../api/gen/model";

export const cinema: WorksContentType = {
  id: "cinema",
  name: "Films",
  enabled: true,
  work: { one: "Film", many: "Films" },
  item: { one: "Film", many: "Films" },
  levels: [],
  fields: [
    { key: "year", label: "Year", tier: "work", kind: "number" },
    { key: "released", label: "Released", tier: "work", kind: "date" },
    { key: "runtime", label: "Runtime", tier: "work", kind: "duration" },
  ],
};

export const orchardId = "3c1e9a52-7b4d-4f0e-8a6c-2d5b9e1f7a03";
export const orchardItemId = "8e2b4d6f-1a3c-4e5f-9b7d-0c2e4a6b8d1f";

export const orchard: WorksTree = {
  id: orchardId,
  contentType: "cinema",
  key: "9000001",
  label: "The Quiet Orchard",
  sortLabel: "Quiet Orchard",
  addedAt: "2026-10-12T09:00:00Z",
  refreshedAt: "2026-10-12T09:00:00Z",
  fields: [
    { key: "year", kind: "number", number: 2021 },
    { key: "released", kind: "date", date: "2021-04-09" },
    { key: "runtime", kind: "duration", ms: 6_720_000 },
  ],
  levels: [],
  items: [
    {
      id: orchardItemId,
      levelId: null,
      key: "film",
      label: "The Quiet Orchard",
      position: 1,
      withdrawnAt: null,
      monitored: true,
      fields: [],
    },
  ],
};
