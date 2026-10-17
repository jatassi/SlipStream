import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  TargetsProfile,
  TargetsProfileList,
  TargetsQualityList,
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

export const getListQualitiesResponseMock = (
  overrideResponse: Partial<Extract<TargetsQualityList, object>> = {},
): TargetsQualityList => ({
  presets: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      key: faker.helpers.arrayElement([
        "disc-or-resolution",
        "anything-better",
        "resolution-only",
      ] as const),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      tiers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () =>
          Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
            faker.string.alpha({ length: { min: 10, max: 20 } }),
          ),
      ),
    }),
  ),
  qualities: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      cinema: faker.datatype.boolean(),
      floor: faker.number.float({ min: 0, fractionDigits: 2 }),
      key: faker.helpers.arrayElement([
        "sdtv",
        "dvd",
        "webrip-480p",
        "hdtv-720p",
        "webrip-720p",
        "webdl-720p",
        "bluray-720p",
        "hdtv-1080p",
        "webrip-1080p",
        "webdl-1080p",
        "bluray-1080p",
        "remux-1080p",
        "hdtv-2160p",
        "webrip-2160p",
        "webdl-2160p",
        "bluray-2160p",
        "remux-2160p",
        "cam",
      ] as const),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      resolution: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["2160p", "1080p", "720p", "480p"] as const),
        null,
      ]),
      source: faker.helpers.arrayElement([
        "tv",
        "hdtv",
        "dvd",
        "webrip",
        "webdl",
        "bluray",
        "remux",
        "cinema",
      ] as const),
    }),
  ),
  ...overrideResponse,
});

export const getListTargetsResponseMock = (
  overrideResponse: Partial<Extract<TargetsTargetList, object>> = {},
): TargetsTargetList => ({
  targets: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      contains: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.uuid(),
      ),
      createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      cutoff: faker.string.alpha({ length: { min: 10, max: 20 } }),
      filters: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          attribute: faker.helpers.arrayElement([
            "hdr",
            "video_codec",
            "audio_codec",
            "channels",
            "language",
          ] as const),
          mode: faker.helpers.arrayElement(["only", "not"] as const),
          values: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
        }),
      ),
      id: faker.string.uuid(),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      profiles: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          id: faker.string.uuid(),
          name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      sizes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          max: faker.helpers.arrayElement([
            faker.number.float({ min: 0, fractionDigits: 2 }),
            null,
          ]),
          min: faker.number.float({ min: 0, fractionDigits: 2 }),
          quality: faker.helpers.arrayElement([
            "sdtv",
            "dvd",
            "webrip-480p",
            "hdtv-720p",
            "webrip-720p",
            "webdl-720p",
            "bluray-720p",
            "hdtv-1080p",
            "webrip-1080p",
            "webdl-1080p",
            "bluray-1080p",
            "remux-1080p",
            "hdtv-2160p",
            "webrip-2160p",
            "webdl-2160p",
            "bluray-2160p",
            "remux-2160p",
            "cam",
          ] as const),
        }),
      ),
      tiers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () =>
          Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
            faker.string.alpha({ length: { min: 10, max: 20 } }),
          ),
      ),
    }),
  ),
  ...overrideResponse,
});

export const getCreateTargetResponseMock = (
  overrideResponse: Partial<Extract<TargetsTarget, object>> = {},
): TargetsTarget => ({
  contains: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.uuid(),
  ),
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  cutoff: faker.string.alpha({ length: { min: 10, max: 20 } }),
  filters: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      attribute: faker.helpers.arrayElement([
        "hdr",
        "video_codec",
        "audio_codec",
        "channels",
        "language",
      ] as const),
      mode: faker.helpers.arrayElement(["only", "not"] as const),
      values: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    }),
  ),
  id: faker.string.uuid(),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  profiles: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({ id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) }),
  ),
  sizes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    max: faker.helpers.arrayElement([faker.number.float({ min: 0, fractionDigits: 2 }), null]),
    min: faker.number.float({ min: 0, fractionDigits: 2 }),
    quality: faker.helpers.arrayElement([
      "sdtv",
      "dvd",
      "webrip-480p",
      "hdtv-720p",
      "webrip-720p",
      "webdl-720p",
      "bluray-720p",
      "hdtv-1080p",
      "webrip-1080p",
      "webdl-1080p",
      "bluray-1080p",
      "remux-1080p",
      "hdtv-2160p",
      "webrip-2160p",
      "webdl-2160p",
      "bluray-2160p",
      "remux-2160p",
      "cam",
    ] as const),
  })),
  tiers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  ),
  ...overrideResponse,
});

