# Transfers are receiver-driven chunk copies between Nodes

Every file copy between Nodes is a Transfer, whether it carries a completed download, an A/V pipeline input or output, or an admin's move. The Controller records each Transfer in its intent. The two Nodes open extra TLS connections with the keys pinned at Pairing, dialed in the same direction as the Node link, or through a relaying Node when neither can reach the other. The receiver asks for chunks over those connections, and the sender hashes each chunk with BLAKE3 as it reads it. The receiver checks every chunk, resumes from the chunks it has verified, and stages files on the destination root's filesystem until import takes each Item. A remote seedbox running an external Download Client needs a SlipStream Worker on it, since no Transfer reaches files without a Node.

Charting Q11 called this a push. The data still flows to the importing Node without a person starting it, but the receiver drives the copy, because every fast tool that splits one file across connections does it as a pull. Decided in [File sync design](https://github.com/jatassi/SlipStream/issues/14#issuecomment-5988990718), Q2, Q5, Q6, Q8 and Q12.

## Considered Options

- **Sender-driven push with offset writes.** Rejected. It needs a receiver that accepts writes at any offset and a second resume design, and buys nothing the receiver-driven form lacks.
- **HTTPS range pull by the storage Node.** Rejected. The sender must accept inbound connections, which fails when the seedbox is the side that dials out.
- **rclone or lftp over SFTP.** Rejected. Segmenting works only as a pull, SFTP pipelines cap one file at roughly 40 to 170 Mbit/s at 100 ms by default (computed in [Fast transfer between Nodes](https://github.com/jatassi/SlipStream/issues/13)), and it means a second binary to ship.
- **BitTorrent between Nodes.** Rejected. BEP 27 forbids peers outside the tracker for private torrents.
- **Torrent piece hashes as the integrity reference.** Rejected. A v1 piece can span two files, and external clients and derived files offer no piece hashes.
- **A Node-less SFTP path for remote seedboxes.** Rejected. It brings back the SFTP ceiling and a second resume and verification design. A seedbox host that forbids user processes is the accepted cost.

## Consequences

- Each new kind of job that moves files names only what to copy, where, and what happens after. Resume, verification, bandwidth caps and the offline fakes stay in one place.
- Transfers inherit the Node link's dial direction and relay fallback, so a relaying Node carries bulk file traffic as well as calls.
- Import works per Item from a staging folder on the library root's filesystem, and its placement decision runs before the Transfer starts.
