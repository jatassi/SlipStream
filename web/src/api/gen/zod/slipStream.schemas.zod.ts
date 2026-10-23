import * as zod from "zod";

export const accountPatchInputBodyDisplayNameMax = 100;

export const accountPatchInputBodyEmailMax = 254;

export const AccountPatchInputBody = zod.object({
  admin: zod.boolean().exactOptional(),
  disabled: zod.boolean().exactOptional().describe("Turning it off ends every session it holds"),
  displayName: zod.string().min(1).max(accountPatchInputBodyDisplayNameMax).exactOptional(),
  email: zod
    .email()
    .max(accountPatchInputBodyEmailMax)
    .exactOptional()
    .describe("Changed at once; the old address is told"),
});

export type AccountPatchInputBody = zod.input<typeof AccountPatchInputBody>;
export type AccountPatchInputBodyOutput = zod.output<typeof AccountPatchInputBody>;

export const checkRootInputBodyPathMax = 4096;

export const CheckRootInputBody = zod.object({
  nodeId: zod.uuid(),
  path: zod.string().min(1).max(checkRootInputBodyPathMax),
  rootId: zod
    .uuid()
    .exactOptional()
    .describe("The root being re-homed, which the folder may overlap"),
});

export type CheckRootInputBody = zod.input<typeof CheckRootInputBody>;
export type CheckRootInputBodyOutput = zod.output<typeof CheckRootInputBody>;

export const DialInputBody = zod.object({
  nodeAddress: zod.string().describe("The SLN1 text the waiting Node printed"),
});

export type DialInputBody = zod.input<typeof DialInputBody>;
export type DialInputBodyOutput = zod.output<typeof DialInputBody>;

export const emailChangeInputBodyEmailMax = 254;

export const EmailChangeInputBody = zod.object({
  email: zod.email().max(emailChangeInputBodyEmailMax),
});

export type EmailChangeInputBody = zod.input<typeof EmailChangeInputBody>;
export type EmailChangeInputBodyOutput = zod.output<typeof EmailChangeInputBody>;

export const FieldError = zod.object({
  location: zod.string(),
  message: zod.string(),
});

export type FieldError = zod.input<typeof FieldError>;
export type FieldErrorOutput = zod.output<typeof FieldError>;

export const FilesUnreleasedCell = zod.object({
  edition: zod.string().describe("any under any cut, else the key of the cut the cell wants"),
  itemId: zod.uuid(),
  state: zod.enum(["unreleased"]).describe("Its Item has no release date, or a later one"),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod.boolean().describe("Missing or failed, and its Item monitored and released"),
});

export type FilesUnreleasedCell = zod.input<typeof FilesUnreleasedCell>;
export type FilesUnreleasedCellOutput = zod.output<typeof FilesUnreleasedCell>;

export const FilesMissingCell = zod.object({
  edition: zod.string().describe("any under any cut, else the key of the cut the cell wants"),
  itemId: zod.uuid(),
  state: zod.enum(["missing"]),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod.boolean().describe("Missing or failed, and its Item monitored and released"),
});

export type FilesMissingCell = zod.input<typeof FilesMissingCell>;
export type FilesMissingCellOutput = zod.output<typeof FilesMissingCell>;

export const FilesDownloadingCell = zod.object({
  edition: zod.string().describe("any under any cut, else the key of the cut the cell wants"),
  itemId: zod.uuid(),
  state: zod.enum(["downloading"]),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod.boolean().describe("Missing or failed, and its Item monitored and released"),
});

export type FilesDownloadingCell = zod.input<typeof FilesDownloadingCell>;
export type FilesDownloadingCellOutput = zod.output<typeof FilesDownloadingCell>;

export const FilesFailedCell = zod.object({
  edition: zod.string().describe("any under any cut, else the key of the cut the cell wants"),
  itemId: zod.uuid(),
  state: zod.enum(["failed"]),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod.boolean().describe("Missing or failed, and its Item monitored and released"),
});

export type FilesFailedCell = zod.input<typeof FilesFailedCell>;
export type FilesFailedCellOutput = zod.output<typeof FilesFailedCell>;

export const filesProbeHeightMin = 0;

export const filesProbeWidthMin = 0;

export const FilesProbe = zod.object({
  height: zod.int().min(filesProbeHeightMin),
  width: zod.int().min(filesProbeWidthMin),
});

export type FilesProbe = zod.input<typeof FilesProbe>;
export type FilesProbeOutput = zod.output<typeof FilesProbe>;

export const filesLibraryFileSizeMin = 0;

export const FilesLibraryFile = zod.object({
  id: zod.uuid(),
  path: zod.string().describe("Relative to its root, / separated"),
  probe: FilesProbe,
  quality: zod.enum([
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
  ]),
  size: zod.int().min(filesLibraryFileSizeMin),
});

export type FilesLibraryFile = zod.input<typeof FilesLibraryFile>;
export type FilesLibraryFileOutput = zod.output<typeof FilesLibraryFile>;

export const FilesUpgradableCell = zod.object({
  edition: zod.string().describe("any under any cut, else the key of the cut the cell wants"),
  file: FilesLibraryFile.describe("The file filling it, below its Target's cutoff"),
  itemId: zod.uuid(),
  state: zod.enum(["upgradable"]),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod.boolean().describe("Missing or failed, and its Item monitored and released"),
});

export type FilesUpgradableCell = zod.input<typeof FilesUpgradableCell>;
export type FilesUpgradableCellOutput = zod.output<typeof FilesUpgradableCell>;

export const FilesMetCell = zod.object({
  edition: zod.string().describe("any under any cut, else the key of the cut the cell wants"),
  file: FilesLibraryFile.describe("The file filling it"),
  itemId: zod.uuid(),
  state: zod.enum(["met"]),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod.boolean().describe("Missing or failed, and its Item monitored and released"),
});

export type FilesMetCell = zod.input<typeof FilesMetCell>;
export type FilesMetCellOutput = zod.output<typeof FilesMetCell>;

export const FilesCell = zod.discriminatedUnion("state", [
  FilesUnreleasedCell,
  FilesMissingCell,
  FilesDownloadingCell,
  FilesFailedCell,
  FilesUpgradableCell,
  FilesMetCell,
]);

export type FilesCell = zod.input<typeof FilesCell>;
export type FilesCellOutput = zod.output<typeof FilesCell>;

export const FilesItemStatus = zod.object({
  itemId: zod.uuid(),
  status: zod
    .union([
      zod.literal("unreleased"),
      zod.literal("missing"),
      zod.literal("downloading"),
      zod.literal("failed"),
      zod.literal("upgradable"),
      zod.literal("met"),
      zod.literal(null),
    ])
    .nullable()
    .describe(
      "The first state its cells hold, in the glossary's order; null for an Item with no cell",
    ),
});

export type FilesItemStatus = zod.input<typeof FilesItemStatus>;
export type FilesItemStatusOutput = zod.output<typeof FilesItemStatus>;

export const FilesCellList = zod.object({
  cells: zod.array(FilesCell).describe("By Item in tree order, then Target in Profile order"),
  items: zod.array(FilesItemStatus).describe("Every Item in tree order"),
});

export type FilesCellList = zod.input<typeof FilesCellList>;
export type FilesCellListOutput = zod.output<typeof FilesCellList>;

export const FilesChanged = zod.object({
  workId: zod.uuid(),
});

export type FilesChanged = zod.input<typeof FilesChanged>;
export type FilesChangedOutput = zod.output<typeof FilesChanged>;

export const GrabsChanged = zod.object({
  grabId: zod.uuid(),
  workId: zod.uuid(),
});

export type GrabsChanged = zod.input<typeof GrabsChanged>;
export type GrabsChangedOutput = zod.output<typeof GrabsChanged>;

export const GrabsEntered = zod.object({
  at: zod.iso.datetime({ offset: true }),
  stage: zod.enum([
    "fetching",
    "downloading",
    "probing",
    "transferring",
    "importing",
    "imported",
    "failed",
  ]),
});

export type GrabsEntered = zod.input<typeof GrabsEntered>;
export type GrabsEnteredOutput = zod.output<typeof GrabsEntered>;

export const GrabsFailure = zod.object({
  code: zod.enum([
    "indexer_unavailable",
    "torrent_unreachable",
    "torrent_invalid",
    "infohash_mismatch",
    "torrent_v2_only",
    "no_download_client",
    "download_failed",
    "no_video_file",
    "probe_failed",
    "fills_nothing",
    "transfer_failed",
    "import_failed",
    "node_written_off",
    "root_changed",
    "needs_transfer",
  ]),
  message: zod.string(),
  stage: zod.enum(["fetching", "downloading", "probing", "transferring", "importing"]),
});

export type GrabsFailure = zod.input<typeof GrabsFailure>;
export type GrabsFailureOutput = zod.output<typeof GrabsFailure>;

export const grabsProgressDoneMin = 0;

export const grabsProgressTotalMin = 0;

export const GrabsProgress = zod.object({
  done: zod
    .int()
    .min(grabsProgressDoneMin)
    .describe("Bytes downloaded, or received and verified while transferring"),
  total: zod.int().min(grabsProgressTotalMin).describe("Bytes wanted"),
});

export type GrabsProgress = zod.input<typeof GrabsProgress>;
export type GrabsProgressOutput = zod.output<typeof GrabsProgress>;

export const grabsGrabReleaseSizeMin = 0;

export const GrabsGrab = zod.object({
  createdAt: zod.iso.datetime({ offset: true }),
  edition: zod
    .string()
    .describe("The Edition it was grabbed to fill: any under any cut, else a cut key"),
  failure: GrabsFailure.exactOptional().describe("Why it failed, absent unless it did"),
  fetchNodeId: zod
    .uuid()
    .nullable()
    .describe("The Searching Node that fetched the .torrent, null until it did"),
  history: zod.array(GrabsEntered).describe("Each stage it entered, oldest first"),
  id: zod.uuid(),
  indexerId: zod.uuid(),
  infoHash: zod
    .string()
    .nullable()
    .describe("The v1 info hash in hex, null until the .torrent is fetched"),
  itemId: zod.uuid(),
  nodeId: zod
    .uuid()
    .describe(
      "The Node that downloads it: the Indexer's home Node until the fetch, then the Downloading Node the fetch handed the .torrent to",
    ),
  progress: GrabsProgress.exactOptional().describe(
    "The download's or the Transfer's progress, absent unless it is downloading and its Node reported it, or transferring",
  ),
  releaseSize: zod.int().min(grabsGrabReleaseSizeMin),
  releaseTitle: zod.string(),
  stage: zod.enum([
    "fetching",
    "downloading",
    "probing",
    "transferring",
    "importing",
    "imported",
    "failed",
  ]),
  targetIds: zod.array(zod.string()).describe("The Targets it was grabbed for, in Profile order"),
  workId: zod.uuid(),
});

export type GrabsGrab = zod.input<typeof GrabsGrab>;
export type GrabsGrabOutput = zod.output<typeof GrabsGrab>;

export const GrabsGrabList = zod.object({
  grabs: zod.array(GrabsGrab).describe("Newest first"),
});

export type GrabsGrabList = zod.input<typeof GrabsGrabList>;
export type GrabsGrabListOutput = zod.output<typeof GrabsGrabList>;

export const GrabsReason = zod.object({
  attribute: zod
    .enum(["hdr", "video_codec", "audio_codec", "channels", "language"])
    .exactOptional()
    .describe("The attribute a failed filter rules on"),
  code: zod.enum([
    "wrong_work",
    "cut_not_wanted",
    "quality_not_in_target",
    "filter",
    "size_below",
    "size_above",
    "size_not_checked",
    "not_an_upgrade",
  ]),
  targetId: zod
    .uuid()
    .exactOptional()
    .describe("The Target whose verdict it is, absent for wrong_work and cut_not_wanted"),
  value: zod
    .string()
    .exactOptional()
    .describe(
      "The value a filter failed on, the quality a Target doesn't accept, or the cut no open cell wants",
    ),
});

export type GrabsReason = zod.input<typeof GrabsReason>;
export type GrabsReasonOutput = zod.output<typeof GrabsReason>;

export const GrabsRefusal = zod.object({
  reasons: zod
    .array(GrabsReason)
    .describe("Each open cell's Target's failures and notes, in Profile order"),
  title: zod.string(),
});

export type GrabsRefusal = zod.input<typeof GrabsRefusal>;
export type GrabsRefusalOutput = zod.output<typeof GrabsRefusal>;

export const grabsSearchCandidatesMin = 0;

export const GrabsSearch = zod.object({
  candidates: zod.int().min(grabsSearchCandidatesMin).describe("The releases the Indexer answered"),
  finishedAt: zod.iso.datetime({ offset: true }).nullable().describe("Null while the run searches"),
  grabId: zod.uuid().nullable().describe("The Grab the run made, null when none"),
  id: zod.uuid(),
  indexerId: zod.uuid(),
  itemId: zod.uuid(),
  notes: zod
    .array(GrabsReason)
    .describe(
      "The grabbed release's notes, such as size_not_checked, for each Target it was grabbed for",
    ),
  outcome: zod
    .union([
      zod.literal("grabbed"),
      zod.literal("nothing_acceptable"),
      zod.literal("indexer_unavailable"),
      zod.literal("indexer_refused"),
      zod.literal(null),
    ])
    .nullable()
    .describe("Null while the run searches"),
  refused: zod
    .array(GrabsRefusal)
    .describe(
      "The releases it didn't grab for a reason, in the Indexer's order; a release it outranked isn't refused",
    ),
  startedAt: zod.iso.datetime({ offset: true }),
});

export type GrabsSearch = zod.input<typeof GrabsSearch>;
export type GrabsSearchOutput = zod.output<typeof GrabsSearch>;

export const GrabsSearchChanged = zod.object({
  searchId: zod.uuid(),
  workId: zod.uuid(),
});

export type GrabsSearchChanged = zod.input<typeof GrabsSearchChanged>;
export type GrabsSearchChangedOutput = zod.output<typeof GrabsSearchChanged>;

export const GrabsSearchList = zod.object({
  searches: zod.array(GrabsSearch).describe("Newest first"),
});

export type GrabsSearchList = zod.input<typeof GrabsSearchList>;
export type GrabsSearchListOutput = zod.output<typeof GrabsSearchList>;

export const GrabsSearchStarted = zod.object({
  searchIds: zod.array(zod.string()).describe("One run per Item with a wanted cell"),
});

export type GrabsSearchStarted = zod.input<typeof GrabsSearchStarted>;
export type GrabsSearchStartedOutput = zod.output<typeof GrabsSearchStarted>;

export const HealthAction = zod.object({
  href: zod.string().exactOptional(),
  kind: zod
    .enum(["check", "run", "link"])
    .describe("check: Check now on the item; run: Run now on taskId; link: a page in the web"),
  label: zod.string().exactOptional().describe("The link's words"),
  taskId: zod.string().exactOptional(),
});

export type HealthAction = zod.input<typeof HealthAction>;
export type HealthActionOutput = zod.output<typeof HealthAction>;

export const HealthCategoryID = zod.enum(["storage", "nodes", "scheduled-tasks"]);

export type HealthCategoryID = zod.input<typeof HealthCategoryID>;
export type HealthCategoryIDOutput = zod.output<typeof HealthCategoryID>;

export const HealthFact = zod.object({
  at: zod.iso.datetime({ offset: true }).exactOptional(),
  bytes: zod.int().exactOptional(),
  label: zod.string(),
  text: zod.string().exactOptional().describe("Exactly one of text, bytes and at is present"),
});

export type HealthFact = zod.input<typeof HealthFact>;
export type HealthFactOutput = zod.output<typeof HealthFact>;

export const HealthItem = zod.object({
  action: zod.union([HealthAction, zod.null()]).describe("Its one action, null when it has none"),
  changedAt: zod.iso.datetime({ offset: true }).describe("When its level last changed"),
  detail: zod.array(HealthFact),
  key: zod.string().describe("Stable within its category and safe as one path segment"),
  level: zod.enum(["warn", "fail"]),
  since: zod.iso.datetime({ offset: true }).describe("When it first appeared at any level"),
  subject: zod.string(),
  text: zod.string().describe("What to do first"),
  unsure: zod.boolean().describe("Its owner can't judge it now, so it reads as it last did"),
});

export type HealthItem = zod.input<typeof HealthItem>;
export type HealthItemOutput = zod.output<typeof HealthItem>;

export const HealthCategory = zod.object({
  canCheck: zod.boolean().describe("It has a Check now"),
  id: HealthCategoryID,
  items: zod.array(HealthItem).describe("Errors first, then the oldest"),
  label: zod.string(),
  level: zod
    .union([zod.literal("warn"), zod.literal("fail"), zod.literal(null)])
    .nullable()
    .describe("Its worst item's level, null while it has none"),
  phrase: zod.string().describe("The label inside a sentence, such as storage"),
  unread: zod
    .string()
    .exactOptional()
    .describe(
      "Why its last answer failed; its items then read as they last did. Absent while it answers",
    ),
});

