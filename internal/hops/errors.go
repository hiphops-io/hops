package hops

import "fmt"

type ErrFailedHopsParse struct {
	message string
}

func (e ErrFailedHopsParse) Error() string {
	return fmt.Sprintf("Unable to parse hops: %s", e.message)
}
