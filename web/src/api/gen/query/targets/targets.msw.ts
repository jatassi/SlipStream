import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  TargetsProfile,
  TargetsProfileList,
  TargetsTarget,
  TargetsTargetList,
} from "../../model";

export const getListProfilesResponseMock = (
  overrideResponse: Partial<Extract<TargetsProfileList, object>> = {},
): TargetsProfileList => ({
  profiles: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
      default: faker.datatype.boolean(),
      id: faker.string.uuid(),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      targetIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.uuid(),
      ),
    }),
  ),
  ...overrideResponse,
});

export const getCreateProfileResponseMock = (
  overrideResponse: Partial<Extract<TargetsProfile, object>> = {},
): TargetsProfile => ({
  contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
  default: faker.datatype.boolean(),
  id: faker.string.uuid(),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  targetIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.uuid(),
  ),
  ...overrideResponse,
});

export const getListTargetsResponseMock = (
  overrideResponse: Partial<Extract<TargetsTargetList, object>> = {},
): TargetsTargetList => ({
  targets: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      id: faker.string.uuid(),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      resolutions: faker.helpers.arrayElements(["2160p", "1080p", "720p", "576p", "480p"] as const),
    }),
  ),
  ...overrideResponse,
});

export const getCreateTargetResponseMock = (
  overrideResponse: Partial<Extract<TargetsTarget, object>> = {},
): TargetsTarget => ({
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  id: faker.string.uuid(),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  resolutions: faker.helpers.arrayElements(["2160p", "1080p", "720p", "576p", "480p"] as const),
  ...overrideResponse,
});

export const getListProfilesMockHandler = (
  overrideResponse?:
    | TargetsProfileList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TargetsProfileList> | TargetsProfileList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/profiles",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListProfilesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCreateProfileMockHandler = (
  overrideResponse?:
    | TargetsProfile
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<TargetsProfile> | TargetsProfile),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/profiles",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCreateProfileResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getListTargetsMockHandler = (
  overrideResponse?:
    | TargetsTargetList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TargetsTargetList> | TargetsTargetList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/targets",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListTargetsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCreateTargetMockHandler = (
  overrideResponse?:
    | TargetsTarget
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<TargetsTarget> | TargetsTarget),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/targets",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCreateTargetResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};
export const getTargetsMock = () => [
  getListProfilesMockHandler(),
  getCreateProfileMockHandler(),
  getListTargetsMockHandler(),
  getCreateTargetMockHandler(),
];
