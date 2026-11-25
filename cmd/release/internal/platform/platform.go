// Package platform lists the platform suite's Go tests: the tests that run
// the code each OS changes. A train or merge check runs only this suite on
// the targets besides linux-amd64, and the platform-tests lint fails a
// package with OS-specific code that no entry names.
package platform

// Test is a package's top-level tests that run OS-specific code.
type Test struct {
	// Package is the package's directory from the repo root.
	Package string
	// Run picks the tests, as go test -run does.
	Run string
	// Covers names the packages with OS-specific code that these tests
	// run and whose own tests don't.
	Covers []string
}

var Tests = []Test{
	{Package: "cmd/fakeworld", Run: "^(TestStartStop|TestStopWithoutSupervisor)$"},
	{
		Package: "cmd/slipstream",
		Run: "^(TestTheTrampolineStartsTheAppAndTheConsoleKeepsItsExitCodes|TestRunNeverOpensTheBrowserNorShowsATray|" +
			"TestRestartExecsInPlace|TestRestartStartsASuccessorProcess|TestSIGTERMStopsTheNodeCleanly|" +
			"TestADataDirAnotherUserOwnsExitsWithTheUsageCode|TestTheMarkerPicksTheDefaultDataDir|TestStopWaitsUntilTheDataDirIsFree|" +
			"TestJoinOffline|TestJoinRunning)$",
		Covers: []string{"cmd/slipstream-app", "fakes/testbin", "internal/node/internal/desktop"},
	},
	{
		Package: "e2e",
		Run:     "^(TestTheOldControllerIsFencedThenJoinsAsAWorker|TestARestoreInPlaceAfterAGrab|TestRestartingAWorkerFromItsHost)$",
	},
	{Package: "fakes/internal/front", Run: "^TestARefusedFrontRefusesOnceTheRuleIsSetAndListensOnceItClears$"},
	{Package: "fakes/internal/fsinfo", Run: "^TestStat$"},
	{Package: "fakes/internal/mediafix", Run: "^TestBuildReadsAsWant$"},
	{
		Package: "fakes/world",
		Run:     "^(TestAScenariosNotifyConnectionsReachTheSinkAndTheScript|TestTheWorldFollowsARestartedNode|TestAddressCheckRefusesTheThirtyThirdAddress)$",
		Covers:  []string{"fakes/world/internal/loopback"},
	},
	{
		Package: "fakes/world/internal/proc",
		Run:     "^(TestAProcessThatIsNotAliveReleasesItsExecutable|TestAWatchedProcessReadsAsGoneOnceItsPIDIsReaped|TestWaitExit|TestKillingAProcessThatHasExitedSucceeds)$",
	},
	{Package: "fakes/world/worldtest", Run: "^TestAHeldFileNamesItsHolder$"},
	{Package: "internal/atomicfile", Run: "^(TestReplaceReplaces|TestRenameRetriesASharingViolation|TestRemoveRetriesASharingViolation)$"},
	{Package: "internal/downloading", Run: "^(TestFreesCountsOnlyFilesWithNoOtherLink|TestPlacePayloadLinksOnOneVolumeAndCopiesAcrossVolumes)$"},
	{Package: "internal/link", Run: "^TestKeyFileIsOwnerOnly$"},
	{Package: "internal/link/internal/mux", Run: "^TestMeasureStream$"},
	{
		Package: "internal/node",
		Run: "^(TestARestoreInPlaceKeepsTheBeforeRestoreBackupNewerBackupsAndCredentials|TestARestoreOnANewMachineFencesTheOldControllerThroughAWorker|" +
			"TestAWorkerJoinsThroughTheSetupGate|TestACrashBeforeEachSwapStepFinishesAtTheNextStart|TestStartRefusesAHeldDataDir|" +
			"TestParseDataDirMakesTheSocketPathAbsoluteAndShort)$",
	},
	{Package: "internal/node/internal/host", Run: "^(TestBelowFloor|TestOnNetworkShareChecksTheNearestExistingAncestor)$"},
	{Package: "internal/notifying", Run: "^(TestAScriptGetsItsArgumentsTheEventAndACleanEnvironment|TestAScriptPastStopIsKilledWithEverythingItStarted)$"},
	{Package: "internal/secret", Run: "^(TestNoRowNoFileCreatesTheKey|TestCreatingTheKeySyncsItsDirectory)$"},
	{Package: "internal/storage", Run: "^(TestSynctestWaitsForSQLiteWaitingOutALock|TestConcurrentWritesAllCommit)$"},
	{
		Package: "internal/storing",
		Run: "^(TestOSDisk.*|TestLongPath|TestUNCPath|TestStableFSName|TestNetworkFSName|TestNetworkMagic|TestStableMagic|" +
			"TestACaseOnlyMoveOnAFoldingVolumeRespellsTheFile|TestAMoveOntoAHardLinkRemovesTheSource|TestAWalkReadsAFoldingRootAndAVolumeWithoutIDs)$",
	},
	{
		Package: "internal/torrent/libtorrent",
		Run: "^(TestTheShimStructsMatchGo|TestEngineRoundTrip|TestFilesListThePathsLibtorrentSanitized|TestEngineListensWhenTheNextTCPPortsAreTakenForUDP|" +
			"TestADownloadThatFillsTheDiskPausesUntilResumed|TestTheListenerFollowsTheDefaultRoute)$",
	},
}
