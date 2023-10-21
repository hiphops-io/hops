package worker

import (
	"errors"
	"strings"

	undist "github.com/hiphops-io/hops/undistribute"
	"github.com/nats-io/nats.go/jetstream"
)

// ParseResponseSubject parses the nats subject to respond on from the original message
func ParseResponseSubject(msg jetstream.Msg) (string, error) {
	// Reply subject is the exact same as the message subject, but with request swapped with 'notify'
	// Subject structure: ACCOUNT_ID.LEASE_ID.request.SEQUENCE_ID.TASK_SLUG
	subjectParts := strings.Split(msg.Subject(), ".")
	if len(subjectParts) != 5 {
		return "", errors.New("Invalid request subject")
	}
	subjectParts[2] = string(undist.Notify)
	replySubject := strings.Join(subjectParts, ".")

	return replySubject, nil
}

func ParseSequenceID(msg jetstream.Msg) (string, error) {
	subjectParts := strings.Split(msg.Subject(), ".")
	if len(subjectParts) != 5 {
		return "", errors.New("Invalid request subject")
	}

	return subjectParts[3], nil
}
