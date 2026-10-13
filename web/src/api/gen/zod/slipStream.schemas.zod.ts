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

export const NodesBuild = zod.object({
  arch: zod.string(),
  installKind: zod.string(),
  os: zod.string(),
  version: zod.string(),
});

export type NodesBuild = zod.input<typeof NodesBuild>;
export type NodesBuildOutput = zod.output<typeof NodesBuild>;

export const NodesChanged = zod.object({
  nodeId: zod.uuid(),
});

export type NodesChanged = zod.input<typeof NodesChanged>;
export type NodesChangedOutput = zod.output<typeof NodesChanged>;

export const NodesFacts = zod.object({
  appliedVersion: zod.string(),
  assigned: zod.array(zod.string()),
  restarting: zod.array(zod.string()),
  running: zod.array(zod.string()),
});

export type NodesFacts = zod.input<typeof NodesFacts>;
export type NodesFactsOutput = zod.output<typeof NodesFacts>;

export const NodesJoinCode = zod.object({
  code: zod.string(),
  expiresAt: zod.iso.datetime({ offset: true }),
});

export type NodesJoinCode = zod.input<typeof NodesJoinCode>;
export type NodesJoinCodeOutput = zod.output<typeof NodesJoinCode>;

export const NodesLink = zod.object({
  build: NodesBuild.exactOptional().describe("Absent while the link is down"),
  since: zod.iso.datetime({ offset: true }),
  state: zod.enum(["local", "up", "down", "mismatched"]),
});

export type NodesLink = zod.input<typeof NodesLink>;
export type NodesLinkOutput = zod.output<typeof NodesLink>;

export const nodesNodeIntentVersionRegExp = new RegExp("^[0-9a-f]{32}$");

export const NodesNode = zod.object({
  addresses: zod.array(zod.string()),
  facts: NodesFacts.exactOptional().describe("Absent while the Node has no session"),
  id: zod.uuid(),
  intentVersion: zod.string().regex(nodesNodeIntentVersionRegExp),
  key: zod.string().describe("The key's fingerprint"),
  link: NodesLink,
  name: zod.string(),
  pairedAt: zod.iso.datetime({ offset: true }),
  roles: zod.array(
    zod.enum([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ]),
  ),
  self: zod.boolean(),
});

export type NodesNode = zod.input<typeof NodesNode>;
export type NodesNodeOutput = zod.output<typeof NodesNode>;

export const NodesNodeList = zod.object({
  nodes: zod.array(NodesNode),
});

export type NodesNodeList = zod.input<typeof NodesNodeList>;
export type NodesNodeListOutput = zod.output<typeof NodesNodeList>;

export const NodesRoles = zod.object({
  roles: zod.array(
    zod.enum([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ]),
  ),
});

export type NodesRoles = zod.input<typeof NodesRoles>;
export type NodesRolesOutput = zod.output<typeof NodesRoles>;

export const nodesSettingsJoinCodeLifetimeSecondsMin = 600;
export const nodesSettingsJoinCodeLifetimeSecondsMax = 604800;

export const NodesSettings = zod.object({
  joinCodeLifetimeSeconds: zod
    .int()
    .min(nodesSettingsJoinCodeLifetimeSecondsMin)
    .max(nodesSettingsJoinCodeLifetimeSecondsMax),
});

export type NodesSettings = zod.input<typeof NodesSettings>;
export type NodesSettingsOutput = zod.output<typeof NodesSettings>;

export const nodesSettingsPatchJoinCodeLifetimeSecondsMin = 600;
export const nodesSettingsPatchJoinCodeLifetimeSecondsMax = 604800;

export const NodesSettingsPatch = zod.object({
  joinCodeLifetimeSeconds: zod
    .int()
    .min(nodesSettingsPatchJoinCodeLifetimeSecondsMin)
    .max(nodesSettingsPatchJoinCodeLifetimeSecondsMax)
    .exactOptional(),
});

export type NodesSettingsPatch = zod.input<typeof NodesSettingsPatch>;
export type NodesSettingsPatchOutput = zod.output<typeof NodesSettingsPatch>;

export const PairingController = zod.object({
  key: zod.string().describe("The Controller key's fingerprint"),
  link: zod.string(),
  name: zod.string(),
});

export type PairingController = zod.input<typeof PairingController>;
export type PairingControllerOutput = zod.output<typeof PairingController>;

export const pairingJoinBodyCodeMax = 1024;

export const PairingJoinBody = zod.object({
  code: zod.string().min(1).max(pairingJoinBodyCodeMax),
});

export type PairingJoinBody = zod.input<typeof PairingJoinBody>;
export type PairingJoinBodyOutput = zod.output<typeof PairingJoinBody>;

