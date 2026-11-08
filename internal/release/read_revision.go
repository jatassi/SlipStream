package release

// readRevision is how often p was re-released: the first release until
// PROPER, REPACK, RERIP, REAL and vN are read.
func readRevision(Prepared) Revision { return Revision{Version: 1} }
