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
    .describe("Changed at once. SlipStream emails the old address about the change"),
});

export type AccountPatchInputBody = zod.input<typeof AccountPatchInputBody>;
export type AccountPatchInputBodyOutput = zod.output<typeof AccountPatchInputBody>;

export const ActInputBody = zod.object({
  action: zod.enum(["pause", "resume", "announce", "recheck", "retry", "mark-failed", "remove"]),
  deleteFiles: zod.boolean().exactOptional().describe("remove: also delete each download copy"),
  dryRun: zod
    .boolean()
    .exactOptional()
    .describe("remove, mark-failed and recheck: return what it would do and change nothing"),
  ids: zod.array(zod.uuid()).min(1),
});

export type ActInputBody = zod.input<typeof ActInputBody>;
export type ActInputBodyOutput = zod.output<typeof ActInputBody>;

export const Bare = zod.union([zod.number(), zod.string(), zod.array(zod.string())]);

export type Bare = zod.input<typeof Bare>;
export type BareOutput = zod.output<typeof Bare>;

export const CancelInputBody = zod.object({
  blocklist: zod
    .boolean()
    .describe(
      "Also put its Release on the Blocklist; only when the Grab's actions.blocklistable is true",
    ),
});

export type CancelInputBody = zod.input<typeof CancelInputBody>;
export type CancelInputBodyOutput = zod.output<typeof CancelInputBody>;

export const checkFolderInputBodyPathMax = 4096;

export const CheckFolderInputBody = zod.object({
  path: zod.string().min(1).max(checkFolderInputBodyPathMax),
  purpose: zod
    .enum(["root", "downloads", "import"])
    .describe(
      "root: a library root's rules, on a Storage Node; downloads: a downloads folder's, writable, on no network share and apart from every root on the Node; import: a folder to import from, inside one of the Node's library roots or download folders",
    ),
  rootId: zod
    .uuid()
    .exactOptional()
    .describe("For root: the root being re-homed, which the folder may overlap"),
});

export type CheckFolderInputBody = zod.input<typeof CheckFolderInputBody>;
export type CheckFolderInputBodyOutput = zod.output<typeof CheckFolderInputBody>;

export const DialInputBody = zod.object({
  nodeAddress: zod.string().describe("The SLN1 text the waiting Node printed"),
});

export type DialInputBody = zod.input<typeof DialInputBody>;
export type DialInputBodyOutput = zod.output<typeof DialInputBody>;

export const DownloadclientsRefusal = zod.object({
  acceptsAt: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("window: when every announce URL accepts an announce"),
  client: zod
    .string()
    .exactOptional()
    .describe("client_away and client_off: the external client's name"),
  code: zod.enum([
    "not_found",
    "removed",
    "importing",
    "not_held",
    "use_not_this_release",
    "reconciling",
    "offline",
    "client_away",
    "client_off",
    "window",
    "not_running",
    "failed",
    "not_retryable",
  ]),
  node: zod
    .string()
    .exactOptional()
    .describe("client_away and client_off: the name of the external client's Node"),
  workId: zod
    .uuid()
    .exactOptional()
    .describe(
      "importing: the Work whose import holds it; use_not_this_release: the Work whose Pending import waits on it",
    ),
});

export type DownloadclientsRefusal = zod.input<typeof DownloadclientsRefusal>;
export type DownloadclientsRefusalOutput = zod.output<typeof DownloadclientsRefusal>;

export const DownloadclientsActResult = zod.object({
  done: zod.boolean(),
  id: zod.uuid(),
  refusal: DownloadclientsRefusal.exactOptional().describe("Why it wasn't done"),
});

export type DownloadclientsActResult = zod.input<typeof DownloadclientsActResult>;
export type DownloadclientsActResultOutput = zod.output<typeof DownloadclientsActResult>;

export const DownloadclientsCanceled = zod.object({
  grabId: zod.uuid(),
  release: zod.string(),
  workId: zod.uuid(),
});

export type DownloadclientsCanceled = zod.input<typeof DownloadclientsCanceled>;
export type DownloadclientsCanceledOutput = zod.output<typeof DownloadclientsCanceled>;

export const downloadclientsOwesBytesMin = 0;

export const downloadclientsOwesSecondsMin = 0;

export const DownloadclientsOwes = zod.object({
  bytes: zod
    .int()
    .min(downloadclientsOwesBytesMin)
    .describe("Upload still needed, 0 when the rule doesn't count ratio or it is reached"),
  seconds: zod
    .int()
    .min(downloadclientsOwesSecondsMin)
    .describe("Seeding time still needed, 0 when the rule doesn't count it or it is reached"),
});

export type DownloadclientsOwes = zod.input<typeof DownloadclientsOwes>;
export type DownloadclientsOwesOutput = zod.output<typeof DownloadclientsOwes>;

export const DownloadclientsTrackerRef = zod.object({
  id: zod.uuid(),
  name: zod.string(),
});

export type DownloadclientsTrackerRef = zod.input<typeof DownloadclientsTrackerRef>;
export type DownloadclientsTrackerRefOutput = zod.output<typeof DownloadclientsTrackerRef>;

export const DownloadclientsHitAndRun = zod.object({
  clears: zod
    .enum(["none", "time", "ratio", "either", "both"])
    .describe("What meets the rule it owes, or none while no rule is known"),
  owes: DownloadclientsOwes.exactOptional().describe(
    "What it still owes, absent while its Tracker's rule isn't known",
  ),
  torrentId: zod.uuid(),
  tracker: zod
    .union([DownloadclientsTrackerRef, zod.null()])
    .describe("The Tracker it owes, null while that Tracker isn't known"),
});

export type DownloadclientsHitAndRun = zod.input<typeof DownloadclientsHitAndRun>;
export type DownloadclientsHitAndRunOutput = zod.output<typeof DownloadclientsHitAndRun>;

export const downloadclientsPreviewFreesMin = 0;

export const DownloadclientsPreview = zod.object({
  cancels: zod
    .array(DownloadclientsCanceled)
    .describe("The Grabs at downloading that a remove fails"),
  frees: zod
    .int()
    .min(downloadclientsPreviewFreesMin)
    .nullable()
    .describe("Bytes deleting the download copies frees, null while a Node can't report it"),
  hitAndRun: zod
    .array(DownloadclientsHitAndRun)
    .describe(
      "The removed torrents whose removal counts as a hit and run, each the last copy of a torrent that still owes its Tracker",
    ),
  linked: zod.boolean().describe("Some file is also a library file, as a recheck reads it"),
  stopSeeding: zod.array(zod.uuid()).describe("The Works whose complete Torrents stop seeding"),
});

export type DownloadclientsPreview = zod.input<typeof DownloadclientsPreview>;
export type DownloadclientsPreviewOutput = zod.output<typeof DownloadclientsPreview>;

export const DownloadclientsActed = zod.object({
  preview: DownloadclientsPreview.exactOptional().describe("What a dry run would do"),
  results: zod.array(DownloadclientsActResult).describe("One for each id, in the order sent"),
});

export type DownloadclientsActed = zod.input<typeof DownloadclientsActed>;
export type DownloadclientsActedOutput = zod.output<typeof DownloadclientsActed>;

export const DownloadclientsAggregate = zod.object({
  allPaused: zod.boolean().describe("There are downloads and every one is paused"),
  done: zod.int().describe("Bytes the downloads hold, live while sampled"),
  wanted: zod.int().describe("Bytes the downloads want"),
});

export type DownloadclientsAggregate = zod.input<typeof DownloadclientsAggregate>;
export type DownloadclientsAggregateOutput = zod.output<typeof DownloadclientsAggregate>;

export const DownloadclientsChanged = zod.object({
  ids: zod.array(zod.string()).describe("The Torrents whose record changed"),
});

export type DownloadclientsChanged = zod.input<typeof DownloadclientsChanged>;
export type DownloadclientsChangedOutput = zod.output<typeof DownloadclientsChanged>;

export const DownloadclientsClientFacts = zod.object({
  bound: zod
    .string()
    .exactOptional()
    .describe("The address and port it listens on, absent while it has no listener"),
  libtorrent: zod.string().describe("libtorrent's version"),
  listen: zod.string().describe("The listen address and port it is set to"),
  listenError: zod.string().exactOptional().describe("Why it has no listener"),
  peerIdPrefix: zod
    .string()
    .describe("The peer ID prefix it announces, which a Tracker's allow list names"),
  reach: zod
    .enum(["reachable", "unproven"])
    .describe(
      "reachable once a peer from the internet has connected to the address it listens on now",
    ),
  reachedAt: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("When the first such peer connected"),
  userAgent: zod.string().describe("The user agent it announces"),
});

export type DownloadclientsClientFacts = zod.input<typeof DownloadclientsClientFacts>;
export type DownloadclientsClientFactsOutput = zod.output<typeof DownloadclientsClientFacts>;

export const downloadclientsDiskGuardErrorBelowMin = 0;

export const DownloadclientsDiskGuard = zod.object({
  errorBelow: zod
    .int()
    .min(downloadclientsDiskGuardErrorBelowMin)
    .describe("The volume's error level, in bytes"),
  hint: zod
    .string()
    .describe(
      "What explains the sentence: the volume, its free space, its error level and what to do",
    ),
  sentence: zod.string().describe("The guard's sentence, as Health shows it"),
});

export type DownloadclientsDiskGuard = zod.input<typeof DownloadclientsDiskGuard>;
export type DownloadclientsDiskGuardOutput = zod.output<typeof DownloadclientsDiskGuard>;

export const downloadclientsDownloadsVolumeFreeMin = 0;

export const downloadclientsDownloadsVolumeTotalMin = 0;

export const DownloadclientsDownloadsVolume = zod.object({
  free: zod.int().min(downloadclientsDownloadsVolumeFreeMin).describe("Bytes"),
  mount: zod.string().describe("As the Node names it, the same as Storage names the volume"),
  name: zod.string().describe("Such as D: (Media)"),
  sampledAt: zod.iso.datetime({ offset: true }),
  total: zod.int().min(downloadclientsDownloadsVolumeTotalMin).describe("Bytes"),
});

export type DownloadclientsDownloadsVolume = zod.input<typeof DownloadclientsDownloadsVolume>;
export type DownloadclientsDownloadsVolumeOutput = zod.output<
  typeof DownloadclientsDownloadsVolume
>;

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

export const downloadclientsClientFolderClientMax = 4096;

export const downloadclientsClientFolderNodeMax = 4096;

export const DownloadclientsClientFolder = zod.object({
  client: zod
    .string()
    .min(1)
    .max(downloadclientsClientFolderClientMax)
    .describe("The folder as the client reports it, such as /downloads"),
  node: zod
    .string()
    .min(1)
    .max(downloadclientsClientFolderNodeMax)
    .describe("The same folder on the client's Node, such as D:\\Torrents"),
});

export type DownloadclientsClientFolder = zod.input<typeof DownloadclientsClientFolder>;
export type DownloadclientsClientFolderOutput = zod.output<typeof DownloadclientsClientFolder>;

export const downloadclientsDaemonHostMax = 253;

export const downloadclientsDaemonPortMax = 65535;

export const DownloadclientsDaemon = zod.object({
  host: zod.string().min(1).max(downloadclientsDaemonHostMax),
  port: zod.int().min(1).max(downloadclientsDaemonPortMax),
});

export type DownloadclientsDaemon = zod.input<typeof DownloadclientsDaemon>;
export type DownloadclientsDaemonOutput = zod.output<typeof DownloadclientsDaemon>;

export const DownloadclientsPollProblem = zod.object({
  cause: zod
    .string()
    .describe("Why the client isn't responding, refused the sign-in or needs attention"),
  fields: zod.array(zod.string()).describe("The form fields an Admin changes to fix it"),
  sentence: zod.string().describe("What happened and what to do, as a sentence"),
});

export type DownloadclientsPollProblem = zod.input<typeof DownloadclientsPollProblem>;
export type DownloadclientsPollProblemOutput = zod.output<typeof DownloadclientsPollProblem>;

export const DownloadclientsRoutedTracker = zod.object({
  id: zod.uuid(),
  name: zod.string(),
  stored: zod
    .boolean()
    .describe(
      "An Admin picked this client for the Tracker's Grabs from some home Node; false while they come here by default",
    ),
});

export type DownloadclientsRoutedTracker = zod.input<typeof DownloadclientsRoutedTracker>;
export type DownloadclientsRoutedTrackerOutput = zod.output<typeof DownloadclientsRoutedTracker>;

export const downloadclientsExternalClientTorrentsMin = 0;

export const DownloadclientsExternalClient = zod.object({
  actions: zod
    .array(zod.enum(["pause", "resume", "recheck", "announce"]))
    .describe("The actions its kind supports"),
  address: zod.string().describe("The whole address, with the kind's usual path filled in"),
  apiKeySaved: zod.boolean().describe("An API key is saved. Secrets are never returned"),
  away: zod
    .boolean()
    .describe(
      "It isn't responding, refused the sign-in or needs attention while its Node is in step, so its card is red",
    ),
  checkEvery: SettingDurationField,
  clientFolders: zod.array(DownloadclientsClientFolder),
  daemon: zod
    .union([DownloadclientsDaemon, zod.null()])
    .describe("Deluge's daemon; null for the only one and for every other kind"),
  folder: zod
    .string()
    .describe("The download folder as the client sees it; empty for the client's default"),
  id: zod.uuid(),
  importDelay: SettingDurationField,
  kind: zod.enum(["qbittorrent", "transmission", "deluge", "rtorrent"]),
  label: zod.string(),
  name: zod.string(),
  nodeId: zod.uuid(),
  on: zod.boolean(),
  passwordSaved: zod.boolean().describe("A password is saved. Secrets are never returned"),
  problem: zod
    .union([DownloadclientsPollProblem, zod.null()])
    .describe("Why its last poll failed; null while it responds and before its first poll"),
  productVersion: zod
    .string()
    .describe("The product's version at the last sign-in, such as 4.1.1; empty before one"),
  reach: zod.enum([
    "responding",
    "not_responding",
    "sign_in_refused",
    "needs_attention",
    "off",
    "node_offline",
    "unchecked",
  ]),
  settingsVersion: zod.int().min(1).describe("Bumped on every edit"),
  since: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When its poll state began; null before its Node first polled it"),
  torrents: zod.int().min(downloadclientsExternalClientTorrentsMin).describe("Its live Torrents"),
  trackers: zod
    .array(DownloadclientsRoutedTracker)
    .describe("The private Trackers whose Grabs download on it now, by name"),
  username: zod.string().describe("Empty when it signs in with none"),
  webUi: zod.string().nullable().describe("The client's own web UI; null for a kind with none"),
});

export type DownloadclientsExternalClient = zod.input<typeof DownloadclientsExternalClient>;
export type DownloadclientsExternalClientOutput = zod.output<typeof DownloadclientsExternalClient>;

export const downloadclientsLimitsActiveDownloadsMin = 0;

export const downloadclientsLimitsDownloadLimitMin = 0;

export const downloadclientsLimitsUploadLimitMin = 0;

export const DownloadclientsLimits = zod.object({
  activeChecks: zod.int().min(1),
  activeDownloads: zod
    .int()
    .min(downloadclientsLimitsActiveDownloadsMin)
    .describe("0 for no limit"),
  connections: zod.int().min(1),
  dht: zod.boolean(),
  downloadLimit: zod
    .int()
    .min(downloadclientsLimitsDownloadLimitMin)
    .describe("Bytes a second, 0 for no limit"),
  stalledAfterSeconds: zod.int().min(1),
  uploadLimit: zod
    .int()
    .min(downloadclientsLimitsUploadLimitMin)
    .describe("Bytes a second, 0 for no limit"),
  utp: zod.boolean(),
});

export type DownloadclientsLimits = zod.input<typeof DownloadclientsLimits>;
export type DownloadclientsLimitsOutput = zod.output<typeof DownloadclientsLimits>;

export const downloadclientsTotalsActiveSecondsMin = 0;

export const downloadclientsTotalsDownloadedMin = 0;

export const downloadclientsTotalsSeedingSecondsMin = 0;

export const downloadclientsTotalsUploadedMin = 0;

export const DownloadclientsTotals = zod.object({
  activeSeconds: zod.int().min(downloadclientsTotalsActiveSecondsMin),
  downloaded: zod.int().min(downloadclientsTotalsDownloadedMin).describe("Bytes"),
  seedingSeconds: zod.int().min(downloadclientsTotalsSeedingSecondsMin),
  uploaded: zod.int().min(downloadclientsTotalsUploadedMin).describe("Bytes"),
});

export type DownloadclientsTotals = zod.input<typeof DownloadclientsTotals>;
export type DownloadclientsTotalsOutput = zod.output<typeof DownloadclientsTotals>;

export const downloadclientsClientCardTorrentsMin = 0;

export const DownloadclientsClientCard = zod.object({
  client: DownloadclientsClientFacts.exactOptional().describe(
    "The built-in client as it reports itself, absent until it does and while the Node is out of step",
  ),
  diskGuard: DownloadclientsDiskGuard.exactOptional().describe(
    "Set while the disk guard holds the Node's new downloads: its downloads volume is below its error level",
  ),
  downloads: DownloadclientsDownloadsVolume.exactOptional().describe(
    "The downloads folder's volume as last sampled, absent until the first sample",
  ),
  externals: zod
    .array(DownloadclientsExternalClient)
    .describe("The Node's external clients, by name"),
  grabsTo: zod
    .uuid()
    .nullable()
    .describe("The external client the Node's new Grabs go to; null for its built-in client"),
  inStep: zod.boolean().describe("The Node is in step; while it isn't the card has no client"),
  limits: DownloadclientsLimits,
  nodeId: zod.uuid(),
  nodeName: zod.string(),
  torrents: zod
    .int()
    .min(downloadclientsClientCardTorrentsMin)
    .describe("The built-in client's live Torrents"),
  totals: DownloadclientsTotals.describe("Added up over the built-in client's live Torrents"),
  trackers: zod
    .array(DownloadclientsRoutedTracker)
    .describe("The private Trackers whose Grabs download on the built-in client now, by name"),
});

export type DownloadclientsClientCard = zod.input<typeof DownloadclientsClientCard>;
export type DownloadclientsClientCardOutput = zod.output<typeof DownloadclientsClientCard>;

export const downloadclientsClientEditAddressMax = 2048;

export const downloadclientsClientEditApiKeyMax = 1024;

export const downloadclientsClientEditClientFoldersMax = 50;

export const downloadclientsClientEditFolderMax = 4096;

export const downloadclientsClientEditLabelMax = 100;

export const downloadclientsClientEditNameMax = 100;

export const downloadclientsClientEditPasswordMax = 1024;

export const downloadclientsClientEditUsernameMax = 256;

export const DownloadclientsClientEdit = zod.object({
  address: zod.string().min(1).max(downloadclientsClientEditAddressMax).exactOptional(),
  apiKey: zod
    .string()
    .max(downloadclientsClientEditApiKeyMax)
    .exactOptional()
    .describe("Write-only; empty keeps the saved one"),
  checkEvery: zod.int().nullish().describe("Seconds; null resets it to its default"),
  clientFolders: zod
    .array(DownloadclientsClientFolder)
    .max(downloadclientsClientEditClientFoldersMax)
    .exactOptional(),
  daemon: zod
    .union([DownloadclientsDaemon, zod.null()])
    .exactOptional()
    .describe("Null for Deluge's only daemon"),
  folder: zod
    .string()
    .max(downloadclientsClientEditFolderMax)
    .exactOptional()
    .describe("Empty for the client's default"),
  importDelay: zod.int().nullish().describe("Seconds; null resets it to its default"),
  kind: zod.enum(["qbittorrent", "transmission", "deluge", "rtorrent"]).exactOptional(),
  label: zod.string().min(1).max(downloadclientsClientEditLabelMax).exactOptional(),
  name: zod
    .string()
    .max(downloadclientsClientEditNameMax)
    .exactOptional()
    .describe("Empty for the kind's name"),
  nodeId: zod
    .uuid()
    .exactOptional()
    .describe("Another Node that holds Downloading; a client with Torrents can't move"),
  on: zod.boolean().exactOptional(),
  password: zod
    .string()
    .max(downloadclientsClientEditPasswordMax)
    .exactOptional()
    .describe("Write-only; empty keeps the saved one"),
  username: zod
    .string()
    .max(downloadclientsClientEditUsernameMax)
    .exactOptional()
    .describe("Empty signs in with none"),
});

export type DownloadclientsClientEdit = zod.input<typeof DownloadclientsClientEdit>;
export type DownloadclientsClientEditOutput = zod.output<typeof DownloadclientsClientEdit>;

export const DownloadclientsClientsChanged = zod.object({
  clientIds: zod
    .array(zod.string())
    .describe("The external clients whose settings or state changed"),
  nodeIds: zod.array(zod.string()).describe("The Nodes whose card or settings changed"),
});

export type DownloadclientsClientsChanged = zod.input<typeof DownloadclientsClientsChanged>;
export type DownloadclientsClientsChangedOutput = zod.output<typeof DownloadclientsClientsChanged>;

export const DownloadclientsKind = zod.object({
  actions: zod.array(zod.enum(["pause", "resume", "recheck", "announce"])),
  apiKey: zod
    .boolean()
    .describe("It may sign in with an API key in place of the username and password"),
  daemon: zod.boolean().describe("Its web UI connects to a daemon the Admin may choose"),
  deletesData: zod.boolean().describe("Its remove deletes data itself"),
  floor: zod.string().describe("The oldest version SlipStream supports, such as 4.0"),
  kind: zod.enum(["qbittorrent", "transmission", "deluge", "rtorrent"]),
  label: zod
    .enum(["any", "no_comma", "lower_word"])
    .describe("Which labels it takes: lower_word is a-z, 0-9, _ and - only"),
  name: zod.string().describe("The product's name, a new client's name by default"),
  password: zod.enum(["none", "optional", "required"]),
  path: zod.string().describe("The path an address with none takes; empty for a kind with none"),
  username: zod.enum(["none", "optional", "required"]),
});

export type DownloadclientsKind = zod.input<typeof DownloadclientsKind>;
export type DownloadclientsKindOutput = zod.output<typeof DownloadclientsKind>;

export const DownloadclientsThresholds = zod.object({
  clientErrorFail: SettingDurationField,
  faultStall: SettingDurationField,
  metadataWait: SettingDurationField,
  unregisteredGap: SettingDurationField,
});

export type DownloadclientsThresholds = zod.input<typeof DownloadclientsThresholds>;
export type DownloadclientsThresholdsOutput = zod.output<typeof DownloadclientsThresholds>;

export const DownloadclientsClientsList = zod.object({
  cards: zod
    .array(DownloadclientsClientCard)
    .describe("Each Node assigned Downloading or draining it, in Node order"),
  kinds: zod
    .array(DownloadclientsKind)
    .describe("The kinds of external client, in the order the form offers them"),
  orphans: zod
    .array(DownloadclientsExternalClient)
    .describe(
      "External clients on Nodes that no longer serve Downloading, which can only be deleted",
    ),
  thresholds: DownloadclientsThresholds,
});

export type DownloadclientsClientsList = zod.input<typeof DownloadclientsClientsList>;
export type DownloadclientsClientsListOutput = zod.output<typeof DownloadclientsClientsList>;

export const downloadclientsNodeCountTorrentsMin = 0;

export const DownloadclientsNodeCount = zod.object({
  nodeId: zod.uuid(),
  torrents: zod.int().min(downloadclientsNodeCountTorrentsMin),
});

export type DownloadclientsNodeCount = zod.input<typeof DownloadclientsNodeCount>;
export type DownloadclientsNodeCountOutput = zod.output<typeof DownloadclientsNodeCount>;

export const downloadclientsStateCountTorrentsMin = 0;

export const DownloadclientsStateCount = zod.object({
  state: zod.enum([
    "removing",
    "failed",
    "tracker-error",
    "paused",
    "checking",
    "queued",
    "stalled",
    "downloading",
    "seeding",
  ]),
  torrents: zod.int().min(downloadclientsStateCountTorrentsMin),
});

export type DownloadclientsStateCount = zod.input<typeof DownloadclientsStateCount>;
export type DownloadclientsStateCountOutput = zod.output<typeof DownloadclientsStateCount>;

export const downloadclientsTabCountTorrentsMin = 0;

export const DownloadclientsTabCount = zod.object({
  tab: zod.enum(["downloading", "seeding", "problems", "goal-met", "all"]),
  torrents: zod.int().min(downloadclientsTabCountTorrentsMin),
});

export type DownloadclientsTabCount = zod.input<typeof DownloadclientsTabCount>;
export type DownloadclientsTabCountOutput = zod.output<typeof DownloadclientsTabCount>;

export const DownloadclientsCounts = zod.object({
  byNode: zod.array(DownloadclientsNodeCount).describe("Each Node holding any, in Node order"),
  byState: zod.array(DownloadclientsStateCount).describe("Each state, in order of precedence"),
  byTab: zod.array(DownloadclientsTabCount).describe("Each tab, in the view's order"),
});

export type DownloadclientsCounts = zod.input<typeof DownloadclientsCounts>;
export type DownloadclientsCountsOutput = zod.output<typeof DownloadclientsCounts>;

export const DownloadclientsFault = zod.object({
  at: zod.iso.datetime({ offset: true }),
  kind: zod.enum(["engine_error", "unregistered", "no_payload"]),
  sentence: zod.string().describe("The fault, as a sentence"),
});

export type DownloadclientsFault = zod.input<typeof DownloadclientsFault>;
export type DownloadclientsFaultOutput = zod.output<typeof DownloadclientsFault>;

export const downloadclientsFileDetailDoneMin = 0;

export const downloadclientsFileDetailIndexMin = 0;

export const downloadclientsFileDetailSizeMin = 0;

export const DownloadclientsFileDetail = zod.object({
  done: zod
    .int()
    .min(downloadclientsFileDetailDoneMin)
    .exactOptional()
    .describe("Bytes done, absent while unknown"),
  index: zod.int().min(downloadclientsFileDetailIndexMin),
  items: zod
    .array(zod.uuid())
    .describe("The Items it filled once a Grab imported it, in order; empty before"),
  path: zod.string().describe("Under the save folder, with / between folders"),
  size: zod.int().min(downloadclientsFileDetailSizeMin),
});

export type DownloadclientsFileDetail = zod.input<typeof DownloadclientsFileDetail>;
export type DownloadclientsFileDetailOutput = zod.output<typeof DownloadclientsFileDetail>;

export const DownloadclientsLiveRow = zod.object({
  done: zod.int().describe("Bytes downloaded"),
  downBps: zod.int().describe("Payload bytes a second it downloads"),
  id: zod.uuid(),
  peers: zod.int(),
  seeds: zod.int(),
  upBps: zod.int().describe("Payload bytes a second it uploads"),
  wanted: zod.int().describe("Bytes its selected files hold"),
});

export type DownloadclientsLiveRow = zod.input<typeof DownloadclientsLiveRow>;
export type DownloadclientsLiveRowOutput = zod.output<typeof DownloadclientsLiveRow>;

export const DownloadclientsLiveTick = zod.object({
  at: zod.iso.datetime({ offset: true }),
  downloads: DownloadclientsAggregate,
  rows: zod
    .array(DownloadclientsLiveRow)
    .describe("The Torrents whose rates, progress or peers moved"),
});

export type DownloadclientsLiveTick = zod.input<typeof DownloadclientsLiveTick>;
export type DownloadclientsLiveTickOutput = zod.output<typeof DownloadclientsLiveTick>;

export const downloadclientsNewClientAddressMax = 2048;

export const downloadclientsNewClientApiKeyMax = 1024;

export const downloadclientsNewClientClientFoldersMax = 50;

export const downloadclientsNewClientFolderMax = 4096;

export const downloadclientsNewClientLabelMax = 100;

export const downloadclientsNewClientNameMax = 100;

export const downloadclientsNewClientPasswordMax = 1024;

export const downloadclientsNewClientUsernameMax = 256;

export const DownloadclientsNewClient = zod.object({
  address: zod
    .string()
    .min(1)
    .max(downloadclientsNewClientAddressMax)
    .describe(
      "The client's whole address, such as http://localhost:9091/transmission/rpc; with no path it takes the kind's usual one",
    ),
  apiKey: zod
    .string()
    .max(downloadclientsNewClientApiKeyMax)
    .exactOptional()
    .describe("qBittorrent 5.2 and later. Write-only"),
  checkEvery: zod.int().exactOptional().describe("Seconds; absent for the default"),
  clientFolders: zod
    .array(DownloadclientsClientFolder)
    .max(downloadclientsNewClientClientFoldersMax)
    .nullish(),
  daemon: DownloadclientsDaemon.exactOptional().describe(
    "Deluge's daemon; absent for the only one",
  ),
  folder: zod
    .string()
    .max(downloadclientsNewClientFolderMax)
    .exactOptional()
    .describe("The download folder as the client sees it; absent for the client's default"),
  importDelay: zod.int().exactOptional().describe("Seconds; absent for the default"),
  kind: zod.enum(["qbittorrent", "transmission", "deluge", "rtorrent"]),
  label: zod
    .string()
    .max(downloadclientsNewClientLabelMax)
    .exactOptional()
    .describe("Absent for slipstream"),
  name: zod
    .string()
    .max(downloadclientsNewClientNameMax)
    .exactOptional()
    .describe("Absent for the kind's name, such as Transmission"),
  nodeId: zod.uuid().describe("A Node that holds Downloading"),
  on: zod.boolean().exactOptional().describe("Absent for on"),
  password: zod
    .string()
    .max(downloadclientsNewClientPasswordMax)
    .exactOptional()
    .describe("Write-only"),
  username: zod.string().max(downloadclientsNewClientUsernameMax).exactOptional(),
});

export type DownloadclientsNewClient = zod.input<typeof DownloadclientsNewClient>;
export type DownloadclientsNewClientOutput = zod.output<typeof DownloadclientsNewClient>;

export const downloadclientsRatioLaneValueMin = 0;

export const DownloadclientsRatioLane = zod.object({
  goal: zod
    .number()
    .exactOptional()
    .describe("The goal's ratio, absent when the goal doesn't count ratio"),
  rule: zod
    .number()
    .exactOptional()
    .describe("The rule's ratio, absent when the rule doesn't count ratio"),
  value: zod.number().min(downloadclientsRatioLaneValueMin),
});

export type DownloadclientsRatioLane = zod.input<typeof DownloadclientsRatioLane>;
export type DownloadclientsRatioLaneOutput = zod.output<typeof DownloadclientsRatioLane>;

export const downloadclientsRetriedFoundMin = 0;

export const DownloadclientsRetried = zod.object({
  at: zod.iso.datetime({ offset: true }),
  found: zod
    .int()
    .min(downloadclientsRetriedFoundMin)
    .exactOptional()
    .describe(
      "The bytes its check found, absent until its Node checked; all of size means it seeds again",
    ),
});

export type DownloadclientsRetried = zod.input<typeof DownloadclientsRetried>;
export type DownloadclientsRetriedOutput = zod.output<typeof DownloadclientsRetried>;

export const downloadclientsTimeLaneGoalMin = 0;

export const downloadclientsTimeLaneMarginMin = 0;

export const downloadclientsTimeLaneRuleMin = 0;

export const downloadclientsTimeLaneValueMin = 0;

export const DownloadclientsTimeLane = zod.object({
  goal: zod
    .int()
    .min(downloadclientsTimeLaneGoalMin)
    .exactOptional()
    .describe("Seconds the goal needs, absent when the goal doesn't count seeding time"),
  margin: zod
    .int()
    .min(downloadclientsTimeLaneMarginMin)
    .exactOptional()
    .describe(
      "Seconds of the margin the rule's mark includes, absent when the rule doesn't count seeding time",
    ),
  rule: zod
    .int()
    .min(downloadclientsTimeLaneRuleMin)
    .exactOptional()
    .describe(
      "Seconds the rule needs plus the margin, absent when the rule doesn't count seeding time",
    ),
  value: zod
    .int()
    .min(downloadclientsTimeLaneValueMin)
    .describe(
      "Seconds of seeding the goal counts: the Tracker's count for a private torrent, the torrent client's for a public one",
    ),
});

export type DownloadclientsTimeLane = zod.input<typeof DownloadclientsTimeLane>;
export type DownloadclientsTimeLaneOutput = zod.output<typeof DownloadclientsTimeLane>;

export const DownloadclientsSeedingView = zod.object({
  action: zod
    .enum(["pause", "remove", "remove-delete"])
    .exactOptional()
    .describe("What its goal does once met, absent for a forever goal"),
  atRisk: zod
    .enum(["no", "yes", "unknown"])
    .describe(
      "Whether a torrent that owes is at risk of a hit and run: yes while no copy of it announces as a seeder, unknown while every copy's Node is offline",
    ),
  clears: zod
    .enum(["none", "time", "ratio", "either", "both"])
    .describe("What meets the rule in force, or none while no rule is known"),
  combine: zod
    .enum(["first", "both", "forever"])
    .describe(
      "How its goal's ratio and seeding time combine. A forever goal seeds until an Admin removes it",
    ),
  endless: zod
    .enum(["none", "forever", "resumed", "no-tracker", "no-rule", "partial"])
    .describe("Why its goal never acts on it, or none while its goal can act"),
  goalMet: zod.boolean().describe("Its seeding goal is met and it owes its Tracker nothing"),
  owes: DownloadclientsOwes.exactOptional().describe(
    "What it still needs to meet its Tracker's rule and the margin, absent unless it owes under a known rule",
  ),
  ratio: DownloadclientsRatioLane,
  state: zod
    .enum(["none", "owing", "met", "missed", "released"])
    .describe("Its hit and run window's state, or none while it owes no Tracker"),
  time: DownloadclientsTimeLane,
});

export type DownloadclientsSeedingView = zod.input<typeof DownloadclientsSeedingView>;
export type DownloadclientsSeedingViewOutput = zod.output<typeof DownloadclientsSeedingView>;

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

