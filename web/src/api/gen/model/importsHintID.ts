import type { ImportsHintIDProvider } from "./importsHintIDProvider.ts";

export interface ImportsHintID {
  provider: ImportsHintIDProvider;
  value: string;
}
