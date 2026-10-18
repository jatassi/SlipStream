import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { PairingWorkerClaim } from "../../model";

export const getGetWorkerClaimResponseMock = (
  overrideResponse: Partial<Extract<PairingWorkerClaim, object>> = {},
): PairingWorkerClaim => ({
  ...(faker.datatype.boolean()
    ? {
        controller: {
          key: faker.string.alpha({ length: { min: 10, max: 20 } }),
          link: faker.string.alpha({ length: { min: 10, max: 20 } }),
          name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        },
      }
    : {}),
  ...(faker.datatype.boolean()
    ? { nodeAddress: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
  ...(faker.datatype.boolean() ? { nodeId: faker.string.uuid() } : {}),
  ...(faker.datatype.boolean()
    ? {
        problem: {
          code: faker.helpers.arrayElement([
            "controller_unreachable",
            "join_code_unknown",
            "join_code_expired",
            "join_code_used",
            "controller_key_mismatch",
            "internal",
          ] as const),
          detail: faker.string.alpha({ length: { min: 10, max: 20 } }),
        },
      }
    : {}),
  state: faker.helpers.arrayElement(["awaiting-dial", "joined", "failed"] as const),
  ...(faker.datatype.boolean()
    ? { until: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  ...overrideResponse,
});

export const getClaimWorkerResponseMock = (
  overrideResponse: Partial<Extract<PairingWorkerClaim, object>> = {},
): PairingWorkerClaim =>
  faker.helpers.arrayElement([
    {
      ...(faker.datatype.boolean()
        ? {
            controller: {
              key: faker.string.alpha({ length: { min: 10, max: 20 } }),
              link: faker.string.alpha({ length: { min: 10, max: 20 } }),
              name: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
          }
        : {}),
      ...(faker.datatype.boolean()
        ? { nodeAddress: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean() ? { nodeId: faker.string.uuid() } : {}),
      ...(faker.datatype.boolean()
        ? {
            problem: {
              code: faker.helpers.arrayElement([
                "controller_unreachable",
                "join_code_unknown",
                "join_code_expired",
                "join_code_used",
                "controller_key_mismatch",
                "internal",
              ] as const),
              detail: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
          }
        : {}),
      state: faker.helpers.arrayElement(["awaiting-dial", "joined", "failed"] as const),
      ...(faker.datatype.boolean()
        ? { until: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      ...overrideResponse,
    },
    {
      ...(faker.datatype.boolean()
        ? {
            controller: {
              key: faker.string.alpha({ length: { min: 10, max: 20 } }),
              link: faker.string.alpha({ length: { min: 10, max: 20 } }),
              name: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
          }
        : {}),
      ...(faker.datatype.boolean()
        ? { nodeAddress: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean() ? { nodeId: faker.string.uuid() } : {}),
      ...(faker.datatype.boolean()
        ? {
            problem: {
              code: faker.helpers.arrayElement([
                "controller_unreachable",
                "join_code_unknown",
                "join_code_expired",
                "join_code_used",
                "controller_key_mismatch",
                "internal",
              ] as const),
              detail: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
          }
        : {}),
      state: faker.helpers.arrayElement(["awaiting-dial", "joined", "failed"] as const),
      ...(faker.datatype.boolean()
        ? { until: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      ...overrideResponse,
    },
  ]);

export const getGetWorkerClaimMockHandler = (
  overrideResponse?:
    | PairingWorkerClaim
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<PairingWorkerClaim> | PairingWorkerClaim),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/setup/worker",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetWorkerClaimResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getClaimWorkerMockHandler = (
  overrideResponse?:
    | PairingWorkerClaim
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<PairingWorkerClaim> | PairingWorkerClaim),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/setup/worker",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getClaimWorkerResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};
export const getPairingMock = () => [getGetWorkerClaimMockHandler(), getClaimWorkerMockHandler()];