export const SettingRateField = zod.object({
  applies: zod.enum(["live", "restart"]),
  default: zod.int().describe("Bytes a second"),
  defaultFrom: zod
    .string()
    .exactOptional()
    .describe("Where the default comes from, when it is another setting's value"),
  isDefault: zod
    .boolean()
    .describe("The setting has no override; Reset to default leaves it unchanged"),
  key: zod.string(),
  kind: zod.enum(["rate"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  max: zod.int().describe("Bytes a second"),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  running: zod.int().describe("Bytes a second"),
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  value: zod.int().describe("Bytes a second"),
  zero: zod.string().exactOptional().describe("The words for 0, such as No limit"),
});

export type SettingRateField = zod.input<typeof SettingRateField>;
export type SettingRateFieldOutput = zod.output<typeof SettingRateField>;

export const DownloadclientsSettings = zod.object({
  activeChecks: SettingCountField,
  activeDownloads: SettingCountField,
  connections: SettingCountField,
  dht: SettingSwitchField,
  downloadLimit: SettingRateField,
  grabsTo: zod
    .uuid()
    .nullable()
    .describe("The external client the Node's new Grabs go to; null for its built-in client"),
  stalledAfter: SettingDurationField,
  uploadLimit: SettingRateField,
  utp: SettingSwitchField,
});

export type DownloadclientsSettings = zod.input<typeof DownloadclientsSettings>;
export type DownloadclientsSettingsOutput = zod.output<typeof DownloadclientsSettings>;

export const DownloadclientsSettingsPatch = zod.object({
  activeChecks: zod.int().nullish().describe("null resets it to its default"),
  activeDownloads: zod.int().nullish().describe("0 for no limit; null resets it to its default"),
  connections: zod.int().nullish().describe("null resets it to its default"),
  dht: zod
    .boolean()
    .nullish()
    .describe(
      "Whether the client runs the DHT, which public torrents use to find peers; null resets it to its default",
    ),
  downloadLimit: zod
    .int()
    .nullish()
    .describe("Bytes a second, 0 for no limit; null resets it to its default"),
  grabsTo: zod
    .uuid()
    .nullish()
    .describe(
      "One of the Node's on external clients, which Grabs for Works whose library root is on the Node go to; null for its built-in client",
    ),
  stalledAfter: zod.int().nullish().describe("Seconds; null resets it to its default"),
  uploadLimit: zod
    .int()
    .nullish()
    .describe("Bytes a second, 0 for no limit; null resets it to its default"),
  utp: zod.boolean().nullish().describe("null resets it to its default"),
});

export type DownloadclientsSettingsPatch = zod.input<typeof DownloadclientsSettingsPatch>;
export type DownloadclientsSettingsPatchOutput = zod.output<typeof DownloadclientsSettingsPatch>;

export const DownloadclientsTestReport = zod.object({
  daemons: zod
    .array(DownloadclientsDaemon)
    .describe("The daemons Deluge's web UI lists; empty for every other kind"),
  nodeFolder: zod.string().describe("The download folder as the Node sees it"),
  product: zod.string().describe("The product's name as it responded, such as Transmission"),
  version: zod.string().describe("Such as 4.1.1"),
  warnings: zod
    .array(zod.string())
    .describe("What could go wrong with these settings, as sentences"),
});

export type DownloadclientsTestReport = zod.input<typeof DownloadclientsTestReport>;
export type DownloadclientsTestReportOutput = zod.output<typeof DownloadclientsTestReport>;

export const DownloadclientsThresholdsPatch = zod.object({
  clientErrorFail: zod
    .int()
    .nullish()
    .describe(
      "Seconds an external client reports an error on a Torrent before its Grab fails; null resets it to its default",
    ),
  faultStall: zod.int().nullish().describe("Seconds; null resets it to its default"),
  metadataWait: zod
    .int()
    .nullish()
    .describe(
      "Seconds a magnet link's metadata search runs before its Grab fails; null resets it to its default",
    ),
  unregisteredGap: zod.int().nullish().describe("Seconds; null resets it to its default"),
});

export type DownloadclientsThresholdsPatch = zod.input<typeof DownloadclientsThresholdsPatch>;
export type DownloadclientsThresholdsPatchOutput = zod.output<
  typeof DownloadclientsThresholdsPatch
>;

export const DownloadclientsTrackerCell = zod.object({
  host: zod.string().describe("Empty when its announce URLs are unknown"),
  message: zod
    .string()
    .exactOptional()
    .describe("The announce URL's failure reason or warning, verbatim"),
  retryAt: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("When it tries a failing announce URL again"),
  status: zod.enum(["working", "updating", "error", "unregistered", "not_contacted"]),
});

export type DownloadclientsTrackerCell = zod.input<typeof DownloadclientsTrackerCell>;
export type DownloadclientsTrackerCellOutput = zod.output<typeof DownloadclientsTrackerCell>;

export const DownloadclientsTorrentGrab = zod.object({
  id: zod.uuid(),
  release: zod.string(),
  stage: zod
    .enum(["downloading", "importing", "waiting", "imported", "failed"])
    .describe("waiting: a file of its Import waits for the Admin and none is sending or placing"),
  workId: zod.uuid(),
});

export type DownloadclientsTorrentGrab = zod.input<typeof DownloadclientsTorrentGrab>;
export type DownloadclientsTorrentGrabOutput = zod.output<typeof DownloadclientsTorrentGrab>;

export const downloadclientsTrackerDetailLeechersMin = 0;

export const downloadclientsTrackerDetailMinIntervalSecondsMin = 0;

export const downloadclientsTrackerDetailSeedersMin = 0;

export const downloadclientsTrackerDetailTierMin = 0;

export const DownloadclientsTrackerDetail = zod.object({
  acceptsAt: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("The earliest it accepts an announce, set while live"),
  answeredAt: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("When it last responded, set while live"),
  leechers: zod
    .int()
    .min(downloadclientsTrackerDetailLeechersMin)
    .exactOptional()
    .describe(
      "The leechers its last announce response counted, set while live and the response carried a count",
    ),
  message: zod
    .string()
    .exactOptional()
    .describe("The announce URL's failure reason or warning, verbatim"),
  minIntervalSeconds: zod
    .int()
    .min(downloadclientsTrackerDetailMinIntervalSecondsMin)
    .exactOptional()
    .describe("Its min interval as the announce policy holds to it, set while live"),
  nextAnnounce: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("When its next announce is due, set while live"),
  retryAt: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("When it tries a failing announce URL again, from the last report"),
  seeders: zod
    .int()
    .min(downloadclientsTrackerDetailSeedersMin)
    .exactOptional()
    .describe(
      "The seeders its last announce response counted, set while live and the response carried a count",
    ),
  status: zod.enum(["working", "updating", "error", "unregistered", "not_contacted"]),
  tier: zod.int().min(downloadclientsTrackerDetailTierMin),
  url: zod.string().describe("With its passkey hidden"),
});

export type DownloadclientsTrackerDetail = zod.input<typeof DownloadclientsTrackerDetail>;
export type DownloadclientsTrackerDetailOutput = zod.output<typeof DownloadclientsTrackerDetail>;

export const downloadclientsTorrentDetailActiveSecondsMin = 0;

export const downloadclientsTorrentDetailDoneMin = 0;

export const downloadclientsTorrentDetailDownloadedMin = 0;

export const downloadclientsTorrentDetailQueuePositionMin = 0;

export const downloadclientsTorrentDetailRatioMin = 0;

export const downloadclientsTorrentDetailSeedingSecondsMin = 0;

export const downloadclientsTorrentDetailSizeMin = 0;

export const downloadclientsTorrentDetailUploadedMin = 0;

export const DownloadclientsTorrentDetail = zod.object({
  activeSeconds: zod
    .int()
    .min(downloadclientsTorrentDetailActiveSecondsMin)
    .describe("All-time seconds it ran"),
  addedAt: zod.iso.datetime({ offset: true }),
  announce: DownloadclientsTrackerCell.exactOptional().describe(
    "Absent when its metainfo names no announce URL",
  ),
  announceAt: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("The earliest every announce URL accepts an announce, set while live"),
  attached: zod
    .boolean()
    .exactOptional()
    .describe(
      "It was in its external client before SlipStream attached to it, so removing it only stops tracking it",
    ),
  clientId: zod
    .uuid()
    .exactOptional()
    .describe("Its external client; absent for its Node's built-in client"),
  clientKind: zod
    .enum(["qbittorrent", "transmission", "deluge", "rtorrent"])
    .exactOptional()
    .describe("Its external client's kind"),
  clientName: zod.string().exactOptional().describe("Its external client's name"),
  contentType: zod.string().describe("The Content Type it was first grabbed for"),
  deleteFiles: zod
    .boolean()
    .describe(
      "A remove may delete its download copy; false when removing it only stops tracking it, attached or in an off client",
    ),
  done: zod
    .int()
    .min(downloadclientsTorrentDetailDoneMin)
    .describe("Bytes done as its Node last reported them; torrents.live carries the live figure"),
  downloaded: zod
    .int()
    .min(downloadclientsTorrentDetailDownloadedMin)
    .describe("All-time bytes downloaded"),
  edition: zod.string().exactOptional().describe("Its newest Grab's Edition"),
  error: zod
    .string()
    .exactOptional()
    .describe(
      "The engine's error message for the failure, or an external client's own words for its error",
    ),
  fault: DownloadclientsFault.exactOptional().describe(
    "The fault found in the release itself, once there is one",
  ),
  files: zod.array(DownloadclientsFileDetail).describe("In its metainfo's order"),
  grabs: zod.array(DownloadclientsTorrentGrab).describe("The Grabs that used it, newest first"),
  held: zod
    .boolean()
    .describe(
      "A Grab that hasn't finished holds it, so Mark failed can put its Release on the Blocklist",
    ),
  hold: zod
    .enum([
      "wanted",
      "window",
      "disk_full",
      "slot",
      "check_slot",
      "disk_guard",
      "draining",
      "controller",
      "client_error",
    ])
    .exactOptional()
    .describe(
      "Why its Node holds it back, as its Node last reported; client_error is an external client's own error, in error",
    ),
  id: zod.uuid(),
  importAt: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("When a finished external torrent imports, set while its client's Import delay runs"),
  infoHash: zod.string().describe("The v1 info hash in hex"),
  lastKnown: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("When its Node last reported it, set while the row is stale for lastKnownCause"),
  lastKnownCause: zod
    .enum(["node", "client", "off"])
    .exactOptional()
    .describe(
      "Why the row is its last report: its Node is out of step, its external client isn't responding or refused the sign-in, or its external client is off",
    ),
  live: zod
    .boolean()
    .describe(
      "Its Node responded, so announce URLs and files are live. Otherwise they are from its last report",
    ),
  name: zod
    .string()
    .describe("The release's name in its metainfo, shown for a Torrent whose Work was deleted"),
  nodeId: zod.uuid(),
  pending: zod
    .enum(["pause", "resume"])
    .exactOptional()
    .describe("An Admin's pause or resume its Node hasn't applied yet"),
  pieceSize: zod.int().min(1).describe("Bytes"),
  private: zod.boolean(),
  quality: zod
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
    .exactOptional()
    .describe("The quality its newest Grab's release title names"),
  queuePosition: zod
    .int()
    .min(downloadclientsTorrentDetailQueuePositionMin)
    .exactOptional()
    .describe("Its place, from 1, among its Node's downloads waiting for a slot"),
  ratio: zod
    .number()
    .min(downloadclientsTorrentDetailRatioMin)
    .describe("Uploaded over the larger of downloaded and size"),
  retry: DownloadclientsRetried.exactOptional().describe("Its last Retry, absent before one"),
  retryable: zod
    .boolean()
    .describe(
      "It failed in its Node's built-in client after it completed, so retry can check its files again",
    ),
  saveFolder: zod
    .string()
    .exactOptional()
    .describe("Where its Node keeps it, absent until its Node reports it"),
  seeding: zod
    .union([DownloadclientsSeedingView, zod.null()])
    .describe(
      "Its seeding against its Tracker's rule and its seeding goal, null in an external client, whose own limits govern its seeding",
    ),
  seedingSeconds: zod
    .int()
    .min(downloadclientsTorrentDetailSeedingSecondsMin)
    .describe("All-time seconds seeded"),
  severity: zod.enum(["none", "warning", "failure"]),
  size: zod.int().min(downloadclientsTorrentDetailSizeMin).describe("Bytes wanted"),
  state: zod.enum([
    "removing",
    "failed",
    "tracker-error",
    "paused",
    "checking",
    "queued",
    "stalled",
    "downloading",
    "seeding",
  ]),
  stoppedAtLimit: zod
    .boolean()
    .exactOptional()
    .describe("Its external client stopped it at its seeding limit"),
  tabs: zod
    .array(zod.enum(["downloading", "seeding", "problems", "goal-met", "all"]))
    .describe("The Torrents view's tabs it shows under"),
  tracker: zod
    .union([DownloadclientsTrackerRef, zod.null()])
    .describe(
      "The Tracker its first Grab's Release reached, null when none did or it is in an external client",
    ),
  trackers: zod.array(DownloadclientsTrackerDetail).describe("In its metainfo's order"),
  until: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("When a window hold starts it, or a check resumes it"),
  uploaded: zod
    .int()
    .min(downloadclientsTorrentDetailUploadedMin)
    .describe("All-time bytes uploaded"),
  workId: zod
    .uuid()
    .exactOptional()
    .describe("Its newest Grab's Work, absent once no Grab is left"),
});

export type DownloadclientsTorrentDetail = zod.input<typeof DownloadclientsTorrentDetail>;
export type DownloadclientsTorrentDetailOutput = zod.output<typeof DownloadclientsTorrentDetail>;

export const downloadclientsTorrentRowDoneMin = 0;

export const downloadclientsTorrentRowQueuePositionMin = 0;

export const downloadclientsTorrentRowRatioMin = 0;

export const downloadclientsTorrentRowSeedingSecondsMin = 0;

export const downloadclientsTorrentRowSizeMin = 0;

export const downloadclientsTorrentRowUploadedMin = 0;

export const DownloadclientsTorrentRow = zod.object({
  addedAt: zod.iso.datetime({ offset: true }),
  announce: DownloadclientsTrackerCell.exactOptional().describe(
    "Absent when its metainfo names no announce URL",
  ),
  attached: zod
    .boolean()
    .exactOptional()
    .describe(
      "It was in its external client before SlipStream attached to it, so removing it only stops tracking it",
    ),
  clientId: zod
    .uuid()
    .exactOptional()
    .describe("Its external client; absent for its Node's built-in client"),
  clientKind: zod
    .enum(["qbittorrent", "transmission", "deluge", "rtorrent"])
    .exactOptional()
    .describe("Its external client's kind"),
  clientName: zod.string().exactOptional().describe("Its external client's name"),
  contentType: zod.string().describe("The Content Type it was first grabbed for"),
  deleteFiles: zod
    .boolean()
    .describe(
      "A remove may delete its download copy; false when removing it only stops tracking it, attached or in an off client",
    ),
  done: zod
    .int()
    .min(downloadclientsTorrentRowDoneMin)
    .describe("Bytes done as its Node last reported them; torrents.live carries the live figure"),
  edition: zod.string().exactOptional().describe("Its newest Grab's Edition"),
  error: zod
    .string()
    .exactOptional()
    .describe(
      "The engine's error message for the failure, or an external client's own words for its error",
    ),
  fault: DownloadclientsFault.exactOptional().describe(
    "The fault found in the release itself, once there is one",
  ),
  held: zod
    .boolean()
    .describe(
      "A Grab that hasn't finished holds it, so Mark failed can put its Release on the Blocklist",
    ),
  hold: zod
    .enum([
      "wanted",
      "window",
      "disk_full",
      "slot",
      "check_slot",
      "disk_guard",
      "draining",
      "controller",
      "client_error",
    ])
    .exactOptional()
    .describe(
      "Why its Node holds it back, as its Node last reported; client_error is an external client's own error, in error",
    ),
  id: zod.uuid(),
  importAt: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("When a finished external torrent imports, set while its client's Import delay runs"),
  lastKnown: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("When its Node last reported it, set while the row is stale for lastKnownCause"),
  lastKnownCause: zod
    .enum(["node", "client", "off"])
    .exactOptional()
    .describe(
      "Why the row is its last report: its Node is out of step, its external client isn't responding or refused the sign-in, or its external client is off",
    ),
  name: zod
    .string()
    .describe("The release's name in its metainfo, shown for a Torrent whose Work was deleted"),
  nodeId: zod.uuid(),
  pending: zod
    .enum(["pause", "resume"])
    .exactOptional()
    .describe("An Admin's pause or resume its Node hasn't applied yet"),
  quality: zod
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
    .exactOptional()
    .describe("The quality its newest Grab's release title names"),
  queuePosition: zod
    .int()
    .min(downloadclientsTorrentRowQueuePositionMin)
    .exactOptional()
    .describe("Its place, from 1, among its Node's downloads waiting for a slot"),
  ratio: zod
    .number()
    .min(downloadclientsTorrentRowRatioMin)
    .describe("Uploaded over the larger of downloaded and size"),
  retry: DownloadclientsRetried.exactOptional().describe("Its last Retry, absent before one"),
  retryable: zod
    .boolean()
    .describe(
      "It failed in its Node's built-in client after it completed, so retry can check its files again",
    ),
  seeding: zod
    .union([DownloadclientsSeedingView, zod.null()])
    .describe(
      "Its seeding against its Tracker's rule and its seeding goal, null in an external client, whose own limits govern its seeding",
    ),
  seedingSeconds: zod
    .int()
    .min(downloadclientsTorrentRowSeedingSecondsMin)
    .describe("All-time seconds seeded"),
  severity: zod.enum(["none", "warning", "failure"]),
  size: zod.int().min(downloadclientsTorrentRowSizeMin).describe("Bytes wanted"),
  state: zod.enum([
    "removing",
    "failed",
    "tracker-error",
    "paused",
    "checking",
    "queued",
    "stalled",
    "downloading",
    "seeding",
  ]),
  stoppedAtLimit: zod
    .boolean()
    .exactOptional()
    .describe("Its external client stopped it at its seeding limit"),
  tabs: zod
    .array(zod.enum(["downloading", "seeding", "problems", "goal-met", "all"]))
    .describe("The Torrents view's tabs it shows under"),
  tracker: zod
    .union([DownloadclientsTrackerRef, zod.null()])
    .describe(
      "The Tracker its first Grab's Release reached, null when none did or it is in an external client",
    ),
  until: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("When a window hold starts it, or a check resumes it"),
  uploaded: zod.int().min(downloadclientsTorrentRowUploadedMin).describe("All-time bytes uploaded"),
  workId: zod
    .uuid()
    .exactOptional()
    .describe("Its newest Grab's Work, absent once no Grab is left"),
});

export type DownloadclientsTorrentRow = zod.input<typeof DownloadclientsTorrentRow>;
export type DownloadclientsTorrentRowOutput = zod.output<typeof DownloadclientsTorrentRow>;

export const DownloadclientsTorrentList = zod.object({
  counts: DownloadclientsCounts.describe("Every Torrent live or Removing, whatever the filter"),
  rows: zod.array(DownloadclientsTorrentRow).describe("In the order they were added"),
});

export type DownloadclientsTorrentList = zod.input<typeof DownloadclientsTorrentList>;
export type DownloadclientsTorrentListOutput = zod.output<typeof DownloadclientsTorrentList>;

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

export const filesAudioTrackChannelsMin = 0;

export const FilesAudioTrack = zod.object({
  atmos: zod.boolean(),
  channels: zod.int().min(filesAudioTrackChannelsMin),
  codec: zod.string().describe("ffprobe's codec name, such as eac3"),
  default: zod.boolean(),
  language: zod.string().nullable().describe("An ISO 639 code; null when the track names none"),
  layout: zod
    .string()
    .describe("ffprobe's channel layout, such as 5.1(side); empty when it names none"),
  profile: zod
    .string()
    .describe("Such as Dolby Digital Plus + Dolby Atmos; empty when ffprobe names none"),
  title: zod.string().describe("Empty when the track has none"),
});

export type FilesAudioTrack = zod.input<typeof FilesAudioTrack>;
export type FilesAudioTrackOutput = zod.output<typeof FilesAudioTrack>;

export const FilesUnreleasedCell = zod.object({
  edition: zod.string().describe("any under any cut, else the key of the cut the cell wants"),
  itemId: zod.uuid(),
  state: zod.enum(["unreleased"]).describe("Its Item has no release date, or a later one"),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod
    .boolean()
    .describe(
      "Missing, failed or upgradable with no Grab in flight, and its Item monitored, listed and released",
    ),
});

export type FilesUnreleasedCell = zod.input<typeof FilesUnreleasedCell>;
export type FilesUnreleasedCellOutput = zod.output<typeof FilesUnreleasedCell>;

export const FilesMissingCell = zod.object({
  edition: zod.string().describe("any under any cut, else the key of the cut the cell wants"),
  itemId: zod.uuid(),
  state: zod.enum(["missing"]),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod
    .boolean()
    .describe(
      "Missing, failed or upgradable with no Grab in flight, and its Item monitored, listed and released",
    ),
});

export type FilesMissingCell = zod.input<typeof FilesMissingCell>;
export type FilesMissingCellOutput = zod.output<typeof FilesMissingCell>;

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

export const FilesStep = zod.enum([
  "fetching",
  "downloading",
  "reading",
  "sending",
  "placing",
  "waiting",
]);

export type FilesStep = zod.input<typeof FilesStep>;
export type FilesStepOutput = zod.output<typeof FilesStep>;

export const FilesDownloadingCell = zod.object({
  edition: zod.string().describe("any under any cut, else the key of the cut the cell wants"),
  file: zod
    .union([FilesLibraryFile, zod.null()])
    .describe("The file the Grab would replace; null when it fills no file"),
  from: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When the step began; null with step"),
  grabId: zod.uuid().describe("The latest Grab in flight for it"),
  itemId: zod.uuid(),
  state: zod.enum(["downloading"]).describe("A Grab is in flight for it, fetching included"),
  step: zod
    .union([FilesStep, zod.null()])
    .describe("What the Grab or its Import is doing for the cell; null until SlipStream reads it"),
  targetId: zod.uuid(),
  targetName: zod.string(),
  until: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When the Import delay ends, for a download that waits for it; null otherwise"),
  wanted: zod
    .boolean()
    .describe(
      "Missing, failed or upgradable with no Grab in flight, and its Item monitored, listed and released",
    ),
});

export type FilesDownloadingCell = zod.input<typeof FilesDownloadingCell>;
export type FilesDownloadingCellOutput = zod.output<typeof FilesDownloadingCell>;

export const FilesFailedCell = zod.object({
  edition: zod.string().describe("any under any cut, else the key of the cut the cell wants"),
  file: zod
    .union([FilesLibraryFile, zod.null()])
    .describe("The file the Grab would have replaced; null when it fills no file"),
  grabId: zod.uuid().describe("The Grab that failed"),
  itemId: zod.uuid(),
  state: zod.enum(["failed"]),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod
    .boolean()
    .describe(
      "Missing, failed or upgradable with no Grab in flight, and its Item monitored, listed and released",
    ),
});

export type FilesFailedCell = zod.input<typeof FilesFailedCell>;
export type FilesFailedCellOutput = zod.output<typeof FilesFailedCell>;

export const FilesHeldCell = zod.object({
  from: zod.iso.datetime({ offset: true }).describe("When the step began"),
  grabId: zod.uuid().describe("The Grab in flight for the cell"),
  step: FilesStep,
  until: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When the Import delay ends, for a download that waits for it; null otherwise"),
});

export type FilesHeldCell = zod.input<typeof FilesHeldCell>;
export type FilesHeldCellOutput = zod.output<typeof FilesHeldCell>;

export const FilesUpgradableCell = zod.object({
  edition: zod.string().describe("any under any cut, else the key of the cut the cell wants"),
  file: FilesLibraryFile.describe(
    "The file filling it, short of its Target's cutoff or upgrade-until score",
  ),
  held: zod
    .union([FilesHeldCell, zod.null()])
    .describe("The Grab in flight for it; null while none is"),
  itemId: zod.uuid(),
  state: zod.enum(["upgradable"]),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod
    .boolean()
    .describe(
      "Missing, failed or upgradable with no Grab in flight, and its Item monitored, listed and released",
    ),
});

export type FilesUpgradableCell = zod.input<typeof FilesUpgradableCell>;
export type FilesUpgradableCellOutput = zod.output<typeof FilesUpgradableCell>;

export const FilesMetCell = zod.object({
  edition: zod.string().describe("any under any cut, else the key of the cut the cell wants"),
  file: FilesLibraryFile.describe("The file filling it"),
  held: zod
    .union([FilesHeldCell, zod.null()])
    .describe("The Grab in flight for it; null while none is"),
  itemId: zod.uuid(),
  state: zod.enum(["met"]),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod
    .boolean()
    .describe(
      "Missing, failed or upgradable with no Grab in flight, and its Item monitored, listed and released",
    ),
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

export const FilesState = zod.enum([
  "failed",
  "missing",
  "downloading",
  "upgradable",
  "unreleased",
  "met",
]);

export type FilesState = zod.input<typeof FilesState>;
export type FilesStateOutput = zod.output<typeof FilesState>;

export const FilesItemStatus = zod.object({
  itemId: zod.uuid(),
  released: zod.boolean().describe("Its availability date is today or earlier"),
  status: zod
    .union([FilesState, zod.null()])
    .describe(
      "The first state its cells hold, in the glossary's order; null for an Item with no cell",
    ),
});

export type FilesItemStatus = zod.input<typeof FilesItemStatus>;
export type FilesItemStatusOutput = zod.output<typeof FilesItemStatus>;

export const FilesStateCounts = zod.object({
  downloading: zod.int().min(1).exactOptional(),
  failed: zod.int().min(1).exactOptional(),
  met: zod.int().min(1).exactOptional(),
  missing: zod.int().min(1).exactOptional(),
  unreleased: zod.int().min(1).exactOptional(),
  upgradable: zod.int().min(1).exactOptional(),
});

export type FilesStateCounts = zod.input<typeof FilesStateCounts>;
export type FilesStateCountsOutput = zod.output<typeof FilesStateCounts>;

export const filesLevelRollupItemsMin = 0;

export const FilesLevelRollup = zod.object({
  cells: FilesStateCounts.describe("Its monitored Items' cells by state"),
  itemStatus: FilesStateCounts.describe(
    "Its monitored Items by status; an Item with no cell isn't counted",
  ),
  items: zod
    .int()
    .min(filesLevelRollupItemsMin)
    .describe("Its monitored Items that its source still lists"),
  levelId: zod.uuid(),
  withdrawn: FilesStateCounts.describe(
    "The cells of its Items its source no longer lists that a file still fills, by state",
  ),
});

export type FilesLevelRollup = zod.input<typeof FilesLevelRollup>;
export type FilesLevelRollupOutput = zod.output<typeof FilesLevelRollup>;

export const FilesFailedNeed = zod.object({
  edition: zod.string().describe("any under any cut, else the key of the cut the cells want"),
  grabId: zod.uuid().describe("The Grab that failed"),
  itemIds: zod.array(zod.string()).describe("A run of Items in tree order whose entries match"),
  kind: zod.enum(["failed"]),
  targetId: zod.uuid().describe("The Target whose download failed"),
  waiting: zod
    .array(zod.string())
    .describe(
      "The Targets of the same Items and Edition, still missing, that wait on the failed one",
    ),
});

export type FilesFailedNeed = zod.input<typeof FilesFailedNeed>;
export type FilesFailedNeedOutput = zod.output<typeof FilesFailedNeed>;

export const FilesUpgradeNeed = zod.object({
  edition: zod.string().describe("any under any cut, else the key of the cut the cells want"),
  itemIds: zod.array(zod.string()).describe("A run of Items in tree order whose entries match"),
  kind: zod.enum(["upgrade"]),
  upgrades: zod.array(zod.string()).describe("Every upgradable Target of the Items and Edition"),
});

export type FilesUpgradeNeed = zod.input<typeof FilesUpgradeNeed>;
export type FilesUpgradeNeedOutput = zod.output<typeof FilesUpgradeNeed>;

export const FilesClientError = zod.object({
  client: zod.string().describe("The external Download Client's name"),
  failsAt: zod.iso
    .datetime({ offset: true })
    .describe("When the Grab fails unless the error clears"),
  message: zod.string().describe("The error as the client words it"),
  node: zod.string().describe("The client's Node's name"),
});

export type FilesClientError = zod.input<typeof FilesClientError>;
export type FilesClientErrorOutput = zod.output<typeof FilesClientError>;

export const filesSpaceFreeMin = 0;

export const FilesSpace = zod.object({
  free: zod
    .int()
    .min(filesSpaceFreeMin)
    .describe("Bytes free on the root's volume above its Storage error level"),
  needs: zod.int().min(1).describe("Bytes the file needs"),
});

export type FilesSpace = zod.input<typeof FilesSpace>;
export type FilesSpaceOutput = zod.output<typeof FilesSpace>;

export const FilesImportWait = zod.object({
  rootId: zod.uuid(),
  space: zod
    .union([FilesSpace, zod.null()])
    .describe(
      "For no_space, the bytes free and the bytes the file needs; null when a clone found the volume full or for root_not_ready",
    ),
});

export type FilesImportWait = zod.input<typeof FilesImportWait>;
export type FilesImportWaitOutput = zod.output<typeof FilesImportWait>;

export const FilesWaitingNeed = zod.object({
  clientError: zod
    .union([FilesClientError, zod.null()])
    .describe("The client's error, for client_error; null otherwise"),
  code: zod
    .enum([
      "node_away",
      "client_away",
      "client_off",
      "no_space",
      "root_not_ready",
      "probe_timeout",
      "client_error",
    ])
    .describe(
      "Why the Grab waits: node_away while its Node, or the Node of the client its fetch goes to, is out of step, client_away while its Torrent's client isn't responding, refused the sign-in or needs attention, client_off while the client it is pinned to is off or deleted, or its Node doesn't hold Downloading. no_space while its root has too little free space for a file, root_not_ready while its root isn't ready, probe_timeout while reading a file timed out three times, and client_error while its external client reports an error",
    ),
  grabId: zod.uuid(),
  importWait: zod
    .union([FilesImportWait, zod.null()])
    .describe("The root the Import waits on, for no_space and root_not_ready; null otherwise"),
  kind: zod.enum(["waiting"]),
});

export type FilesWaitingNeed = zod.input<typeof FilesWaitingNeed>;
export type FilesWaitingNeedOutput = zod.output<typeof FilesWaitingNeed>;

export const FilesNoRootNeed = zod.object({
  kind: zod.enum(["no_root"]).describe("The Work has no root. Its downloads wait until it has one"),
});

export type FilesNoRootNeed = zod.input<typeof FilesNoRootNeed>;
export type FilesNoRootNeedOutput = zod.output<typeof FilesNoRootNeed>;

export const FilesNeed = zod.discriminatedUnion("kind", [
  FilesFailedNeed,
  FilesUpgradeNeed,
  FilesWaitingNeed,
  FilesNoRootNeed,
]);

export type FilesNeed = zod.input<typeof FilesNeed>;
export type FilesNeedOutput = zod.output<typeof FilesNeed>;

export const filesSummaryItemsMin = 0;

export const FilesSummary = zod.object({
  cells: FilesStateCounts.describe("Its monitored Items' cells by state"),
  itemStatus: FilesStateCounts.describe(
    "Its monitored Items by status; an Item with no cell isn't counted",
  ),
  items: zod
    .int()
    .min(filesSummaryItemsMin)
    .describe(
      "Its monitored Items that its source still lists, the only Items the status and counts read",
    ),
  status: zod
    .union([FilesState, zod.null()])
    .describe(
      "The first status its monitored Items hold, in the glossary's order; null while none holds a cell",
    ),
});

export type FilesSummary = zod.input<typeof FilesSummary>;
export type FilesSummaryOutput = zod.output<typeof FilesSummary>;

export const FilesCellList = zod.object({
  cells: zod.array(FilesCell).describe("By Item in tree order, then Target in Profile order"),
  items: zod.array(FilesItemStatus).describe("Every Item in tree order"),
  levels: zod
    .array(FilesLevelRollup)
    .describe(
      "What each level's cells add up to, in the order its first Item comes; empty for a type with no levels",
    ),
  needsYou: zod.array(FilesNeed).describe("What the Work needs from an Admin, failures first"),
  summary: FilesSummary.describe("What the cells add up to for the Work"),
});

export type FilesCellList = zod.input<typeof FilesCellList>;
export type FilesCellListOutput = zod.output<typeof FilesCellList>;

export const FilesChanged = zod.object({
  workId: zod.uuid(),
});

export type FilesChanged = zod.input<typeof FilesChanged>;
export type FilesChangedOutput = zod.output<typeof FilesChanged>;

export const filesConfirmedWorkDigestRegExp = new RegExp("^[0-9a-f]{16}$");

export const FilesConfirmedWork = zod.object({
  digest: zod
    .string()
    .regex(filesConfirmedWorkDigestRegExp)
    .describe("The digest the Work's plan showed"),
  leave: zod
    .array(zod.uuid())
    .nullish()
    .describe(
      "Files to leave as they are; a file that trades or chains paths with another leaves both",
    ),
  work: zod.uuid(),
});

export type FilesConfirmedWork = zod.input<typeof FilesConfirmedWork>;
export type FilesConfirmedWorkOutput = zod.output<typeof FilesConfirmedWork>;

export const filesDolbyVisionCompatibilityMin = 0;

export const filesDolbyVisionLevelMin = 0;

export const filesDolbyVisionProfileMin = 0;

export const FilesDolbyVision = zod.object({
  compatibility: zod
    .int()
    .min(filesDolbyVisionCompatibilityMin)
    .describe("The base layer's signal: 1 HDR10, 2 SDR, 4 HLG, 6 Blu-ray HDR10, 0 none"),
  level: zod.int().min(filesDolbyVisionLevelMin),
  profile: zod.int().min(filesDolbyVisionProfileMin),
});

export type FilesDolbyVision = zod.input<typeof FilesDolbyVision>;
export type FilesDolbyVisionOutput = zod.output<typeof FilesDolbyVision>;

export const FilesFileGrab = zod.object({
  id: zod.uuid(),
  release: zod.string().describe("The Release's title"),
});

export type FilesFileGrab = zod.input<typeof FilesFileGrab>;
export type FilesFileGrabOutput = zod.output<typeof FilesFileGrab>;

export const FilesVideoCodec = zod.enum(["x264", "x265", "AV1", "VP9", "XviD", "MPEG2"]);

export type FilesVideoCodec = zod.input<typeof FilesVideoCodec>;
export type FilesVideoCodecOutput = zod.output<typeof FilesVideoCodec>;

export const FilesFill = zod.object({
  edition: zod.string().describe("any under any cut, else the key of the cut the cell wants"),
  itemId: zod.uuid(),
  targetId: zod.uuid(),
});

export type FilesFill = zod.input<typeof FilesFill>;
export type FilesFillOutput = zod.output<typeof FilesFill>;

export const FilesSubtitleTrack = zod.object({
  codec: zod.string().describe("ffprobe's codec name, such as subrip"),
  default: zod.boolean(),
  forced: zod.boolean(),
  image: zod.boolean().describe("A bitmap format, such as PGS"),
  language: zod.string().nullable().describe("An ISO 639 code; null when the track names none"),
});

export type FilesSubtitleTrack = zod.input<typeof FilesSubtitleTrack>;
export type FilesSubtitleTrackOutput = zod.output<typeof FilesSubtitleTrack>;

export const FilesHdrFormat = zod.enum(["DV", "HDR10+", "HDR10", "HLG", "SDR"]);

export type FilesHdrFormat = zod.input<typeof FilesHdrFormat>;
export type FilesHdrFormatOutput = zod.output<typeof FilesHdrFormat>;

export const filesVideoInfoBitDepthMin = 0;

export const filesVideoInfoFrameRateMin = 0;

export const filesVideoInfoHeightMin = 0;

export const filesVideoInfoWidthMin = 0;

export const FilesVideoInfo = zod.object({
  bitDepth: zod.int().min(filesVideoInfoBitDepthMin).describe("0 when the stream states none"),
  codec: zod.string().describe("ffprobe's codec name, such as hevc"),
  dolbyVision: zod
    .union([FilesDolbyVision, zod.null()])
    .describe("Null when it carries no Dolby Vision"),
  frameRate: zod
    .number()
    .min(filesVideoInfoFrameRateMin)
    .describe("Frames per second to three decimals; 0 when the stream states none"),
  hdr: zod.array(FilesHdrFormat).describe("Every HDR format it carries, or SDR alone"),
  height: zod.int().min(filesVideoInfoHeightMin),
  profile: zod.string().describe("Such as Main 10; empty when ffprobe names none"),
  width: zod.int().min(filesVideoInfoWidthMin),
});

export type FilesVideoInfo = zod.input<typeof FilesVideoInfo>;
export type FilesVideoInfoOutput = zod.output<typeof FilesVideoInfo>;

export const filesMediaInfoBitrateMin = 0;

export const filesMediaInfoDurationSecondsMin = 0;

export const FilesMediaInfo = zod.object({
  audio: zod.array(FilesAudioTrack).describe("In stream order"),
  bitrate: zod
    .int()
    .min(filesMediaInfoBitrateMin)
    .describe("Bits per second; 0 when the container states none"),
  container: zod.string().describe("ffprobe's format name, such as matroska,webm"),
  durationSeconds: zod
    .int()
    .min(filesMediaInfoDurationSecondsMin)
    .describe("0 when the container states none"),
  subtitles: zod.array(FilesSubtitleTrack).describe("In stream order"),
  video: FilesVideoInfo.describe("Its first video stream that isn't cover art"),
});

export type FilesMediaInfo = zod.input<typeof FilesMediaInfo>;
export type FilesMediaInfoOutput = zod.output<typeof FilesMediaInfo>;

export const filesListedFileSizeMin = 0;

export const FilesListedFile = zod.object({
  codec: zod
    .union([FilesVideoCodec, zod.null()])
    .describe(
      "Its video codec in the words a Target filters on; null for a codec they have no word for",
    ),
  cut: zod.string().describe("The key of its own cut"),
  fills: zod.array(FilesFill).describe("The cells it fills; empty for a Kept or gone file"),
  folder: zod.string().describe("The folder it sits in, relative to its root, / separated"),
  goneAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When a library scan found it missing from its root; null while it is there"),
  grab: zod
    .union([FilesFileGrab, zod.null()])
    .describe("The Grab whose download it came from; null for a file no Grab brought"),
  id: zod.uuid(),
  kept: zod.boolean().describe("A Kept file, which fills no cell"),
  label: zod.string().describe("The Target name its file name carries, before any suffix"),
  media: zod
    .union([FilesMediaInfo, zod.null()])
    .describe("What ffprobe read of it; null for a file read before SlipStream kept its tracks"),
  modifiedAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe(
      "Its modification time as a library scan or its import last read it; null until one did",
    ),
  path: zod.string().describe("Relative to its root, / separated"),
  placedBy: zod
    .union([
      zod.literal("link"),
      zod.literal("clone"),
      zod.literal("copy"),
      zod.literal("transfer"),
      zod.literal("move"),
      zod.literal("in_place"),
      zod.literal(null),
    ])
    .nullable()
    .describe("How its import put it into the library; null for a file no import recorded"),
  probeError: zod
    .string()
    .nullable()
    .describe(
      "The first line ffprobe printed when the last library scan couldn't read it, and media is then the read before; null when that read worked",
    ),
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
  size: zod.int().min(filesListedFileSizeMin),
});

export type FilesListedFile = zod.input<typeof FilesListedFile>;
export type FilesListedFileOutput = zod.output<typeof FilesListedFile>;

export const FilesFileList = zod.object({
  files: zod
    .array(FilesListedFile)
    .describe(
      "Filling files first, in Profile order of the first Target each fills, then Kept and gone files, oldest first",
    ),
  folder: zod
    .string()
    .describe(
      "The Work's folder under its root, the one its naming template gives while it has no file",
    ),
});

export type FilesFileList = zod.input<typeof FilesFileList>;
export type FilesFileListOutput = zod.output<typeof FilesFileList>;

export const FilesMoved = zod.object({
  workIds: zod.array(zod.string()),
});

export type FilesMoved = zod.input<typeof FilesMoved>;
export type FilesMovedOutput = zod.output<typeof FilesMoved>;

export const FilesRenameDraft = zod.object({
  contentType: zod.string().describe("The Content Type whose naming settings the draft changes"),
  values: zod
    .record(zod.string(), zod.unknown())
    .describe("Each naming setting's unsaved value by key, over the stored ones"),
});

export type FilesRenameDraft = zod.input<typeof FilesRenameDraft>;
export type FilesRenameDraftOutput = zod.output<typeof FilesRenameDraft>;

export const filesWorkLineBlockedMin = 0;

export const filesWorkLineMovingMin = 0;

export const filesWorkLineRenameMin = 0;

export const FilesWorkLine = zod.object({
  blocked: zod.int().min(filesWorkLineBlockedMin).describe("Files that can't be renamed now"),
  contentType: zod.string(),
  digest: zod.string().describe("A hash of the Work's rows, which a confirm sends back"),
  moving: zod
    .int()
    .min(filesWorkLineMovingMin)
    .describe("Files the disk queue still moves, which the plan leaves out"),
  rename: zod.int().min(filesWorkLineRenameMin).describe("Files that can be renamed now"),
  title: zod.string(),
  workId: zod.uuid(),
});

export type FilesWorkLine = zod.input<typeof FilesWorkLine>;
export type FilesWorkLineOutput = zod.output<typeof FilesWorkLine>;

export const filesRenamePlanBlockedMin = 0;

export const filesRenamePlanFilesMin = 0;

export const FilesRenamePlan = zod.object({
  blocked: zod
    .int()
    .min(filesRenamePlanBlockedMin)
    .describe("Files that can't be renamed now, in every Work"),
  files: zod
    .int()
    .min(filesRenamePlanFilesMin)
    .describe("Files that can be renamed now, in every Work"),
  works: zod
    .array(FilesWorkLine)
    .describe("Each Work with a file to rename, a blocked file or a file still moving"),
});

export type FilesRenamePlan = zod.input<typeof FilesRenamePlan>;
export type FilesRenamePlanOutput = zod.output<typeof FilesRenamePlan>;

export const filesRenameResultQueuedMin = 0;

export const FilesRenameResult = zod.object({
  changed: zod
    .array(zod.uuid())
    .describe("Works left as they are because their plans differ now; plan them again"),
  queued: zod
    .int()
    .min(filesRenameResultQueuedMin)
    .describe("Files queued to move to their new paths"),
});

export type FilesRenameResult = zod.input<typeof FilesRenameResult>;
export type FilesRenameResultOutput = zod.output<typeof FilesRenameResult>;

export const FilesRowBlock = zod.object({
  detail: zod.string().describe("Why the file can't be renamed now, then what to do"),
  otherWorkId: zod
    .uuid()
    .exactOptional()
    .describe("For clash, the Work whose file ends at the same path"),
  reason: zod.enum(["occupied", "too_long", "clash"]),
});

export type FilesRowBlock = zod.input<typeof FilesRowBlock>;
export type FilesRowBlockOutput = zod.output<typeof FilesRowBlock>;

export const filesRenameRowGroupMin = 0;

export const FilesRenameRow = zod.object({
  blocked: FilesRowBlock.exactOptional().describe(
    "Why it can't be renamed now; absent when it can",
  ),
  companions: zod
    .array(zod.string())
    .describe("Files beside it that move with it, such as its subtitles"),
  fileId: zod.uuid(),
  from: zod.string().describe("Where the file is now, relative to its root"),
  group: zod
    .int()
    .min(filesRenameRowGroupMin)
    .describe(
      "0 for a row joined to no other; rows with one number trade or chain paths, so they are renamed or left together",
    ),
  to: zod.string().describe("Where the naming templates put it; empty when its name doesn't fit"),
});

export type FilesRenameRow = zod.input<typeof FilesRenameRow>;
export type FilesRenameRowOutput = zod.output<typeof FilesRenameRow>;

export const FilesRenameScope = zod.object({
  id: zod
    .string()
    .exactOptional()
    .describe("The Work's id for work, the Content Type's id for type; left out for library"),
  kind: zod.enum(["work", "type", "library"]),
});

export type FilesRenameScope = zod.input<typeof FilesRenameScope>;
export type FilesRenameScopeOutput = zod.output<typeof FilesRenameScope>;

export const FilesUnmatchedType = zod.object({
  contentType: zod.string(),
  files: zod.int().min(1),
});

export type FilesUnmatchedType = zod.input<typeof FilesUnmatchedType>;
export type FilesUnmatchedTypeOutput = zod.output<typeof FilesUnmatchedType>;

export const filesWorkRenamePlanMovingMin = 0;

export const FilesWorkRenamePlan = zod.object({
  contentType: zod.string(),
  digest: zod.string().describe("A hash of the Work's rows, which a confirm sends back"),
  moving: zod
    .int()
    .min(filesWorkRenamePlanMovingMin)
    .describe("Files the disk queue still moves, which the plan leaves out"),
  rows: zod
    .array(FilesRenameRow)
    .describe("Each file whose path differs from what the naming templates make"),
  title: zod.string(),
  workId: zod.uuid(),
});

export type FilesWorkRenamePlan = zod.input<typeof FilesWorkRenamePlan>;
export type FilesWorkRenamePlanOutput = zod.output<typeof FilesWorkRenamePlan>;

export const FirstRunStepInputBody = zod.object({
  outcome: zod.enum(["done", "skipped"]).describe("done for Continue, skipped for Skip"),
});

export type FirstRunStepInputBody = zod.input<typeof FirstRunStepInputBody>;
export type FirstRunStepInputBodyOutput = zod.output<typeof FirstRunStepInputBody>;

export const FirstrunChanged = zod.object({});

export type FirstrunChanged = zod.input<typeof FirstrunChanged>;
export type FirstrunChangedOutput = zod.output<typeof FirstrunChanged>;

export const FirstrunStepID = zod.enum([
  "address",
  "passkey",
  "media",
  "language",
  "roots",
  "downloads",
  "targets",
]);

export type FirstrunStepID = zod.input<typeof FirstrunStepID>;
export type FirstrunStepIDOutput = zod.output<typeof FirstrunStepID>;

export const FirstrunStepState = zod.object({
  id: FirstrunStepID,
  state: zod.enum(["todo", "done", "skipped"]),
});

export type FirstrunStepState = zod.input<typeof FirstrunStepState>;
export type FirstrunStepStateOutput = zod.output<typeof FirstrunStepState>;

export const FirstrunFirstRun = zod.object({
  open: zod.boolean().describe("False once an Admin finished first setup, which never reopens"),
  steps: zod.array(FirstrunStepState).describe("The shown steps, in order"),
});

export type FirstrunFirstRun = zod.input<typeof FirstrunFirstRun>;
export type FirstrunFirstRunOutput = zod.output<typeof FirstrunFirstRun>;

export const ForgetInputBody = zod.object({
  fileIds: zod.array(zod.uuid()).min(1),
});

export type ForgetInputBody = zod.input<typeof ForgetInputBody>;
export type ForgetInputBodyOutput = zod.output<typeof ForgetInputBody>;

export const FormatsCondition = zod.object({
  fields: zod
    .record(zod.string(), zod.unknown())
    .describe(
      "Each field by name, as Radarr and Sonarr export them, such as value, or min and max; a value is a number in the radarr and sonarr dialects and a name in slipstream's",
    ),
  kind: zod
    .string()
    .describe("Radarr's and Sonarr's implementation name, such as ReleaseTitleSpecification"),
  name: zod.string(),
  negate: zod.boolean().describe("The condition matches when its test fails"),
  required: zod
    .boolean()
    .describe("The format matches only when this condition does, beside the others of its kind"),
});

export type FormatsCondition = zod.input<typeof FormatsCondition>;
export type FormatsConditionOutput = zod.output<typeof FormatsCondition>;

export const FormatsTargetRef = zod.object({
  id: zod.uuid(),
  name: zod.string(),
});

export type FormatsTargetRef = zod.input<typeof FormatsTargetRef>;
export type FormatsTargetRefOutput = zod.output<typeof FormatsTargetRef>;

export const formatsCustomFormatSlowReleasesMin = 0;

export const FormatsCustomFormat = zod.object({
  conditions: zod.array(FormatsCondition),
  dialect: zod
    .enum(["slipstream", "radarr", "sonarr"])
    .describe(
      "The app whose values its conditions hold: radarr or sonarr for a pasted format, slipstream for one made here",
    ),
  id: zod.uuid(),
  includeWhenRenaming: zod.boolean().describe("The Custom Formats naming token lists it"),
  name: zod.string(),
  scoredBy: zod.array(FormatsTargetRef).describe("Every Target that scores it, by name"),
  slowReleases: zod
    .int()
    .min(formatsCustomFormatSlowReleasesMin)
    .describe(
      "Releases one of its patterns ran past the match limit on since its last save; they count as not matching it",
    ),
});

export type FormatsCustomFormat = zod.input<typeof FormatsCustomFormat>;
export type FormatsCustomFormatOutput = zod.output<typeof FormatsCustomFormat>;

export const FormatsKindValue = zod.object({
  value: Bare.describe(
    "As the value field holds it: a number in the radarr and sonarr dialects, a name in slipstream's",
  ),
  words: zod.string(),
});

export type FormatsKindValue = zod.input<typeof FormatsKindValue>;
export type FormatsKindValueOutput = zod.output<typeof FormatsKindValue>;

export const FormatsOfferedKind = zod.object({
  fields: zod.array(zod.string()).describe("The fields its conditions take, in upstream's order"),
  kind: zod.string(),
  pattern: zod.boolean().describe("Its value is a .NET regular expression, matched ignoring case"),
  values: zod
    .array(FormatsKindValue)
    .describe(
      "What its value field takes, in upstream's order; empty for a kind whose value is a pattern or a number",
    ),
  words: zod.string(),
});

export type FormatsOfferedKind = zod.input<typeof FormatsOfferedKind>;
export type FormatsOfferedKindOutput = zod.output<typeof FormatsOfferedKind>;

export const FormatsDialectSchema = zod.object({
  dialect: zod.enum(["slipstream", "radarr", "sonarr"]),
  kinds: zod
    .array(FormatsOfferedKind)
    .describe("In upstream's order; a kind only the other app has is left out"),
});

export type FormatsDialectSchema = zod.input<typeof FormatsDialectSchema>;
export type FormatsDialectSchemaOutput = zod.output<typeof FormatsDialectSchema>;

export const formatsDraftNameMax = 100;

export const FormatsDraft = zod.object({
  conditions: zod.array(FormatsCondition),
  formatId: zod
    .uuid()
    .exactOptional()
    .describe(
      "The saved Custom Format the draft stands in for, whose app it is written in; left out, the draft is a new format in SlipStream's own values",
    ),
  includeWhenRenaming: zod.boolean().exactOptional(),
  name: zod.string().min(1).max(formatsDraftNameMax),
});

export type FormatsDraft = zod.input<typeof FormatsDraft>;
export type FormatsDraftOutput = zod.output<typeof FormatsDraft>;

export const FormatsEvaluated = zod.object({
  kind: zod.string(),
  matched: zod.boolean().describe("After negate"),
  name: zod.string(),
  negate: zod.boolean(),
  required: zod.boolean(),
  timedOut: zod
    .boolean()
    .describe("Its pattern ran past the match limit, which counts as not matching"),
});

export type FormatsEvaluated = zod.input<typeof FormatsEvaluated>;
export type FormatsEvaluatedOutput = zod.output<typeof FormatsEvaluated>;

export const FormatsExplained = zod.object({
  conditions: zod.array(FormatsEvaluated),
  draft: zod.boolean().describe("It is the request's draft"),
  id: zod.uuid(),
  matched: zod.boolean(),
  name: zod.string(),
  timedOut: zod.boolean(),
});

export type FormatsExplained = zod.input<typeof FormatsExplained>;
export type FormatsExplainedOutput = zod.output<typeof FormatsExplained>;

export const TargetsRenamedFile = zod.object({
  fileId: zod.uuid(),
  from: zod.string().describe("The file's path in its root now"),
  to: zod.string().describe("The file's path in its root after the change"),
});

export type TargetsRenamedFile = zod.input<typeof TargetsRenamedFile>;
export type TargetsRenamedFileOutput = zod.output<typeof TargetsRenamedFile>;

export const TargetsWorkRenames = zod.object({
  files: zod.array(TargetsRenamedFile).describe("In the order the files were added"),
  title: zod.string(),
  workId: zod.uuid(),
});

export type TargetsWorkRenames = zod.input<typeof TargetsWorkRenames>;
export type TargetsWorkRenamesOutput = zod.output<typeof TargetsWorkRenames>;

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
    .describe(
      "Downloads in progress for cells that are no longer wanted. Each waits in Pending imports once it finishes",
    ),
  newlyWanted: zod.int().min(targetsImpactNewlyWantedMin).describe("Cells that become wanted"),
  ownList: zod
    .int()
    .min(targetsImpactOwnListMin)
    .describe("Works that keep their own Edition list through a change to their Profile's"),
  renamed: zod
    .int()
    .min(targetsImpactRenamedMin)
    .describe("Files whose path changes, Kept files included"),
  renames: zod
    .array(TargetsWorkRenames)
    .describe("The first 500 of the renamed files, by Work in title order"),
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

export const FormatsFormatChange = zod.object({
  format: FormatsCustomFormat,
  impact: TargetsImpact,
});

export type FormatsFormatChange = zod.input<typeof FormatsFormatChange>;
export type FormatsFormatChangeOutput = zod.output<typeof FormatsFormatChange>;

export const FormatsFormatDeletion = zod.object({
  impact: TargetsImpact,
  scoredBy: zod
    .array(FormatsTargetRef)
    .describe("The Targets whose scores for it the delete drops, by name"),
});

export type FormatsFormatDeletion = zod.input<typeof FormatsFormatDeletion>;
export type FormatsFormatDeletionOutput = zod.output<typeof FormatsFormatDeletion>;

export const formatsFormatListMatchLimitMillisecondsMin = 0;

export const FormatsFormatList = zod.object({
  formats: zod.array(FormatsCustomFormat).describe("By name"),
  matchLimitMilliseconds: zod
    .int()
    .min(formatsFormatListMatchLimitMillisecondsMin)
    .describe("How long one pattern may run against one name before it counts as not matching"),
});

export type FormatsFormatList = zod.input<typeof FormatsFormatList>;
export type FormatsFormatListOutput = zod.output<typeof FormatsFormatList>;

export const FormatsFormatRef = zod.object({
  dialect: zod.enum(["slipstream", "radarr", "sonarr"]),
  id: zod.uuid(),
  name: zod.string(),
});

export type FormatsFormatRef = zod.input<typeof FormatsFormatRef>;
export type FormatsFormatRefOutput = zod.output<typeof FormatsFormatRef>;

export const formatsNewFormatNameMax = 100;

export const FormatsNewFormat = zod.object({
  conditions: zod
    .array(FormatsCondition)
    .describe(
      "Conditions of a kind match when one of them does and every required one does; the format matches when every kind's do",
    ),
  includeWhenRenaming: zod
    .boolean()
    .exactOptional()
    .describe("The Custom Formats naming token lists it"),
  name: zod
    .string()
    .min(1)
    .max(formatsNewFormatNameMax)
    .describe("Unique among Custom Formats, ignoring case"),
});

export type FormatsNewFormat = zod.input<typeof FormatsNewFormat>;
export type FormatsNewFormatOutput = zod.output<typeof FormatsNewFormat>;

export const formatsParseKindDefault = `release`;
export const formatsParseNamesMax = 100;

export const FormatsParse = zod.object({
  draft: FormatsDraft.exactOptional().describe(
    "A Custom Format matched as if saved, in place of formatId's",
  ),
  kind: zod
    .enum(["release", "file"])
    .default(formatsParseKindDefault)
    .describe(
      "How each name is read. release reads it as a release title. file reads it as a file path, and reads its folder's name when the file name gives no title.",
    ),
  names: zod
    .array(zod.string())
    .min(1)
    .max(formatsParseNamesMax)
    .nullable()
    .describe("Release names, or file paths when kind is file"),
  workId: zod
    .uuid()
    .exactOptional()
    .describe(
      "The Work to judge every name against; left out, each name is matched to the Works whose titles it gives, read as each Content Type in turn",
    ),
});

export type FormatsParse = zod.input<typeof FormatsParse>;
export type FormatsParseOutput = zod.output<typeof FormatsParse>;

export const FormatsReadFacts = zod.object({
  audio: zod.array(zod.string()).describe("Audio codecs, best first"),
  channels: zod.array(zod.string()).describe("Channel layouts, best first"),
  cut: zod.string().describe("The cut's key, standard when the name names none"),
  edition: zod.string().nullable().describe("The edition text as the name gives it"),
  group: zod.string().nullable().describe("The release group"),
  hdr: zod.array(zod.string()).describe("In words, such as Dolby Vision, best first"),
  ids: zod
    .record(zod.string(), zod.string())
    .describe("The id tags the name carries, by Torznab's names: tmdbid, imdbid, tvdbid"),
  languages: zod.array(zod.string()).describe("In words, such as French"),
  marks: zod.array(zod.string()).describe("What the name numbers, such as S01E02, in words"),
  modifier: zod.string().nullable().describe("The quality modifier in words, such as Remux"),
  quality: zod
    .string()
    .nullable()
    .describe("The Quality's name, such as WEBDL-1080p; null when the name gives none"),
  resolution: zod.string().nullable(),
  revision: zod
    .string()
    .nullable()
    .describe("The revision in words, such as v2 REPACK; null for the original release"),
  source: zod.string().nullable().describe("The fine source in words, such as WEB-DL"),
  span: zod
    .enum(["", "one", "several", "level", "levels"])
    .describe(
      "One Item, several Items of one level, a whole level or several levels; empty when the name numbers nothing",
    ),
  titles: zod.array(zod.string()).describe("Every title the name gives, as in A AKA B"),
  video: zod.string().nullable().describe("The video codec"),
  year: zod.int().nullable(),
});

export type FormatsReadFacts = zod.input<typeof FormatsReadFacts>;
export type FormatsReadFactsOutput = zod.output<typeof FormatsReadFacts>;

export const GrabsWordedReason = zod.object({
  attribute: zod
    .enum(["hdr", "video_codec", "audio_codec", "channels", "language"])
    .exactOptional()
    .describe("The attribute a failed filter rules on"),
  code: zod
    .enum([
      "wrong_work",
      "wrong_item",
      "wrong_title",
      "cut_not_wanted",
      "quality_not_in_target",
      "filter",
      "size_below",
      "size_above",
      "size_not_checked",
      "score_below_minimum",
      "not_an_upgrade",
      "repack_other_group",
      "blocklisted",
    ])
    .describe(
      "Why the release was refused, or a note on it. wrong_item is a release whose title names another Item or no Item. wrong_title is a release whose title or year doesn't match the Work's. blocklisted is a release whose info hash is on the Blocklist.",
    ),
  soft: zod
    .boolean()
    .describe(
      "Whether the reason keeps only an automatic search from grabbing the release. A Manual search can grab past it. False for a hard refusal and for a note.",
    ),
  targetId: zod
    .uuid()
    .exactOptional()
    .describe("The Target whose verdict it is, absent for wrong_work, wrong_item and wrong_title"),
  value: zod
    .string()
    .exactOptional()
    .describe(
      "The value a filter failed on, the quality a Target doesn't accept, the cut the cell's Edition doesn't take, the score under the minimum, what a wrong_item title names, such as S01E02, or the title and year a wrong_title release gives, such as Heat (1986)",
    ),
  words: zod.string().describe("The reason as a sentence"),
});

export type GrabsWordedReason = zod.input<typeof GrabsWordedReason>;
export type GrabsWordedReasonOutput = zod.output<typeof GrabsWordedReason>;

export const FormatsTriedTarget = zod.object({
  name: zod.string(),
  reasons: zod
    .array(GrabsWordedReason)
    .describe(
      "The name's own refusals, then the reasons of the first cell it would fill, else of the first cell it was judged at",
    ),
  score: zod.int(),
  targetId: zod.uuid(),
  wouldFill: zod.boolean().describe("An automatic search would grab it for this Target"),
});

export type FormatsTriedTarget = zod.input<typeof FormatsTriedTarget>;
export type FormatsTriedTargetOutput = zod.output<typeof FormatsTriedTarget>;

export const FormatsWorkRef = zod.object({
  id: zod.uuid(),
  label: zod.string(),
});

export type FormatsWorkRef = zod.input<typeof FormatsWorkRef>;
export type FormatsWorkRefOutput = zod.output<typeof FormatsWorkRef>;

export const FormatsParsed = zod.object({
  covers: zod
    .array(zod.uuid())
    .describe("The Items the name covers of the one Work it was judged against"),
  facts: FormatsReadFacts,
  formats: zod.array(FormatsExplained).describe("Every Custom Format, the draft included, by name"),
  name: zod.string(),
  sentence: zod
    .string()
    .describe(
      "What the name reads as, the Works it matches and the Targets it would fill, or why it would fill none",
    ),
  targets: zod
    .array(FormatsTriedTarget)
    .describe(
      "With one Work, its Profile's Targets judged against its held files; with several or none, every Target with no held file",
    ),
  words: zod
    .string()
    .describe(
      "The facts in words, joined with a middle dot: Quality, the revision when it isn't the original, HDR, audio and channels, release group and size",
    ),
  work: zod
    .union([FormatsWorkRef, zod.null()])
    .describe(
      "The one Work the name is judged against: the given Work, or the only Work it matches; null when it matches several or none",
    ),
  works: zod
    .array(FormatsWorkRef)
    .describe(
      "The given Work when the name matches it, else every Work whose title and year it gives",
    ),
});

export type FormatsParsed = zod.input<typeof FormatsParsed>;
export type FormatsParsedOutput = zod.output<typeof FormatsParsed>;

export const FormatsParseResults = zod.object({
  results: zod.array(FormatsParsed).describe("In the order of names"),
});

export type FormatsParseResults = zod.input<typeof FormatsParseResults>;
export type FormatsParseResultsOutput = zod.output<typeof FormatsParseResults>;

export const FormatsPaste = zod.object({
  app: zod
    .enum(["radarr", "sonarr"])
    .exactOptional()
    .describe(
      "The app the text was written for; left out, the app its kinds settle: Edition, Quality modifier or Year mean Radarr and Release type means Sonarr",
    ),
  replace: zod
    .array(zod.uuid())
    .nullish()
    .describe(
      "The Custom Formats of a pasted one's name that it replaces; a pasted one whose name is taken and not listed here is added beside the other, with a number after its name",
    ),
  text: zod
    .string()
    .min(1)
    .describe(
      "One Custom Format's JSON or a list of them, as TRaSH Guides, Radarr and Sonarr export them or as Sonarr's API returns them",
    ),
});

export type FormatsPaste = zod.input<typeof FormatsPaste>;
export type FormatsPasteOutput = zod.output<typeof FormatsPaste>;

export const FormatsPastedFormat = zod.object({
  name: zod.string().describe("Its name in the text"),
  outcome: zod
    .enum(["add", "replace", "keep_both"])
    .describe("Added, replacing sameName, or added beside sameName"),
  sameName: FormatsFormatRef.exactOptional().describe(
    "The Custom Format that already has its name",
  ),
  savedAs: zod
    .string()
    .describe(
      "The name it is saved under: its own, or with a number after it when another Custom Format has its name",
    ),
});

export type FormatsPastedFormat = zod.input<typeof FormatsPastedFormat>;
export type FormatsPastedFormatOutput = zod.output<typeof FormatsPastedFormat>;

export const FormatsPasted = zod.object({
  app: zod
    .enum(["radarr", "sonarr"])
    .describe("The app the formats are read for: the one given, or the one their kinds settle"),
  formats: zod.array(FormatsPastedFormat).describe("In the text's order"),
  impact: TargetsImpact,
});

export type FormatsPasted = zod.input<typeof FormatsPasted>;
export type FormatsPastedOutput = zod.output<typeof FormatsPasted>;

export const FormatsSchema = zod.object({
  dialects: zod
    .array(FormatsDialectSchema)
    .describe("SlipStream's own, then Radarr's and Sonarr's"),
});

export type FormatsSchema = zod.input<typeof FormatsSchema>;
export type FormatsSchemaOutput = zod.output<typeof FormatsSchema>;

export const GrabsClientChoice = zod.object({
  builtInOn: zod
    .uuid()
    .exactOptional()
    .describe("A downloading Node, for its built-in client; send this or clientId"),
  clientId: zod.uuid().exactOptional().describe("An external client; send this or builtInOn"),
});

export type GrabsClientChoice = zod.input<typeof GrabsClientChoice>;
export type GrabsClientChoiceOutput = zod.output<typeof GrabsClientChoice>;

export const GrabsLiftKey = zod.object({
  code: zod.enum([
    "wrong_title",
    "size_below",
    "size_above",
    "score_below_minimum",
    "not_an_upgrade",
    "repack_other_group",
  ]),
  targetId: zod.uuid().exactOptional().describe("Absent for wrong_title"),
});

export type GrabsLiftKey = zod.input<typeof GrabsLiftKey>;
export type GrabsLiftKeyOutput = zod.output<typeof GrabsLiftKey>;

export const grabsGrabPickCellsMax = 100;

export const grabsGrabPickLiftsMax = 100;

export const GrabsGrabPick = zod.object({
  cancelGrabId: zod
    .uuid()
    .exactOptional()
    .describe("The Grab downloading a cell it fills, canceled first"),
  cells: zod
    .array(zod.string())
    .max(grabsGrabPickCellsMax)
    .describe("The cells the row or the Release showed"),
  client: GrabsClientChoice.exactOptional().describe("Absent to go where Tracker routing sends it"),
  edition: zod.string().min(1).exactOptional().describe("The Edition the row showed"),
  lifts: zod.array(GrabsLiftKey).max(grabsGrabPickLiftsMax).describe("The lifts the row showed"),
  releaseId: zod.string(),
  unblock: zod
    .boolean()
    .exactOptional()
    .describe("Remove the Release from the Blocklist and grab it"),
});

export type GrabsGrabPick = zod.input<typeof GrabsGrabPick>;
export type GrabsGrabPickOutput = zod.output<typeof GrabsGrabPick>;

export const grabManualInputBodyPicksMax = 2000;

export const GrabManualInputBody = zod.object({
  manualSearchId: zod.uuid(),
  picks: zod.array(GrabsGrabPick).min(1).max(grabManualInputBodyPicksMax),
});

export type GrabManualInputBody = zod.input<typeof GrabManualInputBody>;
export type GrabManualInputBodyOutput = zod.output<typeof GrabManualInputBody>;

export const GrabsWorkRef = zod.object({
  id: zod.uuid(),
  title: zod.string(),
});

export type GrabsWorkRef = zod.input<typeof GrabsWorkRef>;
export type GrabsWorkRefOutput = zod.output<typeof GrabsWorkRef>;

export const grabsBlockEntryInfoHashRegExp = new RegExp("^[0-9a-f]{40}$");

export const GrabsBlockEntry = zod.object({
  addedAt: zod.iso.datetime({ offset: true }),
  because: zod
    .string()
    .describe("The cause in a sentence, such as TorrentLeech reported it unregistered."),
  cause: zod.enum([
    "unregistered",
    "no_payload",
    "engine_error",
    "marked_failed",
    "by_hand",
    "no_video",
    "unreadable",
    "misdescribed",
  ]),
  indexer: zod
    .string()
    .nullable()
    .describe("The name of the Indexer that listed it; null when none is known"),
  infoHash: zod.string().regex(grabsBlockEntryInfoHashRegExp),
  title: zod.string(),
  words: zod.string().describe("What SlipStream reads from the title, in words"),
  work: zod
    .union([GrabsWorkRef, zod.null()])
    .describe("The Work it was grabbed for; null once the Work is deleted"),
});

export type GrabsBlockEntry = zod.input<typeof GrabsBlockEntry>;
export type GrabsBlockEntryOutput = zod.output<typeof GrabsBlockEntry>;

export const grabsBlocklistTotalMin = 0;

export const GrabsBlocklist = zod.object({
  entries: zod.array(GrabsBlockEntry).describe("Up to 50, newest first"),
  total: zod.int().min(grabsBlocklistTotalMin).describe("How many entries match in all"),
});

export type GrabsBlocklist = zod.input<typeof GrabsBlocklist>;
export type GrabsBlocklistOutput = zod.output<typeof GrabsBlocklist>;

export const GrabsBlocklistChanged = zod.object({});

export type GrabsBlocklistChanged = zod.input<typeof GrabsBlocklistChanged>;
export type GrabsBlocklistChangedOutput = zod.output<typeof GrabsBlocklistChanged>;

export const GrabsChanged = zod.object({
  grabId: zod.uuid(),
  progress: zod.boolean().describe("True when only the download's progress moved"),
  workId: zod.uuid(),
});

export type GrabsChanged = zod.input<typeof GrabsChanged>;
export type GrabsChangedOutput = zod.output<typeof GrabsChanged>;

export const GrabsEditionOverride = zod.object({
  edition: zod.string().min(1).describe("Send it back as the pick's edition"),
  words: zod
    .string()
    .describe(
      "Such as Its name gives the Extended Edition, which the Work doesn't want. It's grabbed as the Standard Edition.",
    ),
});

export type GrabsEditionOverride = zod.input<typeof GrabsEditionOverride>;
export type GrabsEditionOverrideOutput = zod.output<typeof GrabsEditionOverride>;

export const GrabsLostAt = zod.object({
  own: zod
    .string()
    .describe(
      "This Release's value at the step: a Target's name for want, a quality's name for tier, such as v2 or v1 REPACK for revision, a score, whether another Want chose it for chosen_by_another, a count of the Items it covers that aren't wanted for fewer_unwanted, an Indexer priority, a download factor from 0 to 1, a seeder count, bytes, or the title",
    ),
  ref: zod
    .string()
    .describe(
      "The reference Release's value at the step, written as own is. The reference is the grabbed Release, or the first a search could take when the run grabbed none.",
    ),
  step: zod
    .enum([
      "want",
      "tier",
      "revision",
      "score",
      "chosen_by_another",
      "fewer_unwanted",
      "priority",
      "freeleech",
      "seeders",
      "size",
      "title",
    ])
    .describe("The first step of the ranking where the reference Release ranked above this one"),
  targetId: zod
    .uuid()
    .nullable()
    .describe(
      "The Target the tier, revision and score steps compare at; null at the other steps and once the Target is deleted",
    ),
});

export type GrabsLostAt = zod.input<typeof GrabsLostAt>;
export type GrabsLostAtOutput = zod.output<typeof GrabsLostAt>;

export const grabsReleaseDetailsSeedersMin = 0;

export const grabsReleaseDetailsSizeMin = 0;

export const GrabsReleaseDetails = zod.object({
  blocked: zod
    .string()
    .nullable()
    .describe("Since when and why it is on the Blocklist; null when it isn't"),
  formatIds: zod
    .array(zod.uuid())
    .describe("The Custom Formats it matched of those the Work's Targets score"),
  freeleech: zod.boolean(),
  id: zod.string().describe("Send it back as a pick's releaseId"),
  indexers: zod.array(zod.string()).describe("The Indexers that listed it, by priority"),
  lostAt: zod
    .union([GrabsLostAt, zod.null()])
    .describe("Where it lost to the Release that fills its first cell; null when it didn't"),
  publishedAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("Null when the Indexer sent no date"),
  seeders: zod.int().min(grabsReleaseDetailsSeedersMin),
  size: zod.int().min(grabsReleaseDetailsSizeMin).describe("Bytes"),
  title: zod.string(),
  tracker: zod
    .string()
    .exactOptional()
    .describe("The Tracker its link reaches, absent when none is known"),
  verdicts: zod
    .array(zod.string())
    .describe("Its verdict at each Target of its Item, such as It would fill 1080p."),
  words: zod.string().describe("What SlipStream reads from the title, in words"),
});

export type GrabsReleaseDetails = zod.input<typeof GrabsReleaseDetails>;
export type GrabsReleaseDetailsOutput = zod.output<typeof GrabsReleaseDetails>;

export const GrabsChoice = zod.object({
  cells: zod
    .array(zod.string())
    .describe("The cells its row fills once it is pinned, the row's first leading"),
  edition: zod
    .union([GrabsEditionOverride, zod.null()])
    .describe("The Edition it is grabbed as; null unless that is another Edition than its cut's"),
  lifts: zod.array(GrabsWordedReason),
  note: zod
    .string()
    .exactOptional()
    .describe("Such as From TorrentLeech, which automatic search doesn't use."),
  pin: zod.string().describe("Send it in pins to swap this Release into its row"),
  release: GrabsReleaseDetails,
});

export type GrabsChoice = zod.input<typeof GrabsChoice>;
export type GrabsChoiceOutput = zod.output<typeof GrabsChoice>;

export const GrabsGrabClient = zod.object({
  id: zod.uuid(),
  kind: zod.enum(["qbittorrent", "transmission", "deluge", "rtorrent"]),
  name: zod.string(),
});

export type GrabsGrabClient = zod.input<typeof GrabsGrabClient>;
export type GrabsGrabClientOutput = zod.output<typeof GrabsGrabClient>;

export const GrabsClientWait = zod.object({
  client: zod
    .union([GrabsGrabClient, zod.null()])
    .describe("The external client; null for the Node's built-in client"),
  nodeId: zod.uuid().describe("The client's Node"),
  off: zod
    .boolean()
    .describe(
      "The Grab is pinned to this client and the client can't take it: it is off or deleted, or its Node doesn't hold Downloading. False while the client's Node is offline",
    ),
});

export type GrabsClientWait = zod.input<typeof GrabsClientWait>;
export type GrabsClientWaitOutput = zod.output<typeof GrabsClientWait>;

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
    "indexer_removed",
    "indexer_stopped",
    "home_offline",
    "torrent_unreachable",
    "magnet_invalid",
    "no_metadata",
    "no_peer_source",
    "torrent_invalid",
    "infohash_mismatch",
    "torrent_v2_only",
    "download_failed",
    "marked_failed",
    "removed",
    "client_removed",
    "client_deleted",
    "goal_met",
    "no_video_file",
    "probe_failed",
    "fills_nothing",
    "name_taken",
    "name_too_long",
    "transfer_failed",
    "import_failed",
    "node_written_off",
    "root_changed",
    "canceled",
    "blocklisted",
    "release_fault",
    "client_error",
    "declined",
    "unreadable",
  ]),
  current: zod
    .boolean()
    .describe(
      "No later Grab holds any of its cells. Needs you lists a failure only while it is current.",
    ),
  message: zod.string(),
  needsYou: zod
    .boolean()
    .describe(
      "It goes under Needs you. False for a cancel and for a Manual Grab whose Release the Blocklist held after its fetch, which its sheet already said.",
    ),
  stage: zod.enum(["fetching", "downloading", "probing", "transferring", "importing"]),
});