export const PairingWorker = zod.object({
  controller: PairingController,
  nodeId: zod.uuid(),
});

export type PairingWorker = zod.input<typeof PairingWorker>;
export type PairingWorkerOutput = zod.output<typeof PairingWorker>;

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

export const WorksTextValue = zod.object({
  key: zod.string(),
  kind: zod.enum(["text"]),
  text: zod.string(),
});

export type WorksTextValue = zod.input<typeof WorksTextValue>;
export type WorksTextValueOutput = zod.output<typeof WorksTextValue>;

export const WorksNumberValue = zod.object({
  key: zod.string(),
  kind: zod.enum(["number"]),
  number: zod.number(),
});

export type WorksNumberValue = zod.input<typeof WorksNumberValue>;
export type WorksNumberValueOutput = zod.output<typeof WorksNumberValue>;

export const worksDateValueDateRegExp = new RegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$");

export const WorksDateValue = zod.object({
  date: zod.string().regex(worksDateValueDateRegExp).describe("YYYY, YYYY-MM or YYYY-MM-DD"),
  key: zod.string(),
  kind: zod.enum(["date"]),
});

export type WorksDateValue = zod.input<typeof WorksDateValue>;
export type WorksDateValueOutput = zod.output<typeof WorksDateValue>;

export const worksDurationValueMsMin = 0;

export const WorksDurationValue = zod.object({
  key: zod.string(),
  kind: zod.enum(["duration"]),
  ms: zod.int().min(worksDurationValueMsMin),
});

export type WorksDurationValue = zod.input<typeof WorksDurationValue>;
export type WorksDurationValueOutput = zod.output<typeof WorksDurationValue>;

export const WorksFieldValue = zod.discriminatedUnion("kind", [
  WorksTextValue,
  WorksNumberValue,
  WorksDateValue,
  WorksDurationValue,
]);

export type WorksFieldValue = zod.input<typeof WorksFieldValue>;
export type WorksFieldValueOutput = zod.output<typeof WorksFieldValue>;

export const WorksItem = zod.object({
  fields: zod.array(WorksFieldValue),
  id: zod.uuid(),
  key: zod.string(),
  label: zod.string(),
  levelId: zod.uuid().nullable().describe("Null when the Content Type declares no levels"),
  position: zod.int().min(1),
  withdrawnAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When the source stopped listing it; null while it lists it"),
});

export type WorksItem = zod.input<typeof WorksItem>;
export type WorksItemOutput = zod.output<typeof WorksItem>;

export const WorksLevel = zod.object({
  depth: zod.int().min(1),
  fields: zod.array(WorksFieldValue),
  id: zod.uuid(),
  key: zod.string(),
  label: zod.string(),
  parentId: zod.uuid().nullable().describe("Null under the Work"),
  position: zod.int().min(1),
  withdrawnAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When the source stopped listing it; null while it lists it"),
});

export type WorksLevel = zod.input<typeof WorksLevel>;
export type WorksLevelOutput = zod.output<typeof WorksLevel>;

export const WorksTree = zod.object({
  fields: zod.array(WorksFieldValue),
  items: zod.array(WorksItem).describe("By their level's order, then position"),
  levels: zod.array(WorksLevel).describe("By depth, then their parent's order, then position"),
});

export type WorksTree = zod.input<typeof WorksTree>;
export type WorksTreeOutput = zod.output<typeof WorksTree>;

export const WorksAdded = zod.object({
  created: zod.boolean().describe("False when the Work was already in the library"),
  work: WorksTree,
});

export type WorksAdded = zod.input<typeof WorksAdded>;
export type WorksAddedOutput = zod.output<typeof WorksAdded>;

export const WorksCandidate = zod.object({
  fields: zod.array(WorksFieldValue).describe("The Work's fields"),
  key: zod.string(),
  label: zod.string(),
  workId: zod.uuid().nullable().describe("The Work, once it is in the library"),
});

export type WorksCandidate = zod.input<typeof WorksCandidate>;
export type WorksCandidateOutput = zod.output<typeof WorksCandidate>;

export const WorksCandidateList = zod.object({
  candidates: zod.array(WorksCandidate).describe("Best first"),
});

export type WorksCandidateList = zod.input<typeof WorksCandidateList>;
export type WorksCandidateListOutput = zod.output<typeof WorksCandidateList>;

export const WorksFieldSpec = zod.object({
  depth: zod
    .int()
    .min(1)
    .exactOptional()
    .describe("The level's depth, 1 for the outermost; present only on tier level"),
  key: zod.string(),
  kind: zod.enum(["text", "number", "date", "duration"]),
  label: zod.string(),
  tier: zod.enum(["work", "level", "item"]),
});

export type WorksFieldSpec = zod.input<typeof WorksFieldSpec>;
export type WorksFieldSpecOutput = zod.output<typeof WorksFieldSpec>;

