import { z } from "zod";

const Node = z.object({
  name: z.string(),
  setup: z.enum(["none", "controller", "join"]),
  claim: z.enum(["unclaimed", "controller", "worker"]),
  address: z.string(),
  host: z.string(),
  url: z.url(),
  browserUrl: z.url(),
  log: z.string(),
});

export const manifestSchema = z.object({
  runDir: z.string(),
  nodes: z.tuple([Node], Node),
  admin: z.object({ email: z.string(), displayName: z.string(), pin: z.string() }).optional(),
  apiKey: z.string().optional(),
});

export type Manifest = z.infer<typeof manifestSchema>;
export type ManifestNode = z.infer<typeof Node>;