export type GrabsFailure = zod.input<typeof GrabsFailure>;
export type GrabsFailureOutput = zod.output<typeof GrabsFailure>;

export const GrabsFetchWait = zod.object({
  sentence: zod
    .string()
    .describe(
      "Why the fetch waits, such as Waiting to fetch the .torrent from TorrentLeech. SlipStream retries at 14:20.",
    ),
  until: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When SlipStream retries; null while it waits for the Indexer's home Node"),
});

export type GrabsFetchWait = zod.input<typeof GrabsFetchWait>;
export type GrabsFetchWaitOutput = zod.output<typeof GrabsFetchWait>;

export const GrabsFoldGroup = zod.object({
  fault: zod.boolean().describe("A fault in the Releases, as opposed to a refusal"),
  words: zod.string().describe("Such as 8 are 4K, which no Target here takes."),
});

export type GrabsFoldGroup = zod.input<typeof GrabsFoldGroup>;
export type GrabsFoldGroupOutput = zod.output<typeof GrabsFoldGroup>;

export const GrabsFolded = zod.object({
  groups: zod.array(GrabsFoldGroup),
  line: zod.string().describe("Such as SlipStream wouldn't grab 14 more Releases."),
});

export type GrabsFolded = zod.input<typeof GrabsFolded>;
export type GrabsFoldedOutput = zod.output<typeof GrabsFolded>;

