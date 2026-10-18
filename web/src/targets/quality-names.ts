import { useListQualitiesSuspense } from "../api/gen/query/targets/targets";

export function useQualityNames(): (key: string) => string {
  const { qualities } = useListQualitiesSuspense().data;
  return (key) => qualities.find((quality) => quality.key === key)?.name ?? key;
}
