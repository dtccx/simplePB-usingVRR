# Primary-Backup-Replication(diff parts in diff branch)
# Part 1A: Primary-backup in the Normal case

When Start(command) is invoked, the primary should append the command in its log and then send Prepare RPCs to other servers to instruct them to replicate the command in the same index in their log. Note that a server should do the processing for Start only if it believes itself to be the current primary and that its status is NORMAL (as opposed to VIEW-CHANGE or RECOVERY).

Upon receiving a Prepare RPC message, the backup checks whether the message's view and its currentView match and whether the next entry to be added to the log is indeed at the index specified in the message. If so, the backup adds the message's entry to the log and replies Success=ok. Otherwise, the backup replies Success=false. Furthermore, if the backup's state falls behind the primary (e.g. its view is smaller or its log is missing entries), it performs recovery to transfer the primary's log. Note that the backup server needs to process Prepare messages according to their index order, otherwise, it would end up unnecessarily rejecting many messages.

If the primary has received Success=true responses from a majority of servers (including itself), it considers the corresponding log index as "committed". (Since servers process Prepare messages ) It advances thecommittedIndex field locally and also piggybacks this information to backup servers in subsequent Prepares.


# Par1B:
If a backup node has crashed, the system can continue undisturbed since the primary only
needs to receive a majority successful Prepare replies in order to consider a log entry
committed. When the crashed backup node comes back online, it will find that its log is out of
sync with the primary and unable to respond positively to the primary's Prepares. In this case,
the backup needs to synchronize with the primary by sending a Recovery RPC. The primary
replies to the Recovery by sending its log in its entirety. Once the backup catches up with the
primary, it can respond positively to the primary Prepare messages. (Note that our recovery
mechanism here is much simpler than the one described in "Viewstamp: revisited" paper (4.3).
This is because we assume nodes log operations to persistent storage before replying to
primary's Prepare requests. Thus, it is never the case that a recovered node "forgets" the
operations that it has already prepared and relied PrepareOK to. By contrast, VR paper's
recovery protocol handles the case of "forgetful" recovered nodes.)

When you've finished implementing the Recovery aspect, you should be able to pass all three tests of part-1A by typing go test -v -run 1A