export const GrabsGrabActions = zod.object({
  blocklistable: zod
    .boolean()
    .describe("A cancel can also put its Release on the Blocklist: its info hash is known"),
  cancel: zod.boolean().describe("cancelGrab can cancel it"),
  sendTo: zod.boolean().describe("sendGrabTo can send it to another client"),
});

export type GrabsGrabActions = zod.input<typeof GrabsGrabActions>;
export type GrabsGrabActionsOutput = zod.output<typeof GrabsGrabActions>;

export const grabsGrabImportPlacedMin = 0;

export const grabsGrabImportPlannedMin = 0;

export const grabsGrabImportSendingMin = 0;

export const grabsGrabImportWaitingMin = 0;

export const GrabsGrabImport = zod.object({
  id: zod.uuid(),
  placed: zod.int().min(grabsGrabImportPlacedMin).describe("Its files placed into the library"),
  planned: zod
    .int()
    .min(grabsGrabImportPlannedMin)
    .describe("Its files that are or were sending, placing, placed or waiting"),
  sending: zod.int().min(grabsGrabImportSendingMin).describe("Its files a Transfer is sending"),
  waiting: zod.int().min(grabsGrabImportWaitingMin).describe("Its files that wait for the Admin"),
  words: zod.string().describe("Its progress in words, such as Importing 6 of 10 episodes"),
});

export type GrabsGrabImport = zod.input<typeof GrabsGrabImport>;
export type GrabsGrabImportOutput = zod.output<typeof GrabsGrabImport>;

export const grabsMetadataSearchPeersMin = 0;

export const GrabsMetadataSearch = zod.object({
  endsAt: zod.iso
    .datetime({ offset: true })
    .describe("When the Grab fails unless a peer has sent the metadata"),
  peers: zod
    .int()
    .min(grabsMetadataSearchPeersMin)
    .describe("Peers connected at the Downloading Node's last response"),
});

export type GrabsMetadataSearch = zod.input<typeof GrabsMetadataSearch>;
export type GrabsMetadataSearchOutput = zod.output<typeof GrabsMetadataSearch>;

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

export const GrabsGrabTarget = zod.object({
  forced: zod
    .boolean()
    .describe("An Admin forced it: a file goes in even when it isn't an upgrade"),
  itemId: zod.uuid(),
  settled: zod
    .union([zod.literal("placed"), zod.literal("none"), zod.literal("declined"), zod.literal(null)])
    .nullable()
    .describe(
      "How the Grab finished with the cell: a file placed into it, no file filled it, or the Admin declined it. Null while the Grab holds it",
    ),
  targetId: zod.uuid(),
});

export type GrabsGrabTarget = zod.input<typeof GrabsGrabTarget>;
export type GrabsGrabTargetOutput = zod.output<typeof GrabsGrabTarget>;

export const grabsGrabReleaseSizeMin = 0;

export const GrabsGrab = zod.object({
  actions: GrabsGrabActions,
  createdAt: zod.iso.datetime({ offset: true }),
  downloadClient: zod
    .union([GrabsGrabClient, zod.null()])
    .describe(
      "Its Torrent's external Download Client, null for its Node's built-in client or until the .torrent is fetched",
    ),
  edition: zod
    .string()
    .describe("The Edition it was grabbed to fill: any under any cut, else a cut key"),
  failure: GrabsFailure.exactOptional().describe("Why it failed, absent unless it did"),
  fetchNodeId: zod
    .uuid()
    .nullable()
    .describe(
      "The Node that fetched the .torrent: the Indexer's Searching Node, or the Downloading Node that got a magnet link's metadata from peers. Null until one did",
    ),
  fetchWait: GrabsFetchWait.exactOptional().describe(
    "Why fetching the .torrent waits on its Indexer, absent unless it does",
  ),
  grabbedBy: zod.string().nullable().describe("A name, or the API key; null for a search's Grab"),
  history: zod
    .array(GrabsEntered)
    .describe("Each stage it entered, when it last entered it, oldest first"),
  id: zod.uuid(),
  import: zod
    .union([GrabsGrabImport, zod.null()])
    .describe("Its download's Import, from importing on; null before"),
  inFlight: zod
    .boolean()
    .describe("It holds its cells and counts in Activity: it is neither imported nor failed"),
  indexerId: zod
    .uuid()
    .nullable()
    .describe("The Indexer it was grabbed from, null once the Indexer is deleted"),
  infoHash: zod
    .string()
    .nullable()
    .describe("The v1 info hash in hex, null until the .torrent is fetched"),
  itemId: zod.uuid(),
  items: zod
    .array(zod.string())
    .describe("The ids of the Items its Release covers in the Work, in tree order"),
  manualSearchId: zod
    .uuid()
    .nullable()
    .describe(
      "The Manual search it came from; null for a search's Grab and once a later Manual search of the Work replaced it",
    ),
  metadata: GrabsMetadataSearch.exactOptional().describe(
    "The search for the magnet link's metadata, absent unless it is fetching from a magnet link and has asked its Downloading Node",
  ),
  nodeId: zod
    .uuid()
    .describe(
      "The Node that downloads it: the Indexer's home Node until the fetch, then the Downloading Node the fetch handed the .torrent to",
    ),
  overrides: zod
    .array(zod.string())
    .describe("Each choice the Admin made past the decision, as a sentence"),
  progress: GrabsProgress.exactOptional().describe(
    "The download's or the Transfer's progress, absent unless it is downloading and its Node reported it, or transferring",
  ),
  quality: zod
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
    .exactOptional()
    .describe("The quality the release title names, absent when it names none"),
  releaseSize: zod.int().min(grabsGrabReleaseSizeMin),
  releaseTitle: zod.string(),
  releaseWords: zod.string().describe("Its Release in words, such as Bluray-1080p · FKW · 9.8 GB"),
  source: zod
    .enum(["search", "manual"])
    .describe("How it was chosen: by a search's decision, or by an Admin in a Manual search"),
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
  targets: zod.array(GrabsGrabTarget).describe("Each cell it holds, in plan order"),
  torrentId: zod
    .uuid()
    .nullable()
    .describe("The Torrent it downloads, null until the .torrent is fetched"),
  trackerId: zod
    .uuid()
    .nullable()
    .describe("The Tracker its Release reached, null for a Grab older than Trackers"),
  waitsFor: GrabsClientWait.exactOptional().describe(
    "The client a fetching Grab goes to, absent unless it waits for that client: its Node to be back online, or a pinned client that can't take it",
  ),
  workId: zod.uuid(),
});

export type GrabsGrab = zod.input<typeof GrabsGrab>;
export type GrabsGrabOutput = zod.output<typeof GrabsGrab>;

export const GrabsGrabList = zod.object({
  grabs: zod.array(GrabsGrab).describe("Newest first"),
});

export type GrabsGrabList = zod.input<typeof GrabsGrabList>;
export type GrabsGrabListOutput = zod.output<typeof GrabsGrabList>;

export const GrabsHeldCell = zod.object({
  grabId: zod.uuid(),
  stage: zod.enum(["fetching", "downloading", "probing", "transferring", "importing"]),
  words: zod
    .string()
    .describe("Such as A Grab is downloading WEBDL-1080p · FLUX · 6.2 GB for 1080p."),
});

export type GrabsHeldCell = zod.input<typeof GrabsHeldCell>;
export type GrabsHeldCellOutput = zod.output<typeof GrabsHeldCell>;

export const GrabsGrabOffer = zod.object({
  cancels: zod.array(GrabsHeldCell).describe("The Grabs downloading a cell it would also fill"),
  cells: zod
    .array(zod.string())
    .describe(
      "The cells a Grab of it alone fills, any a Grab downloads among them. Send them back as the pick's cells.",
    ),
  edition: zod
    .union([GrabsEditionOverride, zod.null()])
    .describe("The Edition it is grabbed as; null unless that is another Edition than its cut's"),
  grabbable: zod
    .boolean()
    .describe(
      "It fills some cell and its Indexer would take it now. One on the Blocklist is grabbed only with unblock.",
    ),
  lifts: zod.array(GrabsWordedReason),
  outcome: zod.string().exactOptional().describe("What a Grab of it alone gets"),
  refusal: zod.string().exactOptional().describe("Why it can't be grabbed"),
  waits: zod
    .string()
    .exactOptional()
    .describe(
      "Why its Indexer won't take it now, such as TorrentLeech is at its grab limit until 14:05.",
    ),
});

export type GrabsGrabOffer = zod.input<typeof GrabsGrabOffer>;
export type GrabsGrabOfferOutput = zod.output<typeof GrabsGrabOffer>;

export const GrabsIndexerGrabList = zod.object({
  grabs: zod.array(GrabsGrab).describe("Newest first"),
  nextOffset: zod.int().nullable().describe("The offset of the next page; null on the last"),
});

export type GrabsIndexerGrabList = zod.input<typeof GrabsIndexerGrabList>;
export type GrabsIndexerGrabListOutput = zod.output<typeof GrabsIndexerGrabList>;

export const GrabsIndexerLine = zod.object({
  automatic: zod.boolean().describe("Automatic search uses it"),
  done: zod.boolean().describe("It has an outcome for every Item it was asked about"),
  failed: zod.boolean().describe("A fault at the site, as opposed to a limit or no Releases"),
  indexerId: zod.uuid().nullable().describe("Null once the Indexer is deleted"),
  name: zod.string(),
  sentence: zod
    .string()
    .describe(
      "Its outcome in words, such as Limit reached until 14:05, or TorrentLeech searched 6 of 10 episodes. Its limit resets at 14:05.",
    ),
});

export type GrabsIndexerLine = zod.input<typeof GrabsIndexerLine>;
export type GrabsIndexerLineOutput = zod.output<typeof GrabsIndexerLine>;

export const GrabsTargetScore = zod.object({
  score: zod.int(),
  targetId: zod.uuid(),
});

export type GrabsTargetScore = zod.input<typeof GrabsTargetScore>;
export type GrabsTargetScoreOutput = zod.output<typeof GrabsTargetScore>;

export const grabsListedReleaseCopyOfMin = 0;

export const grabsListedReleasePositionMin = 0;

export const grabsListedReleaseSeedersMin = 0;

export const grabsListedReleaseSizeMin = 0;

export const GrabsListedRelease = zod.object({
  copyOf: zod
    .int()
    .min(grabsListedReleaseCopyOfMin)
    .nullable()
    .describe(
      "The position of the Release this is a copy of, listed by an Indexer of no better priority; null when it is no copy",
    ),
  covers: zod.array(zod.uuid()).describe("The Items of the Work it holds"),
  formatIds: zod
    .array(zod.uuid())
    .describe(
      "The Custom Formats it matched of those the Work's Targets score, as the run saw them",
    ),
  indexerId: zod.uuid().nullable().describe("Null once the Indexer is deleted"),
  indexerName: zod.string().describe("The Indexer's name when the run started"),
  lostAt: zod
    .union([GrabsLostAt, zod.null()])
    .describe(
      "Where an outranked or kept Release lost to the reference Release; null for the others and for one that ties it at every step",
    ),
  outcome: zod
    .enum(["grabbed", "not_taken", "outranked", "kept", "refused"])
    .describe(
      "grabbed is the run's Grab. not_taken ranked above it, and its Indexer stopped, reached its limit or dropped it. outranked ranked below it. kept ranked, and the run grabbed nothing. refused has a refusal of its own or is grabbable at no cell.",
    ),
  position: zod
    .int()
    .min(grabsListedReleasePositionMin)
    .describe("Its place in the order the Indexers listed the run's Releases"),
  rank: zod
    .int()
    .min(1)
    .nullable()
    .describe(
      "Its place among the Releases grabbable at some cell, from 1; null for one grabbable at none",
    ),
  reasons: zod
    .array(GrabsWordedReason)
    .describe("Its own refusals, then each cell's failures and notes, in Profile order"),
  scores: zod
    .array(GrabsTargetScore)
    .describe("Its score at each Target it was judged at, in Profile order"),
  seeders: zod.int().min(grabsListedReleaseSeedersMin),
  size: zod.int().min(grabsListedReleaseSizeMin).describe("Bytes"),
  title: zod.string(),
  words: zod
    .string()
    .describe(
      "What SlipStream reads from the title, in words, such as WEBDL-1080p · Dolby Vision · DD+ 5.1 · FLUX · 6.2 GB",
    ),
});

export type GrabsListedRelease = zod.input<typeof GrabsListedRelease>;
export type GrabsListedReleaseOutput = zod.output<typeof GrabsListedRelease>;

export const GrabsManualCell = zod.object({
  edition: zod.string().min(1).describe("The cell's Edition: any, or a cut key"),
  itemId: zod.uuid(),
  targetId: zod.uuid(),
});

export type GrabsManualCell = zod.input<typeof GrabsManualCell>;
export type GrabsManualCellOutput = zod.output<typeof GrabsManualCell>;

export const GrabsManualScope = zod.object({
  cell: GrabsManualCell.exactOptional().describe(
    "The one cell it plans for, absent for a wider scope",
  ),
  itemIds: zod.array(zod.uuid()),
  levelIds: zod.array(zod.uuid()).describe("Each level stands for every Item under it"),
});

export type GrabsManualScope = zod.input<typeof GrabsManualScope>;
export type GrabsManualScopeOutput = zod.output<typeof GrabsManualScope>;

export const GrabsPinned = zod.object({
  key: zod.string().describe("The key of the row it fills"),
  pin: zod.string(),
});

export type GrabsPinned = zod.input<typeof GrabsPinned>;
export type GrabsPinnedOutput = zod.output<typeof GrabsPinned>;

export const GrabsPlanPick = zod.object({
  cells: zod
    .array(zod.string())
    .describe("The cells it fills, its row's first leading. Send them back as the pick's cells."),
  edition: zod
    .union([GrabsEditionOverride, zod.null()])
    .describe("The Edition it is grabbed as; null unless that is another Edition than its cut's"),
  lifts: zod
    .array(GrabsWordedReason)
    .describe(
      "The soft refusals it is grabbed past; send their codes and Targets back as the pick's lifts",
    ),
  release: GrabsReleaseDetails,
  routed: zod
    .string()
    .exactOptional()
    .describe(
      "The client Tracker routing sends it to, such as qBittorrent on Den; absent while no Node can take it",
    ),
});

export type GrabsPlanPick = zod.input<typeof GrabsPlanPick>;
export type GrabsPlanPickOutput = zod.output<typeof GrabsPlanPick>;

export const GrabsPlanRow = zod.object({
  cause: zod
    .string()
    .exactOptional()
    .describe(
      "For a cell nothing fills, its plain cause, such as No Indexer listed a 720p Release.",
    ),
  choices: zod
    .array(GrabsChoice)
    .describe("Every other Release that could fill its first cell, ranked"),
  failure: zod.string().exactOptional().describe("For a failed cell, its last Grab's failure"),
  key: zod
    .string()
    .describe(
      "The row's first cell, stable while that cell leads it. A pin chosen from its choices replaces the pin with this key.",
    ),
  missing: zod
    .string()
    .exactOptional()
    .describe("For a cell nothing fills, such as Missing: 720p for E05."),
  outcome: zod
    .string()
    .exactOptional()
    .describe("What a Grab of it gets, such as It gets you 1080p and 720p for E05."),
  pick: zod.union([GrabsPlanPick, zod.null()]).describe("Null for a cell nothing fills"),
  pinned: zod.boolean().describe("Its Release was swapped in by a pin"),
});

export type GrabsPlanRow = zod.input<typeof GrabsPlanRow>;
export type GrabsPlanRowOutput = zod.output<typeof GrabsPlanRow>;

export const GrabsPlan = zod.object({
  headline: zod.string(),
  held: zod.array(GrabsHeldCell).describe("The cells a Grab downloads, which the plan leaves out"),
  rows: zod.array(GrabsPlanRow).describe("Each Grab and each cell nothing fills, in Want order"),
});

export type GrabsPlan = zod.input<typeof GrabsPlan>;
export type GrabsPlanOutput = zod.output<typeof GrabsPlan>;

export const grabsManualSearchReleaseCountMin = 0;

export const GrabsManualSearch = zod.object({
  folded: GrabsFolded.describe("The Releases SlipStream wouldn't grab, in groups"),
  id: zod.uuid(),
  indexers: zod.array(GrabsIndexerLine),
  line: zod.string().describe("Such as Searching 6 Indexers. 4 have responded."),
  pins: zod.array(GrabsPinned).describe("The pins still in effect; send these next"),
  plan: GrabsPlan,
  query: zod
    .string()
    .nullable()
    .describe("The typed query; null when each run sent its Item's terms"),
  queryable: zod.boolean().describe("The scope is one Item, so a typed query can search it"),
  releaseCount: zod
    .int()
    .min(grabsManualSearchReleaseCountMin)
    .describe("How many Releases listManualSearchReleases lists"),
  scope: GrabsManualScope.describe(
    "What it was started for; no levels, Items or cell for the whole Work",
  ),
  searching: zod.boolean().describe("An Indexer asked hasn't responded yet"),
  startedAt: zod.iso.datetime({ offset: true }),
  startedBy: zod.string().describe("A name, or the API key"),
  title: zod.string().describe("The scope in words"),
});

export type GrabsManualSearch = zod.input<typeof GrabsManualSearch>;
export type GrabsManualSearchOutput = zod.output<typeof GrabsManualSearch>;

export const GrabsManualStarted = zod.object({
  manualSearchId: zod.uuid(),
});

export type GrabsManualStarted = zod.input<typeof GrabsManualStarted>;
export type GrabsManualStartedOutput = zod.output<typeof GrabsManualStarted>;

export const grabsOfferQualityMin = 0;

export const GrabsOffer = zod.object({
  grab: GrabsGrabOffer,
  quality: zod
    .int()
    .min(grabsOfferQualityMin)
    .describe("Its quality's place in its best Target's tiers, for sorting; higher is better"),
  rank: zod
    .int()
    .min(1)
    .nullable()
    .describe("Its place in the ranking from 1; null when no cell's Target takes it"),
  release: GrabsReleaseDetails,
  score: zod.int().describe("Its score at its best Target"),
});

export type GrabsOffer = zod.input<typeof GrabsOffer>;
export type GrabsOfferOutput = zod.output<typeof GrabsOffer>;

export const GrabsOfferList = zod.object({
  releases: zod.array(GrabsOffer).describe("One per group of copies, in rank order"),
});

export type GrabsOfferList = zod.input<typeof GrabsOfferList>;
export type GrabsOfferListOutput = zod.output<typeof GrabsOfferList>;

export const GrabsPickProblem = zod.object({
  code: zod.enum([
    "pick_conflict",
    "cell_held",
    "pick_changed",
    "not_taken",
    "not_grabbable",
    "grab_busy",
    "download_client_field_refused",
    "node_not_downloading",
  ]),
  detail: zod.string(),
  grabId: zod.uuid().exactOptional().describe("cell_held: the Grab downloading the cell"),
});

export type GrabsPickProblem = zod.input<typeof GrabsPickProblem>;
export type GrabsPickProblemOutput = zod.output<typeof GrabsPickProblem>;

export const GrabsPickResult = zod.object({
  grabId: zod.uuid().nullable().describe("Null when the pick was refused"),
  problem: zod.union([GrabsPickProblem, zod.null()]).describe("Null when the pick made a Grab"),
});

export type GrabsPickResult = zod.input<typeof GrabsPickResult>;
export type GrabsPickResultOutput = zod.output<typeof GrabsPickResult>;

export const GrabsPickResults = zod.object({
  results: zod.array(GrabsPickResult).describe("One for each pick, in the order sent"),
});

export type GrabsPickResults = zod.input<typeof GrabsPickResults>;
export type GrabsPickResultsOutput = zod.output<typeof GrabsPickResults>;

export const GrabsReason = zod.object({
  attribute: zod
    .enum(["hdr", "video_codec", "audio_codec", "channels", "language"])
    .exactOptional()
    .describe("The attribute a failed filter rules on"),
  code: zod
    .enum([
      "wrong_work",
      "wrong_item",
      "wrong_title",
      "cut_not_wanted",
      "quality_not_in_target",
      "filter",
      "size_below",
      "size_above",
      "size_not_checked",
      "score_below_minimum",
      "not_an_upgrade",
      "repack_other_group",
      "blocklisted",
    ])
    .describe(
      "Why the release was refused, or a note on it. wrong_item is a release whose title names another Item or no Item. wrong_title is a release whose title or year doesn't match the Work's. blocklisted is a release whose info hash is on the Blocklist.",
    ),
  soft: zod
    .boolean()
    .describe(
      "Whether the reason keeps only an automatic search from grabbing the release. A Manual search can grab past it. False for a hard refusal and for a note.",
    ),
  targetId: zod
    .uuid()
    .exactOptional()
    .describe("The Target whose verdict it is, absent for wrong_work, wrong_item and wrong_title"),
  value: zod
    .string()
    .exactOptional()
    .describe(
      "The value a filter failed on, the quality a Target doesn't accept, the cut the cell's Edition doesn't take, the score under the minimum, what a wrong_item title names, such as S01E02, or the title and year a wrong_title release gives, such as Heat (1986)",
    ),
});

export type GrabsReason = zod.input<typeof GrabsReason>;
export type GrabsReasonOutput = zod.output<typeof GrabsReason>;

export const GrabsReleaseList = zod.object({
  releases: zod
    .array(GrabsListedRelease)
    .describe(
      "Ranked ones first, in rank order, then the rest in the order they were listed. Empty once a later run of the Item finished.",
    ),
});

export type GrabsReleaseList = zod.input<typeof GrabsReleaseList>;
export type GrabsReleaseListOutput = zod.output<typeof GrabsReleaseList>;

export const grabsSearchIndexerReleasesMin = 0;

export const GrabsSearchIndexer = zod.object({
  indexerId: zod.uuid().nullable().describe("Null once the Indexer is deleted"),
  name: zod.string().describe("The Indexer's name when the run started"),
  outcome: zod
    .union([
      zod.literal("found"),
      zod.literal("no_releases"),
      zod.literal("limited"),
      zod.literal("failing"),
      zod.literal("stopped"),
      zod.literal("waiting"),
      zod.literal("late"),
      zod.literal("limit_reached"),
      zod.literal("no_categories"),
      zod.literal("removed"),
      zod.literal("rss_only"),
      zod.literal(null),
    ])
    .nullable()
    .describe("Null while its search runs"),
  releases: zod.int().min(grabsSearchIndexerReleasesMin).describe("The Releases it found"),
  sentence: zod.string().describe("The outcome in words, such as Limit reached until 14:05"),
});

export type GrabsSearchIndexer = zod.input<typeof GrabsSearchIndexer>;
export type GrabsSearchIndexerOutput = zod.output<typeof GrabsSearchIndexer>;

export const grabsSearchCandidatesMin = 0;

export const GrabsSearch = zod.object({
  candidates: zod
    .int()
    .min(grabsSearchCandidatesMin)
    .describe("The Releases the Indexers found, each Release listed twice counted once"),
  finishedAt: zod.iso.datetime({ offset: true }).nullable().describe("Null while the run searches"),
  grabId: zod
    .uuid()
    .nullable()
    .describe(
      "The run's Grab: the decision's, or the rerun's once a dropped Grab reran it; null when none",
    ),
  id: zod.uuid(),
  indexers: zod
    .array(GrabsSearchIndexer)
    .describe("Each Indexer the run asked or skipped, by priority"),
  itemId: zod.uuid(),
  notes: zod
    .array(GrabsReason)
    .describe(
      "The grabbed Release's notes, such as size_not_checked, for each Target it was grabbed for",
    ),
  outcome: zod
    .union([
      zod.literal("grabbed"),
      zod.literal("nothing_acceptable"),
      zod.literal("listed"),
      zod.literal(null),
    ])
    .nullable()
    .describe("Null while the run searches. listed is a Manual search's run, which takes nothing."),
  startedAt: zod.iso.datetime({ offset: true }),
});

export type GrabsSearch = zod.input<typeof GrabsSearch>;
export type GrabsSearchOutput = zod.output<typeof GrabsSearch>;