export const getGetTargetResponseMock = (
  overrideResponse: Partial<Extract<TargetsTarget, object>> = {},
): TargetsTarget => ({
  contains: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.uuid(),
  ),
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  cutoff: faker.string.alpha({ length: { min: 10, max: 20 } }),
  filters: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      attribute: faker.helpers.arrayElement([
        "hdr",
        "video_codec",
        "audio_codec",
        "channels",
        "language",
      ] as const),
      mode: faker.helpers.arrayElement(["only", "not"] as const),
      values: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    }),
  ),
  id: faker.string.uuid(),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  profiles: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({ id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) }),
  ),
  sizes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    max: faker.helpers.arrayElement([faker.number.float({ min: 0, fractionDigits: 2 }), null]),
    min: faker.number.float({ min: 0, fractionDigits: 2 }),
    quality: faker.helpers.arrayElement([
      "sdtv",
      "dvd",
      "webrip-480p",
      "hdtv-720p",
      "webrip-720p",
      "webdl-720p",
      "bluray-720p",
      "hdtv-1080p",
      "webrip-1080p",
      "webdl-1080p",
      "bluray-1080p",
      "remux-1080p",
      "hdtv-2160p",
      "webrip-2160p",
      "webdl-2160p",
      "bluray-2160p",
      "remux-2160p",
      "cam",
    ] as const),
  })),
  tiers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  ),
  ...overrideResponse,
});

export const getDuplicateTargetResponseMock = (
  overrideResponse: Partial<Extract<TargetsTarget, object>> = {},
): TargetsTarget => ({
  contains: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.uuid(),
  ),
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  cutoff: faker.string.alpha({ length: { min: 10, max: 20 } }),
  filters: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      attribute: faker.helpers.arrayElement([
        "hdr",
        "video_codec",
        "audio_codec",
        "channels",
        "language",
      ] as const),
      mode: faker.helpers.arrayElement(["only", "not"] as const),
      values: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    }),
  ),
  id: faker.string.uuid(),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  profiles: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({ id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) }),
  ),
  sizes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    max: faker.helpers.arrayElement([faker.number.float({ min: 0, fractionDigits: 2 }), null]),
    min: faker.number.float({ min: 0, fractionDigits: 2 }),
    quality: faker.helpers.arrayElement([
      "sdtv",
      "dvd",
      "webrip-480p",
      "hdtv-720p",
      "webrip-720p",
      "webdl-720p",
      "bluray-720p",
      "hdtv-1080p",
      "webrip-1080p",
      "webdl-1080p",
      "bluray-1080p",
      "remux-1080p",
      "hdtv-2160p",
      "webrip-2160p",
      "webdl-2160p",
      "bluray-2160p",
      "remux-2160p",
      "cam",
    ] as const),
  })),
  tiers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  ),
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

export const getListQualitiesMockHandler = (
  overrideResponse?:
    | TargetsQualityList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TargetsQualityList> | TargetsQualityList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/qualities",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListQualitiesResponseMock(),
        { status: 200 },
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

export const getDeleteTargetMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/targets/:targetId",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetTargetMockHandler = (
  overrideResponse?:
    | TargetsTarget
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TargetsTarget> | TargetsTarget),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/targets/:targetId",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetTargetResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getDuplicateTargetMockHandler = (
  overrideResponse?:
    | TargetsTarget
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<TargetsTarget> | TargetsTarget),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/targets/:targetId/duplicate",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getDuplicateTargetResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};
export const getTargetsMock = () => [
  getListProfilesMockHandler(),
  getCreateProfileMockHandler(),
  getListQualitiesMockHandler(),
  getListTargetsMockHandler(),
  getCreateTargetMockHandler(),
  getDeleteTargetMockHandler(),
  getGetTargetMockHandler(),
  getDuplicateTargetMockHandler(),
];
