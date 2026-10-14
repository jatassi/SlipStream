package fake

type Online struct{ suite, reason string }

func OnlineSuite(suite string) Online { return Online{suite: suite} }

func NoOnlineRun(reason string) Online { return Online{reason: reason} }

type Spec struct {
	Name   string
	Online Online
}
