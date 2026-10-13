import * as zod from "zod";

export const FieldError = zod.object({
  location: zod.string(),
  message: zod.string(),
});

export type FieldError = zod.input<typeof FieldError>;
export type FieldErrorOutput = zod.output<typeof FieldError>;

export const identityAccountDisplayNameMax = 100;

export const identityAccountEmailMax = 254;

export const IdentityAccount = zod.object({
  admin: zod.boolean(),
  displayName: zod.string().min(1).max(identityAccountDisplayNameMax),
  email: zod.email().max(identityAccountEmailMax),
  id: zod.uuid(),
});

export type IdentityAccount = zod.input<typeof IdentityAccount>;
export type IdentityAccountOutput = zod.output<typeof IdentityAccount>;

export const identityClaimBodyDisplayNameMax = 100;

export const identityClaimBodyEmailMax = 254;

export const identityClaimBodyPinRegExp = new RegExp("^[0-9]{4}$");

export const IdentityClaimBody = zod.object({
  displayName: zod.string().min(1).max(identityClaimBodyDisplayNameMax),
  email: zod.email().max(identityClaimBodyEmailMax),
  pin: zod.string().regex(identityClaimBodyPinRegExp),
});

export type IdentityClaimBody = zod.input<typeof IdentityClaimBody>;
export type IdentityClaimBodyOutput = zod.output<typeof IdentityClaimBody>;

export const identityClaimedApiKeyRegExp = new RegExp("^ssk_[A-Za-z0-9_-]{43}$");

export const IdentityClaimed = zod.object({
  account: IdentityAccount,
  apiKey: zod.string().regex(identityClaimedApiKeyRegExp),
});

export type IdentityClaimed = zod.input<typeof IdentityClaimed>;
export type IdentityClaimedOutput = zod.output<typeof IdentityClaimed>;

export const IdentitySession = zod.object({
  account: IdentityAccount,
});

export type IdentitySession = zod.input<typeof IdentitySession>;
export type IdentitySessionOutput = zod.output<typeof IdentitySession>;

export const IdentitySessionsChanged = zod.object({
  account: zod.uuid(),
});

export type IdentitySessionsChanged = zod.input<typeof IdentitySessionsChanged>;
export type IdentitySessionsChangedOutput = zod.output<typeof IdentitySessionsChanged>;

export const identitySignInBodyEmailMax = 254;

export const identitySignInBodyPinRegExp = new RegExp("^[0-9]{4}$");

export const IdentitySignInBody = zod.object({
  email: zod.email().max(identitySignInBodyEmailMax),
  pin: zod.string().regex(identitySignInBodyPinRegExp),
});

export type IdentitySignInBody = zod.input<typeof IdentitySignInBody>;
export type IdentitySignInBodyOutput = zod.output<typeof IdentitySignInBody>;

export const LiveResync = zod.object({});

export type LiveResync = zod.input<typeof LiveResync>;
export type LiveResyncOutput = zod.output<typeof LiveResync>;

export const LiveSessionEnded = zod.object({
  reason: zod.enum(["signed-out", "expired"]),
});

export type LiveSessionEnded = zod.input<typeof LiveSessionEnded>;
export type LiveSessionEndedOutput = zod.output<typeof LiveSessionEnded>;

export const Problem = zod.object({
  code: zod.string(),
  detail: zod.string().exactOptional(),
  errors: zod.array(FieldError).exactOptional(),
  status: zod.int(),
  title: zod.string(),
});

export type Problem = zod.input<typeof Problem>;
export type ProblemOutput = zod.output<typeof Problem>;

export const SystemBuild = zod.object({
  commit: zod.string().exactOptional(),
  fingerprint: zod.string().exactOptional(),
  libtorrent: zod.string().exactOptional(),
  nativeBundle: zod.string().exactOptional(),
  stamped: zod.boolean(),
  target: zod.string().exactOptional(),
  version: zod.string().exactOptional(),
});

export type SystemBuild = zod.input<typeof SystemBuild>;
export type SystemBuildOutput = zod.output<typeof SystemBuild>;

export const SystemHealth = zod.object({
  status: zod.enum(["ok"]),
});

export type SystemHealth = zod.input<typeof SystemHealth>;
export type SystemHealthOutput = zod.output<typeof SystemHealth>;

export const SystemListeners = zod.object({
  control: zod.string(),
  http: zod.string(),
  link: zod.string(),
  torrent: zod.string(),
});

export type SystemListeners = zod.input<typeof SystemListeners>;
export type SystemListenersOutput = zod.output<typeof SystemListeners>;

export const SystemReport = zod.object({
  build: SystemBuild,
  dataDir: zod.string(),
  listeners: SystemListeners,
  pid: zod.int(),
  seed: zod.string(),
  startedAt: zod.iso.datetime({ offset: true }),
});

export type SystemReport = zod.input<typeof SystemReport>;
export type SystemReportOutput = zod.output<typeof SystemReport>;

export const SystemStatus = zod.object({
  claim: zod.enum(["unclaimed", "controller", "worker"]),
});

export type SystemStatus = zod.input<typeof SystemStatus>;
export type SystemStatusOutput = zod.output<typeof SystemStatus>;

export const LiveEventIdentitySessionsChanged = zod.object({
  event: zod.enum(["identity.sessions-changed"]),
  data: IdentitySessionsChanged,
});

export type LiveEventIdentitySessionsChanged = zod.input<typeof LiveEventIdentitySessionsChanged>;
export type LiveEventIdentitySessionsChangedOutput = zod.output<
  typeof LiveEventIdentitySessionsChanged
>;

export const LiveEventLiveResync = zod.object({
  event: zod.enum(["live.resync"]),
  data: LiveResync,
});

export type LiveEventLiveResync = zod.input<typeof LiveEventLiveResync>;
export type LiveEventLiveResyncOutput = zod.output<typeof LiveEventLiveResync>;

export const LiveEventLiveSessionEnded = zod.object({
  event: zod.enum(["live.session-ended"]),
  data: LiveSessionEnded,
});

export type LiveEventLiveSessionEnded = zod.input<typeof LiveEventLiveSessionEnded>;
export type LiveEventLiveSessionEndedOutput = zod.output<typeof LiveEventLiveSessionEnded>;

export const LiveEvent = zod.discriminatedUnion("event", [
  LiveEventIdentitySessionsChanged,
  LiveEventLiveResync,
  LiveEventLiveSessionEnded,
]);

export type LiveEvent = zod.input<typeof LiveEvent>;
export type LiveEventOutput = zod.output<typeof LiveEvent>;
