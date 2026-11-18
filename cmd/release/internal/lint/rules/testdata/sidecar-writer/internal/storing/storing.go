package storing

type SidecarCalls interface{ List() }

type SidecarRemote func(string) SidecarCalls

func SidecarCallsOf(any) SidecarCalls { return nil }
