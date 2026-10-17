import { useListQualitiesSuspense } from "../api/gen/query/targets/targets";

/** Names each quality as the API does, such as WEBDL-1080p. */
export function useQualityNames(): (key: string) => string {
  const { qualities } = useListQualitiesSuspense().data;
  return (key) => qualities.find((quality) => quality.key === key)?.name ?? key;
}
