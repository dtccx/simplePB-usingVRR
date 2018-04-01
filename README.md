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


## When to recover:
deny prepare (reply.Suceess = false)


# part 2:

The tester prompts a view-change by invoking the PromptViewChange(newView) function on the primary server for the newView. Recall that in VR, the view number uniquely determines the primary server. In our implementation, all servers can be identified by their index in the peers array and we map each view number to the id of the primary as: view-number % total_servers (The auxilary function GetPrimary(view, nservers) performs this calculation).


The primary servers starts the view-change process by sending a ViewChange RPC to every replica server (including itself). Upon receving ViewChange, a replica server checks that the view number included in the message is indeed larger than what it thinks the current view number is. If the check succeeds, it sets its current view number to that in the message and modifies its status to VIEW-CHANGE. It replies Success=true and includes its current log (in its entirety) as well as the latest view-number that has been considered NORMAL. If the check fails, the backup replies Success=false.


If the primary has received successful ViewChange replies from a majority of servers (including itself). It can proceed to start the new view. It needs to start the new-view with a log that contains all the committed entries of the previous view. To maintain this invariant, the primary chooses the log among the majority of successful replies using this rule: it picks the log whose lastest normal view number is the largest. If there are more than one such logs, it picks the longest log among those. Once the primary has determined the log for the new-view, it sends out the StartView RPC to all servers to instruct them to start the new view. Uponreceive StartView, a server sets the new-view as indicated in the message and changes its status to be NORMAL. Note that before setting the new-view according to the StartView RPC message, the server must again check that its current view is no bigger than that in the RPC message, which would mean that there's been no concurrent view-change for a larger view.


# Test Case
run "go test" to run all cases;run "go test 1A" or others to run some certain cases;

the total cases time is 41s.this is average time, but someone optimize it to 17s.

With set the reprepare time Threathold.