export type HealthCategory = zod.input<typeof HealthCategory>;
export type HealthCategoryOutput = zod.output<typeof HealthCategory>;

export const HealthChanged = zod.object({
  categories: zod.array(zod.string()),
});

export type HealthChanged = zod.input<typeof HealthChanged>;
export type HealthChangedOutput = zod.output<typeof HealthChanged>;

export const HealthPointer = zod.object({
  category: HealthCategoryID,
  key: zod.string(),
  text: zod.string(),
});

export type HealthPointer = zod.input<typeof HealthPointer>;
export type HealthPointerOutput = zod.output<typeof HealthPointer>;

export const HealthReport = zod.object({
  categories: zod.array(HealthCategory).describe("In page order"),
  first: zod
    .union([HealthPointer, zod.null()])
    .describe(
      "The item the status sentence leads with: the worst, then the oldest, then in page order; null while nothing is wrong",
    ),
  level: zod
    .union([zod.literal("warn"), zod.literal("fail"), zod.literal(null)])
    .nullable()
    .describe("The worst level over every category, null while nothing is wrong"),
});

export type HealthReport = zod.input<typeof HealthReport>;
export type HealthReportOutput = zod.output<typeof HealthReport>;

export const HealthUnreached = zod.object({
  key: zod.string().exactOptional().describe("The item it was for; absent for a whole category"),
  words: zod.string().describe("What couldn't be reached, as a sentence"),
});

export type HealthUnreached = zod.input<typeof HealthUnreached>;
export type HealthUnreachedOutput = zod.output<typeof HealthUnreached>;

export const HealthChecked = zod.object({
  report: HealthReport.describe("What is wrong right after the check"),
  unreached: zod
    .array(HealthUnreached)
    .describe("What the check couldn't reach; those items stay as they were"),
});

export type HealthChecked = zod.input<typeof HealthChecked>;
export type HealthCheckedOutput = zod.output<typeof HealthChecked>;

export const settingAmountAmountMin = 0;

export const SettingAmount = zod.object({
  amount: zod.int().min(settingAmountAmountMin),
  unit: zod.enum(["bytes", "percent"]),
});

export type SettingAmount = zod.input<typeof SettingAmount>;
export type SettingAmountOutput = zod.output<typeof SettingAmount>;

export const HealthNotReporting = zod.object({
  errorBelow: zod
    .union([SettingAmount, zod.null()])
    .describe("Its own error level, null while it follows the global one"),
  mount: zod.string(),
  nodeId: zod.uuid(),
  nodeName: zod.string(),
  warnBelow: zod
    .union([SettingAmount, zod.null()])
    .describe("Its own warning level, null while it follows the global one"),
});

export type HealthNotReporting = zod.input<typeof HealthNotReporting>;
export type HealthNotReportingOutput = zod.output<typeof HealthNotReporting>;

