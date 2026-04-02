# How does it work

## Client to Client Connections

### Uploading file (Seeding)
When clients upload a file by `upload_file()` function, the client connects
to the Tracker, the tracker then creates a new file for other clients to request from
when they want to download that file.

FILE CONTENTS:
    title: string
    hash: string,
    datecreated: string,
    nodes: []

### Downloading a file (Leeching)
Clients wanting to download a specific file can be read from the Tracker `Tracker Server` for using the 
command --list-files, which returns a list of available files that can be downloadded.

When receiving the file from the tracker, the client sends a leech request 
to list of nodes (cluster) for that file which then each peer in that cluster
stores the information of the newly connected peer. 

each peer aside from the connecting peer will contribute to the sending
of data segments of the file.

### Files

When sending files, we need to consider the throughput of the seeding peer, 
if the seeding peers has a higher throughput, then we adjust the segments that 
a peer can send by N.

# TODO
How do we consider if one peer is faster than the other? and on what position 
should a peer send data segments to fill the nth segment of the 
receiving peer up to m? 

- send first dummy segments from each peer
- and for each peer calculate how many bytes have been sent from starting of the initial byte timer
- (elapsed / chunksSent) / (1024 ** 2) = nPeerThroughput

#### in this example peer2 > peer4 > peer3 > n
 [peer2 segments , peer4 segments n...m, peer3 segments,...]

## Entering cluster

Client communicates with the `Tracker Server` if there are any files available from the comman `--list-files`,
if not then nothing happens.

If client connects to the `Tracker Server` to seed a file, data is created for that file and the client is set to `status` of 
`seeding` for that current file so that we know that we can receive data segments for that file from that seeding client.

If a client wants to leech from a file, it contacts the `Tracker Server`, it receives list of clients that are actively seeding that file,
client enters the cluster and pings every peer and if they respond then they are stored in a hashtable cluster for that file.

    WHEN DOES A CLIENT RESPOND TO A PING:
    - a pinged peer won't respond if it's status is `seeding` only
    else it does not respond to the sender

Once there are responses from the peers whose `status` are `seeding` then a coordinated data segmentation is initialized.
data segmentation happens right afterwards, and then the newly connected peer sets it's `status` to `leecher` for that current file,
once that status is set, it propagates it by pinging the list of active peers in the hashtable to begin sending data segmentation.

This ping is different from the initial ping, it tells pings the peers and calculates the RTT and sorts them based on their response time.
and based on their response time, the leecher will send out how much segments each peer can send based on their RTT value, so if less
then more segments to send else less segments to send.








