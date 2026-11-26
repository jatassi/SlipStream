package access

const (
	plex   = WayPlexTV
	users  = WayUsers
	admin  = ByAdmin
	member = ByMember
	ctl    = ByController
	anyone = ByAdmin | ByMember | ByController
)

// Table is every move a row can make.
var Table = []Transition{
	{None, Offer, NoIdentity, Offered, admin, plex, 0},
	{None, Offer, IdentityHeld, Waiting, admin, plex, 0},
	{None, Offer, Done, Offered, admin, users, 0},
	{Offered, Offer, Done, Offered, admin, plex, 0},
	{Waiting, Offer, Done, Waiting, admin, plex, 0},
	{Removed, Offer, Done, Active, admin, users, 0},
	{Removed, Offer, Gone, Offered, admin, users, ClearUser},
	{Removed, Offer, Failed, Removed, admin, users, KeepProblem},

	{Offered, PlexSignIn, AlreadyThere, Active, ctl, plex, MarkByHand},
	{Offered, PlexSignIn, NotThere, Waiting, ctl, plex, 0},
	{Waiting, PlexSignIn, AlreadyThere, Active, ctl, plex, MarkByHand},
	{Waiting, PlexSignIn, NotThere, Waiting, ctl, plex, 0},
	{Shared, PlexSignIn, Accepted, Active, ctl, plex, 0},
	{Shared, PlexSignIn, NotAccepted, Shared, ctl, plex, 0},
	{Waiting, AnotherPlex, Done, Offered, member, plex, ClearPlex},

	{Waiting, ShareFriend, Accepted, Active, admin, plex, ClearPlexHome},
	{Waiting, ShareFriend, NotAccepted, Shared, admin, plex, ClearPlexHome},
	{Waiting, ShareFriend, Refused, Waiting, admin, plex, KeepProblem},
	{Waiting, ShareHome, Accepted, Active, admin, plex, SetPlexHome},
	{Waiting, ShareHome, NotAccepted, Shared, admin, plex, SetPlexHome},
	{Waiting, ShareHome, Refused, Waiting, admin, plex, KeepProblem},
	{Waiting, MarkAdded, Done, Active, admin, plex, MarkByHand},
	{Shared, Check, Accepted, Active, anyone, plex, 0},
	{Shared, Check, NotAccepted, Shared, anyone, plex, 0},

	{Offered, MakeAccount, Done, Making, member, users, 0},
	{Making, Created, Done, Making, ctl, users, 0},
	{Making, Finish, Done, Active, ctl, users, 0},
	{Making, Unmake, Done, Offered, ctl, users, KeepProblem | ClearUser},
	{Making, Unmake, Failed, Making, ctl, users, KeepProblem},
	{Making, Unmake, Left, None, ctl, users, 0},
	{Offered, Link, Done, Active, admin, users, 0},

	{Offered, Remove, Done, None, admin, plex, 0},
	{Offered, Remove, Done, None, admin, users, 0},
	{Waiting, Remove, Done, None, admin, plex, 0},
	{Waiting, Remove, Failed, Removing, admin, plex, KeepProblem},
	{Shared, Remove, Done, None, admin, plex, 0},
	{Shared, Remove, Failed, Removing, admin, plex, KeepProblem},
	{Active, Remove, Done, None, admin, plex, 0},
	{Active, Remove, Failed, Removing, admin, plex, KeepProblem},
	{Active, Remove, Done, Removed, admin, users, 0},
	{Active, Remove, Failed, Removing, admin, users, KeepProblem},
	{Removing, RetryRemove, Done, None, admin, plex, 0},
	{Removing, RetryRemove, Failed, Removing, admin, plex, KeepProblem},
	{Removing, RetryRemove, Done, Removed, admin, users, 0},
	{Removing, RetryRemove, Left, None, admin, users, 0},
	{Removing, RetryRemove, Failed, Removing, admin, users, KeepProblem},

	{Offered, Leave, Done, None, ctl, plex, 0},
	{Offered, Leave, Done, None, ctl, users, 0},
	{Waiting, Leave, Done, None, ctl, plex, 0},
	{Waiting, Leave, Failed, Removing, ctl, plex, KeepProblem | ClearAccount},
	{Removed, Leave, Done, None, ctl, users, 0},
	{Shared, Leave, Done, None, ctl, plex, 0},
	{Shared, Leave, Failed, Removing, ctl, plex, KeepProblem | ClearAccount},
	{Active, Leave, Done, None, ctl, plex, 0},
	{Active, Leave, Failed, Removing, ctl, plex, KeepProblem | ClearAccount},
	{Active, Leave, Done, None, ctl, users, 0},
	{Active, Leave, Failed, Removing, ctl, users, KeepProblem | ClearAccount},
	{Removing, Leave, Done, None, ctl, plex, 0},
	{Removing, Leave, Failed, Removing, ctl, plex, KeepProblem | ClearAccount},
	{Removing, Leave, Done, None, ctl, users, 0},
	{Removing, Leave, Failed, Removing, ctl, users, KeepProblem | ClearAccount},
	{Making, Leave, Done, None, ctl, users, 0},
	{Making, Leave, Failed, Making, ctl, users, KeepProblem | ClearAccount},
}
