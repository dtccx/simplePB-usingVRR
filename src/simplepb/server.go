package simplepb

//
// This is a outline of primary-backup replication based on a simplifed version of Viewstamp replication.
//
//
//

import (
	"sync"
	//"log"
	"labrpc"
)

// the 3 possible server status
const (
	NORMAL = iota  	//0
	VIEWCHANGE			//1
	RECOVERING			//2
)

// PBServer defines the state of a replica server (either primary or backup)
type PBServer struct {
	mu             sync.Mutex          // Lock to protect shared access to this peer's state
	peers          []*labrpc.ClientEnd // RPC end points of all peers
	me             int                 // this peer's index into peers[]
	currentView    int                 // what this peer believes to be the current active view
	status         int                 // the server's current status (NORMAL, VIEWCHANGE or RECOVERING)
	lastNormalView int                 // the latest view which had a NORMAL status

	log         []interface{} // the log of "commands"
	commitIndex int           // all log entries <= commitIndex are considered to have been committed.

	// ... other state that you might need ...
}

// Prepare defines the arguments for the Prepare RPC
// Note that all field names must start with a capital letter for an RPC args struct
type PrepareArgs struct {
	View          int         // the primary's current view
	PrimaryCommit int         // the primary's commitIndex
	Index         int         // the index position at which the log entry is to be replicated on backups
	Entry         interface{} // the log entry to be replicated
}

// PrepareReply defines the reply for the Prepare RPC
// Note that all field names must start with a capital letter for an RPC reply struct
type PrepareReply struct {
	View    int  // the backup's current view
	Success bool // whether the Prepare request has been accepted or rejected
}

// RecoverArgs defined the arguments for the Recovery RPC
type RecoveryArgs struct {
	View   int // the view that the backup would like to synchronize with
	Server int // the server sending the Recovery RPC (for debugging)
}

type RecoveryReply struct {
	View          int           // the view of the primary
	Entries       []interface{} // the primary's log including entries replicated up to and including the view.
	PrimaryCommit int           // the primary's commitIndex
	Success       bool          // whether the Recovery request has been accepted or rejected
}

type ViewChangeArgs struct {
	View int // the new view to be changed into
}

type ViewChangeReply struct {
	LastNormalView int           // the latest view which had a NORMAL status at the server
	Log            []interface{} // the log at the server
	Success        bool          // whether the ViewChange request has been accepted/rejected
}

type StartViewArgs struct {
	View int           // the new view which has completed view-change
	Log  []interface{} // the log associated with the new new
}

type StartViewReply struct {
}

// GetPrimary is an auxilary function that returns the server index of the
// primary server given the view number (and the total number of replica servers)
func GetPrimary(view int, nservers int) int {
	return view % nservers
}

// IsCommitted is called by tester to check whether an index position
// has been considered committed by this server
func (srv *PBServer) IsCommitted(index int) (committed bool) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.commitIndex >= index {
		return true
	}
	return false
}

// ViewStatus is called by tester to find out the current view of this server
// and whether this view has a status of NORMAL.
func (srv *PBServer) ViewStatus() (currentView int, statusIsNormal bool) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return srv.currentView, srv.status == NORMAL
}

// GetEntryAtIndex is called by tester to return the command replicated at
// a specific log index. If the server's log is shorter than "index", then
// ok = false, otherwise, ok = true
func (srv *PBServer) GetEntryAtIndex(index int) (ok bool, command interface{}) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if len(srv.log) > index {
		return true, srv.log[index]
	}
	return false, command
}

// Kill is called by tester to clean up (e.g. stop the current server)
// before moving on to the next test
func (srv *PBServer) Kill() {
	// Your code here, if necessary
}

// Make is called by tester to create and initalize a PBServer
// peers is the list of RPC endpoints to every server (including self)
// me is this server's index into peers.
// startingView is the initial view (set to be zero) that all servers start in
func Make(peers []*labrpc.ClientEnd, me int, startingView int) *PBServer {
	srv := &PBServer{
		peers:          peers,
		me:             me,
		currentView:    startingView,
		lastNormalView: startingView,
		status:         NORMAL,
	}
	// all servers' log are initialized with a dummy command at index 0
	var v interface{}
	srv.log = append(srv.log, v)

	// Your other initialization code here, if there's any
	return srv
}

