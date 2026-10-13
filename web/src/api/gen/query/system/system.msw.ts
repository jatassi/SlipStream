import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { SystemHealth, SystemReport, SystemStatus } from "../../model";

export const getGetHealthResponseMock = (
  overrideResponse: Partial<Extract<SystemHealth, object>> = {},
): SystemHealth => ({ status: faker.helpers.arrayElement(["ok"] as const), ...overrideResponse });

export const getGetStatusResponseMock = (
  overrideResponse: Partial<Extract<SystemStatus, object>> = {},
): SystemStatus => ({
  claim: faker.helpers.arrayElement(["unclaimed", "controller", "worker"] as const),
  ...overrideResponse,
});

export const getGetSystemResponseMock = (
  overrideResponse: Partial<Extract<SystemReport, object>> = {},
): SystemReport => ({
  build: {
    ...(faker.datatype.boolean()
      ? { commit: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { fingerprint: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { libtorrent: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { nativeBundle: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    stamped: faker.datatype.boolean(),
    ...(faker.datatype.boolean()
      ? { target: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { version: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  dataDir: faker.string.alpha({ length: { min: 10, max: 20 } }),
  listeners: {
    control: faker.string.alpha({ length: { min: 10, max: 20 } }),
    http: faker.string.alpha({ length: { min: 10, max: 20 } }),
    link: faker.string.alpha({ length: { min: 10, max: 20 } }),
    torrent: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  pid: faker.number.int(),
  seed: faker.string.alpha({ length: { min: 10, max: 20 } }),
  startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  ...overrideResponse,
});

export const getGetHealthMockHandler = (
  overrideResponse?:
    | SystemHealth
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<SystemHealth> | SystemHealth),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/health",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetHealthResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetStatusMockHandler = (
  overrideResponse?:
    | SystemStatus
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<SystemStatus> | SystemStatus),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/status",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetStatusResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetSystemMockHandler = (
  overrideResponse?:
    | SystemReport
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<SystemReport> | SystemReport),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/system",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetSystemResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getSystemMock = () => [
  getGetHealthMockHandler(),
  getGetStatusMockHandler(),
  getGetSystemMockHandler(),
];
