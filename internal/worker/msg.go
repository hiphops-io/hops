package worker

import (
	"errors"
	"strings"

	undist "github.com/hiphops-io/hops/undistribute"
	"github.com/nats-io/nats.go/jetstream"
)

// Note. Request subject structure is:
// ACCOUNT_ID.LEASE_ID.request.SEQUENCE_ID.TASK_SLUG.APP.HANDLER

const (
	expectedRequestTokenLen = 7
)

func splitRequestSubject(subject string) ([]string, error) {
	subjectParts := strings.Split(subject, ".")
	if len(subjectParts) != expectedRequestTokenLen {
		return []string{}, errors.New("Invalid request subject")
	}

	return subjectParts, nil
}

// ParseResponseSubject parses the nats subject to respond on from the original request message
func ParseResponseSubject(msg jetstream.Msg) (string, error) {
	// Response subject is same as request subject,
	// but with request swapped to 'notify' and app.handler tokens removed
	subjectParts, err := splitRequestSubject(msg.Subject())
	if err != nil {
		return "", err
	}
	subjectParts[2] = string(undist.Notify)
	replySubject := strings.Join(subjectParts[:len(subjectParts)-2], ".")

	return replySubject, nil
}

// ParseSequenceID parses the sequence ID from a request message
func ParseSequenceID(msg jetstream.Msg) (string, error) {
	subjectParts, err := splitRequestSubject(msg.Subject())
	if err != nil {
		return "", err
	}

	return subjectParts[3], nil
}

// ParseAppHandler parses the app and handler from a request subject
func ParseAppHandler(subject string) (string, string, error) {
	subjectParts, err := splitRequestSubject(subject)
	if err != nil {
		return "", "", err
	}

	return subjectParts[5], subjectParts[6], nil
}