// Start() is invoked by tester on some replica server to replicate a
// command.  Only the primary should process this request by appending
// the command to its log and then return *immediately* (while the log is being replicated to backup servers).
// if this server isn't the primary, returns false.
// Note that since the function returns immediately, there is no guarantee that this command
// will ever be committed upon return, since the primary
// may subsequently fail before replicating the command to all servers
//
// The first return value is the index that the command will appear at
// *if it's eventually committed*. The second return value is the current
// view. The third return value is true if this server believes it is
// the primary.
func (srv *PBServer) Start(command interface{}) (
	index int, view int, ok bool) {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	//log.Printf("status", ok)
	//log.Printf("[%d]status", srv.status)
	//log.Printf("GetPrimary[%d]  me[%d]", GetPrimary(srv.currentView, len(srv.peers)), srv.me)
	// do not process command if status is not NORMAL
	// and if i am not the primary in the current view
	if srv.status != NORMAL {
		return -1, srv.currentView, false
	} else if GetPrimary(srv.currentView, len(srv.peers)) != srv.me {
		return -1, srv.currentView, false
	}

	// Your code here
	//append the command in its log
	srv.log = append(srv.log, command)

	go srv.resendPrepare(command, len(srv.log) - 1, srv.currentView, srv.commitIndex)

	//log.Println(ok)
//	log.Println(srv.IsCommitted(index))
	return len(srv.log) - 1, srv.currentView, true
}

func (srv *PBServer) resendPrepare(command interface{}, index int, currentView int, commitIndex int) {
	//send Prepare RPCs to other servers to instruct them
	//to replicate the command in the same index in their log

	//??how to send to other severs and what is other servers???
	//send prepare and get reply
	replys := make([]*PrepareReply, len(srv.peers))
	for i, _ := range srv.peers {
		if i == srv.me {
			// replys[i] = &PrepareReply{
			// 	View:				srv.currentView,
			// 	Success:		true,
			// }
			continue
		}
		prepareArgs := &PrepareArgs{
			View:         currentView,         // the primary's current view
			PrimaryCommit:commitIndex, 			   // the primary's commitIndex
			Index:        index,        // the index position at which the log entry is to be replicated on backups
			Entry:        command,								 // the log entry to be replicated
		}
		replys[i] = &PrepareReply{}
		srv.sendPrepare(i, prepareArgs, replys[i])
	}
	//Recovery

	//change committedindex
	suc_num := 0
	for i, reply := range replys {
		if i == srv.me {
			continue
		}
		//log.Println(reply)
		if reply.Success == true {
			suc_num += 1
		}
	}
	//log.Printf("[%d] successful number", suc_num)

	if suc_num >= (len(srv.peers) - 1) / 2 {
		for {
			srv.mu.Lock()
			if srv.commitIndex + 1 == index {
				srv.commitIndex += 1
				srv.mu.Unlock()
				break
			}
			srv.mu.Unlock()
		}
	} else {
			//if not committed, resend prepare until index commmit
			go srv.resendPrepare(command, index, currentView, commitIndex)
	}


}

// exmple code to send an AppendEntries RPC to a server.
// server is the index of the target server in srv.peers[].
// expects RPC arguments in args.
// The RPC library fills in *reply with RPC reply, so caller should pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
func (srv *PBServer) sendPrepare(server int, args *PrepareArgs, reply *PrepareReply) bool {
	ok := srv.peers[server].Call("PBServer.Prepare", args, reply)
	return ok
}

// Prepare is the RPC handler for the Prepare RPC
func (srv *PBServer) Prepare(args *PrepareArgs, reply *PrepareReply) {
	// Your code here
	srv.mu.Lock()
	defer srv.mu.Unlock()

	//whether the next entry to be added to the log is indeed at the index specified in the message
	//srv.log[args.Index] == args.Entry??

	//log.Printf("[%d] Server get prepare call", srv.me)
	//log.Printf("[%d] currentView [%d] prepareview", srv.currentView, args.View)

	if srv.currentView == args.View && len(srv.log) == args.Index {
		srv.log = append(srv.log, args.Entry)
		reply.View = srv.currentView
		reply.Success = true
		srv.commitIndex = args.PrimaryCommit
		srv.lastNormalView = srv.currentView
		return
	}	else if srv.currentView == args.View && len(srv.log) > args.Index {
		//log's capaticy is bigger, so should return true, but not append,and not recover
		reply.View = args.View
		reply.Success = true
		return
	} else {
		//reply.View = srv.currentView
		reply.Success = false
		//no way to recover
		if(srv.currentView > args.View) {
			return
		}

		//deal with other prepare, no improve, don't know why.
		//go srv.sendPrepare(srv.me, args, reply)

		srv.status = RECOVERING
		go func() {
			recoveryArgs := &RecoveryArgs {
				View:			args.View,
				Server:		srv.me,
			}
			recoveryReply := &RecoveryReply{}
			ok := srv.peers[GetPrimary(args.View, len(srv.peers))].Call("PBServer.Recovery", recoveryArgs, recoveryReply)
			if ok == true && recoveryReply.Success == true {
				srv.currentView = recoveryReply.View
				srv.lastNormalView = recoveryReply.View
				srv.commitIndex = recoveryReply.PrimaryCommit
				srv.log = recoveryReply.Entries
				srv.status = NORMAL
			}
		}()
	}
	// outdate := srv.currentView < args.View
	// log.Printf("if the date outdated",outdate)
	// if outdate {
	//
	// }


}

