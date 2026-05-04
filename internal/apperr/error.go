package apperr

type CommandError struct {
	Summary string
	Detail  string
}

func (e *CommandError) Error() string {
	if e.Detail == "" {
		return e.Summary
	}
	return e.Summary + ": " + e.Detail
}
