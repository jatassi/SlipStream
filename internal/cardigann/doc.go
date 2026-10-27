// Package cardigann is a port of Prowlarr's Cardigann runtime that sends no
// request. Load reads and compiles a definition, or says why SlipStream
// can't run it, and Summary describes it for the Indexer form and the
// catalog. A Definition's Search is an Exchange: Next hands the caller the
// request to send, Feed takes its response, and Result is the outcome. The
// caller owns the network, cookies, spacing and challenges. Behavior
// follows Prowlarr wherever the catalog depends on it.
package cardigann