// Recovery is the RPC handler for the Recovery RPC
func (srv *PBServer) Recovery(args *RecoveryArgs, reply *RecoveryReply) {
	// Your code here
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.status == NORMAL {
		reply.View = srv.currentView
		reply.Entries = srv.log
		reply.PrimaryCommit = srv.commitIndex
		reply.Success = true
	} else {
		reply.Success = false
	}

}

// Some external oracle prompts the primary of the newView to
// switch to the newView.
// PromptViewChange just kicks start the view change protocol to move to the newView
// It does not block waiting for the view change process to complete.
func (srv *PBServer) PromptViewChange(newView int) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	newPrimary := GetPrimary(newView, len(srv.peers))

	if newPrimary != srv.me { //only primary of newView should do view change
		return
	} else if newView <= srv.currentView {
		return
	}
	vcArgs := &ViewChangeArgs{
		View: newView,
	}
	vcReplyChan := make(chan *ViewChangeReply, len(srv.peers))
	// send ViewChange to all servers including myself
	for i := 0; i < len(srv.peers); i++ {
		go func(server int) {
			var reply ViewChangeReply
			ok := srv.peers[server].Call("PBServer.ViewChange", vcArgs, &reply)
			// fmt.Printf("node-%d (nReplies %d) received reply ok=%v reply=%v\n", srv.me, nReplies, ok, r.reply)
			if ok {
				vcReplyChan <- &reply
			} else {
				vcReplyChan <- nil
			}
		}(i)
	}

	// wait to receive ViewChange replies
	// if view change succeeds, send StartView RPC
	go func() {
		var successReplies []*ViewChangeReply
		var nReplies int
		majority := len(srv.peers)/2 + 1
		for r := range vcReplyChan {
			nReplies++
			if r != nil && r.Success {
				successReplies = append(successReplies, r)
			}
			if nReplies == len(srv.peers) || len(successReplies) == majority {
				break
			}
		}
		ok, log := srv.determineNewViewLog(successReplies)
		if !ok {
			return
		}
		svArgs := &StartViewArgs{
			View: vcArgs.View,
			Log:  log,
		}
		// send StartView to all servers including myself
		for i := 0; i < len(srv.peers); i++ {
			var reply StartViewReply
			go func(server int) {
				// fmt.Printf("node-%d sending StartView v=%d to node-%d\n", srv.me, svArgs.View, server)
				srv.peers[server].Call("PBServer.StartView", svArgs, &reply)
			}(i)
		}
	}()
}

// determineNewViewLog is invoked to determine the log for the newView based on
// the collection of replies for successful ViewChange requests.
// if a quorum of successful replies exist, then ok is set to true.
// otherwise, ok = false.
func (srv *PBServer) determineNewViewLog(successReplies []*ViewChangeReply) (
	ok bool, newViewLog []interface{}) {
		// Your code here

	max := 0
	if len(successReplies) >= len(srv.peers) / 2 {
		//decide new one
		ok = true
		//picks the log whose lastest normal view number is the largest.
		for i := 0; i < len(successReplies); i++ {
			if successReplies[i].LastNormalView > max {
				max = successReplies[i].LastNormalView
				//determined the log for the new-view
				newViewLog = successReplies[i].Log
				continue
			}
			if successReplies[i].LastNormalView == max {
				//more than one such logs, it picks the longest log among those
				if (len(successReplies[i].Log) > len(newViewLog)) {
					newViewLog = successReplies[i].Log
				}
			}
		}
	} else {
		ok = false
	}
	return ok, newViewLog
}

// ViewChange is the RPC handler to process ViewChange RPC.
func (srv *PBServer) ViewChange(args *ViewChangeArgs, reply *ViewChangeReply) {
	// Your code here
	srv.mu.Lock()
	defer srv.mu.Unlock()
	// checks that the view number included in the message is indeed
	//larger than what it thinks the current view number is.
	if args.View > srv.currentView {
		srv.status = VIEWCHANGE
		reply.LastNormalView = srv.lastNormalView
		reply.Log = srv.log
		reply.Success = true
	} else {
		reply.Success = false
	}

}

// StartView is the RPC handler to process StartView RPC.
func (srv *PBServer) StartView(args *StartViewArgs, reply *StartViewReply) {
	// Your code here
	srv.mu.Lock()
	defer srv.mu.Unlock()
	//the server must again check current view is no bigger than in the RPC message,
	//which would mean that there's been no concurrent view-change for a larger view.
	if srv.currentView > args.View {
		return
	} else {
		//sets the new-view as indicated in the message
		//changes its status to be NORMAL
		srv.log = args.Log
		srv.currentView = args.View
		//must set lastnormalview, other wise will return error exit status 1
		srv.lastNormalView = args.View
		srv.status = NORMAL
		srv.commitIndex = len(srv.log) - 1
	}

}


/*
test passed: total time is about 40
... Passed
... Passed
iteration i=0 finished
iteration i=4 finished
iteration i=8 finished
iteration i=12 finished
iteration i=16 finished
 ... Passed
... Passed
PASS
*/