export const SettingSpaceField = zod.object({
  applies: zod.enum(["live", "restart"]),
  default: SettingAmount,
  defaultFrom: zod
    .string()
    .exactOptional()
    .describe("Where the default comes from, when it is another setting's value"),
  isDefault: zod
    .boolean()
    .describe("The setting has no override; Reset to default leaves it unchanged"),
  key: zod.string(),
  kind: zod.enum(["space"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  maxBytes: zod.int().describe("The largest amount in bytes: 1 PB"),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  running: SettingAmount,
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  value: SettingAmount,
});

export type SettingSpaceField = zod.input<typeof SettingSpaceField>;
export type SettingSpaceFieldOutput = zod.output<typeof SettingSpaceField>;

export const HealthSettings = zod.object({
  errorBelow: SettingSpaceField,
  warnBelow: SettingSpaceField,
});

export type HealthSettings = zod.input<typeof HealthSettings>;
export type HealthSettingsOutput = zod.output<typeof HealthSettings>;

export const HealthSettingsInputBody = zod.object({
  errorBelow: zod
    .union([SettingAmount, zod.null()])
    .exactOptional()
    .describe("null resets it to its default"),
  warnBelow: zod
    .union([SettingAmount, zod.null()])
    .exactOptional()
    .describe("null resets it to its default"),
});

export type HealthSettingsInputBody = zod.input<typeof HealthSettingsInputBody>;
export type HealthSettingsInputBodyOutput = zod.output<typeof HealthSettingsInputBody>;

export const HealthVolume = zod.object({
  errorBelow: SettingSpaceField.describe(
    "Its own error level, the default while it follows the global one",
  ),
  errorBytes: zod.int().describe("The error level it follows, in bytes"),
  free: zod.int().describe("The bytes SlipStream may write"),
  level: zod
    .union([zod.literal("warn"), zod.literal("fail"), zod.literal(null)])
    .nullable()
    .describe("Its Storage item's level, null while it has none"),
  live: zod
    .boolean()
    .describe("Its Node is in step, so its figures are current; else they read as of sampledAt"),
  mount: zod.string().describe("As its Node names it, such as D:\\ or /mnt/media"),
  name: zod.string().describe("Such as D: (Media)"),
  nodeId: zod.uuid(),
  nodeName: zod.string(),
  sampledAt: zod.iso.datetime({ offset: true }),
  skipsWarning: zod
    .boolean()
    .describe("Its error level is at or above its warning level, so it goes straight to error"),
  total: zod.int(),
  warnBelow: SettingSpaceField.describe(
    "Its own warning level, the default while it follows the global one",
  ),
  warnBytes: zod.int().describe("The warning level it follows, in bytes"),
});

export type HealthVolume = zod.input<typeof HealthVolume>;
export type HealthVolumeOutput = zod.output<typeof HealthVolume>;

export const HealthVolumeInputBody = zod.object({
  errorBelow: zod
    .union([SettingAmount, zod.null()])
    .exactOptional()
    .describe("null makes it follow the global level"),
  mount: zod.string().min(1),
  nodeId: zod.uuid(),
  warnBelow: zod
    .union([SettingAmount, zod.null()])
    .exactOptional()
    .describe("null makes it follow the global level"),
});

export type HealthVolumeInputBody = zod.input<typeof HealthVolumeInputBody>;
export type HealthVolumeInputBodyOutput = zod.output<typeof HealthVolumeInputBody>;

export const HealthVolumeList = zod.object({
  notReporting: zod
    .array(HealthNotReporting)
    .describe("Volumes with levels of their own that no Node reports now, by Node then mount"),
  volumes: zod.array(HealthVolume).describe("Every volume a Node reports, by Node then mount"),
});

export type HealthVolumeList = zod.input<typeof HealthVolumeList>;
export type HealthVolumeListOutput = zod.output<typeof HealthVolumeList>;

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

export const IdentityPinState = zod.object({
  state: zod.enum(["fine", "waiting", "off"]),
  until: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("When the wait ends, while it waits"),
});

export type IdentityPinState = zod.input<typeof IdentityPinState>;
export type IdentityPinStateOutput = zod.output<typeof IdentityPinState>;

export const identityAccountRowDisplayNameMax = 100;

export const identityAccountRowEmailMax = 254;

export const IdentityAccountRow = zod.object({
  admin: zod.boolean(),
  disabled: zod.boolean().describe("Turned off: it can't sign in and holds no session"),
  displayName: zod.string().min(1).max(identityAccountRowDisplayNameMax),
  email: zod.email().max(identityAccountRowEmailMax),
  id: zod.uuid(),
  lastSignInAt: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("Absent until it first signs in"),
  passkeyCount: zod.int(),
  pin: IdentityPinState,
});

export type IdentityAccountRow = zod.input<typeof IdentityAccountRow>;
export type IdentityAccountRowOutput = zod.output<typeof IdentityAccountRow>;

export const IdentityPasskey = zod.object({
  createdAt: zod.iso.datetime({ offset: true }),
  host: zod.string().describe("The address's host the passkey signs in at"),
  id: zod.uuid(),
  lastUsedAt: zod.iso.datetime({ offset: true }).exactOptional(),
  name: zod.string(),
});

export type IdentityPasskey = zod.input<typeof IdentityPasskey>;
export type IdentityPasskeyOutput = zod.output<typeof IdentityPasskey>;

export const IdentitySessionRow = zod.object({
  address: zod.string().describe("The client address it signed in from"),
  createdAt: zod.iso.datetime({ offset: true }),
  current: zod.boolean().describe("The session making this request"),
  device: zod.string().describe("Such as Safari on iPhone, from the browser at sign-in"),
  id: zod.uuid(),
  lastSeenAt: zod.iso.datetime({ offset: true }),
});

export type IdentitySessionRow = zod.input<typeof IdentitySessionRow>;
export type IdentitySessionRowOutput = zod.output<typeof IdentitySessionRow>;

export const IdentityAccountDetail = zod.object({
  account: IdentityAccountRow,
  passkeys: zod.array(IdentityPasskey),
  sessions: zod.array(IdentitySessionRow),
});

export type IdentityAccountDetail = zod.input<typeof IdentityAccountDetail>;
export type IdentityAccountDetailOutput = zod.output<typeof IdentityAccountDetail>;

export const IdentityAccountRows = zod.object({
  accounts: zod.array(IdentityAccountRow),
});

export type IdentityAccountRows = zod.input<typeof IdentityAccountRows>;
export type IdentityAccountRowsOutput = zod.output<typeof IdentityAccountRows>;

export const IdentityAccountsChanged = zod.object({});

export type IdentityAccountsChanged = zod.input<typeof IdentityAccountsChanged>;
export type IdentityAccountsChangedOutput = zod.output<typeof IdentityAccountsChanged>;

export const IdentityAddress = zod.object({
  address: zod.string().describe("The origin links are built on, empty until set"),
  suggestedAddress: zod
    .string()
    .describe(
      "The address this request reached SlipStream at, or this machine's name when that was an IP address",
    ),
});

export type IdentityAddress = zod.input<typeof IdentityAddress>;
export type IdentityAddressOutput = zod.output<typeof IdentityAddress>;

export const identityAddressChangeAddressMax = 2048;

export const IdentityAddressChange = zod.object({
  address: zod.string().max(identityAddressChangeAddressMax),
});

export type IdentityAddressChange = zod.input<typeof IdentityAddressChange>;
export type IdentityAddressChangeOutput = zod.output<typeof IdentityAddressChange>;

export const IdentityApiKeyMade = zod.object({
  createdAt: zod.iso.datetime({ offset: true }),
});

export type IdentityApiKeyMade = zod.input<typeof IdentityApiKeyMade>;
export type IdentityApiKeyMadeOutput = zod.output<typeof IdentityApiKeyMade>;

export const IdentityCeremony = zod.object({
  ceremony: zod.string().describe("The id the finish carries in its path"),
  options: zod
    .unknown()
    .describe(
      "PublicKeyCredentialCreationOptionsJSON for a registration, PublicKeyCredentialRequestOptionsJSON for an assertion: the browser's parseCreationOptionsFromJSON or parseRequestOptionsFromJSON takes it",
    ),
});

export type IdentityCeremony = zod.input<typeof IdentityCeremony>;
export type IdentityCeremonyOutput = zod.output<typeof IdentityCeremony>;

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

export const IdentityDeadLink = zod.object({
  adminName: zod
    .string()
    .exactOptional()
    .describe(
      "Who to ask: absent for unknown, which names nobody, and for a link the API key or the host command issued",
    ),
  expiresAt: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("Every reason but unknown"),
  issuedAt: zod.iso.datetime({ offset: true }).exactOptional().describe("Every reason but unknown"),
  reason: zod.enum(["unknown", "withdrawn", "expired", "used"]),
  state: zod.enum(["dead"]),
});

export type IdentityDeadLink = zod.input<typeof IdentityDeadLink>;
export type IdentityDeadLinkOutput = zod.output<typeof IdentityDeadLink>;

export const MailOutcome = zod.object({
  detail: zod
    .string()
    .exactOptional()
    .describe("The mail server's reply, or why none answered, when the email failed"),
  outcome: zod.enum(["sent", "failed", "not-set-up"]),
});

export type MailOutcome = zod.input<typeof MailOutcome>;
export type MailOutcomeOutput = zod.output<typeof MailOutcome>;

export const IdentityEmailChangeStarted = zod.object({
  email: MailOutcome.describe("What became of the confirm link's email to the new address"),
});

export type IdentityEmailChangeStarted = zod.input<typeof IdentityEmailChangeStarted>;
export type IdentityEmailChangeStartedOutput = zod.output<typeof IdentityEmailChangeStarted>;

export const IdentityIssuedResetLink = zod.object({
  email: MailOutcome,
  url: zod.url().describe("The link itself, shown only in this answer"),
});

export type IdentityIssuedResetLink = zod.input<typeof IdentityIssuedResetLink>;
export type IdentityIssuedResetLinkOutput = zod.output<typeof IdentityIssuedResetLink>;

export const IdentityPendingLink = zod.object({
  email: zod.email(),
  expiresAt: zod.iso.datetime({ offset: true }),
  id: zod.uuid(),
  issuedAt: zod.iso.datetime({ offset: true }),
  issuedBy: zod
    .string()
    .exactOptional()
    .describe("The Admin who issued it; absent for the API key"),
});

export type IdentityPendingLink = zod.input<typeof IdentityPendingLink>;
export type IdentityPendingLinkOutput = zod.output<typeof IdentityPendingLink>;

export const IdentityIssuedSignupLink = zod.object({
  email: MailOutcome,
  link: IdentityPendingLink,
  url: zod.url().describe("The link itself, to copy"),
});

export type IdentityIssuedSignupLink = zod.input<typeof IdentityIssuedSignupLink>;
export type IdentityIssuedSignupLinkOutput = zod.output<typeof IdentityIssuedSignupLink>;

export const IdentityLiveLink = zod.object({
  adminName: zod
    .string()
    .exactOptional()
    .describe("Who issued the link; absent when the API key or the host command did"),
  email: zod
    .email()
    .describe("Where the link went: the Account's email, or the new one it confirms"),
  expiresAt: zod.iso.datetime({ offset: true }),
  serverName: zod.string(),
  state: zod.enum(["live"]),
});

export type IdentityLiveLink = zod.input<typeof IdentityLiveLink>;
export type IdentityLiveLinkOutput = zod.output<typeof IdentityLiveLink>;

export const IdentityLinkView = zod.discriminatedUnion("state", [
  IdentityLiveLink,
  IdentityDeadLink,
]);

export type IdentityLinkView = zod.input<typeof IdentityLinkView>;
export type IdentityLinkViewOutput = zod.output<typeof IdentityLinkView>;

export const IdentityLinksChanged = zod.object({});

export type IdentityLinksChanged = zod.input<typeof IdentityLinksChanged>;
export type IdentityLinksChangedOutput = zod.output<typeof IdentityLinksChanged>;

export const IdentityPasskeyRows = zod.object({
  passkeys: zod.array(IdentityPasskey),
});

export type IdentityPasskeyRows = zod.input<typeof IdentityPasskeyRows>;
export type IdentityPasskeyRowsOutput = zod.output<typeof IdentityPasskeyRows>;

export const IdentityPasskeysChanged = zod.object({
  account: zod.uuid(),
});

export type IdentityPasskeysChanged = zod.input<typeof IdentityPasskeysChanged>;
export type IdentityPasskeysChangedOutput = zod.output<typeof IdentityPasskeysChanged>;

export const IdentityPendingLinks = zod.object({
  links: zod.array(IdentityPendingLink),
});

export type IdentityPendingLinks = zod.input<typeof IdentityPendingLinks>;
export type IdentityPendingLinksOutput = zod.output<typeof IdentityPendingLinks>;

export const identityPinChangeCurrentRegExp = new RegExp("^[0-9]{4}$");
export const identityPinChangeNewRegExp = new RegExp("^[0-9]{4}$");

export const IdentityPinChange = zod.object({
  current: zod
    .string()
    .regex(identityPinChangeCurrentRegExp)
    .exactOptional()
    .describe("The current PIN, left out only by a session that must set a new PIN"),
  new: zod.string().regex(identityPinChangeNewRegExp),
});

export type IdentityPinChange = zod.input<typeof IdentityPinChange>;
export type IdentityPinChangeOutput = zod.output<typeof IdentityPinChange>;

export const identityReconfirmationPinRegExp = new RegExp("^[0-9]{4}$");

export const IdentityReconfirmation = zod.object({
  pin: zod.string().regex(identityReconfirmationPinRegExp),
});

export type IdentityReconfirmation = zod.input<typeof IdentityReconfirmation>;
export type IdentityReconfirmationOutput = zod.output<typeof IdentityReconfirmation>;

export const identityRegeneratedAPIKeyApiKeyRegExp = new RegExp("^ssk_[A-Za-z0-9_-]{43}$");

export const IdentityRegeneratedAPIKey = zod.object({
  apiKey: zod.string().regex(identityRegeneratedAPIKeyApiKeyRegExp),
  createdAt: zod.iso.datetime({ offset: true }),
});

export type IdentityRegeneratedAPIKey = zod.input<typeof IdentityRegeneratedAPIKey>;
export type IdentityRegeneratedAPIKeyOutput = zod.output<typeof IdentityRegeneratedAPIKey>;

export const IdentitySession = zod.object({
  account: IdentityAccount,
  keyPrompt: zod
    .enum(["ask", "never"])
    .describe("Whether the prompt to add a passkey may show after sign-in"),
  pinRequired: zod
    .boolean()
    .describe(
      "The session must set a new PIN before anything but reading the session and signing out",
    ),
});

export type IdentitySession = zod.input<typeof IdentitySession>;
export type IdentitySessionOutput = zod.output<typeof IdentitySession>;

export const IdentitySessionRows = zod.object({
  sessions: zod.array(IdentitySessionRow),
});

export type IdentitySessionRows = zod.input<typeof IdentitySessionRows>;
export type IdentitySessionRowsOutput = zod.output<typeof IdentitySessionRows>;

export const IdentitySessionsChanged = zod.object({
  account: zod.uuid(),
});

export type IdentitySessionsChanged = zod.input<typeof IdentitySessionsChanged>;
export type IdentitySessionsChangedOutput = zod.output<typeof IdentitySessionsChanged>;

export const SettingNoun = zod.object({
  many: zod.string(),
  one: zod.string(),
});

export type SettingNoun = zod.input<typeof SettingNoun>;
export type SettingNounOutput = zod.output<typeof SettingNoun>;

export const SettingCountField = zod.object({
  applies: zod.enum(["live", "restart"]),
  default: zod.int(),
  defaultFrom: zod
    .string()
    .exactOptional()
    .describe("Where the default comes from, when it is another setting's value"),
  floor: zod
    .string()
    .exactOptional()
    .describe("Why the minimum protects someone outside the install"),
  isDefault: zod
    .boolean()
    .describe("The setting has no override; Reset to default leaves it unchanged"),
  key: zod.string(),
  kind: zod.enum(["count"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  max: zod.int(),
  min: zod.int(),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  running: zod.int(),
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  unit: SettingNoun,
  value: zod.int(),
  zero: zod.string().exactOptional().describe("The words for 0, such as No limit"),
});

export type SettingCountField = zod.input<typeof SettingCountField>;
export type SettingCountFieldOutput = zod.output<typeof SettingCountField>;

export const SettingDurationField = zod.object({
  applies: zod.enum(["live", "restart"]),
  default: zod.int().describe("Seconds"),
  defaultFrom: zod
    .string()
    .exactOptional()
    .describe("Where the default comes from, when it is another setting's value"),
  floor: zod
    .string()
    .exactOptional()
    .describe("Why the minimum protects someone outside the install"),
  isDefault: zod
    .boolean()
    .describe("The setting has no override; Reset to default leaves it unchanged"),
  key: zod.string(),
  kind: zod.enum(["duration"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  max: zod.int().describe("Seconds"),
  min: zod.int().describe("Seconds"),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  running: zod.int().describe("Seconds"),
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  step: zod.int().describe("Seconds: a second, minute, hour or day"),
  value: zod.int().describe("Seconds"),
});

export type SettingDurationField = zod.input<typeof SettingDurationField>;
export type SettingDurationFieldOutput = zod.output<typeof SettingDurationField>;

export const IdentitySettings = zod.object({
  addressFailures: SettingCountField,
  addressWindow: SettingDurationField,
  emailLinkLifetime: SettingDurationField,
  pinOffAfter: SettingCountField,
  pinWait: SettingDurationField,
  pinWaitAfter: SettingCountField,
  reconfirmWindow: SettingDurationField,
  resetLinkLifetime: SettingDurationField,
  sessionLifetime: SettingDurationField,
  signupLinkLifetime: SettingDurationField,
});

export type IdentitySettings = zod.input<typeof IdentitySettings>;
export type IdentitySettingsOutput = zod.output<typeof IdentitySettings>;

export const IdentitySettingsPatch = zod.object({
  addressFailures: zod.int().nullish().describe("null resets it to its default"),
  addressWindow: zod.int().nullish().describe("Seconds; null resets it to its default"),
  emailLinkLifetime: zod.int().nullish().describe("Seconds; null resets it to its default"),
  pinOffAfter: zod.int().nullish().describe("null resets it to its default"),
  pinWait: zod.int().nullish().describe("Seconds; null resets it to its default"),
  pinWaitAfter: zod.int().nullish().describe("null resets it to its default"),
  reconfirmWindow: zod.int().nullish().describe("Seconds; null resets it to its default"),
  resetLinkLifetime: zod.int().nullish().describe("Seconds; null resets it to its default"),
  sessionLifetime: zod.int().nullish().describe("Seconds; null resets it to its default"),
  signupLinkLifetime: zod.int().nullish().describe("Seconds; null resets it to its default"),
});

export type IdentitySettingsPatch = zod.input<typeof IdentitySettingsPatch>;
export type IdentitySettingsPatchOutput = zod.output<typeof IdentitySettingsPatch>;

export const identitySignInBodyEmailMax = 254;

export const identitySignInBodyPinRegExp = new RegExp("^[0-9]{4}$");

export const IdentitySignInBody = zod.object({
  email: zod.email().max(identitySignInBodyEmailMax),
  pin: zod.string().regex(identitySignInBodyPinRegExp),
});

export type IdentitySignInBody = zod.input<typeof IdentitySignInBody>;
export type IdentitySignInBodyOutput = zod.output<typeof IdentitySignInBody>;

export const IdentitySignedIn = zod.object({
  account: IdentityAccount,
  keyPrompt: zod
    .enum(["ask", "never"])
    .describe("Whether the prompt to add a passkey may show after sign-in"),
  passkeyPrompt: zod
    .boolean()
    .describe("Show the prompt to add a passkey for this address, once in this session"),
  pinRequired: zod
    .boolean()
    .describe(
      "The session must set a new PIN before anything but reading the session and signing out",
    ),
});

export type IdentitySignedIn = zod.input<typeof IdentitySignedIn>;
export type IdentitySignedInOutput = zod.output<typeof IdentitySignedIn>;

export const IdentitySignupLinkURL = zod.object({
  link: IdentityPendingLink,
  url: zod.url(),
});

export type IdentitySignupLinkURL = zod.input<typeof IdentitySignupLinkURL>;
export type IdentitySignupLinkURLOutput = zod.output<typeof IdentitySignupLinkURL>;

export const IndexersChanged = zod.object({
  indexerId: zod.uuid(),
});

export type IndexersChanged = zod.input<typeof IndexersChanged>;
export type IndexersChangedOutput = zod.output<typeof IndexersChanged>;

export const IndexersSearchMode = zod.object({
  mode: zod.string().describe("The Torznab t= value"),
  params: zod.array(zod.string()).describe("In the Indexer's order"),
});

export type IndexersSearchMode = zod.input<typeof IndexersSearchMode>;
export type IndexersSearchModeOutput = zod.output<typeof IndexersSearchMode>;

export const IndexersWaiting = zod.object({
  since: zod.iso
    .datetime({ offset: true })
    .describe("The home Node's last Sync frame, or the Controller's start when it sent none since"),
  standing: zod
    .enum(["reconnecting", "offline", "mismatched"])
    .describe(
      "How the home Node stands, out of step within the offline grace: until the grace passes, its searches finish indexer_unavailable",
    ),
});

export type IndexersWaiting = zod.input<typeof IndexersWaiting>;
export type IndexersWaitingOutput = zod.output<typeof IndexersWaiting>;

export const IndexersIndexer = zod.object({
  apiKeySet: zod
    .boolean()
    .describe("False when the key was dropped and an Admin must enter it again"),
  createdAt: zod.iso.datetime({ offset: true }),
  homeNodeId: zod.uuid().describe("The Node that reaches the Indexer"),
  id: zod.uuid(),
  name: zod.string(),
  protocol: zod.enum(["torznab"]),
  searchModes: zod.array(IndexersSearchMode).describe("By mode"),
  url: zod.string(),
  waiting: zod
    .union([IndexersWaiting, zod.null()])
    .describe("Why its calls wait for its home Node, null while they go through"),
});

export type IndexersIndexer = zod.input<typeof IndexersIndexer>;
export type IndexersIndexerOutput = zod.output<typeof IndexersIndexer>;

export const IndexersIndexerList = zod.object({
  indexers: zod.array(IndexersIndexer).describe("By name"),
});

export type IndexersIndexerList = zod.input<typeof IndexersIndexerList>;
export type IndexersIndexerListOutput = zod.output<typeof IndexersIndexerList>;

export const indexersNewAPIKeyApiKeyMax = 256;

export const IndexersNewAPIKey = zod.object({
  apiKey: zod.string().max(indexersNewAPIKeyApiKeyMax),
});

export type IndexersNewAPIKey = zod.input<typeof IndexersNewAPIKey>;
export type IndexersNewAPIKeyOutput = zod.output<typeof IndexersNewAPIKey>;

export const indexersNewIndexerApiKeyMax = 256;

export const indexersNewIndexerNameMax = 100;

export const indexersNewIndexerUrlMax = 2048;

export const indexersNewIndexerUrlRegExp = new RegExp("^https?://[^/?#]+(/[^?#]*)?$");

export const IndexersNewIndexer = zod.object({
  apiKey: zod.string().max(indexersNewIndexerApiKeyMax),
  name: zod.string().min(1).max(indexersNewIndexerNameMax),
  url: zod
    .string()
    .max(indexersNewIndexerUrlMax)
    .regex(indexersNewIndexerUrlRegExp)
    .describe("The Torznab feed's base; SlipStream adds /api"),
});

export type IndexersNewIndexer = zod.input<typeof IndexersNewIndexer>;
export type IndexersNewIndexerOutput = zod.output<typeof IndexersNewIndexer>;

export const joinInputBodyDisplayNameMax = 100;

export const joinInputBodyPinRegExp = new RegExp("^[0-9]{4}$");

export const JoinInputBody = zod.object({
  displayName: zod.string().min(1).max(joinInputBodyDisplayNameMax),
  pin: zod.string().regex(joinInputBodyPinRegExp),
});

export type JoinInputBody = zod.input<typeof JoinInputBody>;
export type JoinInputBodyOutput = zod.output<typeof JoinInputBody>;

export const LiveAccessChanged = zod.object({});

export type LiveAccessChanged = zod.input<typeof LiveAccessChanged>;
export type LiveAccessChangedOutput = zod.output<typeof LiveAccessChanged>;

export const LiveResync = zod.object({});

export type LiveResync = zod.input<typeof LiveResync>;
export type LiveResyncOutput = zod.output<typeof LiveResync>;

export const LiveSessionEnded = zod.object({
  reason: zod.enum(["signed-out", "expired", "revoked"]),
});

export type LiveSessionEnded = zod.input<typeof LiveSessionEnded>;
export type LiveSessionEndedOutput = zod.output<typeof LiveSessionEnded>;

export const MailSettings = zod.object({
  fromAddress: zod.string(),
  fromName: zod.string(),
  host: zod.string().describe("Empty until email is set up"),
  passwordSet: zod.boolean().describe("Whether a password is saved; it is never shown"),
  port: zod.int().min(1),
  security: zod.enum(["none", "starttls", "tls"]),
  username: zod.string().describe("Empty when the server takes no sign-in"),
});

export type MailSettings = zod.input<typeof MailSettings>;
export type MailSettingsOutput = zod.output<typeof MailSettings>;

export const mailSettingsPatchFromAddressMax = 254;

export const mailSettingsPatchFromNameMax = 100;

export const mailSettingsPatchHostMax = 253;

export const mailSettingsPatchPasswordMax = 1024;

export const mailSettingsPatchUsernameMax = 254;

export const MailSettingsPatch = zod.object({
  fromAddress: zod.string().max(mailSettingsPatchFromAddressMax).exactOptional(),
  fromName: zod.string().max(mailSettingsPatchFromNameMax).exactOptional(),
  host: zod.string().max(mailSettingsPatchHostMax).exactOptional(),
  password: zod
    .string()
    .max(mailSettingsPatchPasswordMax)
    .exactOptional()
    .describe("Write-only; empty clears it"),
  port: zod.int().min(1).exactOptional(),
  security: zod.enum(["none", "starttls", "tls"]).exactOptional(),
  username: zod.string().max(mailSettingsPatchUsernameMax).exactOptional(),
});

export type MailSettingsPatch = zod.input<typeof MailSettingsPatch>;
export type MailSettingsPatchOutput = zod.output<typeof MailSettingsPatch>;

export const makeFolderInputBodyNameMax = 255;

export const makeFolderInputBodyParentMax = 4096;

export const MakeFolderInputBody = zod.object({
  name: zod.string().min(1).max(makeFolderInputBodyNameMax).describe("The new folder's name"),
  parent: zod
    .string()
    .min(1)
    .max(makeFolderInputBodyParentMax)
    .describe("An existing folder on the Node"),
});

export type MakeFolderInputBody = zod.input<typeof MakeFolderInputBody>;
export type MakeFolderInputBodyOutput = zod.output<typeof MakeFolderInputBody>;

export const newPINInputBodyPinRegExp = new RegExp("^[0-9]{4}$");

export const NewPINInputBody = zod.object({
  pin: zod.string().regex(newPINInputBodyPinRegExp),
});

export type NewPINInputBody = zod.input<typeof NewPINInputBody>;
export type NewPINInputBodyOutput = zod.output<typeof NewPINInputBody>;

export const newSignupLinkInputBodyEmailMax = 254;

export const NewSignupLinkInputBody = zod.object({
  email: zod.email().max(newSignupLinkInputBodyEmailMax),
});

export type NewSignupLinkInputBody = zod.input<typeof NewSignupLinkInputBody>;
export type NewSignupLinkInputBodyOutput = zod.output<typeof NewSignupLinkInputBody>;

export const NodesAssignment = zod.object({
  nodeId: zod.uuid(),
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

export type NodesAssignment = zod.input<typeof NodesAssignment>;
export type NodesAssignmentOutput = zod.output<typeof NodesAssignment>;

export const NodesProbe = zod.object({
  at: zod.iso.datetime({ offset: true }),
  fit: zod.enum(["fits", "weak", "fails"]),
  probe: zod.enum(["engine-starts", "data-writable", "listener-up"]),
  reason: zod
    .enum([
      "engine_not_listening",
      "downloads_not_writable",
      "downloads_on_network_share",
      "data_not_writable",
      "http_not_listening",
    ])
    .exactOptional()
    .describe("Why it is weak or fails; absent when it fits"),
  words: zod.string().exactOptional().describe("What it found, as a sentence; absent when it fits"),
});

export type NodesProbe = zod.input<typeof NodesProbe>;
export type NodesProbeOutput = zod.output<typeof NodesProbe>;

export const NodesHeld = zod.object({
  bytes: zod.int().describe("0 when sizes don't apply"),
  endless: zod
    .boolean()
    .describe("The Node can't finish it on its own, so it stays until it is moved or written off"),
  noun: zod.string().describe("The noun that fits units: torrent, torrents"),
  units: zod.int(),
});

export type NodesHeld = zod.input<typeof NodesHeld>;
export type NodesHeldOutput = zod.output<typeof NodesHeld>;

export const NodesRoleView = zod.object({
  held: NodesHeld.exactOptional().describe(
    "The tied work the Node holds for the Role; absent when none",
  ),
  role: zod.enum([
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
  state: zod.enum(["holding", "away", "lapsed", "draining", "writing-off", "covering"]),
});

export type NodesRoleView = zod.input<typeof NodesRoleView>;
export type NodesRoleViewOutput = zod.output<typeof NodesRoleView>;

export const NodesBoardNode = zod.object({
  assigned: zod
    .array(
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
    )
    .describe("The Roles ticked on the Node, as a set sends them"),
  away: zod.boolean().describe("Out of step, so its checks can't run"),
  controller: zod.boolean(),
  id: zod.uuid(),
  name: zod.string(),
  probes: zod.array(NodesProbe).describe("The last verdict of each probe the Node ran"),
  roles: zod.array(NodesRoleView).describe("Every Role the Node stands in, in table order"),
  suggested: zod
    .array(
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
    )
    .describe(
      "What a new Node's Roles step ticks first: each Acquire Role that has no probe or whose probe fits on the Node",
    ),
});

export type NodesBoardNode = zod.input<typeof NodesBoardNode>;
export type NodesBoardNodeOutput = zod.output<typeof NodesBoardNode>;

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

export const NodesNamed = zod.object({
  id: zod.uuid(),
  name: zod.string(),
});

export type NodesNamed = zod.input<typeof NodesNamed>;
export type NodesNamedOutput = zod.output<typeof NodesNamed>;

export const NodesCheck = zod.object({
  node: NodesNamed,
  reason: zod.enum([
    "role_taken",
    "probe_fails",
    "probe_weak",
    "offline_unchecked",
    "reach_failed",
    "reached",
    "nothing_to_test",
    "node_is_controller",
    "node_holds_work",
    "nothing_held",
    "keeps_role",
  ]),
  role: zod
    .enum([
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
    ])
    .exactOptional()
    .describe("Absent for a check on the whole Node"),
  verdict: zod.enum(["allow", "warn", "block"]),
  words: zod.string(),
});

export type NodesCheck = zod.input<typeof NodesCheck>;
export type NodesCheckOutput = zod.output<typeof NodesCheck>;

export const NodesDialing = zod.object({
  address: zod.string(),
  addresses: zod.array(zod.string()).describe("Where the Controller dials the Node, in order"),
  since: zod.iso.datetime({ offset: true }),
  unreachable: zod
    .boolean()
    .describe("The Node hasn't paired within timings.nodes.dial_report of the paste"),
});

export type NodesDialing = zod.input<typeof NodesDialing>;
export type NodesDialingOutput = zod.output<typeof NodesDialing>;

export const NodesCodeState = zod.object({
  dialing: NodesDialing.exactOptional().describe(
    "The Node address the Controller dials for an open code; absent without one",
  ),
  nodeId: zod
    .uuid()
    .exactOptional()
    .describe("The Node that paired with the code; absent unless used"),
  state: zod.enum(["open", "used", "expired"]),
});

export type NodesCodeState = zod.input<typeof NodesCodeState>;
export type NodesCodeStateOutput = zod.output<typeof NodesCodeState>;

export const NodesDialView = zod.object({
  by: zod
    .enum(["node", "controller"])
    .exactOptional()
    .describe("The side that dialed the live session; absent with none"),
  controllerDials: zod
    .array(zod.string())
    .describe(
      "The addresses the Controller dials the Node at; empty when the Node dials the Controller",
    ),
});

export type NodesDialView = zod.input<typeof NodesDialView>;
export type NodesDialViewOutput = zod.output<typeof NodesDialView>;

export const NodesTorrentView = zod.object({
  done: zod.int().describe("Bytes downloaded"),
  grabId: zod.uuid(),
  peers: zod.int(),
  title: zod.string(),
  wanted: zod.int().describe("Bytes the torrent's selected files hold"),
});

export type NodesTorrentView = zod.input<typeof NodesTorrentView>;
export type NodesTorrentViewOutput = zod.output<typeof NodesTorrentView>;

export const NodesDownloadingResource = zod.object({
  torrents: zod.array(NodesTorrentView).describe("Each Grab whose torrent the client keeps"),
});

export type NodesDownloadingResource = zod.input<typeof NodesDownloadingResource>;
export type NodesDownloadingResourceOutput = zod.output<typeof NodesDownloadingResource>;

export const NodesDrains = zod.object({
  held: NodesHeld,
  kind: zod.enum(["drains"]),
  node: NodesNamed,
  role: zod.enum([
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
  words: zod.string(),
});

export type NodesDrains = zod.input<typeof NodesDrains>;
export type NodesDrainsOutput = zod.output<typeof NodesDrains>;

export const NodesGains = zod.object({
  kind: zod.enum(["gains"]),
  node: NodesNamed,
  role: zod.enum([
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
  words: zod.string(),
});

export type NodesGains = zod.input<typeof NodesGains>;
export type NodesGainsOutput = zod.output<typeof NodesGains>;

export const NodesLetsGo = zod.object({
  kind: zod.enum(["lets-go"]),
  node: NodesNamed,
  role: zod.enum([
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
  words: zod.string(),
});

export type NodesLetsGo = zod.input<typeof NodesLetsGo>;
export type NodesLetsGoOutput = zod.output<typeof NodesLetsGo>;

export const NodesHandedBack = zod.object({
  from: zod.array(NodesNamed).describe("Every Node that held the Role before"),
  kind: zod.enum(["handed-back"]),
  role: zod.enum([
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
  to: NodesNamed,
  words: zod.string(),
});

export type NodesHandedBack = zod.input<typeof NodesHandedBack>;
export type NodesHandedBackOutput = zod.output<typeof NodesHandedBack>;

export const NodesPinned = zod.object({
  kind: zod.enum(["pinned"]),
  node: NodesNamed,
  role: zod.enum([
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
  words: zod.string(),
});

export type NodesPinned = zod.input<typeof NodesPinned>;
export type NodesPinnedOutput = zod.output<typeof NodesPinned>;

export const NodesMoves = zod.object({
  from: NodesNamed,
  held: NodesHeld,
  kind: zod.enum(["moves"]),
  role: zod.enum([
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
  to: NodesNamed,
  words: zod.string(),
});

export type NodesMoves = zod.input<typeof NodesMoves>;
export type NodesMovesOutput = zod.output<typeof NodesMoves>;

export const NodesWritesOff = zod.object({
  held: NodesHeld,
  kind: zod.enum(["writes-off"]),
  node: NodesNamed,
  role: zod.enum([
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
  words: zod.string(),
});

export type NodesWritesOff = zod.input<typeof NodesWritesOff>;
export type NodesWritesOffOutput = zod.output<typeof NodesWritesOff>;

export const NodesLeaves = zod.object({
  kind: zod.enum(["leaves"]),
  node: NodesNamed,
  words: zod.string(),
});

export type NodesLeaves = zod.input<typeof NodesLeaves>;
export type NodesLeavesOutput = zod.output<typeof NodesLeaves>;

export const NodesEffect = zod.discriminatedUnion("kind", [
  NodesGains,
  NodesLetsGo,
  NodesDrains,
  NodesHandedBack,
  NodesPinned,
  NodesMoves,
  NodesWritesOff,
  NodesLeaves,
]);

export type NodesEffect = zod.input<typeof NodesEffect>;
export type NodesEffectOutput = zod.output<typeof NodesEffect>;

export const NodesFacts = zod.object({
  appliedVersion: zod.string(),
  assigned: zod.array(zod.string()),
  httpAddress: zod
    .string()
    .describe("The address the Node's web server bound; empty until it binds"),
  restarting: zod.array(zod.string()),
  running: zod.array(zod.string()),
});

export type NodesFacts = zod.input<typeof NodesFacts>;
export type NodesFactsOutput = zod.output<typeof NodesFacts>;

export const NodesGesture = zod.object({
  from: zod.uuid().exactOptional().describe("The Node the Role moves from; absent for an add"),
  note: zod.string().exactOptional().describe("The words of a Role the gesture hands back or pins"),
  role: zod.enum([
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
  to: zod.uuid(),
  verdict: zod.enum(["allow", "warn", "block"]),
  word: zod.string(),
});

export type NodesGesture = zod.input<typeof NodesGesture>;
export type NodesGestureOutput = zod.output<typeof NodesGesture>;

export const NodesJoinCode = zod.object({
  code: zod.string(),
  expiresAt: zod.iso.datetime({ offset: true }),
  id: zod.uuid(),
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

export const NodesLinkTraffic = zod.object({
  a: zod.uuid().describe("The Controller"),
  abBps: zod.int().describe("Bytes a second from a to b, TLS records included"),
  b: zod.uuid().describe("The Worker"),
  baBps: zod.int().describe("Bytes a second from b to a, TLS records included"),
  rttMs: zod.int().describe("The link's round trip; 0 until its first ping is answered"),
});

export type NodesLinkTraffic = zod.input<typeof NodesLinkTraffic>;
export type NodesLinkTrafficOutput = zod.output<typeof NodesLinkTraffic>;

export const nodesNodeIntentVersionRegExp = new RegExp("^[0-9a-f]{32}$");

export const NodesNode = zod.object({
  addresses: zod.array(zod.string()),
  dial: NodesDialView,
  facts: NodesFacts.exactOptional().describe("Absent while the Node has no session"),
  id: zod.uuid(),
  intentVersion: zod.string().regex(nodesNodeIntentVersionRegExp),
  key: zod.string().describe("The key's fingerprint"),
  link: NodesLink,
  name: zod.string(),
  pairedAt: zod.iso.datetime({ offset: true }),
  probes: zod
    .array(NodesProbe)
    .describe("The last verdict of each probe the Node ran, kept while it is away"),
  roles: zod.array(NodesRoleView).describe("Every Role the Node stands in, in table order"),
  self: zod.boolean(),
  standing: zod
    .enum(["in-step", "reconnecting", "offline", "lapsed", "mismatched"])
    .describe(
      "How the Node stands with the Controller since link.since: in step, reconnecting within the blip, offline, lapsed past the offline grace, or on another version",
    ),
});

export type NodesNode = zod.input<typeof NodesNode>;
export type NodesNodeOutput = zod.output<typeof NodesNode>;

export const nodesNodeDialAddressesMax = 0;

export const NodesNodeDial = zod.object({
  addresses: zod
    .array(zod.string())
    .max(nodesNodeDialAddressesMax)
    .exactOptional()
    .describe("Send [] to stop dialing the Node, which then dials the Controller"),
  nodeAddress: zod
    .string()
    .exactOptional()
    .describe("Dial the Node at this Node address from now on"),
});

export type NodesNodeDial = zod.input<typeof NodesNodeDial>;
export type NodesNodeDialOutput = zod.output<typeof NodesNodeDial>;

export const NodesNodeList = zod.object({
  nodes: zod.array(NodesNode),
});

export type NodesNodeList = zod.input<typeof NodesNodeList>;
export type NodesNodeListOutput = zod.output<typeof NodesNodeList>;

export const NodesTorrentTraffic = zod.object({
  done: zod.int(),
  grabId: zod.uuid(),
  peers: zod.int(),
});

export type NodesTorrentTraffic = zod.input<typeof NodesTorrentTraffic>;
export type NodesTorrentTrafficOutput = zod.output<typeof NodesTorrentTraffic>;

export const NodesNodeTraffic = zod.object({
  downBps: zod.int().describe("Payload bytes a second its torrents download"),
  id: zod.uuid(),
  torrents: zod.array(NodesTorrentTraffic).describe("Each Grab whose torrent its client keeps"),
  upBps: zod.int().describe("Payload bytes a second its torrents upload"),
});

export type NodesNodeTraffic = zod.input<typeof NodesNodeTraffic>;
export type NodesNodeTrafficOutput = zod.output<typeof NodesNodeTraffic>;

export const NodesOpenCode = zod.object({
  createdAt: zod.iso.datetime({ offset: true }),
  expiresAt: zod.iso.datetime({ offset: true }),
  id: zod.uuid(),
});

export type NodesOpenCode = zod.input<typeof NodesOpenCode>;
export type NodesOpenCodeOutput = zod.output<typeof NodesOpenCode>;

export const NodesOpenCodes = zod.object({
  codes: zod
    .array(NodesOpenCode)
    .describe("Every Join Code that may still pair a Node, oldest first"),
});

export type NodesOpenCodes = zod.input<typeof NodesOpenCodes>;
export type NodesOpenCodesOutput = zod.output<typeof NodesOpenCodes>;

export const NodesOutcome = zod.object({
  basis: zod.string().describe("The basis of the rows it was planned on"),
  checks: zod.array(NodesCheck),
  effects: zod.array(NodesEffect),
  forfeit: zod.array(NodesAssignment).describe("Every Node's written-off Roles once applied"),
  next: zod.array(NodesAssignment).describe("Every Node's Roles once applied"),
});

export type NodesOutcome = zod.input<typeof NodesOutcome>;
export type NodesOutcomeOutput = zod.output<typeof NodesOutcome>;

export const NodesPlanChange = zod.object({
  basis: zod.string().describe("The basis of the board or Outcome the change was planned on"),
  set: zod
    .array(NodesAssignment)
    .describe("The Roles of each Node the change names. A Node it doesn't name keeps its own."),
  test: zod
    .boolean()
    .exactOptional()
    .describe(
      "Run the reach test of each added Role whose target is set, from its Node; setRoles always runs them",
    ),
});

export type NodesPlanChange = zod.input<typeof NodesPlanChange>;
export type NodesPlanChangeOutput = zod.output<typeof NodesPlanChange>;

export const NodesRenameBody = zod.object({
  name: zod
    .string()
    .describe("1 to 63 characters, with no space at either end, that no other Node has"),
});

export type NodesRenameBody = zod.input<typeof NodesRenameBody>;
export type NodesRenameBodyOutput = zod.output<typeof NodesRenameBody>;

export const nodesStorageRootContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const NodesStorageRoot = zod.object({
  contentType: zod.string().regex(nodesStorageRootContentTypeRegExp),
  id: zod.uuid(),
  path: zod.string().describe("The folder as its Node cleaned it"),
  state: zod
    .enum([
      "ready",
      "missing",
      "unmounted",
      "foreign",
      "unwritable",
      "unresponsive",
      "checking",
      "offline",
      "written_off",
    ])
    .describe("As listRoots names it"),
  volumeId: zod
    .string()
    .exactOptional()
    .describe("The volume its Node last found it ready on; absent before it did"),
});

export type NodesStorageRoot = zod.input<typeof NodesStorageRoot>;
export type NodesStorageRootOutput = zod.output<typeof NodesStorageRoot>;

export const NodesStorageVolume = zod.object({
  free: zod.int().describe("The bytes SlipStream may write"),
  id: zod.string().describe("The filesystem id, or the volume serial on Windows"),
  kind: zod.enum(["local", "removable", "network"]),
  mount: zod.string(),
  name: zod.string().describe("The volume as its Node names it, such as D: (Media) or /mnt/media"),
  sampledAt: zod.iso.datetime({ offset: true }).describe("When its Node read these figures"),
  total: zod.int(),
});

export type NodesStorageVolume = zod.input<typeof NodesStorageVolume>;
export type NodesStorageVolumeOutput = zod.output<typeof NodesStorageVolume>;

export const NodesStorageResource = zod.object({
  roots: zod
    .array(NodesStorageRoot)
    .describe("The library roots on the Node, by Content Type, oldest first in each"),
  volumes: zod.array(NodesStorageVolume).describe("The volumes its roots sit on"),
});

export type NodesStorageResource = zod.input<typeof NodesStorageResource>;
export type NodesStorageResourceOutput = zod.output<typeof NodesStorageResource>;

export const NodesResources = zod.object({
  downloading: NodesDownloadingResource.exactOptional().describe(
    "Its torrent client; absent unless the Node stands in Downloading and is in step",
  ),
  storage: NodesStorageResource.exactOptional().describe(
    "Its library roots and the volumes they sit on; absent unless the Node stands in Storage. While it is away, as it last reported them",
  ),
});

export type NodesResources = zod.input<typeof NodesResources>;
export type NodesResourcesOutput = zod.output<typeof NodesResources>;

export const NodesRoleEntry = zod.object({
  label: zod.string(),
  pinned: zod.boolean().describe("The Controller holds it whatever the Admin ticks"),
  probe: zod
    .enum(["engine-starts", "data-writable", "listener-up"])
    .exactOptional()
    .describe("The probe a Node runs before it is given the Role; absent for none"),
  role: zod.enum([
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
});

export type NodesRoleEntry = zod.input<typeof NodesRoleEntry>;
export type NodesRoleEntryOutput = zod.output<typeof NodesRoleEntry>;

export const NodesRoleGroup = zod.object({
  group: zod.enum(["acquire", "store", "process", "connect", "reach"]),
  label: zod.string(),
  roles: zod.array(NodesRoleEntry),
});

export type NodesRoleGroup = zod.input<typeof NodesRoleGroup>;
export type NodesRoleGroupOutput = zod.output<typeof NodesRoleGroup>;

export const NodesRoleBoard = zod.object({
  basis: zod.string(),
  gestures: zod
    .array(NodesGesture)
    .describe("The plan's verdict on every move and add a drag can make"),
  groups: zod
    .array(NodesRoleGroup)
    .describe("Every Role Group and its Roles, in the table's order"),
  nodes: zod.array(NodesBoardNode).describe("Every Node, in id order"),
});

export type NodesRoleBoard = zod.input<typeof NodesRoleBoard>;
export type NodesRoleBoardOutput = zod.output<typeof NodesRoleBoard>;

export const NodesRolesChange = zod.object({
  basis: zod.string().describe("The basis of the board or Outcome the change was planned on"),
  set: zod
    .array(NodesAssignment)
    .describe("The Roles of each Node the change names. A Node it doesn't name keeps its own."),
});

export type NodesRolesChange = zod.input<typeof NodesRolesChange>;
export type NodesRolesChangeOutput = zod.output<typeof NodesRolesChange>;

export const NodesSettings = zod.object({
  joinCodeLifetime: SettingDurationField,
  offlineGrace: SettingDurationField,
});

export type NodesSettings = zod.input<typeof NodesSettings>;
export type NodesSettingsOutput = zod.output<typeof NodesSettings>;

export const NodesSettingsChanged = zod.object({
  nodeId: zod.uuid(),
});

export type NodesSettingsChanged = zod.input<typeof NodesSettingsChanged>;
export type NodesSettingsChangedOutput = zod.output<typeof NodesSettingsChanged>;

export const NodesSettingsPatch = zod.object({
  joinCodeLifetime: zod.int().nullish().describe("Seconds; null resets it to its default"),
  offlineGrace: zod.int().nullish().describe("Seconds; null resets it to its default"),
});

export type NodesSettingsPatch = zod.input<typeof NodesSettingsPatch>;
export type NodesSettingsPatchOutput = zod.output<typeof NodesSettingsPatch>;

export const NodesTopologyLink = zod.object({
  a: zod.uuid().describe("The Controller"),
  b: zod.uuid().describe("The Worker"),
  dialer: zod
    .uuid()
    .exactOptional()
    .describe("The Node that dialed the live session; absent while the link is down"),
  rttMs: zod
    .int()
    .exactOptional()
    .describe("The live session's round trip; absent until its first ping is answered"),
  state: zod.enum(["up", "down", "mismatched"]),
});

export type NodesTopologyLink = zod.input<typeof NodesTopologyLink>;
export type NodesTopologyLinkOutput = zod.output<typeof NodesTopologyLink>;

export const NodesTopologyNode = zod.object({
  controller: zod.boolean(),
  id: zod.uuid(),
  lastSeenAt: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("Its last Sync frame; absent while it is online"),
  name: zod.string(),
  probes: zod
    .array(NodesProbe)
    .describe("The last verdict of each probe the Node ran, kept while it is away"),
  resources: NodesResources,
  roles: zod.array(NodesRoleView).describe("Every Role the Node stands in, in table order"),
  self: zod.boolean().describe("The Node answering, which is the Controller"),
  since: zod.iso
    .datetime({ offset: true })
    .describe("When it came in step; while away, its last Sync frame"),
  state: zod.enum(["online", "reconnecting", "offline", "lapsed", "mismatched"]),
});

export type NodesTopologyNode = zod.input<typeof NodesTopologyNode>;
export type NodesTopologyNodeOutput = zod.output<typeof NodesTopologyNode>;

export const NodesTopology = zod.object({
  at: zod.iso.datetime({ offset: true }),
  links: zod
    .array(NodesTopologyLink)
    .describe("The link between the Controller and each Worker, in the Workers' id order"),
  nodes: zod.array(NodesTopologyNode).describe("Every Node, in id order"),
  sentence: zod.string().describe("What stands out across the Nodes, in plain words"),
});

export type NodesTopology = zod.input<typeof NodesTopology>;
export type NodesTopologyOutput = zod.output<typeof NodesTopology>;

export const NodesTraffic = zod.object({
  at: zod.iso.datetime({ offset: true }),
  links: zod
    .array(NodesLinkTraffic)
    .describe("Every live link, between the Controller and a Worker"),
  nodes: zod
    .array(NodesNodeTraffic)
    .describe("Every Node whose torrent client reports, in id order"),
});

export type NodesTraffic = zod.input<typeof NodesTraffic>;
export type NodesTrafficOutput = zod.output<typeof NodesTraffic>;

export const NodesWriteOffBody = zod.object({
  roles: zod
    .array(
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
    )
    .min(1)
    .describe(
      "The Roles whose work the Node holds to give up. A Role still ticked on the Node is unticked.",
    ),
});

export type NodesWriteOffBody = zod.input<typeof NodesWriteOffBody>;
export type NodesWriteOffBodyOutput = zod.output<typeof NodesWriteOffBody>;

export const ownPatchInputBodyDisplayNameMax = 100;

export const OwnPatchInputBody = zod.object({
  displayName: zod.string().min(1).max(ownPatchInputBodyDisplayNameMax).exactOptional(),
  keyPrompt: zod
    .enum(["ask", "never"])
    .exactOptional()
    .describe("Whether the prompt to add a passkey may show after sign-in"),
});

export type OwnPatchInputBody = zod.input<typeof OwnPatchInputBody>;
export type OwnPatchInputBodyOutput = zod.output<typeof OwnPatchInputBody>;

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

export const PairingProblem = zod.object({
  code: zod.enum([
    "controller_unreachable",
    "join_code_unknown",
    "join_code_expired",
    "join_code_used",
    "controller_key_mismatch",
    "internal",
  ]),
  detail: zod.string(),
});

export type PairingProblem = zod.input<typeof PairingProblem>;
export type PairingProblemOutput = zod.output<typeof PairingProblem>;

export const PairingWorkerClaim = zod.object({
  controller: PairingController.exactOptional().describe("Set once joined"),
  nodeAddress: zod
    .string()
    .exactOptional()
    .describe(
      "The SLN1 text to paste into the Controller's Add a Node dialog; absent once joined without a wait",
    ),
  nodeId: zod.uuid().exactOptional().describe("This Node's id; set once joined"),
  problem: PairingProblem.exactOptional().describe("Why a failed Join failed"),
  state: zod.enum(["awaiting-dial", "joined", "failed"]),
  until: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("When this Node stops waiting for the dial"),
});

export type PairingWorkerClaim = zod.input<typeof PairingWorkerClaim>;
export type PairingWorkerClaimOutput = zod.output<typeof PairingWorkerClaim>;

export const passkeyRenameInputBodyNameMax = 100;

export const PasskeyRenameInputBody = zod.object({
  name: zod.string().min(1).max(passkeyRenameInputBodyNameMax),
});

export type PasskeyRenameInputBody = zod.input<typeof PasskeyRenameInputBody>;
export type PasskeyRenameInputBodyOutput = zod.output<typeof PasskeyRenameInputBody>;

export const Problem = zod.object({
  code: zod.string(),
  detail: zod.string().exactOptional(),
  errors: zod.array(FieldError).exactOptional(),
  status: zod.int(),
  title: zod.string(),
});

export type Problem = zod.input<typeof Problem>;
export type ProblemOutput = zod.output<typeof Problem>;

export const rehomeRootInputBodyForgetMin = 0;

export const rehomeRootInputBodyPathMax = 4096;

export const RehomeRootInputBody = zod.object({
  forget: zod
    .int()
    .min(rehomeRootInputBodyForgetMin)
    .exactOptional()
    .describe(
      "The missing file count the Admin confirmed from a rehome_missing answer: those files are forgotten, so their cells read missing and wanted. A different count is refused with rehome_missing again",
    ),
  nodeId: zod
    .uuid()
    .exactOptional()
    .describe("The Storage Node whose disk holds it now; left out, its Node"),
  path: zod
    .string()
    .max(rehomeRootInputBodyPathMax)
    .exactOptional()
    .describe("The folder that holds it now, in its Node's path style; left out, its folder"),
});

export type RehomeRootInputBody = zod.input<typeof RehomeRootInputBody>;
export type RehomeRootInputBodyOutput = zod.output<typeof RehomeRootInputBody>;

export const ResetLinkInputBody = zod.object({
  removePasskeys: zod
    .boolean()
    .exactOptional()
    .describe("Also remove the Account's passkeys, for when someone else may have had access"),
});

export type ResetLinkInputBody = zod.input<typeof ResetLinkInputBody>;
export type ResetLinkInputBodyOutput = zod.output<typeof ResetLinkInputBody>;

export const RootSettingsInputBody = zod.object({
  sampleEvery: zod.int().nullish().describe("Seconds; null resets it to its default"),
});

export type RootSettingsInputBody = zod.input<typeof RootSettingsInputBody>;
export type RootSettingsInputBodyOutput = zod.output<typeof RootSettingsInputBody>;

export const RootsChanged = zod.object({
  rootIds: zod.array(zod.string()),
});

export type RootsChanged = zod.input<typeof RootsChanged>;
export type RootsChangedOutput = zod.output<typeof RootsChanged>;

export const RootsFolder = zod.object({
  name: zod.string(),
  path: zod.string(),
  readable: zod.boolean().describe("SlipStream can list it"),
});

export type RootsFolder = zod.input<typeof RootsFolder>;
export type RootsFolderOutput = zod.output<typeof RootsFolder>;

export const RootsFolderProblem = zod.object({
  account: zod
    .string()
    .exactOptional()
    .describe("For denied, the account SlipStream runs as on the Node"),
  kind: zod.enum(["not_found", "not_a_folder", "denied", "unresponsive", "io"]),
  osText: zod.string().describe("The error as the Node's OS put it"),
});

export type RootsFolderProblem = zod.input<typeof RootsFolderProblem>;
export type RootsFolderProblemOutput = zod.output<typeof RootsFolderProblem>;

export const RootsInUse = zod.object({
  files: zod.int().describe("The files it holds, those waiting to be deleted counted"),
  grabs: zod.int().describe("The Grabs in flight for its Works or into it"),
});

export type RootsInUse = zod.input<typeof RootsInUse>;
export type RootsInUseOutput = zod.output<typeof RootsInUse>;

export const RootsPlace = zod.object({
  free: zod.int(),
  kind: zod
    .enum(["local", "removable", "network"])
    .exactOptional()
    .describe("Absent for a root whose volume its Node hasn't read yet"),
  name: zod.string(),
  path: zod.string(),
  rootId: zod.uuid().exactOptional().describe("The root this place is"),
  total: zod.int(),
});

export type RootsPlace = zod.input<typeof RootsPlace>;
export type RootsPlaceOutput = zod.output<typeof RootsPlace>;

export const RootsSegment = zod.object({
  name: zod.string(),
  path: zod.string(),
});

export type RootsSegment = zod.input<typeof RootsSegment>;
export type RootsSegmentOutput = zod.output<typeof RootsSegment>;

export const RootsVolume = zod.object({
  free: zod.int().describe("The bytes SlipStream may write"),
  id: zod.string().describe("The filesystem id, or the volume serial on Windows"),
  kind: zod.enum(["local", "removable", "network"]),
  mount: zod.string(),
  name: zod.string().describe("The volume as its Node names it, such as D: (Media) or /mnt/media"),
  sampledAt: zod.iso.datetime({ offset: true }),
  total: zod.int(),
});

export type RootsVolume = zod.input<typeof RootsVolume>;
export type RootsVolumeOutput = zod.output<typeof RootsVolume>;

export const RootsListing = zod.object({
  folders: zod.array(RootsFolder).describe("Its subfolders, in natural order without case"),
  hidden: zod.int().describe("The hidden subfolders left out"),
  parent: zod
    .string()
    .describe("The folder above it; empty at a drive, a share or /, and for the places"),
  path: zod.string().describe("The folder as its Node cleaned it; empty for the Node's places"),
  places: zod
    .array(RootsPlace)
    .describe("For the empty path: the roots on the Node, then its drives or mounts"),
  segments: zod
    .array(RootsSegment)
    .describe("Its breadcrumbs as its Node splits it; a drive or a share is one"),
  service: zod
    .boolean()
    .describe("SlipStream runs as a Windows service there, which sees no mapped drive letters"),
  volume: RootsVolume.exactOptional().describe("The volume it sits on, when the Node read it"),
});

export type RootsListing = zod.input<typeof RootsListing>;
export type RootsListingOutput = zod.output<typeof RootsListing>;

export const RootsMadeFolder = zod.object({
  path: zod.string().describe("The new folder, joined by the Node"),
});

export type RootsMadeFolder = zod.input<typeof RootsMadeFolder>;
export type RootsMadeFolderOutput = zod.output<typeof RootsMadeFolder>;

export const RootsMissingFile = zod.object({
  path: zod
    .string()
    .describe("Where the file records its bytes, relative to the root and / separated"),
  reason: zod
    .enum(["missing", "size_differs"])
    .describe("missing: nothing is at that path; size_differs: a file is there at another size"),
});

export type RootsMissingFile = zod.input<typeof RootsMissingFile>;
export type RootsMissingFileOutput = zod.output<typeof RootsMissingFile>;

export const RootsMissingFiles = zod.object({
  count: zod.int().describe("The recorded files not at the new place"),
  first: zod.array(RootsMissingFile).describe("Up to the first 20 of them, in file order"),
});

export type RootsMissingFiles = zod.input<typeof RootsMissingFiles>;
export type RootsMissingFilesOutput = zod.output<typeof RootsMissingFiles>;

export const rootsNewRootContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");
export const rootsNewRootPathMax = 4096;

export const RootsNewRoot = zod.object({
  contentType: zod.string().regex(rootsNewRootContentTypeRegExp),
  makeDefault: zod
    .boolean()
    .exactOptional()
    .describe(
      "Make it its Content Type's default; a Content Type's first root is its default whatever this says",
    ),
  nodeId: zod.uuid().describe("A Node that holds Storage"),
  path: zod
    .string()
    .min(1)
    .max(rootsNewRootPathMax)
    .describe("An absolute path to an existing folder on the Node, in its own path style"),
});

export type RootsNewRoot = zod.input<typeof RootsNewRoot>;
export type RootsNewRootOutput = zod.output<typeof RootsNewRoot>;

export const RootsRootVolume = zod.object({
  free: zod.int().describe("The bytes SlipStream may write"),
  id: zod.string().describe("The filesystem id, or the volume serial on Windows"),
  kind: zod.enum(["local", "removable", "network"]),
  mount: zod.string(),
  name: zod.string().describe("The volume as its Node names it, such as D: (Media) or /mnt/media"),
  rootIds: zod.array(zod.string()).describe("The roots its Node last found ready on it"),
  sampledAt: zod.iso.datetime({ offset: true }).describe("When its Node read these figures"),
  total: zod.int(),
});

export type RootsRootVolume = zod.input<typeof RootsRootVolume>;
export type RootsRootVolumeOutput = zod.output<typeof RootsRootVolume>;

export const RootsNodeVolumes = zod.object({
  live: zod
    .boolean()
    .describe(
      "The Node is online, so its figures are current; else each volume reads as of its sampledAt",
    ),
  nodeId: zod.uuid(),
  volumes: zod.array(RootsRootVolume).describe("Its volumes that hold a root, by name"),
});

export type RootsNodeVolumes = zod.input<typeof RootsNodeVolumes>;
export type RootsNodeVolumesOutput = zod.output<typeof RootsNodeVolumes>;

export const RootsProgress = zod.object({
  checked: zod.int().describe("The recorded files checked so far"),
  total: zod.int().describe("The recorded files to check"),
});

export type RootsProgress = zod.input<typeof RootsProgress>;
export type RootsProgressOutput = zod.output<typeof RootsProgress>;

export const RootsRefusal = zod.object({
  reason: zod
    .enum([
      "not_absolute",
      "not_found",
      "not_a_folder",
      "unwritable",
      "inside_root",
      "contains_root",
      "inside_data",
      "contains_data",
      "inside_downloads",
      "contains_downloads",
      "unresponsive",
      "node_offline",
    ])
    .describe("Why the folder can't be a root"),
  rootId: zod
    .uuid()
    .exactOptional()
    .describe("The root the folder is inside, contains or carries the marker of"),
});

export type RootsRefusal = zod.input<typeof RootsRefusal>;
export type RootsRefusalOutput = zod.output<typeof RootsRefusal>;

export const rootsRootContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const RootsRoot = zod.object({
  contentType: zod.string().regex(rootsRootContentTypeRegExp),
  default: zod.boolean().describe("Where its Content Type's new Works go"),
  fileBytes: zod.int().describe("The bytes of its files"),
  id: zod.uuid(),
  nodeId: zod.uuid().describe("The Storage Node whose disk holds it"),
  path: zod.string().describe("The folder as its Node cleaned it"),
  rehome: RootsProgress.exactOptional().describe(
    "The re-home checking its files at the new place now; absent while none is",
  ),
  state: zod
    .enum([
      "ready",
      "missing",
      "unmounted",
      "foreign",
      "unwritable",
      "unresponsive",
      "checking",
      "offline",
      "written_off",
    ])
    .describe(
      "Only ready takes imports. checking: its Node hasn't reported it yet; offline and written_off come from its Node's standing",
    ),
  volumeId: zod
    .string()
    .exactOptional()
    .describe("The volume its Node last found it ready on; absent before it did"),
  works: zod.int().describe("The Works that live in it"),
});

export type RootsRoot = zod.input<typeof RootsRoot>;
export type RootsRootOutput = zod.output<typeof RootsRoot>;

export const RootsRootList = zod.object({
  roots: zod.array(RootsRoot).describe("By Content Type, oldest first in each"),
});

export type RootsRootList = zod.input<typeof RootsRootList>;
export type RootsRootListOutput = zod.output<typeof RootsRootList>;

export const RootsRootSettings = zod.object({
  sampleEvery: SettingDurationField,
});

export type RootsRootSettings = zod.input<typeof RootsRootSettings>;
export type RootsRootSettingsOutput = zod.output<typeof RootsRootSettings>;

export const RootsSuggestions = zod.object({
  suggestions: zod
    .array(zod.string())
    .describe("Up to 25 folders completing the text, in the separators it was typed with"),
});

export type RootsSuggestions = zod.input<typeof RootsSuggestions>;
export type RootsSuggestionsOutput = zod.output<typeof RootsSuggestions>;

export const RootsVerdict = zod.object({
  path: zod.string().describe("The folder as its Node cleaned it"),
  refusal: RootsRefusal.exactOptional().describe("Why it can't be a root; absent when usable"),
  usable: zod.boolean(),
  volume: RootsVolume.exactOptional().describe("The volume it sits on, when the Node read it"),
});

export type RootsVerdict = zod.input<typeof RootsVerdict>;
export type RootsVerdictOutput = zod.output<typeof RootsVerdict>;

export const RootsVolumeList = zod.object({
  nodes: zod
    .array(RootsNodeVolumes)
    .describe("Each Storage Node with a volume that holds a root, in id order"),
});

export type RootsVolumeList = zod.input<typeof RootsVolumeList>;
export type RootsVolumeListOutput = zod.output<typeof RootsVolumeList>;

export const RootsVolumesChanged = zod.object({
  nodeId: zod.uuid(),
  volumeIds: zod.array(zod.string()),
});

export type RootsVolumesChanged = zod.input<typeof RootsVolumesChanged>;
export type RootsVolumesChangedOutput = zod.output<typeof RootsVolumesChanged>;

export const RootsWorkRoot = zod.object({
  rootId: zod.uuid(),
  workId: zod.uuid(),
});

export type RootsWorkRoot = zod.input<typeof RootsWorkRoot>;
export type RootsWorkRootOutput = zod.output<typeof RootsWorkRoot>;

export const ServingWorker = zod.object({
  controller: zod.string().describe("Its Controller's name"),
  fingerprint: zod.string().describe("This Worker's key fingerprint, as its Controller lists it"),
  holding: zod
    .array(zod.string())
    .describe("The labels of the Roles it holds beside Serving, in table order"),
  name: zod.string().describe("This Worker's name"),
  seeding: zod.int().describe("How many torrents it seeds"),
  since: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("Since when this Worker can't reach its Controller; absent while it can"),
});

export type ServingWorker = zod.input<typeof ServingWorker>;
export type ServingWorkerOutput = zod.output<typeof ServingWorker>;

export const SetWorkRootInputBody = zod.object({
  rootId: zod.uuid(),
});

export type SetWorkRootInputBody = zod.input<typeof SetWorkRootInputBody>;
export type SetWorkRootInputBodyOutput = zod.output<typeof SetWorkRootInputBody>;

export const SettingChoiceOption = zod.object({
  label: zod.string(),
  value: zod.string(),
});

export type SettingChoiceOption = zod.input<typeof SettingChoiceOption>;
export type SettingChoiceOptionOutput = zod.output<typeof SettingChoiceOption>;

export const SettingChoiceField = zod.object({
  applies: zod.enum(["live", "restart"]),
  default: zod.string(),
  defaultFrom: zod
    .string()
    .exactOptional()
    .describe("Where the default comes from, when it is another setting's value"),
  isDefault: zod
    .boolean()
    .describe("The setting has no override; Reset to default leaves it unchanged"),
  key: zod.string(),
  kind: zod.enum(["choice"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  options: zod.array(SettingChoiceOption),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  running: zod.string(),
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  value: zod.string(),
});

export type SettingChoiceField = zod.input<typeof SettingChoiceField>;
export type SettingChoiceFieldOutput = zod.output<typeof SettingChoiceField>;

export const SettingTimes = zod.object({
  end: zod.string().describe("HH:MM, before start when the window crosses midnight"),
  start: zod.string().describe("HH:MM"),
});

export type SettingTimes = zod.input<typeof SettingTimes>;
export type SettingTimesOutput = zod.output<typeof SettingTimes>;

export const SettingWindowField = zod.object({
  applies: zod.enum(["live", "restart"]),
  default: SettingTimes,
  defaultFrom: zod
    .string()
    .exactOptional()
    .describe("Where the default comes from, when it is another setting's value"),
  isDefault: zod
    .boolean()
    .describe("The setting has no override; Reset to default leaves it unchanged"),
  key: zod.string(),
  kind: zod.enum(["window"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  running: SettingTimes,
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  value: SettingTimes,
  zone: zod.string().describe("The time zone's name, such as Europe/London"),
});

export type SettingWindowField = zod.input<typeof SettingWindowField>;
export type SettingWindowFieldOutput = zod.output<typeof SettingWindowField>;

export const SettingPortField = zod.object({
  applies: zod.enum(["live", "restart"]),
  default: zod.int(),
  defaultFrom: zod
    .string()
    .exactOptional()
    .describe("Where the default comes from, when it is another setting's value"),
  isDefault: zod
    .boolean()
    .describe("The setting has no override; Reset to default leaves it unchanged"),
  key: zod.string(),
  kind: zod.enum(["port"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  running: zod.int(),
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  value: zod.int(),
});

export type SettingPortField = zod.input<typeof SettingPortField>;
export type SettingPortFieldOutput = zod.output<typeof SettingPortField>;

export const SettingSwitchField = zod.object({
  applies: zod.enum(["live", "restart"]),
  default: zod.boolean(),
  defaultFrom: zod
    .string()
    .exactOptional()
    .describe("Where the default comes from, when it is another setting's value"),
  isDefault: zod
    .boolean()
    .describe("The setting has no override; Reset to default leaves it unchanged"),
  key: zod.string(),
  kind: zod.enum(["switch"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  running: zod.boolean(),
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  value: zod.boolean(),
});

export type SettingSwitchField = zod.input<typeof SettingSwitchField>;
export type SettingSwitchFieldOutput = zod.output<typeof SettingSwitchField>;

export const SettingField = zod.discriminatedUnion("kind", [
  SettingDurationField,
  SettingCountField,
  SettingChoiceField,
  SettingWindowField,
  SettingPortField,
  SettingSwitchField,
  SettingSpaceField,
]);

export type SettingField = zod.input<typeof SettingField>;
export type SettingFieldOutput = zod.output<typeof SettingField>;

export const ShellAlerts = zod.object({
  health: zod
    .union([zod.literal("warn"), zod.literal("fail"), zod.literal(null)])
    .nullable()
    .describe("The worst level across every Health category, null while nothing is wrong"),
  nodes: zod
    .union([zod.literal("warn"), zod.literal("fail"), zod.literal(null)])
    .nullable()
    .describe(
      "warn while a Node is away past the blip or the Controller covers a Role for one, fail while a Node is out of step past the offline grace, null otherwise",
    ),
  tasks: zod
    .union([zod.literal("warn"), zod.literal("fail"), zod.literal(null)])
    .nullable()
    .describe(
      "The worst level in the Scheduled tasks Health category, null while nothing is wrong",
    ),
});

export type ShellAlerts = zod.input<typeof ShellAlerts>;
export type ShellAlertsOutput = zod.output<typeof ShellAlerts>;

export const shellCountsActivityMin = 0;

export const ShellCounts = zod.object({
  activity: zod.int().min(shellCountsActivityMin).describe("The Grabs neither imported nor failed"),
  alerts: ShellAlerts,
});

export type ShellCounts = zod.input<typeof ShellCounts>;
export type ShellCountsOutput = zod.output<typeof ShellCounts>;

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

export const SystemConfigSection = zod.object({
  fields: zod.array(SettingField),
  id: zod.enum(["web", "log"]),
});

export type SystemConfigSection = zod.input<typeof SystemConfigSection>;
export type SystemConfigSectionOutput = zod.output<typeof SystemConfigSection>;

export const SystemHealth = zod.object({
  status: zod.enum(["ok"]),
});

export type SystemHealth = zod.input<typeof SystemHealth>;
export type SystemHealthOutput = zod.output<typeof SystemHealth>;

export const SystemHttpFacts = zod.object({
  configured: zod
    .string()
    .describe("http.listen at start; listeners.http differs after a fallback to a later port"),
});

export type SystemHttpFacts = zod.input<typeof SystemHttpFacts>;
export type SystemHttpFactsOutput = zod.output<typeof SystemHttpFacts>;

export const SystemListeners = zod.object({
  control: zod.string(),
  http: zod.string(),
  link: zod.string(),
  torrent: zod.string(),
});

export type SystemListeners = zod.input<typeof SystemListeners>;
export type SystemListenersOutput = zod.output<typeof SystemListeners>;

export const SystemLogRecord = zod.object({
  attrs: zod.unknown().describe("The record's other attributes"),
  level: zod.enum(["debug", "info", "warn", "error"]),
  msg: zod.string(),
  seq: zod.int(),
  subsystem: zod.string(),
  time: zod.iso.datetime({ offset: true }),
});

export type SystemLogRecord = zod.input<typeof SystemLogRecord>;
export type SystemLogRecordOutput = zod.output<typeof SystemLogRecord>;

export const SystemLogReset = zod.object({
  reason: zod.enum(["lapped", "restarted"]),
  startedAt: zod.iso.datetime({ offset: true }).describe("When the serving process started"),
});

export type SystemLogReset = zod.input<typeof SystemLogReset>;
export type SystemLogResetOutput = zod.output<typeof SystemLogReset>;

export const SystemNodeConfig = zod.object({
  sections: zod.array(SystemConfigSection),
});

export type SystemNodeConfig = zod.input<typeof SystemNodeConfig>;
export type SystemNodeConfigOutput = zod.output<typeof SystemNodeConfig>;

export const SystemPasskeys = zod.object({
  available: zod.boolean(),
  reason: zod
    .enum(["ip_address", "plain_http", "not_a_domain"])
    .exactOptional()
    .describe(
      "Why it can't: an IP address, a name over plain HTTP other than localhost, or a host that is not a domain name",
    ),
});

export type SystemPasskeys = zod.input<typeof SystemPasskeys>;
export type SystemPasskeysOutput = zod.output<typeof SystemPasskeys>;

export const SystemRemoved = zod.object({
  at: zod.iso.datetime({ offset: true }),
  controller: zod.string().describe("The name of the Controller that removed the Node"),
});

export type SystemRemoved = zod.input<typeof SystemRemoved>;
export type SystemRemovedOutput = zod.output<typeof SystemRemoved>;

export const SystemReport = zod.object({
  build: SystemBuild,
  dataDir: zod.string(),
  http: SystemHttpFacts,
  listeners: SystemListeners,
  logsDir: zod.string(),
  pid: zod.int(),
  seed: zod.string(),
  startedAt: zod.iso.datetime({ offset: true }),
});

export type SystemReport = zod.input<typeof SystemReport>;
export type SystemReportOutput = zod.output<typeof SystemReport>;

export const SystemRestarting = zod.object({
  boundPort: zod.int().describe("The port the stopping process serves on"),
  nextPort: zod
    .int()
    .exactOptional()
    .describe("The port the successor binds, when http.listen waits for this restart"),
  startedAt: zod.iso
    .datetime({ offset: true })
    .describe("When the stopping process started; its successor's startedAt is later"),
});

export type SystemRestarting = zod.input<typeof SystemRestarting>;
export type SystemRestartingOutput = zod.output<typeof SystemRestarting>;

export const SystemStatus = zod.object({
  claim: zod.enum(["unclaimed", "controller", "worker"]),
  passkeys: SystemPasskeys.describe(
    "Whether the address this request reached SlipStream at can hold a passkey",
  ),
  removed: SystemRemoved.exactOptional().describe(
    "Set on an Unclaimed Node its Controller removed, until it joins again",
  ),
});

export type SystemStatus = zod.input<typeof SystemStatus>;
export type SystemStatusOutput = zod.output<typeof SystemStatus>;

export const TargetsAnyCut = zod.object({
  kind: zod.enum(["any"]).describe("Any cut: one file per Target, of whatever cut"),
});

export type TargetsAnyCut = zod.input<typeof TargetsAnyCut>;
export type TargetsAnyCutOutput = zod.output<typeof TargetsAnyCut>;

export const TargetsChanged = zod.object({});

export type TargetsChanged = zod.input<typeof TargetsChanged>;
export type TargetsChangedOutput = zod.output<typeof TargetsChanged>;

export const targetsCutContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const TargetsCut = zod.object({
  aliases: zod
    .array(zod.string())
    .describe(
      "The phrases that name it in a title or file name, a custom cut's name first; Standard is read where none is named",
    ),
  contentType: zod
    .string()
    .regex(targetsCutContentTypeRegExp)
    .exactOptional()
    .describe("The Content Type that adds it, for origin type"),
  key: zod.string(),
  name: zod.string(),
  origin: zod
    .enum(["builtin", "type", "custom"])
    .describe("Built into SlipStream, added by a Content Type, or added by an Admin"),
});

export type TargetsCut = zod.input<typeof TargetsCut>;
export type TargetsCutOutput = zod.output<typeof TargetsCut>;

export const targetsUnwantedFilesBytesMin = 0;

export const targetsUnwantedFilesFilesMin = 0;

export const TargetsUnwantedFiles = zod.object({
  bytes: zod.int().min(targetsUnwantedFilesBytesMin),
  files: zod.int().min(targetsUnwantedFilesFilesMin),
});

export type TargetsUnwantedFiles = zod.input<typeof TargetsUnwantedFiles>;
export type TargetsUnwantedFilesOutput = zod.output<typeof TargetsUnwantedFiles>;

export const targetsImpactDownloadingMin = 0;

export const targetsImpactNewlyWantedMin = 0;

export const targetsImpactOwnListMin = 0;

export const targetsImpactRenamedMin = 0;

export const targetsImpactWorksMin = 0;

export const TargetsImpact = zod.object({
  downloading: zod
    .int()
    .min(targetsImpactDownloadingMin)
    .describe("Downloads in flight for cells that go; their files follow the save's choice"),
  newlyWanted: zod.int().min(targetsImpactNewlyWantedMin).describe("Cells that become wanted"),
  ownList: zod
    .int()
    .min(targetsImpactOwnListMin)
    .describe("Works that keep their own Edition list through a change to their Profile's"),
  renamed: zod
    .int()
    .min(targetsImpactRenamedMin)
    .describe("Files that still fill a cell and change name"),
  unwanted: TargetsUnwantedFiles.describe(
    "Files that fill a cell now and none after: deleted, or kept as Kept files, as the save chooses",
  ),
  works: zod
    .int()
    .min(targetsImpactWorksMin)
    .describe("Works whose cells, fills or file names change"),
});

export type TargetsImpact = zod.input<typeof TargetsImpact>;
export type TargetsImpactOutput = zod.output<typeof TargetsImpact>;

export const TargetsCutChange = zod.object({
  cut: TargetsCut,
  impact: TargetsImpact,
});

export type TargetsCutChange = zod.input<typeof TargetsCutChange>;
export type TargetsCutChangeOutput = zod.output<typeof TargetsCutChange>;

export const TargetsCutList = zod.object({
  cuts: zod
    .array(TargetsCut)
    .describe(
      "The built-in cuts, Standard first, then each Content Type's by type, then the custom ones by name",
    ),
});

export type TargetsCutList = zod.input<typeof TargetsCutList>;
export type TargetsCutListOutput = zod.output<typeof TargetsCutList>;

export const TargetsNamedCuts = zod.object({
  cuts: zod.array(zod.string()).min(1).describe("Cut keys in list order"),
  kind: zod
    .enum(["cuts"])
    .describe(
      "Named cuts: with Standard, each other cut once seen for an Item; without it, every cut outright",
    ),
});

export type TargetsNamedCuts = zod.input<typeof TargetsNamedCuts>;
export type TargetsNamedCutsOutput = zod.output<typeof TargetsNamedCuts>;

export const TargetsEditions = zod.discriminatedUnion("kind", [TargetsAnyCut, TargetsNamedCuts]);

export type TargetsEditions = zod.input<typeof TargetsEditions>;
export type TargetsEditionsOutput = zod.output<typeof TargetsEditions>;

export const TargetsEntry = zod.object({
  sourcing: zod.enum(["download"]).describe("How the Target gets its file"),
  targetId: zod.uuid(),
});

export type TargetsEntry = zod.input<typeof TargetsEntry>;
export type TargetsEntryOutput = zod.output<typeof TargetsEntry>;

export const targetsEstimateBytesMin = 0;

export const targetsEstimateMibPerMinuteMin = 0;

export const TargetsEstimate = zod.object({
  bytes: zod
    .int()
    .min(targetsEstimateBytesMin)
    .nullable()
    .describe("That rate over the runtime of the Profile's Works, null with no estimate yet"),
  mibPerMinute: zod
    .number()
    .min(targetsEstimateMibPerMinuteMin)
    .nullable()
    .describe("What the files filling the Target average, null with no estimate yet"),
  targetId: zod.uuid(),
});

export type TargetsEstimate = zod.input<typeof TargetsEstimate>;
export type TargetsEstimateOutput = zod.output<typeof TargetsEstimate>;

export const TargetsFilter = zod.object({
  attribute: zod.enum(["hdr", "video_codec", "audio_codec", "channels", "language"]),
  mode: zod
    .enum(["only", "not"])
    .describe(
      "Only passes a fact holding one of the values, not one holding none; an unknown fact passes",
    ),
  values: zod
    .array(zod.string())
    .min(1)
    .describe("The attribute's values by name; a language is an ISO 639 code or original"),
});

export type TargetsFilter = zod.input<typeof TargetsFilter>;
export type TargetsFilterOutput = zod.output<typeof TargetsFilter>;

export const targetsNewCutAliasesMax = 20;

export const targetsNewCutNameMax = 100;

export const TargetsNewCut = zod.object({
  aliases: zod
    .array(zod.string())
    .max(targetsNewCutAliasesMax)
    .exactOptional()
    .describe("Other phrases that name it in a title; its name always does"),
  name: zod
    .string()
    .min(1)
    .max(targetsNewCutNameMax)
    .describe("Its key is the name's words in lower case joined by hyphens, fixed once made"),
});

export type TargetsNewCut = zod.input<typeof TargetsNewCut>;
export type TargetsNewCutOutput = zod.output<typeof TargetsNewCut>;

export const targetsNewProfileContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");
export const targetsNewProfileNameMax = 100;

export const TargetsNewProfile = zod.object({
  contentType: zod.string().regex(targetsNewProfileContentTypeRegExp),
  editions: TargetsEditions,
  name: zod.string().min(1).max(targetsNewProfileNameMax),
  targets: zod
    .array(TargetsEntry)
    .describe("In Profile order; none is a Profile whose Works want nothing"),
});

export type TargetsNewProfile = zod.input<typeof TargetsNewProfile>;
export type TargetsNewProfileOutput = zod.output<typeof TargetsNewProfile>;

export const targetsSizeMaxMin = 0;

export const targetsSizeMinMin = 0;

export const TargetsSize = zod.object({
  max: zod.number().min(targetsSizeMaxMin).nullable().describe("MiB per minute; null is no limit"),
  min: zod
    .number()
    .min(targetsSizeMinMin)
    .describe("MiB per minute of the runtime the file covers"),
  quality: zod.enum([
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
  ]),
});

export type TargetsSize = zod.input<typeof TargetsSize>;
export type TargetsSizeOutput = zod.output<typeof TargetsSize>;

export const targetsNewTargetNameMax = 100;

export const TargetsNewTarget = zod.object({
  cutoff: zod
    .enum([
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
    ])
    .describe("Any quality of the tier at which a file is final"),
  filters: zod.array(TargetsFilter).nullish().describe("An attribute left out is Any"),
  name: zod.string().min(1).max(targetsNewTargetNameMax),
  sizes: zod
    .array(TargetsSize)
    .nullish()
    .describe("An accepted quality left out takes its floor and no maximum"),
  tiers: zod
    .array(zod.array(zod.string()).nullable())
    .min(1)
    .describe("Quality keys, best tier first; a tier's qualities rank as equal"),
});

export type TargetsNewTarget = zod.input<typeof TargetsNewTarget>;
export type TargetsNewTargetOutput = zod.output<typeof TargetsNewTarget>;

export const TargetsNewWants = zod.object({
  editions: zod
    .union([TargetsEditions, zod.null()])
    .describe("The Work's own Edition list, or null for its Profile's"),
  profileId: zod.uuid(),
});

export type TargetsNewWants = zod.input<typeof TargetsNewWants>;
export type TargetsNewWantsOutput = zod.output<typeof TargetsNewWants>;

export const TargetsPreset = zod.object({
  key: zod.enum(["disc-or-resolution", "anything-better", "resolution-only"]),
  name: zod.string(),
  tiers: zod
    .array(zod.array(zod.string()).nullable())
    .describe(
      "Every quality, best tier first. Over chosen qualities, keep these tiers to them; with no upgrades, put the cutoff at the bottom tier",
    ),
});

export type TargetsPreset = zod.input<typeof TargetsPreset>;
export type TargetsPresetOutput = zod.output<typeof TargetsPreset>;

export const targetsProfileContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");
export const targetsProfileWorksMin = 0;

export const TargetsProfile = zod.object({
  contentType: zod.string().regex(targetsProfileContentTypeRegExp),
  default: zod.boolean().describe("Whether a Work added with no Profile takes it"),
  editions: TargetsEditions,
  id: zod.uuid(),
  name: zod.string(),
  targets: zod
    .array(TargetsEntry)
    .describe("In Profile order: a file is named after the first it fills"),
  works: zod.int().min(targetsProfileWorksMin).describe("The Works wanting from it"),
});

export type TargetsProfile = zod.input<typeof TargetsProfile>;
export type TargetsProfileOutput = zod.output<typeof TargetsProfile>;

export const TargetsProfileChange = zod.object({
  impact: TargetsImpact,
  profile: TargetsProfile,
});

export type TargetsProfileChange = zod.input<typeof TargetsProfileChange>;
export type TargetsProfileChangeOutput = zod.output<typeof TargetsProfileChange>;

export const targetsProfileDetailContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");
export const targetsProfileDetailWorksMin = 0;

export const TargetsProfileDetail = zod.object({
  contentType: zod.string().regex(targetsProfileDetailContentTypeRegExp),
  default: zod.boolean().describe("Whether a Work added with no Profile takes it"),
  editions: TargetsEditions,
  estimates: zod
    .array(TargetsEstimate)
    .describe(
      "One per Target, in Profile order. A Target no file of known runtime fills yet has no estimate",
    ),
  id: zod.uuid(),
  name: zod.string(),
  targets: zod
    .array(TargetsEntry)
    .describe("In Profile order: a file is named after the first it fills"),
  works: zod.int().min(targetsProfileDetailWorksMin).describe("The Works wanting from it"),
});

export type TargetsProfileDetail = zod.input<typeof TargetsProfileDetail>;
export type TargetsProfileDetailOutput = zod.output<typeof TargetsProfileDetail>;

export const TargetsProfileList = zod.object({
  profiles: zod.array(TargetsProfile).describe("By Content Type, then name"),
});

export type TargetsProfileList = zod.input<typeof TargetsProfileList>;
export type TargetsProfileListOutput = zod.output<typeof TargetsProfileList>;

export const TargetsProfileRef = zod.object({
  id: zod.uuid(),
  name: zod.string(),
});

export type TargetsProfileRef = zod.input<typeof TargetsProfileRef>;
export type TargetsProfileRefOutput = zod.output<typeof TargetsProfileRef>;

export const targetsQualityFloorMin = 0;

export const TargetsQuality = zod.object({
  cinema: zod
    .boolean()
    .describe("A cinema recording: kept out of a Target unless an Admin adds it"),
  floor: zod
    .number()
    .min(targetsQualityFloorMin)
    .describe("A new Target's minimum size, in MiB per minute"),
  key: zod.enum([
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
  ]),
  name: zod.string(),
  resolution: zod
    .union([
      zod.literal("2160p"),
      zod.literal("1080p"),
      zod.literal("720p"),
      zod.literal("480p"),
      zod.literal(null),
    ])
    .nullable()
    .describe("Null for cinema sources"),
  source: zod.enum(["tv", "hdtv", "dvd", "webrip", "webdl", "bluray", "remux", "cinema"]),
});

export type TargetsQuality = zod.input<typeof TargetsQuality>;
export type TargetsQualityOutput = zod.output<typeof TargetsQuality>;

export const TargetsQualityList = zod.object({
  presets: zod.array(TargetsPreset).describe("The default first"),
  qualities: zod.array(TargetsQuality).describe("Low to high, the presets' default order"),
});

export type TargetsQualityList = zod.input<typeof TargetsQualityList>;
export type TargetsQualityListOutput = zod.output<typeof TargetsQualityList>;

export const TargetsTarget = zod.object({
  contains: zod
    .array(zod.uuid())
    .describe("The other Targets every file of which also meets this one"),
  createdAt: zod.iso.datetime({ offset: true }),
  cutoff: zod.string().describe("The first quality of the tier at which a file is final"),
  filters: zod.array(TargetsFilter).describe("Every attribute that isn't Any, in attribute order"),
  id: zod.uuid(),
  name: zod.string(),
  profiles: zod.array(TargetsProfileRef).describe("The Profiles holding it, by name"),
  sizes: zod.array(TargetsSize).describe("Every accepted quality's size, in tier order"),
  tiers: zod
    .array(zod.array(zod.string()).nullable())
    .describe("Quality keys, best tier first; a tier's qualities rank as equal"),
});

export type TargetsTarget = zod.input<typeof TargetsTarget>;
export type TargetsTargetOutput = zod.output<typeof TargetsTarget>;

export const TargetsTargetChange = zod.object({
  impact: TargetsImpact,
  target: TargetsTarget,
});

export type TargetsTargetChange = zod.input<typeof TargetsTargetChange>;
export type TargetsTargetChangeOutput = zod.output<typeof TargetsTargetChange>;

export const TargetsTargetList = zod.object({
  targets: zod.array(TargetsTarget).describe("By name"),
});

export type TargetsTargetList = zod.input<typeof TargetsTargetList>;
export type TargetsTargetListOutput = zod.output<typeof TargetsTargetList>;

export const TargetsWants = zod.object({
  editions: TargetsEditions.describe("The effective Edition list"),
  from: zod
    .enum(["profile", "work"])
    .describe("Whether the Edition list is the Profile's or the Work's own"),
  profileId: zod.uuid(),
  profileName: zod.string(),
});

export type TargetsWants = zod.input<typeof TargetsWants>;
export type TargetsWantsOutput = zod.output<typeof TargetsWants>;

export const TargetsWantsChange = zod.object({
  impact: TargetsImpact,
  wants: TargetsWants,
});

export type TargetsWantsChange = zod.input<typeof TargetsWantsChange>;
export type TargetsWantsChangeOutput = zod.output<typeof TargetsWantsChange>;

export const TasksCadence = zod.object({
  field: SettingDurationField,
  inWindow: zod
    .boolean()
    .describe("The task starts in the Maintenance window, every field.value seconds in whole days"),
  words: zod.string().describe("The cadence as the Tasks page says it, such as Every 8 h"),
});

export type TasksCadence = zod.input<typeof TasksCadence>;
export type TasksCadenceOutput = zod.output<typeof TasksCadence>;

export const TasksChanged = zod.object({
  taskId: zod.string(),
});

export type TasksChanged = zod.input<typeof TasksChanged>;
export type TasksChangedOutput = zod.output<typeof TasksChanged>;

export const TasksRunUnit = zod.object({
  cause: zod.string().nullable().describe("Why it failed or was skipped, null when it succeeded"),
  key: zod.string(),
  label: zod.string(),
  node: zod
    .string()
    .describe("The Node it ran on, still named once removed; empty when no Node has its id"),
  nodeId: zod.uuid().describe("The Node it ran on"),
  nodeRemoved: zod.boolean().describe("The Node it ran on was removed since"),
  outcome: zod.enum(["succeeded", "failed", "skipped", "canceled"]),
});

export type TasksRunUnit = zod.input<typeof TasksRunUnit>;
export type TasksRunUnitOutput = zod.output<typeof TasksRunUnit>;

export const TasksRun = zod.object({
  cause: zod.string().nullable().describe("The first failure's cause, null when none"),
  endedAt: zod.iso.datetime({ offset: true }).nullable().describe("Null while it runs"),
  id: zod.int(),
  nodes: zod.array(zod.string()).describe("The Nodes its units ran on"),
  result: zod
    .union([
      zod.literal("succeeded"),
      zod.literal("partial"),
      zod.literal("failed"),
      zod.literal("skipped"),
      zod.literal("canceled"),
      zod.literal(null),
    ])
    .nullable()
    .describe("Null while it runs"),
  startedAt: zod.iso.datetime({ offset: true }),
  summary: zod.string().describe("Such as 5 of 6 steps, seedbox offline; empty while it runs"),
  trigger: zod.enum(["schedule", "catch-up", "run-now", "returned"]),
  units: zod
    .array(TasksRunUnit)
    .describe(
      "Each unit's outcome as the run ended, kept after the unit is gone; empty while it runs",
    ),
});

export type TasksRun = zod.input<typeof TasksRun>;
export type TasksRunOutput = zod.output<typeof TasksRun>;

export const TasksRunNow = zod.object({
  available: zod.boolean(),
  reason: zod
    .enum(["running", "waiting"])
    .exactOptional()
    .describe("Why Run now is unavailable; absent when it is available"),
});

export type TasksRunNow = zod.input<typeof TasksRunNow>;
export type TasksRunNowOutput = zod.output<typeof TasksRunNow>;

export const TasksIdleState = zod.object({
  kind: zod.enum(["idle"]),
  next: zod.iso
    .datetime({ offset: true })
    .describe("When the next run is due; now when it is due already"),
});

export type TasksIdleState = zod.input<typeof TasksIdleState>;
export type TasksIdleStateOutput = zod.output<typeof TasksIdleState>;

export const tasksRunningStateDoneMin = 0;

export const tasksRunningStateTotalMin = 0;

export const TasksRunningState = zod.object({
  done: zod.int().min(tasksRunningStateDoneMin),
  kind: zod.enum(["running"]),
  runId: zod.int(),
  since: zod.iso.datetime({ offset: true }),
  total: zod.int().min(tasksRunningStateTotalMin),
  trigger: zod.enum(["schedule", "catch-up", "run-now", "returned"]),
});

export type TasksRunningState = zod.input<typeof TasksRunningState>;
export type TasksRunningStateOutput = zod.output<typeof TasksRunningState>;

export const TasksWaitingState = zod.object({
  kind: zod.enum(["waiting"]),
  node: zod.string().describe("The Node every unit waits for"),
  since: zod.iso
    .datetime({ offset: true })
    .describe("The Node's last Sync frame, or the Controller's start when it sent none since"),
  standing: zod.enum(["reconnecting", "offline", "lapsed", "mismatched"]),
});

export type TasksWaitingState = zod.input<typeof TasksWaitingState>;
export type TasksWaitingStateOutput = zod.output<typeof TasksWaitingState>;

export const TasksState = zod.discriminatedUnion("kind", [
  TasksIdleState,
  TasksRunningState,
  TasksWaitingState,
]);

export type TasksState = zod.input<typeof TasksState>;
export type TasksStateOutput = zod.output<typeof TasksState>;

export const TasksTask = zod.object({
  about: zod.string(),
  cadence: TasksCadence,
  group: zod.enum(["controller", "acquire", "store", "process", "connect", "reach"]),
  id: zod.string(),
  lastRun: TasksRun.exactOptional().describe("The latest finished run, absent before one has"),
  name: zod.string(),
  nodes: zod.array(zod.string()).describe("The Nodes its units run on"),
  noun: SettingNoun.describe("What its progress counts: 3 of 12 steps"),
  overdue: SettingChoiceField,
  runNow: TasksRunNow,
  state: TasksState,
});

export type TasksTask = zod.input<typeof TasksTask>;
export type TasksTaskOutput = zod.output<typeof TasksTask>;

export const TasksUnit = zod.object({
  cause: zod.string().nullable().describe("Why it last failed or was skipped, null otherwise"),
  dueOnReturn: zod.boolean().describe("It runs once when its Node returns"),
  endedAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When it last ended, null before it has run"),
  key: zod.string(),
  label: zod.string(),
  lastSuccessAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("Null before it has succeeded"),
  node: zod.string().describe("The Node it runs on"),
  nodeId: zod.uuid().describe("The Node it runs on now: where the roster routes portable work"),
  nodeRemoved: zod.boolean().describe("The Node it runs on was removed"),
  outcome: zod
    .union([
      zod.literal("succeeded"),
      zod.literal("failed"),
      zod.literal("skipped"),
      zod.literal("canceled"),
      zod.literal(null),
    ])
    .nullable()
    .describe("Its last outcome, null before it has run"),
});

export type TasksUnit = zod.input<typeof TasksUnit>;
export type TasksUnitOutput = zod.output<typeof TasksUnit>;

export const TasksDetail = zod.object({
  runs: zod.array(TasksRun).describe("Its runs, newest first, up to the runs kept"),
  task: TasksTask,
  units: zod.array(TasksUnit).describe("Each unit it lists now, in run order"),
});

export type TasksDetail = zod.input<typeof TasksDetail>;
export type TasksDetailOutput = zod.output<typeof TasksDetail>;

export const TasksFailurePart = zod.object({
  href: zod.string().describe("The task's page"),
  kind: zod.enum(["failure"]),
  text: zod.string(),
});

export type TasksFailurePart = zod.input<typeof TasksFailurePart>;
export type TasksFailurePartOutput = zod.output<typeof TasksFailurePart>;

export const TasksGroup = zod.object({
  group: zod.enum(["controller", "acquire", "store", "process", "connect", "reach"]),
  tasks: zod.array(TasksTask).min(1),
});

export type TasksGroup = zod.input<typeof TasksGroup>;
export type TasksGroupOutput = zod.output<typeof TasksGroup>;

export const tasksOpenRunDoneMin = 0;

export const tasksOpenRunTotalMin = 0;

export const TasksOpenRun = zod.object({
  done: zod.int().min(tasksOpenRunDoneMin),
  name: zod.string().describe("The task's name"),
  nodes: zod.array(zod.string()).describe("The Nodes its units run on"),
  noun: SettingNoun.describe("What its progress counts: 3 of 12 steps"),
  runId: zod.int(),
  startedAt: zod.iso.datetime({ offset: true }),
  taskId: zod.string(),
  total: zod.int().min(tasksOpenRunTotalMin),
  trigger: zod.enum(["schedule", "catch-up", "run-now", "returned"]),
});

export type TasksOpenRun = zod.input<typeof TasksOpenRun>;
export type TasksOpenRunOutput = zod.output<typeof TasksOpenRun>;

export const TasksOpenRuns = zod.object({
  headerDelaySeconds: zod
    .int()
    .describe(
      "How long a scheduled run is open before the header shows it; a run-now run shows at once",
    ),
  runs: zod.array(TasksOpenRun).describe("Oldest first"),
});

export type TasksOpenRuns = zod.input<typeof TasksOpenRuns>;
export type TasksOpenRunsOutput = zod.output<typeof TasksOpenRuns>;

export const TasksPlainPart = zod.object({
  kind: zod.enum(["plain"]),
  text: zod.string(),
});

export type TasksPlainPart = zod.input<typeof TasksPlainPart>;
export type TasksPlainPartOutput = zod.output<typeof TasksPlainPart>;

export const TasksWarningPart = zod.object({
  href: zod.string().describe("The task's page"),
  kind: zod.enum(["warning"]),
  text: zod.string(),
});

export type TasksWarningPart = zod.input<typeof TasksWarningPart>;
export type TasksWarningPartOutput = zod.output<typeof TasksWarningPart>;

export const TasksPart = zod.discriminatedUnion("kind", [
  TasksPlainPart,
  TasksWarningPart,
  TasksFailurePart,
]);

export type TasksPart = zod.input<typeof TasksPart>;
export type TasksPartOutput = zod.output<typeof TasksPart>;

export const TasksSchedule = zod.object({
  headerDelay: SettingDurationField,
  jitterCap: SettingDurationField,
  jitterPercent: SettingCountField,
  overdue: SettingChoiceField,
  partialWarnIntervals: SettingCountField,
  runsKept: SettingCountField,
  window: SettingWindowField,
});

export type TasksSchedule = zod.input<typeof TasksSchedule>;
export type TasksScheduleOutput = zod.output<typeof TasksSchedule>;

export const TasksOverview = zod.object({
  groups: zod.array(TasksGroup).describe("Each group with a shown task, in the page's fixed order"),
  multinode: zod
    .boolean()
    .describe("The install has more than one Node, so the Tasks page shows a Node column"),
  sentence: zod.array(TasksPart).min(1).describe("The status sentence's parts, in order"),
  settings: TasksSchedule,
});

export type TasksOverview = zod.input<typeof TasksOverview>;
export type TasksOverviewOutput = zod.output<typeof TasksOverview>;

export const TasksProgressed = zod.object({
  done: zod.int(),
  runId: zod.int(),
  taskId: zod.string(),
  total: zod.int(),
});

export type TasksProgressed = zod.input<typeof TasksProgressed>;
export type TasksProgressedOutput = zod.output<typeof TasksProgressed>;

export const TasksRunStarted = zod.object({
  runId: zod.int(),
});

export type TasksRunStarted = zod.input<typeof TasksRunStarted>;
export type TasksRunStartedOutput = zod.output<typeof TasksRunStarted>;

export const TasksSchedulePatch = zod.object({
  headerDelay: zod.int().nullish().describe("Seconds; null resets it to its default"),
  jitterCap: zod.int().nullish().describe("Seconds; null resets it to its default"),
  jitterPercent: zod.int().nullish().describe("null resets it to its default"),
  overdue: zod.string().nullish().describe("catch-up or wait; null resets it to its default"),
  partialWarnIntervals: zod.int().nullish().describe("null resets it to its default"),
  runsKept: zod.int().nullish().describe("null resets it to its default"),
  window: zod
    .union([SettingTimes, zod.null()])
    .exactOptional()
    .describe("null resets it to its default"),
});

export type TasksSchedulePatch = zod.input<typeof TasksSchedulePatch>;
export type TasksSchedulePatchOutput = zod.output<typeof TasksSchedulePatch>;

export const TasksTaskPatch = zod.object({
  cadence: zod.int().nullish().describe("Seconds; null resets it to its default"),
  overdue: zod.string().nullish().describe("catch-up or wait; null resets it to its default"),
});

export type TasksTaskPatch = zod.input<typeof TasksTaskPatch>;
export type TasksTaskPatchOutput = zod.output<typeof TasksTaskPatch>;

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
  monitored: zod.boolean(),
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

export const worksTreeContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const WorksTree = zod.object({
  addedAt: zod.iso.datetime({ offset: true }),
  contentType: zod.string().regex(worksTreeContentTypeRegExp),
  fields: zod.array(WorksFieldValue),
  id: zod.uuid(),
  items: zod
    .array(WorksItem)
    .describe("By their level's order, live before withdrawn, then position"),
  key: zod.string(),
  label: zod.string(),
  levels: zod
    .array(WorksLevel)
    .describe("By depth, then their parent's order, live before withdrawn, then position"),
  refreshedAt: zod.iso.datetime({ offset: true }),
  sortLabel: zod.string(),
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
  icon: zod.enum(["film", "tv", "note", "open-book", "headphones", "panels", "image", "shapes"]),
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

export const worksListedWorkContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const WorksListedWork = zod.object({
  addedAt: zod.iso.datetime({ offset: true }),
  contentType: zod.string().regex(worksListedWorkContentTypeRegExp),
  fields: zod.array(WorksFieldValue).describe("The Work's own fields"),
  id: zod.uuid(),
  key: zod.string(),
  label: zod.string(),
  refreshedAt: zod.iso.datetime({ offset: true }),
  sortLabel: zod.string(),
});

export type WorksListedWork = zod.input<typeof WorksListedWork>;
export type WorksListedWorkOutput = zod.output<typeof WorksListedWork>;

export const worksNewWorkContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");
export const worksNewWorkKeyMax = 200;

export const WorksNewWork = zod.object({
  contentType: zod.string().regex(worksNewWorkContentTypeRegExp),
  key: zod
    .string()
    .min(1)
    .max(worksNewWorkKeyMax)
    .describe("The Work's key at its metadata source"),
  monitor: zod
    .boolean()
    .exactOptional()
    .describe("Monitor a new Work's Items; adding a Work already in the library changes nothing"),
  profileId: zod
    .uuid()
    .exactOptional()
    .describe(
      "The Profile of the Content Type a new Work wants from; left out, the type's default",
    ),
  rootId: zod
    .uuid()
    .exactOptional()
    .describe(
      "The root of the Content Type a new Work lives in; left out, the type's default, or none while the type has no root",
    ),
});

export type WorksNewWork = zod.input<typeof WorksNewWork>;
export type WorksNewWorkOutput = zod.output<typeof WorksNewWork>;

export const WorksWorkChanged = zod.object({
  workId: zod.uuid(),
});

export type WorksWorkChanged = zod.input<typeof WorksWorkChanged>;
export type WorksWorkChangedOutput = zod.output<typeof WorksWorkChanged>;

export const WorksWorkList = zod.object({
  works: zod.array(WorksListedWork).describe("By sort label, then id"),
});

export type WorksWorkList = zod.input<typeof WorksWorkList>;
export type WorksWorkListOutput = zod.output<typeof WorksWorkList>;

export const LiveEventFilesChanged = zod.object({
  event: zod.enum(["files.changed"]),
  data: FilesChanged,
});

export type LiveEventFilesChanged = zod.input<typeof LiveEventFilesChanged>;
export type LiveEventFilesChangedOutput = zod.output<typeof LiveEventFilesChanged>;

export const LiveEventGrabsChanged = zod.object({
  event: zod.enum(["grabs.changed"]),
  data: GrabsChanged,
});

export type LiveEventGrabsChanged = zod.input<typeof LiveEventGrabsChanged>;
export type LiveEventGrabsChangedOutput = zod.output<typeof LiveEventGrabsChanged>;

export const LiveEventGrabsSearchChanged = zod.object({
  event: zod.enum(["grabs.search-changed"]),
  data: GrabsSearchChanged,
});

export type LiveEventGrabsSearchChanged = zod.input<typeof LiveEventGrabsSearchChanged>;
export type LiveEventGrabsSearchChangedOutput = zod.output<typeof LiveEventGrabsSearchChanged>;

export const LiveEventHealthChanged = zod.object({
  event: zod.enum(["health.changed"]),
  data: HealthChanged,
});

export type LiveEventHealthChanged = zod.input<typeof LiveEventHealthChanged>;
export type LiveEventHealthChangedOutput = zod.output<typeof LiveEventHealthChanged>;

export const LiveEventIdentityAccountsChanged = zod.object({
  event: zod.enum(["identity.accounts-changed"]),
  data: IdentityAccountsChanged,
});

export type LiveEventIdentityAccountsChanged = zod.input<typeof LiveEventIdentityAccountsChanged>;
export type LiveEventIdentityAccountsChangedOutput = zod.output<
  typeof LiveEventIdentityAccountsChanged
>;

export const LiveEventIdentityLinksChanged = zod.object({
  event: zod.enum(["identity.links-changed"]),
  data: IdentityLinksChanged,
});

export type LiveEventIdentityLinksChanged = zod.input<typeof LiveEventIdentityLinksChanged>;
export type LiveEventIdentityLinksChangedOutput = zod.output<typeof LiveEventIdentityLinksChanged>;

export const LiveEventIdentityPasskeysChanged = zod.object({
  event: zod.enum(["identity.passkeys-changed"]),
  data: IdentityPasskeysChanged,
});

export type LiveEventIdentityPasskeysChanged = zod.input<typeof LiveEventIdentityPasskeysChanged>;
export type LiveEventIdentityPasskeysChangedOutput = zod.output<
  typeof LiveEventIdentityPasskeysChanged
>;

export const LiveEventIdentitySessionsChanged = zod.object({
  event: zod.enum(["identity.sessions-changed"]),
  data: IdentitySessionsChanged,
});

export type LiveEventIdentitySessionsChanged = zod.input<typeof LiveEventIdentitySessionsChanged>;
export type LiveEventIdentitySessionsChangedOutput = zod.output<
  typeof LiveEventIdentitySessionsChanged
>;

export const LiveEventIndexersChanged = zod.object({
  event: zod.enum(["indexers.changed"]),
  data: IndexersChanged,
});

export type LiveEventIndexersChanged = zod.input<typeof LiveEventIndexersChanged>;
export type LiveEventIndexersChangedOutput = zod.output<typeof LiveEventIndexersChanged>;

export const LiveEventLiveAccessChanged = zod.object({
  event: zod.enum(["live.access-changed"]),
  data: LiveAccessChanged,
});

export type LiveEventLiveAccessChanged = zod.input<typeof LiveEventLiveAccessChanged>;
export type LiveEventLiveAccessChangedOutput = zod.output<typeof LiveEventLiveAccessChanged>;

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

export const LiveEventNodeSettingsChanged = zod.object({
  event: zod.enum(["node.settings-changed"]),
  data: NodesSettingsChanged,
});

export type LiveEventNodeSettingsChanged = zod.input<typeof LiveEventNodeSettingsChanged>;
export type LiveEventNodeSettingsChangedOutput = zod.output<typeof LiveEventNodeSettingsChanged>;

export const LiveEventNodesChanged = zod.object({
  event: zod.enum(["nodes.changed"]),
  data: NodesChanged,
});

export type LiveEventNodesChanged = zod.input<typeof LiveEventNodesChanged>;
export type LiveEventNodesChangedOutput = zod.output<typeof LiveEventNodesChanged>;

export const LiveEventNodesTraffic = zod.object({
  event: zod.enum(["nodes.traffic"]),
  data: NodesTraffic,
});

export type LiveEventNodesTraffic = zod.input<typeof LiveEventNodesTraffic>;
export type LiveEventNodesTrafficOutput = zod.output<typeof LiveEventNodesTraffic>;

export const LiveEventRootsChanged = zod.object({
  event: zod.enum(["roots.changed"]),
  data: RootsChanged,
});

export type LiveEventRootsChanged = zod.input<typeof LiveEventRootsChanged>;
export type LiveEventRootsChangedOutput = zod.output<typeof LiveEventRootsChanged>;

export const LiveEventTargetsChanged = zod.object({
  event: zod.enum(["targets.changed"]),
  data: TargetsChanged,
});

export type LiveEventTargetsChanged = zod.input<typeof LiveEventTargetsChanged>;
export type LiveEventTargetsChangedOutput = zod.output<typeof LiveEventTargetsChanged>;

export const LiveEventTasksChanged = zod.object({
  event: zod.enum(["tasks.changed"]),
  data: TasksChanged,
});

export type LiveEventTasksChanged = zod.input<typeof LiveEventTasksChanged>;
export type LiveEventTasksChangedOutput = zod.output<typeof LiveEventTasksChanged>;

export const LiveEventTasksProgress = zod.object({
  event: zod.enum(["tasks.progress"]),
  data: TasksProgressed,
});

export type LiveEventTasksProgress = zod.input<typeof LiveEventTasksProgress>;
export type LiveEventTasksProgressOutput = zod.output<typeof LiveEventTasksProgress>;

export const LiveEventVolumesChanged = zod.object({
  event: zod.enum(["volumes.changed"]),
  data: RootsVolumesChanged,
});

export type LiveEventVolumesChanged = zod.input<typeof LiveEventVolumesChanged>;
export type LiveEventVolumesChangedOutput = zod.output<typeof LiveEventVolumesChanged>;

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
  LiveEventFilesChanged,
  LiveEventGrabsChanged,
  LiveEventGrabsSearchChanged,
  LiveEventHealthChanged,
  LiveEventIdentityAccountsChanged,
  LiveEventIdentityLinksChanged,
  LiveEventIdentityPasskeysChanged,
  LiveEventIdentitySessionsChanged,
  LiveEventIndexersChanged,
  LiveEventLiveAccessChanged,
  LiveEventLiveResync,
  LiveEventLiveSessionEnded,
  LiveEventNodeSettingsChanged,
  LiveEventNodesChanged,
  LiveEventNodesTraffic,
  LiveEventRootsChanged,
  LiveEventTargetsChanged,
  LiveEventTasksChanged,
  LiveEventTasksProgress,
  LiveEventVolumesChanged,
  LiveEventWorksChanged,
  LiveEventWorksContentTypesChanged,
]);

export type LiveEvent = zod.input<typeof LiveEvent>;
export type LiveEventOutput = zod.output<typeof LiveEvent>;

export const LogEventLogRecord = zod.object({
  event: zod.enum(["log.record"]),
  data: SystemLogRecord,
});

export type LogEventLogRecord = zod.input<typeof LogEventLogRecord>;
export type LogEventLogRecordOutput = zod.output<typeof LogEventLogRecord>;

export const LogEventLogReset = zod.object({
  event: zod.enum(["log.reset"]),
  data: SystemLogReset,
});

export type LogEventLogReset = zod.input<typeof LogEventLogReset>;
export type LogEventLogResetOutput = zod.output<typeof LogEventLogReset>;

export const LogEvent = zod.discriminatedUnion("event", [LogEventLogRecord, LogEventLogReset]);

export type LogEvent = zod.input<typeof LogEvent>;
export type LogEventOutput = zod.output<typeof LogEvent>;