export const WorksNoun = zod.object({
  many: zod.string(),
  one: zod.string(),
});

export type WorksNoun = zod.input<typeof WorksNoun>;
export type WorksNounOutput = zod.output<typeof WorksNoun>;

export const worksContentTypeIdRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const WorksContentType = zod.object({
  enabled: zod.boolean(),
  fields: zod.array(WorksFieldSpec),
  id: zod.string().regex(worksContentTypeIdRegExp),
  item: WorksNoun,
  levels: zod.array(WorksNoun).describe("Outermost first; empty when Items hang off the Work"),
  name: zod.string(),
  work: WorksNoun,
});

export type WorksContentType = zod.input<typeof WorksContentType>;
export type WorksContentTypeOutput = zod.output<typeof WorksContentType>;

export const WorksContentTypeList = zod.object({
  contentTypes: zod.array(WorksContentType),
});

export type WorksContentTypeList = zod.input<typeof WorksContentTypeList>;
export type WorksContentTypeListOutput = zod.output<typeof WorksContentTypeList>;

export const worksContentTypesChangedContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const WorksContentTypesChanged = zod.object({
  contentType: zod.string().regex(worksContentTypesChangedContentTypeRegExp),
});

export type WorksContentTypesChanged = zod.input<typeof WorksContentTypesChanged>;
export type WorksContentTypesChangedOutput = zod.output<typeof WorksContentTypesChanged>;

export const WorksEnabled = zod.object({
  enabled: zod.boolean(),
});

export type WorksEnabled = zod.input<typeof WorksEnabled>;
export type WorksEnabledOutput = zod.output<typeof WorksEnabled>;

export const worksNewWorkContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");
export const worksNewWorkKeyMax = 200;

export const WorksNewWork = zod.object({
  contentType: zod.string().regex(worksNewWorkContentTypeRegExp),
  key: zod
    .string()
    .min(1)
    .max(worksNewWorkKeyMax)
    .describe("The Work's key at its metadata source"),
});

export type WorksNewWork = zod.input<typeof WorksNewWork>;
export type WorksNewWorkOutput = zod.output<typeof WorksNewWork>;

export const worksWorkContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const WorksWork = zod.object({
  addedAt: zod.iso.datetime({ offset: true }),
  contentType: zod.string().regex(worksWorkContentTypeRegExp),
  id: zod.uuid(),
  key: zod.string(),
  label: zod.string(),
  refreshedAt: zod.iso.datetime({ offset: true }),
  sortLabel: zod.string(),
});

export type WorksWork = zod.input<typeof WorksWork>;
export type WorksWorkOutput = zod.output<typeof WorksWork>;

export const WorksWorkChanged = zod.object({
  workId: zod.uuid(),
});

export type WorksWorkChanged = zod.input<typeof WorksWorkChanged>;
export type WorksWorkChangedOutput = zod.output<typeof WorksWorkChanged>;

export const WorksWorkList = zod.object({
  works: zod.array(WorksWork).describe("By sort label, then id"),
});

export type WorksWorkList = zod.input<typeof WorksWorkList>;
export type WorksWorkListOutput = zod.output<typeof WorksWorkList>;

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

export const LiveEventNodesChanged = zod.object({
  event: zod.enum(["nodes.changed"]),
  data: NodesChanged,
});

export type LiveEventNodesChanged = zod.input<typeof LiveEventNodesChanged>;
export type LiveEventNodesChangedOutput = zod.output<typeof LiveEventNodesChanged>;

export const LiveEventWorksChanged = zod.object({
  event: zod.enum(["works.changed"]),
  data: WorksWorkChanged,
});

export type LiveEventWorksChanged = zod.input<typeof LiveEventWorksChanged>;
export type LiveEventWorksChangedOutput = zod.output<typeof LiveEventWorksChanged>;

export const LiveEventWorksContentTypesChanged = zod.object({
  event: zod.enum(["works.content-types-changed"]),
  data: WorksContentTypesChanged,
});

export type LiveEventWorksContentTypesChanged = zod.input<typeof LiveEventWorksContentTypesChanged>;
export type LiveEventWorksContentTypesChangedOutput = zod.output<
  typeof LiveEventWorksContentTypesChanged
>;

export const LiveEvent = zod.discriminatedUnion("event", [
  LiveEventIdentitySessionsChanged,
  LiveEventLiveResync,
  LiveEventLiveSessionEnded,
  LiveEventNodesChanged,
  LiveEventWorksChanged,
  LiveEventWorksContentTypesChanged,
]);

export type LiveEvent = zod.input<typeof LiveEvent>;
export type LiveEventOutput = zod.output<typeof LiveEvent>;