export const GrabsSearchChanged = zod.object({
  manualSearchId: zod
    .uuid()
    .nullable()
    .describe("The Manual search the run belongs to; null for an automatic run"),
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

export const grabsSearchScopeItemIdsMax = 20000;

export const grabsSearchScopeLevelIdsMax = 20000;

export const GrabsSearchScope = zod.object({
  itemIds: zod.array(zod.uuid()).max(grabsSearchScopeItemIdsMax),
  levelIds: zod
    .array(zod.uuid())
    .max(grabsSearchScopeLevelIdsMax)
    .describe("Each level stands for every Item under it"),
});

export type GrabsSearchScope = zod.input<typeof GrabsSearchScope>;
export type GrabsSearchScopeOutput = zod.output<typeof GrabsSearchScope>;

export const GrabsSearchStarted = zod.object({
  searchIds: zod.array(zod.string()).describe("One run per Item in the scope with a wanted cell"),
});

export type GrabsSearchStarted = zod.input<typeof GrabsSearchStarted>;
export type GrabsSearchStartedOutput = zod.output<typeof GrabsSearchStarted>;

export const grabsStartManualItemIdsMax = 20000;

export const grabsStartManualLevelIdsMax = 20000;

export const grabsStartManualQueryMax = 200;

export const GrabsStartManual = zod.object({
  again: zod
    .boolean()
    .exactOptional()
    .describe("Start a new search even when the Work's Manual search has this scope and query"),
  cell: GrabsManualCell.exactOptional().describe(
    "One cell to plan for; send it without levelIds and itemIds",
  ),
  itemIds: zod.array(zod.uuid()).max(grabsStartManualItemIdsMax).exactOptional(),
  levelIds: zod
    .array(zod.uuid())
    .max(grabsStartManualLevelIdsMax)
    .exactOptional()
    .describe("Each level stands for every Item under it"),
  query: zod
    .string()
    .max(grabsStartManualQueryMax)
    .exactOptional()
    .describe(
      "Text every Indexer searches for in place of the Item's terms; only for a scope of one Item",
    ),
});

export type GrabsStartManual = zod.input<typeof GrabsStartManual>;
export type GrabsStartManualOutput = zod.output<typeof GrabsStartManual>;

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

export const HealthCategoryID = zod.enum([
  "storage",
  "library",
  "nodes",
  "download-clients",
  "seeding",
  "indexers",
  "scheduled-tasks",
]);

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
  unsure: zod.boolean().describe("SlipStream couldn't check it, so it shows the last known state"),
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
      "Why its last check failed. Its items then show their last known state. Absent while checks succeed",
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
    .describe("Its Node is in step, so its figures are current. Otherwise they are from sampledAt"),
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

export const IdentityAddressChanged = zod.object({});

export type IdentityAddressChanged = zod.input<typeof IdentityAddressChanged>;
export type IdentityAddressChangedOutput = zod.output<typeof IdentityAddressChanged>;

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
    .describe("The mail server's reply, or why it didn't respond, when the email failed"),
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
  url: zod.url().describe("The link itself, shown only in this response"),
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
  firstRun: zod
    .enum(["open", "finished"])
    .exactOptional()
    .describe("An Admin's first setup; absent for a Member"),
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
  firstRun: zod
    .enum(["open", "finished"])
    .exactOptional()
    .describe("An Admin's first setup; absent for a Member"),
  keyPrompt: zod
    .enum(["ask", "never"])
    .describe("Whether the prompt to add a passkey may show after sign-in"),
  passkeyPrompt: zod
    .boolean()
    .describe(
      "Show the prompt to add a passkey for this address, once in this session. False while first setup is open, because its passkey step asks instead.",
    ),
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

export const ImportFolderInputBody = zod.object({
  folder: zod
    .string()
    .min(1)
    .describe("Absolute on the Node, inside one of its library roots or download folders"),
  nodeId: zod.uuid(),
});

export type ImportFolderInputBody = zod.input<typeof ImportFolderInputBody>;
export type ImportFolderInputBodyOutput = zod.output<typeof ImportFolderInputBody>;

export const ImportIDsInputBody = zod.object({
  importIds: zod.array(zod.uuid()).min(1),
});

export type ImportIDsInputBody = zod.input<typeof ImportIDsInputBody>;
export type ImportIDsInputBodyOutput = zod.output<typeof ImportIDsInputBody>;

export const ImportsAction = zod.enum([
  "confirm",
  "retry",
  "decline",
  "force",
  "match",
  "notThisRelease",
  "changeEditions",
  "turnOnRenaming",
  "changeSetting",
]);

export type ImportsAction = zod.input<typeof ImportsAction>;
export type ImportsActionOutput = zod.output<typeof ImportsAction>;

export const ImportsAddedOutcome = zod.object({
  id: zod.uuid(),
  outcome: zod.enum(["added"]),
  workId: zod.uuid().describe("The Work its files joined"),
});

export type ImportsAddedOutcome = zod.input<typeof ImportsAddedOutcome>;
export type ImportsAddedOutcomeOutput = zod.output<typeof ImportsAddedOutcome>;

export const ImportsOutcomeProblem = zod.object({
  code: zod
    .string()
    .describe("The code a request for this Import alone would have been refused with"),
  detail: zod.string().exactOptional(),
  title: zod.string(),
});

export type ImportsOutcomeProblem = zod.input<typeof ImportsOutcomeProblem>;
export type ImportsOutcomeProblemOutput = zod.output<typeof ImportsOutcomeProblem>;

export const ImportsFailedOutcome = zod.object({
  id: zod.uuid(),
  outcome: zod.enum(["failed"]),
  problem: ImportsOutcomeProblem,
});

export type ImportsFailedOutcome = zod.input<typeof ImportsFailedOutcome>;
export type ImportsFailedOutcomeOutput = zod.output<typeof ImportsFailedOutcome>;

export const ImportsOutcome = zod.discriminatedUnion("outcome", [
  ImportsAddedOutcome,
  ImportsFailedOutcome,
]);

export type ImportsOutcome = zod.input<typeof ImportsOutcome>;
export type ImportsOutcomeOutput = zod.output<typeof ImportsOutcome>;

export const ImportsAddedList = zod.object({
  outcomes: zod.array(ImportsOutcome).describe("One per Import, in request order"),
});

export type ImportsAddedList = zod.input<typeof ImportsAddedList>;
export type ImportsAddedListOutput = zod.output<typeof ImportsAddedList>;

export const ImportsCellRef = zod.object({
  edition: zod.string().describe("any under any cut, else the key of the cut the cell wants"),
  itemId: zod.uuid(),
  targetId: zod.uuid(),
});

export type ImportsCellRef = zod.input<typeof ImportsCellRef>;
export type ImportsCellRefOutput = zod.output<typeof ImportsCellRef>;

export const ImportsChanged = zod.object({
  rootIds: zod.array(zod.string()),
});

export type ImportsChanged = zod.input<typeof ImportsChanged>;
export type ImportsChangedOutput = zod.output<typeof ImportsChanged>;

export const importsChosenFilePositionMin = 0;

export const ImportsChosenFile = zod.object({
  itemIds: zod
    .array(zod.string())
    .min(1)
    .describe("The Items of the Work the file holds, in any order"),
  position: zod.int().min(importsChosenFilePositionMin).describe("A waiting file of the Import"),
});

export type ImportsChosenFile = zod.input<typeof ImportsChosenFile>;
export type ImportsChosenFileOutput = zod.output<typeof ImportsChosenFile>;

export const ImportsConfirmFile = zod.object({
  action: zod.enum(["confirm"]).describe("Import the file with its current reading"),
});

export type ImportsConfirmFile = zod.input<typeof ImportsConfirmFile>;
export type ImportsConfirmFileOutput = zod.output<typeof ImportsConfirmFile>;

export const ImportsDeclineFile = zod.object({
  action: zod.enum(["decline"]).describe("Don't import the file"),
});

export type ImportsDeclineFile = zod.input<typeof ImportsDeclineFile>;
export type ImportsDeclineFileOutput = zod.output<typeof ImportsDeclineFile>;

export const importsExtensionImpactFilesMin = 0;

export const importsExtensionImpactPendingMin = 0;

export const ImportsExtensionImpact = zod.object({
  files: zod
    .int()
    .min(importsExtensionImpactFilesMin)
    .describe("The library files that stop counting as video"),
  pending: zod
    .int()
    .min(importsExtensionImpactPendingMin)
    .describe("The files of unfinished Imports that stop counting as video"),
});

export type ImportsExtensionImpact = zod.input<typeof ImportsExtensionImpact>;
export type ImportsExtensionImpactOutput = zod.output<typeof ImportsExtensionImpact>;

export const ImportsRetryFile = zod.object({
  action: zod.enum(["retry"]).describe("Try the file again"),
});

export type ImportsRetryFile = zod.input<typeof ImportsRetryFile>;
export type ImportsRetryFileOutput = zod.output<typeof ImportsRetryFile>;

export const ImportsNotThisReleaseFile = zod.object({
  action: zod
    .enum(["notThisRelease"])
    .describe("Put the download's Release on the Blocklist and search again; a Grab's Import only"),
});

export type ImportsNotThisReleaseFile = zod.input<typeof ImportsNotThisReleaseFile>;
export type ImportsNotThisReleaseFileOutput = zod.output<typeof ImportsNotThisReleaseFile>;

export const ImportsForceFile = zod.object({
  action: zod
    .enum(["force"])
    .describe("Put the file into a Target even though it isn't an upgrade there"),
  cell: ImportsCellRef,
});

export type ImportsForceFile = zod.input<typeof ImportsForceFile>;
export type ImportsForceFileOutput = zod.output<typeof ImportsForceFile>;

export const ImportsMatchFile = zod.object({
  action: zod.enum(["match"]).describe("Change what the file is"),
  edition: zod
    .string()
    .nullable()
    .describe("The key of the cut it is; null for the one its name carries"),
  itemIds: zod
    .array(zod.uuid())
    .min(1)
    .describe("The Items of the Work the file holds, in any order"),
  releaseName: zod.string().min(1).describe("The name SlipStream reads its quality and cut from"),
  workId: zod.uuid(),
});

export type ImportsMatchFile = zod.input<typeof ImportsMatchFile>;
export type ImportsMatchFileOutput = zod.output<typeof ImportsMatchFile>;

export const ImportsFileAction = zod.discriminatedUnion("action", [
  ImportsConfirmFile,
  ImportsRetryFile,
  ImportsDeclineFile,
  ImportsNotThisReleaseFile,
  ImportsForceFile,
  ImportsMatchFile,
]);

export type ImportsFileAction = zod.input<typeof ImportsFileAction>;
export type ImportsFileActionOutput = zod.output<typeof ImportsFileAction>;

export const importsFileHoldAttemptsMin = 0;

export const ImportsFileHold = zod.object({
  attempts: zod.int().min(importsFileHoldAttemptsMin).describe("The tries that ran into the hold"),
  code: zod.enum(["no_space", "root_not_ready", "node_away", "probe_timeout", "probe_error"]),
  sentence: zod.string().describe("What it waits on and when SlipStream retries"),
});

export type ImportsFileHold = zod.input<typeof ImportsFileHold>;
export type ImportsFileHoldOutput = zod.output<typeof ImportsFileHold>;

export const ImportsForceOption = zod.object({
  cell: ImportsCellRef,
  deletes: zod
    .array(zod.string())
    .describe("The paths of the library files the force deletes, relative to their root"),
});

export type ImportsForceOption = zod.input<typeof ImportsForceOption>;
export type ImportsForceOptionOutput = zod.output<typeof ImportsForceOption>;

export const ImportsHintID = zod.object({
  provider: zod.enum(["tmdb", "imdb", "tvdb"]),
  value: zod.string(),
});

export type ImportsHintID = zod.input<typeof ImportsHintID>;
export type ImportsHintIDOutput = zod.output<typeof ImportsHintID>;

export const ImportsHint = zod.object({
  ids: zod.array(ImportsHintID).describe("The provider ids the name's tags carry"),
  title: zod.string().describe("The title the folder or file name carries, as written"),
  year: zod.int().nullable().describe("Null when the name carries none"),
});

export type ImportsHint = zod.input<typeof ImportsHint>;
export type ImportsHintOutput = zod.output<typeof ImportsHint>;

export const ImportsImportFailure = zod.object({
  code: zod.enum(["no_video_file", "unreadable"]),
  sentence: zod.string(),
});

export type ImportsImportFailure = zod.input<typeof ImportsImportFailure>;
export type ImportsImportFailureOutput = zod.output<typeof ImportsImportFailure>;

export const ImportsRootLabel = zod.object({
  id: zod.uuid(),
  label: zod
    .string()
    .describe("The root as copy names it, such as Movies library root on Media server"),
});

export type ImportsRootLabel = zod.input<typeof ImportsRootLabel>;
export type ImportsRootLabelOutput = zod.output<typeof ImportsRootLabel>;

export const ImportsReason = zod.object({
  code: zod
    .enum([
      "sample",
      "bonus",
      "hidden_folder",
      "too_small",
      "extension",
      "runtime_sample",
      "sibling_better",
      "kept_better",
      "meets_nothing",
      "not_an_upgrade",
      "edition_not_wanted",
      "other_items",
      "no_number",
      "parts",
      "name_taken",
      "name_too_long",
      "settings",
      "busy",
      "confirm",
      "several",
      "no_match",
      "no_year",
      "elsewhere",
      "unlisted",
      "no_such_item",
      "unreadable",
      "left",
      "no_rule",
      "import_failed",
      "root_changed",
      "by_admin",
      "removed",
      "canceled",
      "blocklisted",
      "work_deleted",
    ])
    .describe("Why the file is left out, waits, isn't better or failed"),
  count: zod.int().exactOptional().describe("For several: the candidates that match"),
  line: zod.string().exactOptional().describe("For unreadable: the first line ffprobe printed"),
  names: zod
    .string()
    .exactOptional()
    .describe("For no_such_item: the Item the file name names, such as S03E14"),
  root: ImportsRootLabel.exactOptional().describe("For elsewhere: the root the Work lives in"),
  sentence: zod
    .string()
    .nullable()
    .describe(
      "The one sentence the row leads with; null for a root Import's reason, which its other fields word",
    ),
  title: zod
    .string()
    .exactOptional()
    .describe(
      "For several, no_match, no_year, elsewhere and unlisted: the name's title or the Work's label",
    ),
  year: zod.int().exactOptional().describe("The year beside title, when it has one"),
});

export type ImportsReason = zod.input<typeof ImportsReason>;
export type ImportsReasonOutput = zod.output<typeof ImportsReason>;

export const ImportsTargetVerdict = zod.object({
  sentence: zod
    .string()
    .describe("How the Target judges the file, such as Fills 2160p for Heat (1995)."),
  targetId: zod.uuid(),
});

export type ImportsTargetVerdict = zod.input<typeof ImportsTargetVerdict>;
export type ImportsTargetVerdictOutput = zod.output<typeof ImportsTargetVerdict>;

export const importsImportFilePositionMin = 0;

export const importsImportFileSizeMin = 0;

export const importsImportFileStagedMin = 0;

export const ImportsImportFile = zod.object({
  actions: zod.array(ImportsAction).describe("The other actions its row offers"),
  cut: zod.string().describe("The key of the cut its name carries"),
  fileId: zod.uuid().nullable().describe("The library file it became; null until it is placed"),
  fills: zod.array(ImportsCellRef).describe("The cells it fills, or would fill once confirmed"),
  forces: zod
    .array(ImportsForceOption)
    .describe("Each Target it can be forced into, with the files that deletes"),
  hold: zod
    .union([ImportsFileHold, zod.null()])
    .describe(
      "What it waits on that SlipStream retries automatically; null while nothing holds it",
    ),
  itemIds: zod
    .array(zod.string())
    .describe("The Items of the Work it holds, in tree order; empty until a Work places it"),
  movable: zod.boolean().describe("Move files may take it: no tracked torrent seeds it"),
  outcome: zod.enum([
    "pending",
    "left_out",
    "not_better",
    "waiting",
    "sending",
    "placing",
    "placed",
    "declined",
    "failed",
  ]),
  path: zod.string().describe("For a root Import relative to the root, / separated"),
  placedBy: zod
    .union([
      zod.literal("link"),
      zod.literal("clone"),
      zod.literal("copy"),
      zod.literal("transfer"),
      zod.literal("move"),
      zod.literal("in_place"),
      zod.literal(null),
    ])
    .nullable()
    .describe("How it went into the library; null until it is placing"),
  position: zod
    .int()
    .min(importsImportFilePositionMin)
    .describe("The file's place in its Import, which names it in actions"),
  quality: zod
    .union([
      zod.literal("sdtv"),
      zod.literal("dvd"),
      zod.literal("webrip-480p"),
      zod.literal("hdtv-720p"),
      zod.literal("webrip-720p"),
      zod.literal("webdl-720p"),
      zod.literal("bluray-720p"),
      zod.literal("hdtv-1080p"),
      zod.literal("webrip-1080p"),
      zod.literal("webdl-1080p"),
      zod.literal("bluray-1080p"),
      zod.literal("remux-1080p"),
      zod.literal("hdtv-2160p"),
      zod.literal("webrip-2160p"),
      zod.literal("webdl-2160p"),
      zod.literal("bluray-2160p"),
      zod.literal("remux-2160p"),
      zod.literal("cam"),
      zod.literal(null),
    ])
    .nullable()
    .describe("The quality its name and media info read as; null when ffprobe can't read it"),
  reason: zod
    .union([ImportsReason, zod.null()])
    .describe("Why it is left out, waits, isn't better or failed; null otherwise"),
  recommended: zod
    .union([ImportsAction, zod.null()])
    .describe("The action its row offers first; null when it offers none"),
  size: zod.int().min(importsImportFileSizeMin),
  staged: zod
    .int()
    .min(importsImportFileStagedMin)
    .describe("Bytes its holding or staged copy uses while it waits"),
  verdicts: zod
    .array(ImportsTargetVerdict)
    .describe("Each Target's verdict on it, for its Details"),
});

export type ImportsImportFile = zod.input<typeof ImportsImportFile>;
export type ImportsImportFileOutput = zod.output<typeof ImportsImportFile>;

export const ImportsImportGrab = zod.object({
  id: zod.uuid(),
  release: zod.string().describe("The Release's title"),
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

export type ImportsImportGrab = zod.input<typeof ImportsImportGrab>;
export type ImportsImportGrabOutput = zod.output<typeof ImportsImportGrab>;

export const ImportsSuggestion = zod.object({
  confident: zod
    .boolean()
    .describe("It meets the rules that let a library scan add a Work by itself"),
  key: zod.string().describe("The record's key at its Content Type's source"),
  label: zod.string(),
  posterUrl: zod
    .string()
    .nullable()
    .describe("The artwork route for its poster, without a size; null when it has none"),
  workId: zod.uuid().nullable().describe("The Work, once it is in the library"),
  year: zod.int().nullable().describe("Null when the record has none"),
});

export type ImportsSuggestion = zod.input<typeof ImportsSuggestion>;
export type ImportsSuggestionOutput = zod.output<typeof ImportsSuggestion>;

export const importsImportEntryContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const ImportsImportEntry = zod.object({
  contentType: zod.string().regex(importsImportEntryContentTypeRegExp),
  createdAt: zod.iso.datetime({ offset: true }),
  edition: zod
    .string()
    .nullable()
    .describe("The key of the cut its files fill; null for any under any cut"),
  failure: zod
    .union([ImportsImportFailure, zod.null()])
    .describe("Why the whole Import failed; null unless it did"),
  files: zod.array(ImportsImportFile).describe("For a root Import in path order, else by position"),
  finishedAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("Null until every file is placed or settled"),
  folder: zod
    .string()
    .describe(
      "For a root Import, its Work folder relative to the root, or the file's path when it sits loose at the root's top. The folder on its Node for a folder Import, and empty for a download, whose file paths are relative to its torrent",
    ),
  grab: zod
    .union([ImportsImportGrab, zod.null()])
    .describe("The Grab whose download this is; null for a folder or a root"),
  hint: ImportsHint.describe("What the folder or file name says"),
  id: zod.uuid(),
  ignoredAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("Null while it needs the Admin"),
  moveFiles: zod
    .boolean()
    .describe(
      "A folder Import moves its files into the library and deletes the folder once it is empty",
    ),
  nodeId: zod
    .uuid()
    .nullable()
    .describe("The Node the files are on, for a download or a folder; null for a root"),
  root: zod
    .union([ImportsRootLabel, zod.null()])
    .describe("The library root the folder is in; null outside a root Import"),
  seeding: zod.boolean().describe("The download keeps seeding while a file waits"),
  source: zod
    .enum(["grab", "root", "folder"])
    .describe("root: a folder a library scan found in a root and couldn't place"),
  stage: zod.enum(["probing", "waiting", "placing", "done", "failed"]),
  suggestion: zod
    .union([ImportsSuggestion, zod.null()])
    .describe("The best candidate; null when none matched"),
  workId: zod
    .uuid()
    .nullable()
    .describe("The Work its files go into; null for a root Import, whose suggestion names one"),
});

export type ImportsImportEntry = zod.input<typeof ImportsImportEntry>;
export type ImportsImportEntryOutput = zod.output<typeof ImportsImportEntry>;

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

export const SettingAddressField = zod.object({
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
  kind: zod.enum(["address"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  running: zod.string(),
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  value: zod.string(),
});

export type SettingAddressField = zod.input<typeof SettingAddressField>;
export type SettingAddressFieldOutput = zod.output<typeof SettingAddressField>;

export const SettingFolderField = zod.object({
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
  kind: zod.enum(["folder"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  nodeId: zod.uuid().describe("The Node the path is on, which browses and judges it"),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  purpose: zod
    .enum(["downloads"])
    .describe("The rules the Node judges the folder by, as checkFolder names them"),
  running: zod.string(),
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  value: zod.string(),
});

export type SettingFolderField = zod.input<typeof SettingFolderField>;
export type SettingFolderFieldOutput = zod.output<typeof SettingFolderField>;

export const SettingTemplateField = zod.object({
  applies: zod.enum(["live", "restart"]),
  default: zod.string(),
  defaultFrom: zod
    .string()
    .exactOptional()
    .describe("Where the default comes from, when it is another setting's value"),
  field: zod.string().describe("The template field whose tokens the control offers"),
  isDefault: zod
    .boolean()
    .describe("The setting has no override; Reset to default leaves it unchanged"),
  key: zod.string(),
  kind: zod.enum(["template"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  plain: zod.boolean().describe("The text holds no tokens"),
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  value: zod.string(),
});

export type SettingTemplateField = zod.input<typeof SettingTemplateField>;
export type SettingTemplateFieldOutput = zod.output<typeof SettingTemplateField>;

export const settingListFieldMinMin = 0;

export const SettingListField = zod.object({
  applies: zod.enum(["live", "restart"]),
  default: zod.array(zod.string()),
  defaultFrom: zod
    .string()
    .exactOptional()
    .describe("Where the default comes from, when it is another setting's value"),
  isDefault: zod
    .boolean()
    .describe("The setting has no override; Reset to default leaves it unchanged"),
  item: zod
    .enum(["extension"])
    .describe(
      "What each entry is: an extension is lowercase letters and digits with no leading dot, such as mkv",
    ),
  key: zod.string(),
  kind: zod.enum(["list"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  min: zod.int().min(settingListFieldMinMin).describe("The fewest entries the list takes"),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  running: zod.array(zod.string()),
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  value: zod.array(zod.string()),
});

export type SettingListField = zod.input<typeof SettingListField>;
export type SettingListFieldOutput = zod.output<typeof SettingListField>;

export const SettingSizeField = zod.object({
  applies: zod.enum(["live", "restart"]),
  default: zod.int().describe("Bytes"),
  defaultFrom: zod
    .string()
    .exactOptional()
    .describe("Where the default comes from, when it is another setting's value"),
  isDefault: zod
    .boolean()
    .describe("The setting has no override; Reset to default leaves it unchanged"),
  key: zod.string(),
  kind: zod.enum(["size"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  max: zod.int().describe("Bytes"),
  min: zod.int().describe("Bytes"),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  running: zod.int().describe("Bytes"),
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  value: zod.int().describe("Bytes"),
});

export type SettingSizeField = zod.input<typeof SettingSizeField>;
export type SettingSizeFieldOutput = zod.output<typeof SettingSizeField>;

export const SettingField = zod.discriminatedUnion("kind", [
  SettingDurationField,
  SettingCountField,
  SettingChoiceField,
  SettingWindowField,
  SettingPortField,
  SettingSwitchField,
  SettingRateField,
  SettingAddressField,
  SettingFolderField,
  SettingSpaceField,
  SettingTemplateField,
  SettingListField,
  SettingSizeField,
]);

export type SettingField = zod.input<typeof SettingField>;
export type SettingFieldOutput = zod.output<typeof SettingField>;

export const ImportsImportingSettings = zod.object({
  settings: zod
    .array(SettingField)
    .describe(
      "The video file extensions, the smallest video file, then the runtime under which a video is a sample",
    ),
});

export type ImportsImportingSettings = zod.input<typeof ImportsImportingSettings>;
export type ImportsImportingSettingsOutput = zod.output<typeof ImportsImportingSettings>;

export const ImportsImportsList = zod.object({
  imports: zod.array(ImportsImportEntry),
});

export type ImportsImportsList = zod.input<typeof ImportsImportsList>;
export type ImportsImportsListOutput = zod.output<typeof ImportsImportsList>;

export const ImportsIntoAction = zod.object({
  action: zod.enum(["into"]),
  files: zod.array(ImportsChosenFile).min(1),
  workId: zod.uuid().describe("A Work in the Import's root"),
});

export type ImportsIntoAction = zod.input<typeof ImportsIntoAction>;
export type ImportsIntoActionOutput = zod.output<typeof ImportsIntoAction>;

export const importsLeadAddedMin = 0;

export const importsLeadContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");
export const importsLeadNeedsYouMin = 0;

export const ImportsLead = zod.object({
  added: zod
    .int()
    .min(importsLeadAddedMin)
    .describe("The Works the root's last library scan added"),
  contentType: zod.string().regex(importsLeadContentTypeRegExp),
  needsYou: zod
    .int()
    .min(importsLeadNeedsYouMin)
    .describe("The root's unfinished root Imports that aren't ignored"),
  root: ImportsRootLabel,
});

export type ImportsLead = zod.input<typeof ImportsLead>;
export type ImportsLeadOutput = zod.output<typeof ImportsLead>;

export const ImportsList = zod.object({
  imports: zod
    .array(ImportsImportEntry)
    .describe("The unfinished Imports, in root and folder order"),
  leads: zod
    .array(ImportsLead)
    .describe(
      "One per root a library scan compared, in root order; empty when source leaves out root",
    ),
  next: zod.uuid().nullable().describe("Pass it as after for the next page; null on the last page"),
});

export type ImportsList = zod.input<typeof ImportsList>;
export type ImportsListOutput = zod.output<typeof ImportsList>;

export const ImportsMatchAction = zod.object({
  action: zod.enum(["match"]),
  key: zod
    .string()
    .min(1)
    .describe(
      "A record's key at the Content Type's source, as searchCatalog lists it; it becomes the suggestion",
    ),
});

export type ImportsMatchAction = zod.input<typeof ImportsMatchAction>;
export type ImportsMatchActionOutput = zod.output<typeof ImportsMatchAction>;

export const ImportsRootAction = zod.discriminatedUnion("action", [
  ImportsMatchAction,
  ImportsIntoAction,
]);

export type ImportsRootAction = zod.input<typeof ImportsRootAction>;
export type ImportsRootActionOutput = zod.output<typeof ImportsRootAction>;

export const ImportsSettings = zod.object({
  settings: zod
    .array(SettingField)
    .describe("The stop share, then the files each Node probes at once"),
});

export type ImportsSettings = zod.input<typeof ImportsSettings>;
export type ImportsSettingsOutput = zod.output<typeof ImportsSettings>;

export const indexersCaptchaAnswerAnswerMax = 64;

export const indexersCaptchaAnswerAttemptMax = 64;

export const IndexersCaptchaAnswer = zod.object({
  answer: zod.string().max(indexersCaptchaAnswerAnswerMax).describe("The code the captcha shows"),
  attempt: zod
    .string()
    .min(1)
    .max(indexersCaptchaAnswerAttemptMax)
    .describe("The attempt the captcha was loaded with"),
});

export type IndexersCaptchaAnswer = zod.input<typeof IndexersCaptchaAnswer>;
export type IndexersCaptchaAnswerOutput = zod.output<typeof IndexersCaptchaAnswer>;

export const IndexersCaptchaImage = zod.object({
  attempt: zod.string().describe("Send it back with the code the image shows"),
  expiresAt: zod.iso
    .datetime({ offset: true })
    .describe("When the attempt closes. Load a new captcha after it."),
  image: zod.string().describe("The captcha as a data URL"),
});

export type IndexersCaptchaImage = zod.input<typeof IndexersCaptchaImage>;
export type IndexersCaptchaImageOutput = zod.output<typeof IndexersCaptchaImage>;

export const indexersCatalogAddressAddressMax = 2048;

export const IndexersCatalogAddress = zod.object({
  address: zod
    .string()
    .max(indexersCatalogAddressAddressMax)
    .nullable()
    .describe(
      "The base address, to which SlipStream adds the schema version; null resets it to the default",
    ),
});

export type IndexersCatalogAddress = zod.input<typeof IndexersCatalogAddress>;
export type IndexersCatalogAddressOutput = zod.output<typeof IndexersCatalogAddress>;

export const IndexersCatalogChanged = zod.object({
  refreshedAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("Null until a refresh has read the catalog"),
});

export type IndexersCatalogChanged = zod.input<typeof IndexersCatalogChanged>;
export type IndexersCatalogChangedOutput = zod.output<typeof IndexersCatalogChanged>;

export const IndexersCatalogEntry = zod.object({
  categories: zod.array(zod.int()).describe("The Newznab category ids it maps to"),
  description: zod.string().describe("The definition's own words"),
  id: zod.string(),
  language: zod.string().describe("A language tag such as en-US"),
  name: zod.string(),
  origin: zod.enum(["catalog", "custom", "native"]),
  privacy: zod.enum(["private", "semi-private", "public"]),
  reason: zod.string().nullable().describe("Why SlipStream can't run it; null when it can"),
  replacedByCustom: zod
    .boolean()
    .describe("A custom definition has this catalog entry's id and runs in its place"),
  version: zod.iso.datetime({ offset: true }).describe("When SlipStream downloaded this version"),
});

export type IndexersCatalogEntry = zod.input<typeof IndexersCatalogEntry>;
export type IndexersCatalogEntryOutput = zod.output<typeof IndexersCatalogEntry>;

export const indexersChallengeSolverTimeoutMin = 10;
export const indexersChallengeSolverTimeoutMax = 180;

export const IndexersChallengeSolver = zod.object({
  address: zod
    .string()
    .nullable()
    .describe(
      "The address of the service that clears browser challenges for the Node; null when it has none",
    ),
  timeout: zod
    .int()
    .min(indexersChallengeSolverTimeoutMin)
    .max(indexersChallengeSolverTimeoutMax)
    .describe("Seconds the solver may take to clear a page"),
});

export type IndexersChallengeSolver = zod.input<typeof IndexersChallengeSolver>;
export type IndexersChallengeSolverOutput = zod.output<typeof IndexersChallengeSolver>;

export const IndexersChanged = zod.object({
  indexerId: zod.uuid(),
});

export type IndexersChanged = zod.input<typeof IndexersChanged>;
export type IndexersChangedOutput = zod.output<typeof IndexersChanged>;

export const indexersCustomDefinitionYamlMax = 1048576;

export const IndexersCustomDefinition = zod.object({
  yaml: zod
    .string()
    .min(1)
    .max(indexersCustomDefinitionYamlMax)
    .describe("The definition's YAML file"),
});

export type IndexersCustomDefinition = zod.input<typeof IndexersCustomDefinition>;
export type IndexersCustomDefinitionOutput = zod.output<typeof IndexersCustomDefinition>;

export const IndexersOption = zod.object({
  label: zod.string(),
  value: zod.string(),
});

export type IndexersOption = zod.input<typeof IndexersOption>;
export type IndexersOptionOutput = zod.output<typeof IndexersOption>;

export const IndexersDefinitionField = zod.object({
  default: zod.string().nullable().describe("Null when the field has none"),
  help: zod
    .string()
    .nullable()
    .describe("An info field's text, written by the definition's authors, or a ported site's help"),
  label: zod.string(),
  max: zod.int().nullable().describe("A number's greatest value; null for none"),
  min: zod.int().nullable().describe("A number's least value; null for none"),
  name: zod.string().describe("May hold hyphens or start with a digit"),
  options: zod.array(IndexersOption).describe("A select's or a multiselect's choices, by value"),
  required: zod
    .boolean()
    .describe(
      "An Indexer can't be saved with the value empty; an edit that sends a sealed one empty keeps the saved value",
    ),
  sealed: zod.boolean().describe("Its saved value is never returned"),
  type: zod.enum([
    "text",
    "password",
    "checkbox",
    "select",
    "number",
    "multiselect",
    "info",
    "info_cookie",
    "info_flaresolverr",
    "info_useragent",
    "info_category_8000",
    "cardigannCaptcha",
  ]),
});

export type IndexersDefinitionField = zod.input<typeof IndexersDefinitionField>;
export type IndexersDefinitionFieldOutput = zod.output<typeof IndexersDefinitionField>;

export const indexersLimitCountMin = 0;
export const indexersLimitCountMax = 100000;

export const IndexersLimit = zod.object({
  count: zod.int().min(indexersLimitCountMin).max(indexersLimitCountMax).describe("0 is no limit"),
  per: zod.enum(["hour", "day"]).describe("The sliding window it counts in"),
});

export type IndexersLimit = zod.input<typeof IndexersLimit>;
export type IndexersLimitOutput = zod.output<typeof IndexersLimit>;

export const IndexersPortFacts = zod.object({
  grabs: IndexersLimit.describe("A new Indexer's grab limit unless the form sets one"),
  recent: zod.boolean().describe("The site lists its new uploads, so RSS sync can use it"),
  searches: IndexersLimit.describe("A new Indexer's search limit unless the form sets one"),
  spacingSeconds: zod.number().describe("The least time between two requests to the site"),
  synced: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When the Prowlarr commit SlipStream's support for the site was synced to was made"),
  userAgent: zod
    .boolean()
    .describe("The pasted cookie works only with the User-Agent of the browser it came from"),
});

export type IndexersPortFacts = zod.input<typeof IndexersPortFacts>;
export type IndexersPortFactsOutput = zod.output<typeof IndexersPortFacts>;

export const IndexersDefinitionForm = zod.object({
  captcha: zod.boolean().describe("Signing in asks for an image captcha"),
  code: zod.boolean().describe("Signing in may ask for a two-factor code"),
  cookie: zod
    .boolean()
    .describe(
      "Its Indexers sign in with a browser cookie and its User-Agent, pasted as the cookie member, in place of a cookie field",
    ),
  entry: IndexersCatalogEntry,
  fields: zod
    .array(IndexersDefinitionField)
    .describe("In the definition's order, the captcha last"),
  links: zod.array(zod.string()).describe("The site's addresses, the first being the default"),
  port: zod
    .union([IndexersPortFacts, zod.null()])
    .describe("What an included site's form says beyond a definition's; null for a definition"),
  session: zod.boolean().describe("Its Indexers sign in and keep a session on their Node"),
  solver: zod
    .boolean()
    .describe("The site sits behind a browser challenge that a Challenge solver can clear"),
});

export type IndexersDefinitionForm = zod.input<typeof IndexersDefinitionForm>;
export type IndexersDefinitionFormOutput = zod.output<typeof IndexersDefinitionForm>;

export const IndexersFailure = zod.object({
  at: zod.iso.datetime({ offset: true }),
  text: zod.string(),
});

export type IndexersFailure = zod.input<typeof IndexersFailure>;
export type IndexersFailureOutput = zod.output<typeof IndexersFailure>;

export const IndexersPin = zod.object({
  baseAddress: zod.string(),
  fetchedAt: zod.iso
    .datetime({ offset: true })
    .describe("When SlipStream downloaded the pinned version"),
  id: zod.string(),
  sealed: zod
    .array(zod.string())
    .describe("The names of the sealed settings that hold a value, in order"),
  values: zod
    .record(zod.string(), zod.string())
    .describe("The settings saved in plain, by name. Sealed settings are never returned"),
  version: zod.string().describe("The pinned version's git blob sha"),
});

export type IndexersPin = zod.input<typeof IndexersPin>;
export type IndexersPinOutput = zod.output<typeof IndexersPin>;

export const indexersLimitUseCountMin = 0;

export const indexersLimitUseUsedMin = 0;

export const IndexersLimitUse = zod.object({
  count: zod.int().min(indexersLimitUseCountMin).describe("0 is no limit"),
  freeAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When the oldest counted call leaves the window; null while under the limit"),
  per: zod.enum(["hour", "day"]),
  used: zod.int().min(indexersLimitUseUsedMin).describe("The calls counted in the window now"),
});

export type IndexersLimitUse = zod.input<typeof IndexersLimitUse>;
export type IndexersLimitUseOutput = zod.output<typeof IndexersLimitUse>;

export const IndexersPortPin = zod.object({
  baseAddress: zod.string(),
  id: zod.string(),
  name: zod.string().describe("The site's name"),
  sealed: zod
    .array(zod.string())
    .describe("The names of the sealed settings that hold a value, in order"),
  synced: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe(
      "When the Prowlarr commit the port was synced to was made; null once SlipStream no longer includes the site",
    ),
  values: zod
    .record(zod.string(), zod.string())
    .describe("The settings saved in plain, by name. Sealed settings are never returned"),
});

export type IndexersPortPin = zod.input<typeof IndexersPortPin>;
export type IndexersPortPinOutput = zod.output<typeof IndexersPortPin>;

export const IndexersSearchMode = zod.object({
  mode: zod.string().describe("The Torznab t= value"),
  params: zod.array(zod.string()).describe("In the Indexer's order"),
});

export type IndexersSearchMode = zod.input<typeof IndexersSearchMode>;
export type IndexersSearchModeOutput = zod.output<typeof IndexersSearchMode>;

export const IndexersSession = zod.object({
  earnedAt: zod.iso.datetime({ offset: true }),
  nodeId: zod.uuid(),
  pasted: zod.boolean().describe("Seeded from a browser cookie an Admin pasted"),
});

export type IndexersSession = zod.input<typeof IndexersSession>;
export type IndexersSessionOutput = zod.output<typeof IndexersSession>;

export const indexersStatusRungMin = 0;

export const IndexersStatus = zod.object({
  cause: zod
    .union([
      zod.literal("credentials_rejected"),
      zod.literal("captcha_needed"),
      zod.literal("challenge_unsolved"),
      zod.literal("cookie_expired"),
      zod.literal("definition_unloadable"),
      zod.literal("account_refused"),
      zod.literal("request_rejected"),
      zod.literal("code_needed"),
      zod.literal("port_retired"),
      zod.literal(null),
    ])
    .nullable()
    .describe("Null unless stopped or home_only"),
  lastFailure: zod.union([IndexersFailure, zod.null()]),
  nodeId: zod.uuid().nullable().describe("The Node the stop was met on"),
  rung: zod
    .int()
    .min(indexersStatusRungMin)
    .describe("The backoff ladder's position, stepping down one on each success"),
  since: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When it began failing, stopped or began waiting for home"),
  solverTried: zod
    .boolean()
    .describe("A challenge_unsolved that the Node's Challenge solver couldn't clear"),
  state: zod
    .enum(["ok", "limited", "failing", "stopped", "home_only"])
    .describe("home_only: a stop met on a failover Node, which waits for home"),
  successAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("The last answer the site responded to"),
  text: zod.string().nullable().describe("The site's or feed's own words"),
  until: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When a limited or failing Indexer is asked again"),
});

export type IndexersStatus = zod.input<typeof IndexersStatus>;
export type IndexersStatusOutput = zod.output<typeof IndexersStatus>;

export const indexersOutcomeReleasesMin = 0;

export const IndexersOutcome = zod.object({
  cause: zod.string().nullable().describe("Set for stopped"),
  kind: zod.enum([
    "found",
    "no_releases",
    "limited",
    "failing",
    "stopped",
    "waiting",
    "late",
    "limit_reached",
    "no_categories",
    "removed",
  ]),
  lapsed: zod.boolean().describe("Waiting past the offline grace"),
  nodeId: zod
    .uuid()
    .nullable()
    .describe("Set for waiting: the Node waited for; for stopped: the Node the stop was met on"),
  releases: zod.int().min(indexersOutcomeReleasesMin),
  since: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe(
      "Set for waiting: since when the home Node is away; null when it holds no Role to search with",
    ),
  text: zod.string().nullable().describe("Set for stopped: the site's own words"),
  until: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("Set for limited, failing and limit_reached"),
});

export type IndexersOutcome = zod.input<typeof IndexersOutcome>;
export type IndexersOutcomeOutput = zod.output<typeof IndexersOutcome>;

export const indexersIndexerPriorityMax = 50;

export const IndexersIndexer = zod.object({
  allowFailover: zod.boolean().describe("Read only for an Indexer that signs in"),
  allowNoSize: zod.boolean().describe("An RSS feed whose items may have no size"),
  automaticSearch: zod.boolean(),
  categories: zod
    .array(zod.int())
    .nullable()
    .describe("Newznab category ids; null for every category it advertises"),
  createdAt: zod.iso.datetime({ offset: true }),
  definition: zod
    .union([IndexersPin, zod.null()])
    .describe("A catalog Indexer's definition; null for every other kind"),
  feedHost: zod
    .string()
    .nullable()
    .describe("The host of a feed's address, which is never returned"),
  fix: zod
    .string()
    .nullable()
    .describe("What the Admin can do about its status, in words; null when nothing"),
  grabs: IndexersLimitUse,
  homeNodeId: zod.uuid().describe("The Node it searches from"),
  id: zod.uuid(),
  kind: zod.enum(["catalog", "torznab", "rss", "native"]),
  media: zod
    .array(zod.string())
    .nullable()
    .describe("The Content Types an RSS feed carries; null for every other kind"),
  name: zod.string(),
  notes: zod
    .array(zod.string())
    .describe(
      "What changed when SlipStream last moved it to a new definition version or from a ported site to its catalog definition, for the Indexer page",
    ),
  on: zod.boolean(),
  port: zod
    .union([IndexersPortPin, zod.null()])
    .describe("A native Indexer's ported site; null for every other kind"),
  priority: zod.int().min(1).max(indexersIndexerPriorityMax).describe("Lower wins a tie"),
  revision: zod.int().min(1),
  rss: zod.boolean(),
  searchModes: zod.array(IndexersSearchMode).describe("By mode"),
  searches: IndexersLimitUse,
  sentence: zod.string().describe("Its status in words"),
  session: zod
    .union([IndexersSession, zod.null()])
    .describe("Its newest session on any Node, null for none"),
  signsIn: zod
    .boolean()
    .describe(
      "It holds a login session, so it waits for its home rather than failing over unless allowFailover",
    ),
  status: IndexersStatus,
  trackerIds: zod.array(zod.string()).describe("The Trackers it reaches"),
  waits: zod
    .union([IndexersOutcome, zod.null()])
    .describe("Why its searches wait now, null when they don't"),
});

export type IndexersIndexer = zod.input<typeof IndexersIndexer>;
export type IndexersIndexerOutput = zod.output<typeof IndexersIndexer>;

export const IndexersIndexerCatalog = zod.object({
  address: zod
    .string()
    .describe("The catalog's base address, to which SlipStream adds the schema version"),
  addressIsDefault: zod.boolean(),
  entries: zod
    .array(IndexersCatalogEntry)
    .describe("The catalog's definitions by name, then the custom definitions by id"),
  nextRefreshAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When the Indexer catalog refresh runs next; null while it runs or waits"),
  notice: zod.string().nullable().describe("Why the catalog lists no site yet; null once it does"),
  refreshedAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("Null until a refresh has downloaded the catalog"),
});

export type IndexersIndexerCatalog = zod.input<typeof IndexersIndexerCatalog>;
export type IndexersIndexerCatalogOutput = zod.output<typeof IndexersIndexerCatalog>;

export const indexersPastedCookieCookieMax = 8192;

export const indexersPastedCookieUserAgentMax = 512;

export const IndexersPastedCookie = zod.object({
  cookie: zod
    .string()
    .min(1)
    .max(indexersPastedCookieCookieMax)
    .describe(
      "The Cookie header's value from a signed-in browser: name=value pairs separated by semicolons",
    ),
  userAgent: zod
    .string()
    .max(indexersPastedCookieUserAgentMax)
    .exactOptional()
    .describe("The User-Agent of that browser, sent with the cookie"),
});

export type IndexersPastedCookie = zod.input<typeof IndexersPastedCookie>;
export type IndexersPastedCookieOutput = zod.output<typeof IndexersPastedCookie>;

export const indexersIndexerEditAddressMax = 2048;

export const indexersIndexerEditAddressRegExp = new RegExp(
  "^https?://[^/?#]+(/[^?#]*)?(\\?[^#]*)?$",
);
export const indexersIndexerEditApiKeyMax = 256;

export const indexersIndexerEditBaseAddressMax = 2048;

export const indexersIndexerEditCodeMax = 16;

export const indexersIndexerEditNameMax = 100;

export const indexersIndexerEditPriorityMax = 50;

export const IndexersIndexerEdit = zod.object({
  address: zod
    .string()
    .max(indexersIndexerEditAddressMax)
    .regex(indexersIndexerEditAddressRegExp)
    .exactOptional()
    .describe("Write-only"),
  allowFailover: zod.boolean().nullish().describe("Null resets it to off"),
  allowNoSize: zod.boolean().exactOptional().describe("An RSS feed's"),
  apiKey: zod.string().max(indexersIndexerEditApiKeyMax).exactOptional().describe("Write-only"),
  automaticSearch: zod.boolean().nullish().describe("Null resets it to on"),
  baseAddress: zod
    .string()
    .max(indexersIndexerEditBaseAddressMax)
    .exactOptional()
    .describe(
      "A catalog Indexer's: one of its definition's addresses. A native Indexer's: one its port knows or a typed http or https one",
    ),
  captcha: IndexersCaptchaAnswer.exactOptional().describe(
    "The answer to a captcha loaded for this Indexer, for the Test the change runs",
  ),
  categories: zod.array(zod.int()).nullish().describe("Null for every category it advertises"),
  code: zod
    .string()
    .max(indexersIndexerEditCodeMax)
    .exactOptional()
    .describe("The current two-factor code, for the Test the change runs"),
  cookie: IndexersPastedCookie.exactOptional().describe("A new browser cookie. Write-only"),
  grabs: zod
    .union([IndexersLimit, zod.null()])
    .exactOptional()
    .describe("Null turns the limit off"),
  homeNodeId: zod.uuid().exactOptional().describe("A Node that holds Searching"),
  media: zod.array(zod.string()).exactOptional().describe("An RSS feed's Content Types"),
  name: zod.string().min(1).max(indexersIndexerEditNameMax).exactOptional(),
  on: zod.boolean().nullish().describe("Null resets it to on"),
  priority: zod
    .int()
    .min(1)
    .max(indexersIndexerEditPriorityMax)
    .nullish()
    .describe("Null resets it to 25"),
  rss: zod.boolean().nullish().describe("Null resets it to on"),
  searches: zod
    .union([IndexersLimit, zod.null()])
    .exactOptional()
    .describe("Null turns the limit off"),
  values: zod
    .record(zod.string(), zod.string())
    .exactOptional()
    .describe(
      "A catalog or native Indexer's settings to change, by name; a sealed one sent empty keeps its saved value",
    ),
});

export type IndexersIndexerEdit = zod.input<typeof IndexersIndexerEdit>;
export type IndexersIndexerEditOutput = zod.output<typeof IndexersIndexerEdit>;

export const IndexersIndexerList = zod.object({
  indexers: zod.array(IndexersIndexer).describe("By name"),
});

export type IndexersIndexerList = zod.input<typeof IndexersIndexerList>;
export type IndexersIndexerListOutput = zod.output<typeof IndexersIndexerList>;

export const indexersNewIndexerAddressMax = 2048;

export const indexersNewIndexerAddressRegExp = new RegExp(
  "^https?://[^/?#]+(/[^?#]*)?(\\?[^#]*)?$",
);
export const indexersNewIndexerApiKeyMax = 256;

export const indexersNewIndexerBaseAddressMax = 2048;

export const indexersNewIndexerCodeMax = 16;

export const indexersNewIndexerDefinitionMax = 200;

export const indexersNewIndexerNameMax = 100;

export const indexersNewIndexerPortMax = 200;

export const indexersNewIndexerPriorityDefault = 25;
export const indexersNewIndexerPriorityMax = 50;

export const indexersNewIndexerTrackerNameMax = 100;

export const IndexersNewIndexer = zod.object({
  address: zod
    .string()
    .max(indexersNewIndexerAddressMax)
    .regex(indexersNewIndexerAddressRegExp)
    .exactOptional()
    .describe(
      "A Torznab feed's base, to which SlipStream adds /api, or an RSS feed's whole address; required for both. Write-only",
    ),
  allowFailover: zod.boolean().exactOptional(),
  allowNoSize: zod.boolean().exactOptional().describe("Use an RSS feed whose items have no size"),
  apiKey: zod
    .string()
    .max(indexersNewIndexerApiKeyMax)
    .exactOptional()
    .describe("A Torznab feed's. Write-only"),
  automaticSearch: zod.boolean().exactOptional().describe("Absent for on"),
  baseAddress: zod
    .string()
    .max(indexersNewIndexerBaseAddressMax)
    .exactOptional()
    .describe(
      "One of the definition's addresses, absent for its first; or a native Indexer's address, one the port knows or a typed http or https one",
    ),
  captcha: IndexersCaptchaAnswer.exactOptional().describe(
    "The answer to a captcha loaded for this draft",
  ),
  categories: zod
    .array(zod.int())
    .nullish()
    .describe("Newznab category ids; absent for every category it advertises"),
  code: zod
    .string()
    .max(indexersNewIndexerCodeMax)
    .exactOptional()
    .describe("The current two-factor code, for a site whose sign-in asks for one"),
  cookie: IndexersPastedCookie.exactOptional().describe(
    "A browser cookie for an RSS feed that needs one, or for a catalog or native Indexer whose site signs in with one. Write-only",
  ),
  definition: zod
    .string()
    .max(indexersNewIndexerDefinitionMax)
    .exactOptional()
    .describe("A catalog Indexer's definition id; required for one"),
  grabs: IndexersLimit.exactOptional(),
  homeNodeId: zod
    .uuid()
    .exactOptional()
    .describe("A Node that holds Searching; absent for the default home"),
  kind: zod.enum(["catalog", "torznab", "rss", "native"]),
  media: zod
    .array(zod.string())
    .nullish()
    .describe(
      "The Content Types an RSS feed carries, in place of categories; required for an RSS feed",
    ),
  name: zod.string().min(1).max(indexersNewIndexerNameMax),
  on: zod.boolean().exactOptional().describe("Absent for on"),
  port: zod
    .string()
    .max(indexersNewIndexerPortMax)
    .exactOptional()
    .describe("A native Indexer's port id; required for one"),
  priority: zod
    .int()
    .min(1)
    .max(indexersNewIndexerPriorityMax)
    .default(indexersNewIndexerPriorityDefault),
  rss: zod.boolean().exactOptional().describe("Absent for on"),
  searches: IndexersLimit.exactOptional(),
  trackerName: zod
    .string()
    .max(indexersNewIndexerTrackerNameMax)
    .exactOptional()
    .describe(
      "The name of the Tracker the feed reaches when its items name no source; required for an RSS feed, absent for a Torznab feed's Indexer name",
    ),
  values: zod
    .record(zod.string(), zod.string())
    .exactOptional()
    .describe(
      "A catalog or native Indexer's settings by name, a checkbox's as true or false and a multiselect's as comma-separated option values; a setting left out takes its default. Sealed ones are write-only",
    ),
});

export type IndexersNewIndexer = zod.input<typeof IndexersNewIndexer>;
export type IndexersNewIndexerOutput = zod.output<typeof IndexersNewIndexer>;

export const indexersQueryReleasesMin = 0;

export const IndexersQuery = zod.object({
  completedAt: zod.iso.datetime({ offset: true }).nullable().describe("Null while it runs"),
  id: zod.uuid(),
  outcome: zod
    .union([
      zod.literal("ok"),
      zod.literal("limited"),
      zod.literal("failing"),
      zod.literal("stopped"),
      zod.literal("lost"),
      zod.literal(null),
    ])
    .nullable()
    .describe("Null while it runs; lost when its Node never answered"),
  releases: zod.int().min(indexersQueryReleasesMin),
  reservedAt: zod.iso.datetime({ offset: true }),
  startedAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When its Node sent its first request; null before, or when it sent none"),
  use: zod.enum(["search", "recent", "fetch", "test"]),
});

export type IndexersQuery = zod.input<typeof IndexersQuery>;
export type IndexersQueryOutput = zod.output<typeof IndexersQuery>;

export const IndexersQueryList = zod.object({
  nextOffset: zod.int().nullable().describe("The offset of the next page; null on the last"),
  queries: zod.array(IndexersQuery).describe("Newest first"),
});

export type IndexersQueryList = zod.input<typeof IndexersQueryList>;
export type IndexersQueryListOutput = zod.output<typeof IndexersQueryList>;

export const indexersReportItemsMin = 0;

export const IndexersReport = zod.object({
  items: zod.int().min(indexersReportItemsMin).describe("The newest items it listed"),
  searchModes: zod.array(IndexersSearchMode).describe("By mode"),
  sources: zod
    .array(zod.string())
    .describe(
      "The different sources a Torznab feed's items named: more than one is an aggregate feed",
    ),
});

export type IndexersReport = zod.input<typeof IndexersReport>;
export type IndexersReportOutput = zod.output<typeof IndexersReport>;

export const IndexersSettings = zod.object({
  settings: zod
    .array(SettingField)
    .describe(
      "How long a search waits for each Indexer, and how long every request to an Indexer may fail before Health warns",
    ),
});

export type IndexersSettings = zod.input<typeof IndexersSettings>;
export type IndexersSettingsOutput = zod.output<typeof IndexersSettings>;

export const IndexersSettingsChange = zod.object({
  values: zod
    .record(zod.string(), zod.unknown())
    .describe("Each setting's new value by key, all or none; null resets one to its default"),
});

export type IndexersSettingsChange = zod.input<typeof IndexersSettingsChange>;
export type IndexersSettingsChangeOutput = zod.output<typeof IndexersSettingsChange>;

export const IndexersSettingsChanged = zod.object({});

export type IndexersSettingsChanged = zod.input<typeof IndexersSettingsChanged>;
export type IndexersSettingsChangedOutput = zod.output<typeof IndexersSettingsChanged>;

export const indexersSolverChangeAddressMax = 2048;

export const IndexersSolverChange = zod.object({
  address: zod
    .string()
    .max(indexersSolverChangeAddressMax)
    .describe(
      "An http or https address of a service that speaks FlareSolverr's API; empty removes the solver",
    ),
  timeout: zod.int().exactOptional().describe("Seconds from 10 to 180; null or left out is 60"),
});

export type IndexersSolverChange = zod.input<typeof IndexersSolverChange>;
export type IndexersSolverChangeOutput = zod.output<typeof IndexersSolverChange>;

export const IndexersStatusChanged = zod.object({
  indexerId: zod.uuid(),
});

export type IndexersStatusChanged = zod.input<typeof IndexersStatusChanged>;
export type IndexersStatusChangedOutput = zod.output<typeof IndexersStatusChanged>;

export const indexersTestBodyCodeMax = 16;

export const IndexersTestBody = zod.object({
  captcha: IndexersCaptchaAnswer.exactOptional().describe(
    "The answer to a captcha loaded for this Indexer",
  ),
  code: zod
    .string()
    .max(indexersTestBodyCodeMax)
    .exactOptional()
    .describe("The current two-factor code, for a site whose sign-in asks for one"),
});

export type IndexersTestBody = zod.input<typeof IndexersTestBody>;
export type IndexersTestBodyOutput = zod.output<typeof IndexersTestBody>;

export const IndexersTrackerChanged = zod.object({
  trackerId: zod.uuid(),
});

export type IndexersTrackerChanged = zod.input<typeof IndexersTrackerChanged>;
export type IndexersTrackerChangedOutput = zod.output<typeof IndexersTrackerChanged>;

export const joinInputBodyDisplayNameMax = 100;

export const joinInputBodyPinRegExp = new RegExp("^[0-9]{4}$");

export const JoinInputBody = zod.object({
  displayName: zod.string().min(1).max(joinInputBodyDisplayNameMax),
  pin: zod.string().regex(joinInputBodyPinRegExp),
});

export type JoinInputBody = zod.input<typeof JoinInputBody>;
export type JoinInputBodyOutput = zod.output<typeof JoinInputBody>;

export const LibraryScanSettingsInputBody = zod.object({
  values: zod
    .record(zod.string(), zod.unknown())
    .describe("Each setting's new value by key, all or none; null resets one to its default"),
});

export type LibraryScanSettingsInputBody = zod.input<typeof LibraryScanSettingsInputBody>;
export type LibraryScanSettingsInputBodyOutput = zod.output<typeof LibraryScanSettingsInputBody>;

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

export const MergeTrackerInputBody = zod.object({
  into: zod.uuid().describe("The Tracker it merges into, which is kept"),
});

export type MergeTrackerInputBody = zod.input<typeof MergeTrackerInputBody>;
export type MergeTrackerInputBodyOutput = zod.output<typeof MergeTrackerInputBody>;

export const MetadataAttribution = zod.object({
  logo: zod
    .union([zod.literal("tmdb"), zod.literal(null)])
    .nullable()
    .describe(
      "The logo the web app bundles to show beside it; null when it bundles none, so the text shows alone",
    ),
  text: zod.string().describe("The sentence to show, word for word"),
});

export type MetadataAttribution = zod.input<typeof MetadataAttribution>;
export type MetadataAttributionOutput = zod.output<typeof MetadataAttribution>;

export const MetadataChanged = zod.object({
  provider: zod.string().describe("The provider whose status changed"),
});

export type MetadataChanged = zod.input<typeof MetadataChanged>;
export type MetadataChangedOutput = zod.output<typeof MetadataChanged>;

export const MetadataLastError = zod.object({
  at: zod.iso.datetime({ offset: true }),
  words: zod.string().describe("In plain words, the provider named"),
});

export type MetadataLastError = zod.input<typeof MetadataLastError>;
export type MetadataLastErrorOutput = zod.output<typeof MetadataLastError>;

export const MetadataReach = zod.object({
  at: zod.iso.datetime({ offset: true }),
  nodeId: zod.uuid(),
  reached: zod.boolean().describe("Whether the Node reached the provider on its latest call"),
});

export type MetadataReach = zod.input<typeof MetadataReach>;
export type MetadataReachOutput = zod.output<typeof MetadataReach>;

export const MetadataDetails = zod.object({
  lastError: zod.union([MetadataLastError, zod.null()]),
  lastSuccess: zod.iso.datetime({ offset: true }).nullable(),
  nodes: zod
    .array(MetadataReach)
    .describe(
      "Each holder that called the provider since the Controller started or the key changed, by id",
    ),
});

export type MetadataDetails = zod.input<typeof MetadataDetails>;
export type MetadataDetailsOutput = zod.output<typeof MetadataDetails>;

export const metadataNewKeyKeyMax = 256;

export const metadataNewKeyPinMax = 256;

export const MetadataNewKey = zod.object({
  key: zod.string().min(1).max(metadataNewKeyKeyMax),
  pin: zod
    .string()
    .max(metadataNewKeyPinMax)
    .exactOptional()
    .describe(
      "A subscriber PIN to send beside the key, for a provider that takes one; left out, none",
    ),
});

export type MetadataNewKey = zod.input<typeof MetadataNewKey>;
export type MetadataNewKeyOutput = zod.output<typeof MetadataNewKey>;

export const MetadataProvider = zod.object({
  attribution: zod
    .union([MetadataAttribution, zod.null()])
    .describe(
      "What the provider's terms require SlipStream to show; null when they require nothing",
    ),
  details: MetadataDetails,
  id: zod.string().describe("The provider, such as tmdb"),
  key: zod
    .enum(["none", "bundled", "yours"])
    .describe("Whose key calls carry: none in this build, SlipStream's, or the Admin's"),
  keyPage: zod.url().describe("Where an Admin gets a key, opened in the browser"),
  keySetAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When the Admin set the key in use; null unless it is theirs"),
  label: zod.string(),
  pinSet: zod.boolean().describe("The Admin's key in use carries a PIN"),
  state: zod
    .enum(["unused", "working", "key_refused", "rate_limited", "out_of_allowance", "unreachable"])
    .describe("From the latest outcome of a call or test, unused until one"),
  summary: zod
    .string()
    .describe(
      "How the provider stands and what to do, in plain words: the sentence a page leads with, the rest under Details",
    ),
  takesPin: zod.boolean().describe("Set key offers a subscriber PIN beside the key"),
  until: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When a rate limited provider, or one out of its daily allowance, takes calls again"),
});

export type MetadataProvider = zod.input<typeof MetadataProvider>;
export type MetadataProviderOutput = zod.output<typeof MetadataProvider>;

export const MetadataProviderList = zod.object({
  providers: zod.array(MetadataProvider).describe("In display order"),
});

export type MetadataProviderList = zod.input<typeof MetadataProviderList>;
export type MetadataProviderListOutput = zod.output<typeof MetadataProviderList>;

export const MetadataRefreshAllRun = zod.object({
  lastRunAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When the latest Refresh all of the type ran; null until one has"),
  running: zod.boolean().describe("Whether a run that refreshes every Work of the type is open"),
});

export type MetadataRefreshAllRun = zod.input<typeof MetadataRefreshAllRun>;
export type MetadataRefreshAllRunOutput = zod.output<typeof MetadataRefreshAllRun>;

export const MetadataRefreshStarted = zod.object({
  runId: zod.int().describe("The Metadata refresh run that refreshes every Work of the type"),
});

export type MetadataRefreshStarted = zod.input<typeof MetadataRefreshStarted>;
export type MetadataRefreshStartedOutput = zod.output<typeof MetadataRefreshStarted>;

export const MetadataSettings = zod.object({
  settings: zod
    .array(SettingField)
    .describe("Language and region first, then the cache, OMDb's ratings and the artwork store"),
});

export type MetadataSettings = zod.input<typeof MetadataSettings>;
export type MetadataSettingsOutput = zod.output<typeof MetadataSettings>;

export const MetadataSettingsChange = zod.object({
  values: zod
    .record(zod.string(), zod.unknown())
    .describe(
      "Each setting's new value by key, all or none; null resets one to its default. A new language or region makes every Work due at the next Metadata refresh.",
    ),
});

export type MetadataSettingsChange = zod.input<typeof MetadataSettingsChange>;
export type MetadataSettingsChangeOutput = zod.output<typeof MetadataSettingsChange>;

export const MoveFilesInputBody = zod.object({
  importIds: zod.array(zod.uuid()).min(1),
  on: zod.boolean(),
});

export type MoveFilesInputBody = zod.input<typeof MoveFilesInputBody>;
export type MoveFilesInputBodyOutput = zod.output<typeof MoveFilesInputBody>;

export const movingOutputBodyFilesMin = 0;

export const MovingOutputBody = zod.object({
  files: zod.int().min(movingOutputBodyFilesMin),
});

export type MovingOutputBody = zod.input<typeof MovingOutputBody>;
export type MovingOutputBodyOutput = zod.output<typeof MovingOutputBody>;

export const namingChangedContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const NamingChanged = zod.object({
  contentType: zod.string().regex(namingChangedContentTypeRegExp),
});

export type NamingChanged = zod.input<typeof NamingChanged>;
export type NamingChangedOutput = zod.output<typeof NamingChanged>;

export const NamingSamplePath = zod.object({
  label: zod.string().describe("The sample, and for a file template the file it names"),
  path: zod.string().describe('The path in the root, with "/" between folders'),
});

export type NamingSamplePath = zod.input<typeof NamingSamplePath>;
export type NamingSamplePathOutput = zod.output<typeof NamingSamplePath>;

export const NamingFieldPreview = zod.object({
  key: zod.string(),
  problems: zod
    .array(zod.string())
    .describe("What the save check refuses in the setting, each a sentence; empty when it passes"),
  samples: zod
    .array(NamingSamplePath)
    .describe(
      "The full paths a folder or file template names for the type's samples, with renaming on; empty for any other setting and for a template that doesn't parse",
    ),
});

export type NamingFieldPreview = zod.input<typeof NamingFieldPreview>;
export type NamingFieldPreviewOutput = zod.output<typeof NamingFieldPreview>;

export const NamingToken = zod.object({
  about: zod.string().describe("What the token prints, as a sentence"),
  aliases: zod.array(zod.string()).describe("Other names that match the same token"),
  example: zod.string().describe("A value the token prints"),
  group: zod
    .enum(["work", "level", "item", "target", "release", "media"])
    .describe("The token builder's heading"),
  kind: zod.enum(["title", "year", "number", "date", "text", "id", "list"]),
  modifier: zod
    .union([
      zod.literal("pad"),
      zod.literal("cut"),
      zod.literal("languages"),
      zod.literal("formats"),
      zod.literal("format"),
      zod.literal(null),
    ])
    .nullable()
    .describe("What it takes after a colon; null when it takes nothing"),
  name: zod.string(),
  scope: zod.enum(["work", "level", "item", "file"]).describe("The tier the value comes from"),
  tags: zod
    .array(zod.string())
    .describe("The words of the tags that may wrap it, as in {imdb-{ImdbId}}"),
});

export type NamingToken = zod.input<typeof NamingToken>;
export type NamingTokenOutput = zod.output<typeof NamingToken>;

export const NamingFieldTokens = zod.object({
  key: zod.string().describe("The template setting's key"),
  tokens: zod
    .array(NamingToken)
    .describe("Every token the template may name, the Content Type's own first"),
});

export type NamingFieldTokens = zod.input<typeof NamingFieldTokens>;
export type NamingFieldTokensOutput = zod.output<typeof NamingFieldTokens>;

export const NamingProblem = zod.object({
  key: zod.string().describe("The setting key, such as <type>.naming.file.standard"),
  sentence: zod.string().describe("What is wrong, then what passes"),
});

export type NamingProblem = zod.input<typeof NamingProblem>;
export type NamingProblemOutput = zod.output<typeof NamingProblem>;

export const namingTypeNamingContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const NamingTypeNaming = zod.object({
  contentType: zod.string().regex(namingTypeNamingContentTypeRegExp),
  problems: zod
    .array(NamingProblem)
    .describe(
      "What the save check refuses in the stored settings once renaming is on; empty when every file of a Work shows as one item on Plex, Jellyfin and Emby",
    ),
  settings: zod
    .array(SettingField)
    .describe(
      "The rename switch, the folder templates outermost first, the file templates, the colon mode and its text, and the multi-episode style when the type's file templates number its Items",
    ),
});

export type NamingTypeNaming = zod.input<typeof NamingTypeNaming>;
export type NamingTypeNamingOutput = zod.output<typeof NamingTypeNaming>;

export const NamingNamingList = zod.object({
  contentTypes: zod.array(NamingTypeNaming).describe("In id order"),
});

export type NamingNamingList = zod.input<typeof NamingNamingList>;
export type NamingNamingListOutput = zod.output<typeof NamingNamingList>;

export const NamingUnplaced = zod.object({
  code: zod
    .enum(["no_number", "no_such_item", "no_rule"])
    .describe("Why the Content Type can't tell which Item the name holds"),
  names: zod
    .string()
    .exactOptional()
    .describe("For no_such_item: the Item the name names, such as S03E14"),
});

export type NamingUnplaced = zod.input<typeof NamingUnplaced>;
export type NamingUnplacedOutput = zod.output<typeof NamingUnplaced>;

export const NamingTestedName = zod.object({
  detail: zod
    .string()
    .exactOptional()
    .describe("Why it would go nowhere, then what to do, for any other reason"),
  name: zod.string(),
  path: zod
    .string()
    .describe(
      "Where a file of that name would go if a library scan found it now, relative to the root; empty when it would go nowhere",
    ),
  unplaced: NamingUnplaced.exactOptional().describe(
    "Why it would go nowhere, when no Item of the Work holds the name",
  ),
});

export type NamingTestedName = zod.input<typeof NamingTestedName>;
export type NamingTestedNameOutput = zod.output<typeof NamingTestedName>;

export const NamingWorkFile = zod.object({
  detail: zod.string().exactOptional().describe("Why its name doesn't fit, then what to do"),
  from: zod.string().describe("Where the file is now, relative to its root"),
  to: zod.string().describe("Where the draft puts it; empty when its name doesn't fit"),
});

export type NamingWorkFile = zod.input<typeof NamingWorkFile>;
export type NamingWorkFileOutput = zod.output<typeof NamingWorkFile>;

export const NamingPreview = zod.object({
  fields: zod.array(NamingFieldPreview).describe("Every setting, in the settings' order"),
  tested: zod
    .array(NamingTestedName)
    .describe("Each of names, in order; empty while a setting has a problem"),
  work: zod
    .array(NamingWorkFile)
    .describe(
      "Each live file of workId's Work, in the order it was added; empty without workId and while a setting has a problem",
    ),
});

export type NamingPreview = zod.input<typeof NamingPreview>;
export type NamingPreviewOutput = zod.output<typeof NamingPreview>;

export const NamingTestName = zod.object({
  name: zod.string().min(1).describe("A file name, or a path inside the Work's folder"),
  workId: zod
    .uuid()
    .describe("The Work the name matches, as POST /releases/parse with kind file reads it"),
});

export type NamingTestName = zod.input<typeof NamingTestName>;
export type NamingTestNameOutput = zod.output<typeof NamingTestName>;

export const namingPreviewInputBodyNamesMax = 100;

export const NamingPreviewInputBody = zod.object({
  draft: zod
    .record(zod.string(), zod.unknown())
    .describe(
      "Values over the stored settings by key, as a save would send them; null reads a setting's default",
    ),
  names: zod
    .array(NamingTestName)
    .max(namingPreviewInputBodyNamesMax)
    .nullish()
    .describe("File names to name under the draft, each in the Work it matches"),
  workId: zod
    .uuid()
    .exactOptional()
    .describe(
      "A Work whose files to name under the draft. A Work of another Content Type is named under its own stored settings.",
    ),
});

export type NamingPreviewInputBody = zod.input<typeof NamingPreviewInputBody>;
export type NamingPreviewInputBodyOutput = zod.output<typeof NamingPreviewInputBody>;

export const NamingTokenList = zod.object({
  fields: zod.array(NamingFieldTokens).describe("Each template setting, in the settings' order"),
});

export type NamingTokenList = zod.input<typeof NamingTokenList>;
export type NamingTokenListOutput = zod.output<typeof NamingTokenList>;

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

export const nodesHeldEndlessUnitsMin = 0;

export const NodesHeld = zod.object({
  bytes: zod.int().describe("0 when sizes don't apply"),
  endlessUnits: zod
    .int()
    .min(nodesHeldEndlessUnitsMin)
    .describe(
      "How many units the Node can't finish on its own, which stay until they are removed, moved or written off",
    ),
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
    "keeps_config",
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
  done: zod.int().describe("Bytes downloaded, live while an Admin listens"),
  peers: zod.int().describe("0 while its Node isn't sampled"),
  title: zod.string().describe("Its newest Grab's release, or its name once no Grab is left"),
  torrentId: zod.uuid(),
  wanted: zod.int().describe("Bytes the torrent's selected files hold"),
});

export type NodesTorrentView = zod.input<typeof NodesTorrentView>;
export type NodesTorrentViewOutput = zod.output<typeof NodesTorrentView>;

export const NodesDownloadingResource = zod.object({
  torrents: zod.array(NodesTorrentView).describe("Each Torrent the client keeps, in added order"),
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
  abBps: zod
    .int()
    .describe(
      "Bytes a second from a to b over the link and their Transfer lanes, TLS records included",
    ),
  b: zod.uuid().describe("The Worker"),
  baBps: zod
    .int()
    .describe(
      "Bytes a second from b to a over the link and their Transfer lanes, TLS records included",
    ),
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
  peers: zod.int(),
  torrentId: zod.uuid(),
  wanted: zod.int(),
});

export type NodesTorrentTraffic = zod.input<typeof NodesTorrentTraffic>;
export type NodesTorrentTrafficOutput = zod.output<typeof NodesTorrentTraffic>;

export const NodesNodeTraffic = zod.object({
  downBps: zod.int().describe("Payload bytes a second its torrents download"),
  id: zod.uuid(),
  torrents: zod
    .array(NodesTorrentTraffic)
    .describe("Each Torrent its client sampled moving since an Admin began listening, in id order"),
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
  self: zod.boolean().describe("The Node that responded, which is the Controller"),
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
    .describe("Every Node whose torrent client is sampled, in id order"),
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

export const PlanInputBody = zod.object({
  draft: FilesRenameDraft.exactOptional().describe(
    "Unsaved naming settings to plan under instead of the stored ones",
  ),
  scope: FilesRenameScope,
});

export type PlanInputBody = zod.input<typeof PlanInputBody>;
export type PlanInputBodyOutput = zod.output<typeof PlanInputBody>;

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
      "The missing file count the Admin confirmed from a rehome_missing response: those files are forgotten, so their cells read missing and wanted. A different count is refused with rehome_missing again",
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

export const RenameInputBody = zod.object({
  works: zod.array(FilesConfirmedWork).min(1).describe("Each Work to rename as its plan showed"),
});

export type RenameInputBody = zod.input<typeof RenameInputBody>;
export type RenameInputBodyOutput = zod.output<typeof RenameInputBody>;

export const renameTrackerInputBodyNameMax = 100;

export const RenameTrackerInputBody = zod.object({
  name: zod.string().min(1).max(renameTrackerInputBodyNameMax),
});

export type RenameTrackerInputBody = zod.input<typeof RenameTrackerInputBody>;
export type RenameTrackerInputBodyOutput = zod.output<typeof RenameTrackerInputBody>;

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
      "The Node is online, so its figures are current. Otherwise each volume's figures are from its sampledAt",
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
      "network_share",
      "outside",
      "node_offline",
    ])
    .describe("Why the folder can't serve its purpose"),
  rootId: zod
    .uuid()
    .exactOptional()
    .describe("The root the folder is inside, contains or carries the marker of"),
});

export type RootsRefusal = zod.input<typeof RootsRefusal>;
export type RootsRefusalOutput = zod.output<typeof RootsRefusal>;

export const rootsRunningDoneMin = 0;

export const rootsRunningTotalMin = 0;

export const RootsRunning = zod.object({
  done: zod.int().min(rootsRunningDoneMin).describe("The files listed, compared or probed so far"),
  phase: zod
    .enum(["listing", "comparing", "probing"])
    .describe(
      "listing: reading the folder, total null; comparing: matching the listed files with the records; probing: reading media info",
    ),
  runId: zod.int().describe("The run Cancel library scan cancels"),
  total: zod
    .int()
    .min(rootsRunningTotalMin)
    .nullable()
    .describe("The files to compare or probe; null while listing"),
});

export type RootsRunning = zod.input<typeof RootsRunning>;
export type RootsRunningOutput = zod.output<typeof RootsRunning>;

export const RootsStopped = zod.object({
  gone: zod.int().min(1).describe("The recorded files the scan didn't find"),
  of: zod.int().min(1).describe("The files the root records"),
});

export type RootsStopped = zod.input<typeof RootsStopped>;
export type RootsStoppedOutput = zod.output<typeof RootsStopped>;

export const RootsScan = zod.object({
  lastAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When the last library scan of its folder finished; null until one has"),
  running: zod
    .union([RootsRunning, zod.null()])
    .describe("Its unit of the open Library scan run while that unit runs; null otherwise"),
  stopped: zod
    .union([RootsStopped, zod.null()])
    .describe(
      "The gone files that stopped that scan, which Mark them gone confirms; null when it didn't stop",
    ),
  summary: zod
    .string()
    .describe("What that scan found, or why it stopped, as a sentence; empty until one finished"),
});

export type RootsScan = zod.input<typeof RootsScan>;
export type RootsScanOutput = zod.output<typeof RootsScan>;

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
  scan: RootsScan.describe("Its library scan at its folder"),
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

export const RootsScanStarted = zod.object({
  runId: zod.int().describe("The Library scan run that scans the root"),
});

export type RootsScanStarted = zod.input<typeof RootsScanStarted>;
export type RootsScanStartedOutput = zod.output<typeof RootsScanStarted>;

export const RootsSuggestions = zod.object({
  suggestions: zod
    .array(zod.string())
    .describe("Up to 25 folders completing the text, in the separators it was typed with"),
});

export type RootsSuggestions = zod.input<typeof RootsSuggestions>;
export type RootsSuggestionsOutput = zod.output<typeof RootsSuggestions>;

export const RootsVerdict = zod.object({
  path: zod.string().describe("The folder as its Node cleaned it"),
  refusal: RootsRefusal.exactOptional().describe(
    "Why it can't serve its purpose; absent when usable",
  ),
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

export const SendToInputBody = zod.object({
  builtInOn: zod
    .uuid()
    .exactOptional()
    .describe("A downloading Node, for its built-in client; send this or clientId"),
  clientId: zod.uuid().exactOptional().describe("An external client; send this or builtInOn"),
});

export type SendToInputBody = zod.input<typeof SendToInputBody>;
export type SendToInputBodyOutput = zod.output<typeof SendToInputBody>;

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

export const SetNamingInputBody = zod.object({
  values: zod
    .record(zod.string(), zod.unknown())
    .describe("Each setting's new value by key, all or none; null resets one to its default"),
});

export type SetNamingInputBody = zod.input<typeof SetNamingInputBody>;
export type SetNamingInputBodyOutput = zod.output<typeof SetNamingInputBody>;

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

export const shellCountsPendingImportsMin = 0;

export const ShellCounts = zod.object({
  activity: zod.int().min(shellCountsActivityMin).describe("The Grabs neither imported nor failed"),
  alerts: ShellAlerts,
  downloads: DownloadclientsAggregate.describe(
    "Every download at once; torrents.live keeps it current",
  ),
  pendingImports: zod
    .int()
    .min(shellCountsPendingImportsMin)
    .describe("The Imports, of any source, with a file waiting for the Admin, ignored ones aside"),
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
  id: zod.enum(["web", "log", "torrent"]),
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

export const TargetsChanged = zod.object({
  workIds: zod.array(zod.string()),
});

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

export const TargetsFormatScore = zod.object({
  formatId: zod.uuid(),
  score: zod.int().describe("Added to a Release's score when it matches the Custom Format"),
});

export type TargetsFormatScore = zod.input<typeof TargetsFormatScore>;
export type TargetsFormatScoreOutput = zod.output<typeof TargetsFormatScore>;

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

export const targetsNewScoresIncrementDefault = 1;

export const TargetsNewScores = zod.object({
  formats: zod.array(TargetsFormatScore).nullish().describe("A Custom Format left out scores 0"),
  increment: zod
    .int()
    .min(1)
    .default(targetsNewScoresIncrementDefault)
    .describe("How far a Release's score must pass the file's to replace it in the same tier"),
  minimum: zod.int().exactOptional().describe("A Release scoring below it isn't grabbed"),
  upgradeUntil: zod
    .int()
    .exactOptional()
    .describe(
      "A file at or past the cutoff upgrades to better-scored Releases until it scores this",
    ),
});

export type TargetsNewScores = zod.input<typeof TargetsNewScores>;
export type TargetsNewScoresOutput = zod.output<typeof TargetsNewScores>;

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

export const targetsNewTargetPropersDefault = `prefer_and_upgrade`;

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
  propers: zod
    .enum(["prefer_and_upgrade", "prefer_no_upgrade", "dont_prefer"])
    .default(targetsNewTargetPropersDefault)
    .describe(
      "prefer_and_upgrade ranks propers and repacks higher and has one replace a file of its Quality even past the cutoff. prefer_no_upgrade only ranks them higher. dont_prefer leaves them to the scores",
    ),
  scores: TargetsNewScores.exactOptional().describe(
    "Left out, every score is 0 and the increment is 1",
  ),
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

export const TargetsScores = zod.object({
  formats: zod
    .array(TargetsFormatScore)
    .describe("Every Custom Format the Target scores; a format left out scores 0"),
  increment: zod
    .int()
    .min(1)
    .describe("How far a Release's score must pass the file's to replace it in the same tier"),
  minimum: zod.int().describe("A Release scoring below it isn't grabbed"),
  upgradeUntil: zod
    .int()
    .describe(
      "A file at or past the cutoff upgrades to better-scored Releases until it scores this",
    ),
});

export type TargetsScores = zod.input<typeof TargetsScores>;
export type TargetsScoresOutput = zod.output<typeof TargetsScores>;

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
  propers: zod
    .enum(["prefer_and_upgrade", "prefer_no_upgrade", "dont_prefer"])
    .describe(
      "prefer_and_upgrade ranks propers and repacks higher and has one replace a file of its Quality even past the cutoff. prefer_no_upgrade only ranks them higher. dont_prefer leaves them to the scores",
    ),
  scores: TargetsScores,
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
  trigger: zod.enum(["schedule", "catch-up", "run-now", "returned", "event"]),
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
  trigger: zod.enum(["schedule", "catch-up", "run-now", "returned", "event"]),
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
  tuning: zod
    .array(SettingField)
    .describe(
      "Settings its owner stores that tune what it does; PATCH sets them under tuning by key",
    ),
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
  trigger: zod.enum(["schedule", "catch-up", "run-now", "returned", "event"]),
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

export const tasksUnitStepDoneMin = 0;

export const tasksUnitStepTotalMin = 0;

export const TasksUnitStep = zod.object({
  done: zod.int().min(tasksUnitStepDoneMin),
  key: zod.string(),
  phase: zod.string(),
  total: zod.int().min(tasksUnitStepTotalMin).nullable(),
});

export type TasksUnitStep = zod.input<typeof TasksUnitStep>;
export type TasksUnitStepOutput = zod.output<typeof TasksUnitStep>;

export const TasksProgressed = zod.object({
  done: zod.int(),
  runId: zod.int(),
  taskId: zod.string(),
  total: zod.int(),
  unit: TasksUnitStep.exactOptional().describe(
    "The unit whose step moved; absent when a unit ended",
  ),
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
  tuning: zod
    .record(zod.string(), zod.unknown())
    .exactOptional()
    .describe(
      "Each tuning setting's new value by key, all or none; null resets one to its default",
    ),
});

export type TasksTaskPatch = zod.input<typeof TasksTaskPatch>;
export type TasksTaskPatchOutput = zod.output<typeof TasksTaskPatch>;

export const TrackersTrackerRef = zod.object({
  id: zod.uuid(),
  name: zod.string(),
});

export type TrackersTrackerRef = zod.input<typeof TrackersTrackerRef>;
export type TrackersTrackerRefOutput = zod.output<typeof TrackersTrackerRef>;

export const TrackersPlannedTorrent = zod.object({
  id: zod.uuid(),
  name: zod.string(),
  tracker: zod
    .union([TrackersTrackerRef, zod.null()])
    .describe("null for a torrent with no known Tracker"),
});

export type TrackersPlannedTorrent = zod.input<typeof TrackersPlannedTorrent>;
export type TrackersPlannedTorrentOutput = zod.output<typeof TrackersPlannedTorrent>;

export const TrackersAct = zod.object({
  action: zod.enum(["pause", "remove", "remove-delete"]),
  torrent: TrackersPlannedTorrent,
});

export type TrackersAct = zod.input<typeof TrackersAct>;
export type TrackersActOutput = zod.output<typeof TrackersAct>;

export const TrackersAttention = zod.object({
  level: zod.enum(["warn", "info"]),
  text: zod.string().describe("One sentence"),
});

export type TrackersAttention = zod.input<typeof TrackersAttention>;
export type TrackersAttentionOutput = zod.output<typeof TrackersAttention>;

export const TrackersChoice = zod.object({
  builtInOn: zod
    .uuid()
    .exactOptional()
    .describe("A Node, for its built-in client; set this or clientId"),
  clientId: zod.uuid().exactOptional().describe("An external client; set this or builtInOn"),
});

export type TrackersChoice = zod.input<typeof TrackersChoice>;
export type TrackersChoiceOutput = zod.output<typeof TrackersChoice>;

export const trackersCountsAtRiskMin = 0;

export const trackersCountsGoalMetMin = 0;

export const trackersCountsOwingMin = 0;

export const trackersCountsSeedingMin = 0;

export const trackersCountsTorrentsMin = 0;

export const TrackersCounts = zod.object({
  atRisk: zod
    .int()
    .min(trackersCountsAtRiskMin)
    .describe("The ones that owe it seeding and aren't seeding"),
  goalMet: zod.int().min(trackersCountsGoalMetMin).describe("The ones whose seeding goal is met"),
  owing: zod.int().min(trackersCountsOwingMin).describe("The ones that still owe it seeding"),
  seeding: zod.int().min(trackersCountsSeedingMin),
  torrents: zod.int().min(trackersCountsTorrentsMin),
});

export type TrackersCounts = zod.input<typeof TrackersCounts>;
export type TrackersCountsOutput = zod.output<typeof TrackersCounts>;

export const TrackersGoal = zod.object({
  action: zod
    .union([
      zod.literal("pause"),
      zod.literal("remove"),
      zod.literal("remove-delete"),
      zod.literal(null),
    ])
    .nullable()
    .describe(
      "What happens to the torrent once the goal is met, where remove-delete also deletes its download copy; null when combine is forever",
    ),
  combine: zod
    .enum(["first", "both", "forever"])
    .describe(
      "first is whichever of the ratio and seeding time is reached first; both needs both of them; forever seeds until an Admin removes the torrent. A goal that ends counts a ratio, a seeding time or both, and with only one of them first and both each wait for it",
    ),
  ratio: zod
    .number()
    .nullable()
    .describe("At most two decimals; null when combine is forever or the goal doesn't count ratio"),
  seedSeconds: zod
    .int()
    .nullable()
    .describe("Seconds; null when combine is forever or the goal doesn't count seeding time"),
});

export type TrackersGoal = zod.input<typeof TrackersGoal>;
export type TrackersGoalOutput = zod.output<typeof TrackersGoal>;

export const TrackersKept = zod.object({
  torrent: TrackersPlannedTorrent,
  why: zod
    .enum(["not-imported", "draining"])
    .describe(
      "not-imported: a Grab on it didn't import, so removing it keeps the download copy; draining: its Node is draining Downloading, so it is removed keeping its download copy in place of a pause",
    ),
});

export type TrackersKept = zod.input<typeof TrackersKept>;
export type TrackersKeptOutput = zod.output<typeof TrackersKept>;

export const TrackersWait = zod.object({
  torrent: TrackersPlannedTorrent,
  why: zod
    .enum([
      "removing",
      "resumed",
      "ran",
      "incomplete",
      "grab-downloading",
      "grab-importing",
      "import-waiting",
      "reconciling",
    ])
    .describe(
      "What holds its met goal back: removing, it is being removed; resumed, an Admin resumed it after its goal paused it, so it seeds with no end; ran, its goal already acted on it; incomplete, its selected files aren't all there; grab-downloading or grab-importing, a Grab on it is still downloading or importing; import-waiting, a file of its Import waits for the Admin; reconciling, SlipStream is reconciling the library after a restore",
    ),
});

export type TrackersWait = zod.input<typeof TrackersWait>;
export type TrackersWaitOutput = zod.output<typeof TrackersWait>;

export const TrackersGoalsPreview = zod.object({
  acts: zod.array(TrackersAct).describe("The torrents whose goal acts, in added order"),
  kept: zod
    .array(TrackersKept)
    .describe("The torrents removed keeping their download copy, in added order"),
  waits: zod
    .array(TrackersWait)
    .describe("The torrents whose goal is met and waits, in added order"),
});

export type TrackersGoalsPreview = zod.input<typeof TrackersGoalsPreview>;
export type TrackersGoalsPreviewOutput = zod.output<typeof TrackersGoalsPreview>;

export const trackersOwedBytesMin = 0;

export const trackersOwedSecondsMin = 0;

export const TrackersOwed = zod.object({
  bytes: zod
    .int()
    .min(trackersOwedBytesMin)
    .describe("Upload still needed, 0 when the rule doesn't count ratio or it is reached"),
  seconds: zod
    .int()
    .min(trackersOwedSecondsMin)
    .describe("Seeding time still needed, 0 when the rule doesn't count it or it is reached"),
});

export type TrackersOwed = zod.input<typeof TrackersOwed>;
export type TrackersOwedOutput = zod.output<typeof TrackersOwed>;

export const TrackersTarget = zod.object({
  clientId: zod
    .uuid()
    .exactOptional()
    .describe("The external client; absent for the Node's built-in client"),
  nodeId: zod.uuid(),
});

export type TrackersTarget = zod.input<typeof TrackersTarget>;
export type TrackersTargetOutput = zod.output<typeof TrackersTarget>;

export const TrackersRoute = zod.object({
  default: zod
    .union([TrackersTarget, zod.null()])
    .describe("The home Node's built-in client; null while the home Node doesn't hold Downloading"),
  homeNodeId: zod.uuid().describe("The home Node of the Indexers that find the Grabs"),
  skipped: zod
    .boolean()
    .describe(
      "True while the stored client can't take a Grab because it is off or deleted, or its Node doesn't hold Downloading",
    ),
  stored: zod
    .union([TrackersChoice, zod.null()])
    .describe("The client an Admin picked; null for none"),
  to: zod
    .union([TrackersTarget, zod.null()])
    .describe(
      "Where the Grabs download now: the stored client while it can take them, else the default; null while they download where the Work's library root is",
    ),
});

export type TrackersRoute = zod.input<typeof TrackersRoute>;
export type TrackersRouteOutput = zod.output<typeof TrackersRoute>;

export const TrackersRouteList = zod.object({
  routes: zod
    .array(TrackersRoute)
    .describe(
      "One for each home Node of the Tracker's Indexers, by Node name; empty unless the Tracker is private",
    ),
});

export type TrackersRouteList = zod.input<typeof TrackersRouteList>;
export type TrackersRouteListOutput = zod.output<typeof TrackersRouteList>;

export const TrackersRule = zod.object({
  clears: zod
    .enum(["time", "ratio", "either", "both"])
    .describe("What meets the rule: its seeding time, its ratio, either of them or both"),
  ratio: zod
    .number()
    .nullable()
    .describe("At most two decimals; null when the rule doesn't count ratio"),
  seedSeconds: zod
    .int()
    .nullable()
    .describe("Seconds; null when the rule doesn't count seeding time"),
  source: zod
    .enum(["releases", "admin"])
    .describe(
      "releases is the latest rule the Tracker's Releases stated; admin is the rule an Admin set, which Releases never replace",
    ),
});

export type TrackersRule = zod.input<typeof TrackersRule>;
export type TrackersRuleOutput = zod.output<typeof TrackersRule>;

export const TrackersRuleBody = zod.object({
  clears: zod
    .enum(["time", "ratio", "either", "both"])
    .describe("What meets the rule: its seeding time, its ratio, either of them or both"),
  ratio: zod
    .number()
    .nullable()
    .describe("At most two decimals; null when the rule doesn't count ratio"),
  seedSeconds: zod
    .int()
    .nullable()
    .describe("Seconds; null when the rule doesn't count seeding time"),
});

export type TrackersRuleBody = zod.input<typeof TrackersRuleBody>;
export type TrackersRuleBodyOutput = zod.output<typeof TrackersRuleBody>;

export const TrackersRuleShape = zod.object({
  clears: zod.enum(["time", "ratio", "either", "both"]),
  ratio: zod.number().nullable().describe("null when the rule doesn't count ratio"),
  seedSeconds: zod
    .int()
    .nullable()
    .describe("Seconds; null when the rule doesn't count seeding time"),
});

export type TrackersRuleShape = zod.input<typeof TrackersRuleShape>;
export type TrackersRuleShapeOutput = zod.output<typeof TrackersRuleShape>;

export const TrackersSeedingChanged = zod.object({
  trackerId: zod
    .uuid()
    .nullable()
    .describe(
      "The Tracker whose seeding changed; null for the default goal, the seeding settings and torrents with no known Tracker",
    ),
});

export type TrackersSeedingChanged = zod.input<typeof TrackersSeedingChanged>;
export type TrackersSeedingChangedOutput = zod.output<typeof TrackersSeedingChanged>;

export const TrackersSettings = zod.object({
  settings: zod
    .array(SettingField)
    .describe(
      "The margin added to every rule's seeding time, and how long a torrent that owes may go without announcing before Health warns",
    ),
});

export type TrackersSettings = zod.input<typeof TrackersSettings>;
export type TrackersSettingsOutput = zod.output<typeof TrackersSettings>;

export const TrackersSettingsChange = zod.object({
  values: zod
    .record(zod.string(), zod.unknown())
    .describe("Each setting's new value by key, all or none; null resets one to its default"),
});

export type TrackersSettingsChange = zod.input<typeof TrackersSettingsChange>;
export type TrackersSettingsChangeOutput = zod.output<typeof TrackersSettingsChange>;

export const trackersStatsDownloadedMin = 0;

export const trackersStatsGrabs30dMin = 0;

export const trackersStatsOwingMin = 0;

export const trackersStatsRatioMin = 0;

export const trackersStatsSeedingBytesMin = 0;

export const trackersStatsTorrentsMin = 0;

export const trackersStatsUploadedMin = 0;

export const TrackersStats = zod.object({
  downloaded: zod
    .int()
    .min(trackersStatsDownloadedMin)
    .describe("Bytes, counting each torrent once, live or removed"),
  grabs30d: zod.int().min(trackersStatsGrabs30dMin).describe("Its Grabs in the last 30 days"),
  owing: zod.int().min(trackersStatsOwingMin).describe("Its torrents that still owe it seeding"),
  ratio: zod
    .number()
    .min(trackersStatsRatioMin)
    .describe("Uploaded over downloaded, or over size where a torrent's size is larger"),
  seedingBytes: zod
    .int()
    .min(trackersStatsSeedingBytesMin)
    .describe("The size of its torrents that are seeding"),
  torrents: zod.int().min(trackersStatsTorrentsMin).describe("Its torrents, live or being removed"),
  uploaded: zod
    .int()
    .min(trackersStatsUploadedMin)
    .describe("Bytes, counting each torrent once, live or removed"),
});

export type TrackersStats = zod.input<typeof TrackersStats>;
export type TrackersStatsOutput = zod.output<typeof TrackersStats>;

export const TrackersTracker = zod.object({
  goal: zod
    .union([TrackersGoal, zod.null()])
    .describe("Its own seeding goal; null while it follows the default goal"),
  hosts: zod.array(zod.string()).describe("The announce hosts of its private torrents"),
  id: zod.uuid(),
  indexerIds: zod.array(zod.string()).describe("The Indexers that reach it, by name"),
  keys: zod
    .array(zod.string())
    .describe(
      "How a Release finds it: definition:<id>, prowlarr:<feed Indexer>:<source id> or named:<Indexer>",
    ),
  name: zod.string(),
  privacy: zod.enum(["private", "semi-private", "public", "unknown"]),
  rule: zod
    .union([TrackersRule, zod.null()])
    .describe(
      "Its seeding rule in force; null while none is known, and always for a public Tracker",
    ),
});

export type TrackersTracker = zod.input<typeof TrackersTracker>;
export type TrackersTrackerOutput = zod.output<typeof TrackersTracker>;

export const TrackersTrackerItem = zod.object({
  counts: TrackersCounts.describe(
    "Its torrents, live or being removed, as Activity > Torrents counts them",
  ),
  goal: zod
    .union([TrackersGoal, zod.null()])
    .describe("Its own seeding goal; null while it follows the default goal"),
  hosts: zod.array(zod.string()).describe("The announce hosts of its private torrents"),
  id: zod.uuid(),
  indexerIds: zod.array(zod.string()).describe("The Indexers that reach it, by name"),
  keys: zod
    .array(zod.string())
    .describe(
      "How a Release finds it: definition:<id>, prowlarr:<feed Indexer>:<source id> or named:<Indexer>",
    ),
  name: zod.string(),
  privacy: zod.enum(["private", "semi-private", "public", "unknown"]),
  rule: zod
    .union([TrackersRule, zod.null()])
    .describe(
      "Its seeding rule in force; null while none is known, and always for a public Tracker",
    ),
});

export type TrackersTrackerItem = zod.input<typeof TrackersTrackerItem>;
export type TrackersTrackerItemOutput = zod.output<typeof TrackersTrackerItem>;

export const TrackersTrackerList = zod.object({
  trackers: zod.array(TrackersTrackerItem).describe("By name"),
});

export type TrackersTrackerList = zod.input<typeof TrackersTrackerList>;
export type TrackersTrackerListOutput = zod.output<typeof TrackersTrackerList>;

export const TrackersTrackerPage = zod.object({
  attention: zod.array(TrackersAttention).describe("What needs a look on its page, worst first"),
  goal: zod
    .union([TrackersGoal, zod.null()])
    .describe("Its own seeding goal; null while it follows the default goal"),
  hosts: zod.array(zod.string()).describe("The announce hosts of its private torrents"),
  id: zod.uuid(),
  indexerIds: zod.array(zod.string()).describe("The Indexers that reach it, by name"),
  keys: zod
    .array(zod.string())
    .describe(
      "How a Release finds it: definition:<id>, prowlarr:<feed Indexer>:<source id> or named:<Indexer>",
    ),
  name: zod.string(),
  privacy: zod.enum(["private", "semi-private", "public", "unknown"]),
  rule: zod
    .union([TrackersRule, zod.null()])
    .describe(
      "Its seeding rule in force; null while none is known, and always for a public Tracker",
    ),
  stats: TrackersStats.describe("SlipStream's own sums over its torrents"),
});

export type TrackersTrackerPage = zod.input<typeof TrackersTrackerPage>;
export type TrackersTrackerPageOutput = zod.output<typeof TrackersTrackerPage>;

export const trackersWindowInfoHashRegExp = new RegExp("^[0-9a-f]{40}$");

export const TrackersWindow = zod.object({
  atRisk: zod
    .enum(["no", "yes", "unknown"])
    .describe(
      "Whether it is at risk of a hit and run: yes while no copy of it announces as a seeder, unknown while every copy's Node is offline",
    ),
  closedAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("Its latest miss or release; a regrab keeps it"),
  infoHash: zod.string().regex(trackersWindowInfoHashRegExp),
  metAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When its Tracker's rule and the margin were met; null unless met"),
  name: zod.string().describe("Its latest torrent's name"),
  openedAt: zod.iso.datetime({ offset: true }).describe("Its torrent's first downloaded byte"),
  owed: zod
    .union([TrackersOwed, zod.null()])
    .describe("What its latest miss still owed; null before a miss"),
  owes: zod
    .union([TrackersOwed, zod.null()])
    .describe(
      "What it still needs to meet its Tracker's rule and the margin; null unless it owes under a known rule",
    ),
  quietSince: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When it stopped seeding while it owes; null while it seeds"),
  regrabbedAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When a torrent of it came back after a miss"),
  rule: zod
    .union([TrackersRuleShape, zod.null()])
    .describe("The rule it snapshot when it opened; null while none is known"),
  state: zod.enum(["owing", "met", "missed", "released"]),
  torrents: zod
    .array(zod.string())
    .describe("Its live Torrents' ids, in added order; empty once none is live"),
});

export type TrackersWindow = zod.input<typeof TrackersWindow>;
export type TrackersWindowOutput = zod.output<typeof TrackersWindow>;

export const TrackersWindowList = zod.object({
  nextOffset: zod.int().nullable().describe("The offset of the next page; null on the last"),
  windows: zod.array(TrackersWindow).describe("Newest opened first, each as it reads now"),
});

export type TrackersWindowList = zod.input<typeof TrackersWindowList>;
export type TrackersWindowListOutput = zod.output<typeof TrackersWindowList>;

export const TransfersChanged = zod.object({
  nodeIds: zod.array(zod.string()),
  transferIds: zod.array(zod.string()),
});

export type TransfersChanged = zod.input<typeof TransfersChanged>;
export type TransfersChangedOutput = zod.output<typeof TransfersChanged>;

export const TransfersDestination = zod.object({
  kind: zod.enum(["root"]).describe("root is a library root"),
  nodeId: zod.uuid().describe("The Node that holds the root now"),
  nodeName: zod.string(),
  rootId: zod.uuid(),
});

export type TransfersDestination = zod.input<typeof TransfersDestination>;
export type TransfersDestinationOutput = zod.output<typeof TransfersDestination>;

export const transfersDetailsLanesMin = 0;

export const transfersDetailsRetriedMin = 0;

export const transfersDetailsRttMsMin = 0;

export const TransfersDetails = zod.object({
  lanes: zod.int().min(transfersDetailsLanesMin).describe("Connections that carry its chunks now"),
  retried: zod
    .int()
    .min(transfersDetailsRetriedMin)
    .describe("Chunks asked for again since the receiving Node last started it"),
  rttMs: zod.int().min(transfersDetailsRttMsMin).describe("The lanes' round trip"),
});

export type TransfersDetails = zod.input<typeof TransfersDetails>;
export type TransfersDetailsOutput = zod.output<typeof TransfersDetails>;

export const TransfersEnd = zod.object({
  nodeId: zod.uuid(),
  nodeName: zod.string(),
});

export type TransfersEnd = zod.input<typeof TransfersEnd>;
export type TransfersEndOutput = zod.output<typeof TransfersEnd>;

export const transfersFilePositionMin = 0;

export const transfersFileSizeMin = 0;

export const TransfersFile = zod.object({
  name: zod.string(),
  placed: zod.boolean().describe("Its job moved it to where it belongs"),
  position: zod.int().min(transfersFilePositionMin),
  size: zod.int().min(transfersFileSizeMin).describe("Bytes"),
  verified: zod.boolean().describe("Every byte is verified at the destination"),
});

export type TransfersFile = zod.input<typeof TransfersFile>;
export type TransfersFileOutput = zod.output<typeof TransfersFile>;

export const TransfersSettings = zod.object({
  sendLimit: SettingRateField,
});

export type TransfersSettings = zod.input<typeof TransfersSettings>;
export type TransfersSettingsOutput = zod.output<typeof TransfersSettings>;

export const TransfersSettingsPatch = zod.object({
  sendLimit: zod
    .int()
    .nullish()
    .describe("Bytes a second, 0 for no limit; null resets it to its default"),
});

export type TransfersSettingsPatch = zod.input<typeof TransfersSettingsPatch>;
export type TransfersSettingsPatchOutput = zod.output<typeof TransfersSettingsPatch>;

export const transfersSpeedRateMin = 0;

export const TransfersSpeed = zod.object({
  destinationNodeId: zod.uuid(),
  rate: zod
    .int()
    .min(transfersSpeedRateMin)
    .describe("Bytes a second, the speed of the last Transfer from the source to the destination"),
  rateAt: zod.iso.datetime({ offset: true }),
  route: zod
    .enum(["receiver_dials", "sender_dials", "node_link"])
    .exactOptional()
    .describe("The last route that moved bytes; absent while none has"),
  routeAt: zod.iso.datetime({ offset: true }).exactOptional(),
  sourceNodeId: zod.uuid(),
});

export type TransfersSpeed = zod.input<typeof TransfersSpeed>;
export type TransfersSpeedOutput = zod.output<typeof TransfersSpeed>;

export const TransfersSpeedList = zod.object({
  speeds: zod
    .array(TransfersSpeed)
    .describe("One for each source and destination that a Transfer moved bytes between"),
});

export type TransfersSpeedList = zod.input<typeof TransfersSpeedList>;
export type TransfersSpeedListOutput = zod.output<typeof TransfersSpeedList>;

export const TransfersWait = zod.object({
  kind: zod.enum([
    "node_away",
    "node_other_build",
    "connecting",
    "no_route",
    "root_not_ready",
    "storage_draining",
    "source_checking",
    "source_not_ready",
  ]),
  nodeId: zod.uuid().describe("The Node it waits for"),
});

export type TransfersWait = zod.input<typeof TransfersWait>;
export type TransfersWaitOutput = zod.output<typeof TransfersWait>;

export const transfersTransferEstimateSecondsMin = 0;

export const transfersTransferRateMin = 0;

export const transfersTransferTotalMin = 0;

export const transfersTransferVerifiedMin = 0;

export const TransfersTransfer = zod.object({
  destination: TransfersDestination,
  details: TransfersDetails,
  estimateSeconds: zod
    .int()
    .min(transfersTransferEstimateSecondsMin)
    .describe("Seconds until its files are verified; 0 while unknown"),
  files: zod.array(TransfersFile).describe("In the order they copy"),
  id: zod.uuid(),
  job: zod.string().describe("The id of the job that asked for it, such as its Grab's"),
  jobName: zod.string().describe("The job's name, such as a Work's title and year"),
  purpose: zod.enum(["grab", "import"]).describe("The kind of job that asked for it"),
  rate: zod
    .int()
    .min(transfersTransferRateMin)
    .describe(
      "Bytes a second it copies at, else the last speed between its two Nodes; 0 while unknown",
    ),
  reason: zod
    .string()
    .exactOptional()
    .describe("Why it isn't copying, as a sentence; absent while it copies"),
  route: zod
    .enum(["receiver_dials", "sender_dials", "node_link"])
    .exactOptional()
    .describe(
      "Which Node opens its lanes, or node_link when they run over the Node link; absent unless it's first in line",
    ),
  source: TransfersEnd.describe("The Node it copies from"),
  state: zod
    .enum(["queued", "waiting", "needs_space", "copying", "failed"])
    .describe(
      "copying also covers files that are all verified and wait for their job to place them",
    ),
  total: zod.int().min(transfersTransferTotalMin).describe("Bytes"),
  verified: zod
    .int()
    .min(transfersTransferVerifiedMin)
    .describe("Bytes verified at the destination"),
  wait: TransfersWait.exactOptional().describe(
    "What it waits for; absent unless its state is waiting",
  ),
});

export type TransfersTransfer = zod.input<typeof TransfersTransfer>;
export type TransfersTransferOutput = zod.output<typeof TransfersTransfer>;

export const TransfersTransferList = zod.object({
  transfers: zod.array(TransfersTransfer).describe("Oldest first"),
});

export type TransfersTransferList = zod.input<typeof TransfersTransferList>;
export type TransfersTransferListOutput = zod.output<typeof TransfersTransferList>;

export const unblockInputBodyInfoHashesItemRegExp = new RegExp("^[0-9a-f]{40}$");
export const unblockInputBodyInfoHashesMax = 1000;

export const UnblockInputBody = zod.object({
  infoHashes: zod
    .array(zod.string().regex(unblockInputBodyInfoHashesItemRegExp))
    .min(1)
    .max(unblockInputBodyInfoHashesMax),
});

export type UnblockInputBody = zod.input<typeof UnblockInputBody>;
export type UnblockInputBodyOutput = zod.output<typeof UnblockInputBody>;

export const UnmatchedOutputBody = zod.object({
  contentTypes: zod.array(FilesUnmatchedType).describe("Each Content Type with such a file"),
});

export type UnmatchedOutputBody = zod.input<typeof UnmatchedOutputBody>;
export type UnmatchedOutputBodyOutput = zod.output<typeof UnmatchedOutputBody>;

export const WorkPlanInputBody = zod.object({
  draft: FilesRenameDraft.exactOptional().describe(
    "Unsaved naming settings to plan under instead of the stored ones",
  ),
});

export type WorkPlanInputBody = zod.input<typeof WorkPlanInputBody>;
export type WorkPlanInputBodyOutput = zod.output<typeof WorkPlanInputBody>;

export const WorksArt = zod.object({
  kind: zod.string().describe("An artwork kind of the Content Type"),
  url: zod.string().describe("The artwork route for the image, without a size"),
});

export type WorksArt = zod.input<typeof WorksArt>;
export type WorksArtOutput = zod.output<typeof WorksArt>;

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

export const WorksTextsValue = zod.object({
  key: zod.string(),
  kind: zod.enum(["texts"]),
  texts: zod.array(zod.string()).describe("In the source's order"),
});

export type WorksTextsValue = zod.input<typeof WorksTextsValue>;
export type WorksTextsValueOutput = zod.output<typeof WorksTextsValue>;

export const WorksPerson = zod.object({
  department: zod.string().describe("Empty when the source names none"),
  name: zod.string(),
  photoUrl: zod
    .string()
    .nullable()
    .describe(
      "The artwork route for the person's photo, without a size; null when the source has none",
    ),
  role: zod
    .string()
    .describe("The character played or the job done; empty when the source names none"),
});

export type WorksPerson = zod.input<typeof WorksPerson>;
export type WorksPersonOutput = zod.output<typeof WorksPerson>;

export const WorksPeopleValue = zod.object({
  key: zod.string(),
  kind: zod.enum(["people"]),
  people: zod.array(WorksPerson).describe("In the source's order"),
});

export type WorksPeopleValue = zod.input<typeof WorksPeopleValue>;
export type WorksPeopleValueOutput = zod.output<typeof WorksPeopleValue>;

export const WorksLinkValue = zod.object({
  key: zod.string(),
  kind: zod.enum(["link"]),
  label: zod.string(),
  url: zod.url(),
});

export type WorksLinkValue = zod.input<typeof WorksLinkValue>;
export type WorksLinkValueOutput = zod.output<typeof WorksLinkValue>;

export const WorksFieldValue = zod.discriminatedUnion("kind", [
  WorksTextValue,
  WorksNumberValue,
  WorksDateValue,
  WorksDurationValue,
  WorksTextsValue,
  WorksPeopleValue,
  WorksLinkValue,
]);

export type WorksFieldValue = zod.input<typeof WorksFieldValue>;
export type WorksFieldValueOutput = zod.output<typeof WorksFieldValue>;

export const worksAvailabilityDateRegExp = new RegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$");

export const WorksAvailability = zod.object({
  basis: zod
    .string()
    .describe(
      "The rule that chose the date, in words, such as Digital release in United States; empty when the Content Type names none",
    ),
  date: zod.string().regex(worksAvailabilityDateRegExp).describe("YYYY, YYYY-MM or YYYY-MM-DD"),
});

export type WorksAvailability = zod.input<typeof WorksAvailability>;
export type WorksAvailabilityOutput = zod.output<typeof WorksAvailability>;

export const worksRatingScaleExclusiveMin = 0;

export const worksRatingValueMin = 0;

export const worksRatingVotesMin = 0;

export const WorksRating = zod.object({
  fetchedAt: zod.iso.datetime({ offset: true }),
  label: zod.string().describe("The source's name as a person reads it"),
  scale: zod.number().gt(worksRatingScaleExclusiveMin).describe("The value's maximum"),
  source: zod.string().describe("The ratings source, such as imdb"),
  value: zod.number().min(worksRatingValueMin),
  votes: zod.int().min(worksRatingVotesMin).nullable().describe("Null when the source counts none"),
});

export type WorksRating = zod.input<typeof WorksRating>;
export type WorksRatingOutput = zod.output<typeof WorksRating>;

export const WorksItem = zod.object({
  artwork: zod.array(WorksArt).describe("By kind"),
  aside: zod.boolean().describe("It sits in an aside level"),
  availability: zod
    .union([WorksAvailability, zod.null()])
    .describe(
      "The day the Item counts as released and the rule that chose it; null while no date is known",
    ),
  fields: zod.array(WorksFieldValue),
  id: zod.uuid(),
  key: zod.string(),
  label: zod.string(),
  levelId: zod.uuid().nullable().describe("Null when the Content Type declares no levels"),
  monitored: zod.boolean(),
  position: zod.int().min(1),
  ratings: zod.array(WorksRating).describe("By source"),
  withdrawnAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When the source stopped listing it; null while it lists it"),
});

export type WorksItem = zod.input<typeof WorksItem>;
export type WorksItemOutput = zod.output<typeof WorksItem>;

export const WorksLevel = zod.object({
  artwork: zod.array(WorksArt).describe("By kind"),
  aside: zod.boolean().describe("An outermost level outside the Work's main run, such as Specials"),
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

export const WorksWebLink = zod.object({
  key: zod.string().describe("The link's key, such as imdb"),
  kind: zod.enum(["page", "trailer"]),
  label: zod.string().describe("The site's name as a person reads it"),
  url: zod.url(),
});

export type WorksWebLink = zod.input<typeof WorksWebLink>;
export type WorksWebLinkOutput = zod.output<typeof WorksWebLink>;

export const WorksWorkProfile = zod.object({
  id: zod.uuid(),
  name: zod.string(),
});

export type WorksWorkProfile = zod.input<typeof WorksWorkProfile>;
export type WorksWorkProfileOutput = zod.output<typeof WorksWorkProfile>;

export const WorksWorkRoot = zod.object({
  id: zod.uuid(),
  nodeId: zod.uuid().describe("The Storage Node whose disk holds it"),
  path: zod.string().describe("The folder's path on that Node, in that Node's format"),
});

export type WorksWorkRoot = zod.input<typeof WorksWorkRoot>;
export type WorksWorkRootOutput = zod.output<typeof WorksWorkRoot>;

export const worksWorkViewContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const WorksWorkView = zod.object({
  addedAt: zod.iso.datetime({ offset: true }),
  artwork: zod.array(WorksArt).describe("The Work's own images, by kind"),
  contentType: zod.string().regex(worksWorkViewContentTypeRegExp),
  fields: zod.array(WorksFieldValue),
  goneUpstreamAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe(
      "When the metadata source last reported that it no longer has this Work; null while it has it",
    ),
  id: zod.uuid(),
  items: zod
    .array(WorksItem)
    .describe("By their level's order, live before withdrawn, then position"),
  key: zod.string(),
  label: zod.string(),
  levelFolders: zod
    .boolean()
    .describe("New files sit in a folder per level; unread for a Content Type with no levels"),
  levels: zod
    .array(WorksLevel)
    .describe("By depth, then their parent's order, live before withdrawn, then position"),
  links: zod
    .array(WorksWebLink)
    .describe("Pages about the Work on other sites, in its Content Type's order"),
  monitorAsides: zod
    .boolean()
    .describe("The Items of aside levels are part of the Work's monitoring choice"),
  monitorNew: zod
    .boolean()
    .describe(
      "Items a refresh adds arrive monitored, unless they sit in an aside level and monitorAsides is off",
    ),
  options: zod
    .array(SettingField)
    .describe(
      "The Content Type's own options for the Work, in Edit's order; empty when it has none",
    ),
  profile: WorksWorkProfile.describe("The Profile the Work wants from"),
  ratings: zod.array(WorksRating).describe("The Work's own, by source"),
  refreshedAt: zod.iso.datetime({ offset: true }),
  root: zod
    .union([WorksWorkRoot, zod.null()])
    .describe("The root the Work lives in; null while it has none"),
  sortLabel: zod.string(),
});

export type WorksWorkView = zod.input<typeof WorksWorkView>;
export type WorksWorkViewOutput = zod.output<typeof WorksWorkView>;

export const WorksAdded = zod.object({
  created: zod.boolean().describe("False when the Work was already in the library"),
  work: WorksWorkView,
});

export type WorksAdded = zod.input<typeof WorksAdded>;
export type WorksAddedOutput = zod.output<typeof WorksAdded>;

export const WorksArtRoles = zod.object({
  backdrop: zod
    .string()
    .nullable()
    .describe("The kind shown behind the Work page's header; null when none"),
  poster: zod
    .string()
    .nullable()
    .describe("The kind the Library and the Work page show as its poster; null when none"),
  title: zod
    .string()
    .nullable()
    .describe("The kind of logo shown as the Work's title; null when none"),
});

export type WorksArtRoles = zod.input<typeof WorksArtRoles>;
export type WorksArtRolesOutput = zod.output<typeof WorksArtRoles>;

export const WorksArtworkKind = zod.object({
  depth: zod
    .int()
    .min(1)
    .exactOptional()
    .describe("The level's depth, 1 for the outermost; present only on tier level"),
  key: zod.string(),
  label: zod.string(),
  shape: zod.enum(["poster", "backdrop", "logo", "photo", "cover"]),
  tier: zod.enum(["work", "level", "item"]),
});

export type WorksArtworkKind = zod.input<typeof WorksArtworkKind>;
export type WorksArtworkKindOutput = zod.output<typeof WorksArtworkKind>;

export const WorksCandidate = zod.object({
  fields: zod.array(WorksFieldValue).describe("The Work's fields"),
  key: zod.string(),
  label: zod.string(),
  original: zod.string().describe("The label in its original language; empty when it is the label"),
  overview: zod.string().describe("One line; empty when the source has none"),
  posterUrl: zod
    .string()
    .nullable()
    .describe(
      "The artwork route for the result's poster, without a size; null when the source has none",
    ),
  unlisted: zod
    .boolean()
    .describe(
      "Another source found it and the Content Type's source doesn't list it, so it can't be added",
    ),
  workId: zod.uuid().nullable().describe("The Work, once it is in the library"),
});

export type WorksCandidate = zod.input<typeof WorksCandidate>;
export type WorksCandidateOutput = zod.output<typeof WorksCandidate>;

export const WorksCandidateList = zod.object({
  candidates: zod.array(WorksCandidate).describe("Best first"),
});

export type WorksCandidateList = zod.input<typeof WorksCandidateList>;
export type WorksCandidateListOutput = zod.output<typeof WorksCandidateList>;

export const WorksChanged = zod.object({
  workIds: zod.array(zod.string()),
});

export type WorksChanged = zod.input<typeof WorksChanged>;
export type WorksChangedOutput = zod.output<typeof WorksChanged>;

export const WorksNoun = zod.object({
  many: zod.string(),
  one: zod.string(),
});

export type WorksNoun = zod.input<typeof WorksNoun>;
export type WorksNounOutput = zod.output<typeof WorksNoun>;

export const WorksListing = zod.object({
  column: zod.boolean().describe("A column of the Library's table"),
  filter: zod
    .boolean()
    .describe("A filter: picked values for text and texts, a range for number, date and duration"),
  sort: zod.boolean().describe("A sort, with sections by its value"),
});

export type WorksListing = zod.input<typeof WorksListing>;
export type WorksListingOutput = zod.output<typeof WorksListing>;

export const WorksFieldSpec = zod.object({
  art: zod
    .string()
    .nullable()
    .describe(
      "The key of a Work artwork kind shown beside it, such as a studio's logo; null when none",
    ),
  depth: zod
    .int()
    .min(1)
    .exactOptional()
    .describe("The level's depth, 1 for the outermost; present only on tier level"),
  in: zod
    .array(zod.enum(["facts", "tagline", "overview", "about", "credits", "details", "code"]))
    .describe(
      "Every place the Work page shows it. Code is the short identifier the many-Items layout shows before an Item's label",
    ),
  key: zod.string(),
  kind: zod.enum(["text", "number", "date", "duration", "texts", "people", "link"]),
  label: zod.string(),
  list: WorksListing.describe(
    "Where the Library uses it; all false for a field of a level or an Item",
  ),
  tier: zod.enum(["work", "level", "item"]),
  with: zod
    .string()
    .nullable()
    .describe(
      "The key of a text field of the same tier shown in brackets after it; null when none",
    ),
});

export type WorksFieldSpec = zod.input<typeof WorksFieldSpec>;
export type WorksFieldSpecOutput = zod.output<typeof WorksFieldSpec>;

export const worksContentTypeIdRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const WorksContentType = zod.object({
  art: WorksArtRoles,
  artwork: zod.array(WorksArtworkKind).describe("The kinds of image its Works and Items carry"),
  aside: zod
    .union([WorksNoun, zod.null()])
    .describe(
      "What the type calls its outermost levels outside a Work's main run, such as Specials; null when it marks none",
    ),
  availability: zod
    .string()
    .describe(
      "The label of the Library's availability column and sort, such as Next airing; empty reads Available",
    ),
  enabled: zod.boolean(),
  fields: zod.array(WorksFieldSpec),
  icon: zod.enum(["film", "tv", "note", "open-book", "headphones", "panels", "image", "shapes"]),
  id: zod.string().regex(worksContentTypeIdRegExp),
  item: WorksNoun,
  items: zod
    .enum(["one", "many"])
    .describe("How many Items a Work holds, which sets the Work page's layout"),
  levels: zod.array(WorksNoun).describe("Outermost first; empty when Items hang off the Work"),
  name: zod.string(),
  ratings: zod
    .array(zod.string())
    .describe(
      "The ratings sources its Works show, in order. The first a Work holds is its hero rating",
    ),
  releases: zod
    .string()
    .describe(
      "The verb for unreleased Items coming out, as in Episodes 4 to 7 air 12 October; empty reads come out",
    ),
  source: zod.string().describe("Its metadata source's name, such as TMDB"),
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

export const worksDeletePlanBytesMin = 0;

export const worksDeletePlanDownloadingMin = 0;

export const worksDeletePlanFilesMin = 0;

export const worksDeletePlanKeptMin = 0;

export const worksDeletePlanWorksMin = 0;

export const WorksDeletePlan = zod.object({
  bytes: zod.int().min(worksDeletePlanBytesMin).describe("The size of their library files"),
  downloading: zod
    .int()
    .min(worksDeletePlanDownloadingMin)
    .describe(
      "Their unfinished downloads, whose torrents keep downloading and seeding in the client",
    ),
  files: zod
    .int()
    .min(worksDeletePlanFilesMin)
    .describe("Their library files, Kept files included"),
  kept: zod.int().min(worksDeletePlanKeptMin).describe("Their Kept files"),
  works: zod.int().min(worksDeletePlanWorksMin).describe("The Works it removes"),
});

export type WorksDeletePlan = zod.input<typeof WorksDeletePlan>;
export type WorksDeletePlanOutput = zod.output<typeof WorksDeletePlan>;

export const WorksDoneOutcome = zod.object({
  outcome: zod.enum(["done"]),
  workId: zod.uuid(),
});

export type WorksDoneOutcome = zod.input<typeof WorksDoneOutcome>;
export type WorksDoneOutcomeOutput = zod.output<typeof WorksDoneOutcome>;

export const WorksSkippedOutcome = zod.object({
  outcome: zod.enum(["skipped"]),
  reason: zod
    .enum(["already", "work_has_files", "work_gone_upstream", "work_not_found"])
    .describe("Why the action left the Work as it was"),
  workId: zod.uuid(),
});

export type WorksSkippedOutcome = zod.input<typeof WorksSkippedOutcome>;
export type WorksSkippedOutcomeOutput = zod.output<typeof WorksSkippedOutcome>;

export const WorksOutcomeProblem = zod.object({
  code: zod
    .string()
    .describe("The code a request for this Work alone would have been refused with"),
  detail: zod.string().exactOptional(),
  title: zod.string(),
});

export type WorksOutcomeProblem = zod.input<typeof WorksOutcomeProblem>;
export type WorksOutcomeProblemOutput = zod.output<typeof WorksOutcomeProblem>;

export const WorksFailedOutcome = zod.object({
  outcome: zod.enum(["failed"]),
  problem: WorksOutcomeProblem,
  workId: zod.uuid(),
});

export type WorksFailedOutcome = zod.input<typeof WorksFailedOutcome>;
export type WorksFailedOutcomeOutput = zod.output<typeof WorksFailedOutcome>;

export const WorksOutcome = zod.discriminatedUnion("outcome", [
  WorksDoneOutcome,
  WorksSkippedOutcome,
  WorksFailedOutcome,
]);

export type WorksOutcome = zod.input<typeof WorksOutcome>;
export type WorksOutcomeOutput = zod.output<typeof WorksOutcome>;

export const WorksDeleted = zod.object({
  outcomes: zod.array(WorksOutcome).describe("One per Work, in request order"),
  plan: WorksDeletePlan,
});

export type WorksDeleted = zod.input<typeof WorksDeleted>;
export type WorksDeletedOutput = zod.output<typeof WorksDeleted>;

export const worksDeletingWorkIdsMax = 20000;

export const WorksDeleting = zod.object({
  files: zod
    .enum(["delete", "leave"])
    .describe("Delete the Works' library files from disk, or leave them where they are"),
  preview: zod
    .boolean()
    .exactOptional()
    .describe("Return the plan and the outcomes it predicts, and delete nothing"),
  workIds: zod
    .array(zod.uuid())
    .min(1)
    .max(worksDeletingWorkIdsMax)
    .describe("A repeated id counts once"),
});

export type WorksDeleting = zod.input<typeof WorksDeleting>;
export type WorksDeletingOutput = zod.output<typeof WorksDeleting>;

export const WorksEnabled = zod.object({
  enabled: zod.boolean(),
});

export type WorksEnabled = zod.input<typeof WorksEnabled>;
export type WorksEnabledOutput = zod.output<typeof WorksEnabled>;

export const worksRowRatingScaleExclusiveMin = 0;

export const worksRowRatingValueMin = 0;

export const WorksRowRating = zod.object({
  scale: zod.number().gt(worksRowRatingScaleExclusiveMin).describe("The value's maximum"),
  source: zod.string().describe("The ratings source, such as imdb"),
  value: zod.number().min(worksRowRatingValueMin),
});

export type WorksRowRating = zod.input<typeof WorksRowRating>;
export type WorksRowRatingOutput = zod.output<typeof WorksRowRating>;

export const worksLibraryRowAvailableOnRegExp = new RegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$");
export const worksLibraryRowBytesMin = 0;

export const worksLibraryRowContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");
export const worksLibraryRowFilesMin = 0;

export const worksLibraryRowGoneFilesMin = 0;

export const worksLibraryRowItemsMin = 0;

export const worksLibraryRowKeptMin = 0;

export const worksLibraryRowNeedsYouMin = 0;

export const WorksLibraryRow = zod.object({
  addedAt: zod.iso.datetime({ offset: true }),
  addedBy: zod
    .enum(["admin", "scan"])
    .describe("Who added it: an Admin, or a library scan that matched a folder on disk"),
  availableOn: zod
    .string()
    .regex(worksLibraryRowAvailableOnRegExp)
    .nullable()
    .describe(
      "Its one Item's availability date, or for many Items the earliest not yet out, else the latest out, read over its monitored Items, or over every Item while none is monitored; null while none of those has a date",
    ),
  bytes: zod
    .int()
    .min(worksLibraryRowBytesMin)
    .describe("The size of its library files, Kept files included"),
  cells: FilesStateCounts.describe("Its monitored Items' cells by state"),
  contentType: zod.string().regex(worksLibraryRowContentTypeRegExp),
  files: zod.int().min(worksLibraryRowFilesMin).describe("Its library files, Kept files included"),
  goneFiles: zod
    .int()
    .min(worksLibraryRowGoneFilesMin)
    .describe(
      "Its files a library scan found missing from its root, which the other counts leave out",
    ),
  goneUpstreamAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe(
      "When the metadata source last reported that it no longer has this Work; null while it has it",
    ),
  id: zod.uuid(),
  itemStatus: FilesStateCounts.describe(
    "Its monitored Items by status; an Item with no cell isn't counted",
  ),
  items: zod
    .int()
    .min(worksLibraryRowItemsMin)
    .describe(
      "Its monitored Items that its source still lists, the only Items the status and counts read",
    ),
  kept: zod.int().min(worksLibraryRowKeptMin).describe("Its Kept files"),
  key: zod.string(),
  label: zod.string(),
  monitored: zod
    .enum(["all", "some", "none"])
    .describe("How many of the Items its source still lists are monitored"),
  needsYou: zod.int().min(worksLibraryRowNeedsYouMin).describe("The entries of its Needs you"),
  posterUrl: zod
    .string()
    .nullable()
    .describe("The artwork route for its poster, without a size; null when it has none"),
  profileId: zod.uuid(),
  rating: zod
    .union([WorksRowRating, zod.null()])
    .describe(
      "The first of its Content Type's ratings sources that it holds; null when it holds none",
    ),
  refreshedAt: zod.iso.datetime({ offset: true }),
  rootId: zod.uuid().nullable().describe("Null while the Work has no root"),
  sortLabel: zod.string(),
  status: zod
    .union([FilesState, zod.null()])
    .describe(
      "The first status its monitored Items hold, in the glossary's order; null while none holds a cell",
    ),
  values: zod
    .record(zod.string(), Bare)
    .describe(
      "Its listed fields by key: a string for text and date, a number for number and duration in ms, a list of strings for texts; a field it lacks is left out",
    ),
});

export type WorksLibraryRow = zod.input<typeof WorksLibraryRow>;
export type WorksLibraryRowOutput = zod.output<typeof WorksLibraryRow>;

export const WorksLibraryRows = zod.object({
  rows: zod.array(WorksLibraryRow).describe("By sort label, then id"),
});

export type WorksLibraryRows = zod.input<typeof WorksLibraryRows>;
export type WorksLibraryRowsOutput = zod.output<typeof WorksLibraryRows>;

export const worksMonitoredChangeWorkIdsMax = 20000;

export const WorksMonitoredChange = zod.object({
  on: zod
    .boolean()
    .describe(
      "Apply the All preset to each Work, keeping its own aside choice, or None with asides off",
    ),
  workIds: zod
    .array(zod.uuid())
    .min(1)
    .max(worksMonitoredChangeWorkIdsMax)
    .describe("A repeated id counts once"),
});

export type WorksMonitoredChange = zod.input<typeof WorksMonitoredChange>;
export type WorksMonitoredChangeOutput = zod.output<typeof WorksMonitoredChange>;

export const worksMonitoringChangeWorkIdsMax = 20000;

export const WorksMonitoringChange = zod.object({
  includeAsides: zod
    .boolean()
    .describe("Monitor the aside Items the preset picks, and new ones when it monitors new Items"),
  monitorNew: zod
    .boolean()
    .exactOptional()
    .describe("Whether to monitor Items a refresh adds; left out, the preset decides"),
  preset: zod.enum(["all", "future", "missing", "existing", "first", "latest", "none"]),
  preview: zod.boolean().describe("Answer what the change would move and write nothing"),
  workIds: zod
    .array(zod.uuid())
    .min(1)
    .max(worksMonitoringChangeWorkIdsMax)
    .describe("A repeated id counts once"),
});

export type WorksMonitoringChange = zod.input<typeof WorksMonitoringChange>;
export type WorksMonitoringChangeOutput = zod.output<typeof WorksMonitoringChange>;

export const WorksPresetCount = zod.object({
  monitors: zod.int().describe("Live Items the preset monitors, asides left out"),
  monitorsNew: zod.boolean().describe("Whether the preset monitors Items a refresh adds"),
  preset: zod.enum(["all", "future", "missing", "existing", "first", "latest", "none"]),
});

export type WorksPresetCount = zod.input<typeof WorksPresetCount>;
export type WorksPresetCountOutput = zod.output<typeof WorksPresetCount>;

export const WorksPresetCounts = zod.object({
  asides: zod.int().describe("Items in aside levels, which Include adds to a preset"),
  choices: zod.array(WorksPresetCount).describe("One per preset, in the order the presets show"),
});

export type WorksPresetCounts = zod.input<typeof WorksPresetCounts>;
export type WorksPresetCountsOutput = zod.output<typeof WorksPresetCounts>;

export const WorksMonitoringDoneOutcome = zod.object({
  counts: zod
    .union([WorksPresetCounts, zod.null()])
    .describe("What each preset would monitor of the Work; null unless a preview"),
  outcome: zod.enum(["done"]),
  start: zod.int().describe("Items the change starts monitoring"),
  stop: zod.int().describe("Items the change stops monitoring"),
  workId: zod.uuid(),
});

export type WorksMonitoringDoneOutcome = zod.input<typeof WorksMonitoringDoneOutcome>;
export type WorksMonitoringDoneOutcomeOutput = zod.output<typeof WorksMonitoringDoneOutcome>;

export const WorksMonitoringSkippedOutcome = zod.object({
  counts: zod
    .union([WorksPresetCounts, zod.null()])
    .describe("What each preset would monitor of the Work; null unless a preview"),
  outcome: zod.enum(["skipped"]),
  reason: zod.enum(["already"]).describe("Why the change left the Work as it was"),
  workId: zod.uuid(),
});

export type WorksMonitoringSkippedOutcome = zod.input<typeof WorksMonitoringSkippedOutcome>;
export type WorksMonitoringSkippedOutcomeOutput = zod.output<typeof WorksMonitoringSkippedOutcome>;

export const WorksMonitoringOutcome = zod.discriminatedUnion("outcome", [
  WorksMonitoringDoneOutcome,
  WorksMonitoringSkippedOutcome,
  WorksFailedOutcome,
]);

export type WorksMonitoringOutcome = zod.input<typeof WorksMonitoringOutcome>;
export type WorksMonitoringOutcomeOutput = zod.output<typeof WorksMonitoringOutcome>;

export const WorksMonitoringList = zod.object({
  outcomes: zod.array(WorksMonitoringOutcome).describe("One per Work, in request order"),
});

export type WorksMonitoringList = zod.input<typeof WorksMonitoringList>;
export type WorksMonitoringListOutput = zod.output<typeof WorksMonitoringList>;

export const worksNewWorkContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");
export const worksNewWorkKeyMax = 200;

export const WorksNewWork = zod.object({
  contentType: zod.string().regex(worksNewWorkContentTypeRegExp),
  includeAsides: zod
    .boolean()
    .exactOptional()
    .describe(
      "Add every Item of the aside levels, such as Specials, to what the preset monitors, now and as new ones arrive; left out, the type's add default",
    ),
  key: zod
    .string()
    .min(1)
    .max(worksNewWorkKeyMax)
    .describe("The Work's key at its metadata source"),
  levelFolders: zod
    .boolean()
    .exactOptional()
    .describe(
      "Put a new Work's files in a folder per level; left out, the type's add default. Unread for a Content Type with no levels",
    ),
  preset: zod
    .enum(["all", "future", "missing", "existing", "first", "latest", "none"])
    .exactOptional()
    .describe(
      "The monitoring preset a new Work's Items follow; left out, the type's add default. Adding a Work already in the library changes nothing",
    ),
  profileId: zod
    .uuid()
    .exactOptional()
    .describe(
      "The Profile of the Content Type a new Work wants from; left out, the type's default",
    ),
  remember: zod
    .boolean()
    .exactOptional()
    .describe(
      "Make each choice given here its type's default for the next add. Adding a Work already in the library changes no default",
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

export const WorksOptionsChange = zod.object({
  levelFolders: zod
    .boolean()
    .exactOptional()
    .describe("Whether the Work's files go in a folder per level; left out, it keeps its own"),
  preview: zod.boolean().describe("Answer what the change would withdraw and write nothing"),
  type: zod
    .record(zod.string(), zod.unknown())
    .exactOptional()
    .describe("The Content Type's own options to change, by key"),
});

export type WorksOptionsChange = zod.input<typeof WorksOptionsChange>;
export type WorksOptionsChangeOutput = zod.output<typeof WorksOptionsChange>;

export const WorksOptionsChanged = zod.object({
  levelFolders: zod.boolean(),
  moves: zod
    .boolean()
    .describe("The change fetches the Work again, so its Items may move between levels"),
  options: zod.array(SettingField).describe("The Content Type's own options for the Work"),
  withdraws: zod
    .int()
    .describe(
      "Live Items the change withdraws: the source doesn't list them under the new options, and they keep their files",
    ),
});

export type WorksOptionsChanged = zod.input<typeof WorksOptionsChanged>;
export type WorksOptionsChangedOutput = zod.output<typeof WorksOptionsChanged>;

export const WorksOutcomeList = zod.object({
  outcomes: zod.array(WorksOutcome).describe("One per Work, in request order"),
});

export type WorksOutcomeList = zod.input<typeof WorksOutcomeList>;
export type WorksOutcomeListOutput = zod.output<typeof WorksOutcomeList>;

export const worksTreeContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const WorksTree = zod.object({
  addedAt: zod.iso.datetime({ offset: true }),
  artwork: zod.array(WorksArt).describe("The Work's own images, by kind"),
  contentType: zod.string().regex(worksTreeContentTypeRegExp),
  fields: zod.array(WorksFieldValue),
  goneUpstreamAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe(
      "When the metadata source last reported that it no longer has this Work; null while it has it",
    ),
  id: zod.uuid(),
  items: zod
    .array(WorksItem)
    .describe("By their level's order, live before withdrawn, then position"),
  key: zod.string(),
  label: zod.string(),
  levels: zod
    .array(WorksLevel)
    .describe("By depth, then their parent's order, live before withdrawn, then position"),
  ratings: zod.array(WorksRating).describe("The Work's own, by source"),
  refreshedAt: zod.iso.datetime({ offset: true }),
  sortLabel: zod.string(),
});

export type WorksTree = zod.input<typeof WorksTree>;
export type WorksTreeOutput = zod.output<typeof WorksTree>;

export const WorksPreviewed = zod.object({
  inLibrary: zod
    .boolean()
    .describe("Whether the Work is in the library; while it isn't, the ids in work name nothing"),
  presets: zod
    .union([WorksPresetCounts, zod.null()])
    .describe(
      "What each monitoring preset would monitor of the Work; null while it is in the library",
    ),
  work: WorksTree,
});

export type WorksPreviewed = zod.input<typeof WorksPreviewed>;
export type WorksPreviewedOutput = zod.output<typeof WorksPreviewed>;

export const worksProfileChangeWorkIdsMax = 20000;

export const WorksProfileChange = zod.object({
  profileId: zod
    .uuid()
    .describe("The Profile each Work wants from; a Work with its own Edition list keeps it"),
  workIds: zod
    .array(zod.uuid())
    .min(1)
    .max(worksProfileChangeWorkIdsMax)
    .describe("A repeated id counts once"),
});

export type WorksProfileChange = zod.input<typeof WorksProfileChange>;
export type WorksProfileChangeOutput = zod.output<typeof WorksProfileChange>;

export const WorksProfileChanged = zod.object({
  impact: TargetsImpact,
  outcomes: zod.array(WorksOutcome).describe("One per Work, in request order"),
});

export type WorksProfileChanged = zod.input<typeof WorksProfileChanged>;
export type WorksProfileChangedOutput = zod.output<typeof WorksProfileChanged>;

export const worksRefreshingWorkIdsMax = 20000;

export const WorksRefreshing = zod.object({
  workIds: zod
    .array(zod.uuid())
    .min(1)
    .max(worksRefreshingWorkIdsMax)
    .describe("A repeated id counts once"),
});

export type WorksRefreshing = zod.input<typeof WorksRefreshing>;
export type WorksRefreshingOutput = zod.output<typeof WorksRefreshing>;

export const WorksRemoved = zod.object({
  workIds: zod.array(zod.string()),
});

export type WorksRemoved = zod.input<typeof WorksRemoved>;
export type WorksRemovedOutput = zod.output<typeof WorksRemoved>;

export const worksRootChangeWorkIdsMax = 20000;

export const WorksRootChange = zod.object({
  rootId: zod.uuid().describe("The root each Work moves to; a Work with files stays where it is"),
  workIds: zod
    .array(zod.uuid())
    .min(1)
    .max(worksRootChangeWorkIdsMax)
    .describe("A repeated id counts once"),
});

export type WorksRootChange = zod.input<typeof WorksRootChange>;
export type WorksRootChangeOutput = zod.output<typeof WorksRootChange>;

export const WorksSearchResult = zod.object({
  rows: zod
    .array(WorksLibraryRow)
    .describe("The rows of the first limit of them, in the same order"),
  workIds: zod.array(zod.string()).describe("Every matching Work, best first"),
});

export type WorksSearchResult = zod.input<typeof WorksSearchResult>;
export type WorksSearchResultOutput = zod.output<typeof WorksSearchResult>;

export const WorksSettingsChange = zod.object({
  values: zod
    .record(zod.string(), zod.unknown())
    .describe("Each setting's new value by key, all or none; null resets one to its default"),
});

export type WorksSettingsChange = zod.input<typeof WorksSettingsChange>;
export type WorksSettingsChangeOutput = zod.output<typeof WorksSettingsChange>;

export const worksTypeChangedContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const WorksTypeChanged = zod.object({
  contentType: zod.string().regex(worksTypeChangedContentTypeRegExp),
});

export type WorksTypeChanged = zod.input<typeof WorksTypeChanged>;
export type WorksTypeChangedOutput = zod.output<typeof WorksTypeChanged>;

export const WorksTypeSettings = zod.object({
  settings: zod.array(SettingField).describe("In the Content Type's order; empty when it has none"),
});

export type WorksTypeSettings = zod.input<typeof WorksTypeSettings>;
export type WorksTypeSettingsOutput = zod.output<typeof WorksTypeSettings>;

export const worksWorkMonitoredItemIdsMax = 20000;

export const worksWorkMonitoredLevelIdsMax = 20000;

export const WorksWorkMonitored = zod.object({
  itemIds: zod.array(zod.uuid()).max(worksWorkMonitoredItemIdsMax),
  levelIds: zod
    .array(zod.uuid())
    .max(worksWorkMonitoredLevelIdsMax)
    .describe("Each level stands for every Item under it"),
  on: zod.boolean().describe("Monitor the levels and Items, or stop"),
});

export type WorksWorkMonitored = zod.input<typeof WorksWorkMonitored>;
export type WorksWorkMonitoredOutput = zod.output<typeof WorksWorkMonitored>;

export const LiveEventDownloadClientsChanged = zod.object({
  event: zod.enum(["download-clients.changed"]),
  data: DownloadclientsClientsChanged,
});

export type LiveEventDownloadClientsChanged = zod.input<typeof LiveEventDownloadClientsChanged>;
export type LiveEventDownloadClientsChangedOutput = zod.output<
  typeof LiveEventDownloadClientsChanged
>;

export const LiveEventFilesChanged = zod.object({
  event: zod.enum(["files.changed"]),
  data: FilesChanged,
});

export type LiveEventFilesChanged = zod.input<typeof LiveEventFilesChanged>;
export type LiveEventFilesChangedOutput = zod.output<typeof LiveEventFilesChanged>;

export const LiveEventFilesMoved = zod.object({
  event: zod.enum(["files.moved"]),
  data: FilesMoved,
});

export type LiveEventFilesMoved = zod.input<typeof LiveEventFilesMoved>;
export type LiveEventFilesMovedOutput = zod.output<typeof LiveEventFilesMoved>;

export const LiveEventFirstrunChanged = zod.object({
  event: zod.enum(["firstrun.changed"]),
  data: FirstrunChanged,
});

export type LiveEventFirstrunChanged = zod.input<typeof LiveEventFirstrunChanged>;
export type LiveEventFirstrunChangedOutput = zod.output<typeof LiveEventFirstrunChanged>;

export const LiveEventGrabsBlocklistChanged = zod.object({
  event: zod.enum(["grabs.blocklist-changed"]),
  data: GrabsBlocklistChanged,
});

export type LiveEventGrabsBlocklistChanged = zod.input<typeof LiveEventGrabsBlocklistChanged>;
export type LiveEventGrabsBlocklistChangedOutput = zod.output<
  typeof LiveEventGrabsBlocklistChanged
>;

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

export const LiveEventIdentityAddressChanged = zod.object({
  event: zod.enum(["identity.address-changed"]),
  data: IdentityAddressChanged,
});

export type LiveEventIdentityAddressChanged = zod.input<typeof LiveEventIdentityAddressChanged>;
export type LiveEventIdentityAddressChangedOutput = zod.output<
  typeof LiveEventIdentityAddressChanged
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

export const LiveEventImportsChanged = zod.object({
  event: zod.enum(["imports.changed"]),
  data: ImportsChanged,
});

export type LiveEventImportsChanged = zod.input<typeof LiveEventImportsChanged>;
export type LiveEventImportsChangedOutput = zod.output<typeof LiveEventImportsChanged>;

export const LiveEventIndexersCatalogChanged = zod.object({
  event: zod.enum(["indexers.catalog-changed"]),
  data: IndexersCatalogChanged,
});

export type LiveEventIndexersCatalogChanged = zod.input<typeof LiveEventIndexersCatalogChanged>;
export type LiveEventIndexersCatalogChangedOutput = zod.output<
  typeof LiveEventIndexersCatalogChanged
>;

export const LiveEventIndexersChanged = zod.object({
  event: zod.enum(["indexers.changed"]),
  data: IndexersChanged,
});

export type LiveEventIndexersChanged = zod.input<typeof LiveEventIndexersChanged>;
export type LiveEventIndexersChangedOutput = zod.output<typeof LiveEventIndexersChanged>;

export const LiveEventIndexersSettingsChanged = zod.object({
  event: zod.enum(["indexers.settings-changed"]),
  data: IndexersSettingsChanged,
});

export type LiveEventIndexersSettingsChanged = zod.input<typeof LiveEventIndexersSettingsChanged>;
export type LiveEventIndexersSettingsChangedOutput = zod.output<
  typeof LiveEventIndexersSettingsChanged
>;

export const LiveEventIndexersStatusChanged = zod.object({
  event: zod.enum(["indexers.status-changed"]),
  data: IndexersStatusChanged,
});

export type LiveEventIndexersStatusChanged = zod.input<typeof LiveEventIndexersStatusChanged>;
export type LiveEventIndexersStatusChangedOutput = zod.output<
  typeof LiveEventIndexersStatusChanged
>;

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

export const LiveEventMetadataChanged = zod.object({
  event: zod.enum(["metadata.changed"]),
  data: MetadataChanged,
});

export type LiveEventMetadataChanged = zod.input<typeof LiveEventMetadataChanged>;
export type LiveEventMetadataChangedOutput = zod.output<typeof LiveEventMetadataChanged>;

export const LiveEventNamingChanged = zod.object({
  event: zod.enum(["naming.changed"]),
  data: NamingChanged,
});

export type LiveEventNamingChanged = zod.input<typeof LiveEventNamingChanged>;
export type LiveEventNamingChangedOutput = zod.output<typeof LiveEventNamingChanged>;

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

export const LiveEventTorrentsChanged = zod.object({
  event: zod.enum(["torrents.changed"]),
  data: DownloadclientsChanged,
});

export type LiveEventTorrentsChanged = zod.input<typeof LiveEventTorrentsChanged>;
export type LiveEventTorrentsChangedOutput = zod.output<typeof LiveEventTorrentsChanged>;

export const LiveEventTorrentsLive = zod.object({
  event: zod.enum(["torrents.live"]),
  data: DownloadclientsLiveTick,
});

export type LiveEventTorrentsLive = zod.input<typeof LiveEventTorrentsLive>;
export type LiveEventTorrentsLiveOutput = zod.output<typeof LiveEventTorrentsLive>;

export const LiveEventTrackersChanged = zod.object({
  event: zod.enum(["trackers.changed"]),
  data: IndexersTrackerChanged,
});

export type LiveEventTrackersChanged = zod.input<typeof LiveEventTrackersChanged>;
export type LiveEventTrackersChangedOutput = zod.output<typeof LiveEventTrackersChanged>;

export const LiveEventTrackersSeedingChanged = zod.object({
  event: zod.enum(["trackers.seeding-changed"]),
  data: TrackersSeedingChanged,
});

export type LiveEventTrackersSeedingChanged = zod.input<typeof LiveEventTrackersSeedingChanged>;
export type LiveEventTrackersSeedingChangedOutput = zod.output<
  typeof LiveEventTrackersSeedingChanged
>;

export const LiveEventTransfersChanged = zod.object({
  event: zod.enum(["transfers.changed"]),
  data: TransfersChanged,
});

export type LiveEventTransfersChanged = zod.input<typeof LiveEventTransfersChanged>;
export type LiveEventTransfersChangedOutput = zod.output<typeof LiveEventTransfersChanged>;

export const LiveEventVolumesChanged = zod.object({
  event: zod.enum(["volumes.changed"]),
  data: RootsVolumesChanged,
});

export type LiveEventVolumesChanged = zod.input<typeof LiveEventVolumesChanged>;
export type LiveEventVolumesChangedOutput = zod.output<typeof LiveEventVolumesChanged>;

export const LiveEventWorksChanged = zod.object({
  event: zod.enum(["works.changed"]),
  data: WorksChanged,
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

export const LiveEventWorksRemoved = zod.object({
  event: zod.enum(["works.removed"]),
  data: WorksRemoved,
});

export type LiveEventWorksRemoved = zod.input<typeof LiveEventWorksRemoved>;
export type LiveEventWorksRemovedOutput = zod.output<typeof LiveEventWorksRemoved>;

export const LiveEventWorksTypeChanged = zod.object({
  event: zod.enum(["works.type-changed"]),
  data: WorksTypeChanged,
});

export type LiveEventWorksTypeChanged = zod.input<typeof LiveEventWorksTypeChanged>;
export type LiveEventWorksTypeChangedOutput = zod.output<typeof LiveEventWorksTypeChanged>;

export const LiveEvent = zod.discriminatedUnion("event", [
  LiveEventDownloadClientsChanged,
  LiveEventFilesChanged,
  LiveEventFilesMoved,
  LiveEventFirstrunChanged,
  LiveEventGrabsBlocklistChanged,
  LiveEventGrabsChanged,
  LiveEventGrabsSearchChanged,
  LiveEventHealthChanged,
  LiveEventIdentityAccountsChanged,
  LiveEventIdentityAddressChanged,
  LiveEventIdentityLinksChanged,
  LiveEventIdentityPasskeysChanged,
  LiveEventIdentitySessionsChanged,
  LiveEventImportsChanged,
  LiveEventIndexersCatalogChanged,
  LiveEventIndexersChanged,
  LiveEventIndexersSettingsChanged,
  LiveEventIndexersStatusChanged,
  LiveEventLiveAccessChanged,
  LiveEventLiveResync,
  LiveEventLiveSessionEnded,
  LiveEventMetadataChanged,
  LiveEventNamingChanged,
  LiveEventNodeSettingsChanged,
  LiveEventNodesChanged,
  LiveEventNodesTraffic,
  LiveEventRootsChanged,
  LiveEventTargetsChanged,
  LiveEventTasksChanged,
  LiveEventTasksProgress,
  LiveEventTorrentsChanged,
  LiveEventTorrentsLive,
  LiveEventTrackersChanged,
  LiveEventTrackersSeedingChanged,
  LiveEventTransfersChanged,
  LiveEventVolumesChanged,
  LiveEventWorksChanged,
  LiveEventWorksContentTypesChanged,
  LiveEventWorksRemoved,
  LiveEventWorksTypeChanged,
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
