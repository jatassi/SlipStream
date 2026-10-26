package worksapi

type outcome struct{}

func (outcome) Variants() (discriminator string, variants []any) {
	return "outcome", []any{doneOutcome{}, skippedOutcome{}, failedOutcome{}}
}

type doneOutcome struct {
	WorkID  string `json:"workId" format:"uuid"`
	Outcome string `json:"outcome" enum:"done"`
}

type skippedOutcome struct {
	WorkID  string `json:"workId" format:"uuid"`
	Outcome string `json:"outcome" enum:"skipped"`
	Reason  string `json:"reason" enum:"already,work_has_files,work_gone_upstream,work_not_found" doc:"Why the action left the Work as it was"`
}

type failedOutcome struct {
	WorkID  string         `json:"workId" format:"uuid"`
	Outcome string         `json:"outcome" enum:"failed"`
	Problem outcomeProblem `json:"problem"`
}

type outcomeProblem struct {
	Code   string `json:"code" doc:"The code a request for this Work alone would have been refused with"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}
