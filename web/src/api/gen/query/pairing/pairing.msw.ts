import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { PairingWorker } from "../../model";

export const getClaimWorkerResponseMock = (
  overrideResponse: Partial<Extract<PairingWorker, object>> = {},
): PairingWorker => ({
  controller: {
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    link: faker.string.alpha({ length: { min: 10, max: 20 } }),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  nodeId: faker.string.uuid(),
  ...overrideResponse,
});

export const getClaimWorkerMockHandler = (
  overrideResponse?:
    | PairingWorker
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<PairingWorker> | PairingWorker),
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
export const getPairingMock = () => [getClaimWorkerMockHandler()];
