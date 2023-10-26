package undistribute

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Durability int

const (
	Default Durability = iota
	Durable
	Ephemeral
)

// Remap jetstream.DeliverPolicy iota to make DeliverNewPolicy the default
// plus remove unsupported policy types
type DeliverPolicy int

const (
	DeliverNewPolicy DeliverPolicy = iota
	DeliverAllPolicy
	DeliverLastPolicy
	DeliverLastPerSubjectPolicy
)

var deliverPolicyMappings = map[DeliverPolicy]jetstream.DeliverPolicy{
	DeliverNewPolicy:            jetstream.DeliverNewPolicy,
	DeliverAllPolicy:            jetstream.DeliverAllPolicy,
	DeliverLastPolicy:           jetstream.DeliverLastPolicy,
	DeliverLastPerSubjectPolicy: jetstream.DeliverLastPerSubjectPolicy,
}

type LeaseConfig struct {
	NatsUrl              string
	StreamName           string
	RequestDurability    Durability
	RequestDeliverPolicy DeliverPolicy
	LeaseSubject         string
	LeaseDurability      Durability
	LeaseFilter          string
	LeaseDeliverPolicy   DeliverPolicy
	SourceSubject        string
	SourceDurability     Durability
	SourceFilter         string
	SourceDeliverPolicy  DeliverPolicy
	SourceConsumerName   string
	RootDir              string
	Seed                 []byte
}

var DefaultLeaseConfig = LeaseConfig{
	NatsUrl:              nats.DefaultURL,
	StreamName:           "hops",
	RequestDurability:    Durable,
	RequestDeliverPolicy: DeliverNewPolicy,
	LeaseDurability:      Durable,
	LeaseFilter:          ">",
	LeaseDeliverPolicy:   DeliverNewPolicy,
	SourceSubject:        "any",
	SourceDurability:     Durable,
	SourceFilter:         ">",
	SourceDeliverPolicy:  DeliverNewPolicy,
	Seed:                 []byte(""),
}

// MergeLeaseConfigs merges a LeaseConfig into this LeaseConfig
//
// Values in the receiver will take precedence
func (l *LeaseConfig) MergeLeaseConfig(mergeConf LeaseConfig) {
	if l.NatsUrl == "" {
		l.NatsUrl = mergeConf.NatsUrl
	}
	if l.StreamName == "" {
		l.StreamName = mergeConf.StreamName
	}
	if l.LeaseSubject == "" {
		l.LeaseSubject = mergeConf.LeaseSubject
	}
	if l.LeaseDurability == 0 {
		l.LeaseDurability = mergeConf.LeaseDurability
	}
	if l.LeaseFilter == "" {
		l.LeaseFilter = mergeConf.LeaseFilter
	}
	if l.SourceSubject == "" {
		l.SourceSubject = mergeConf.SourceSubject
	}
	if l.SourceDurability == 0 {
		l.SourceDurability = mergeConf.SourceDurability
	}
	if l.SourceFilter == "" {
		l.SourceFilter = mergeConf.SourceFilter
	}
	if l.RootDir == "" {
		l.RootDir = mergeConf.RootDir
	}
	if l.Seed == nil {
		l.Seed = mergeConf.Seed
	}
}

// BuildSubject is a simple helper for constructing a message subject from tokens
func BuildSubject(tokens []string, extraTokens []string) string {
	tokens = append(tokens, extraTokens...)
	return strings.Join(tokens, ".")
}

func (l *LeaseConfig) RequestConsumerSubject(appendTokens ...string) string {
	tokens := []string{
		l.StreamName,
		"*",
		"request",
		"*",
		"*",
	}

	return BuildSubject(tokens, appendTokens)
}

func (l *LeaseConfig) LeaseConsumerSubject() string {
	return fmt.Sprintf("%s.%s.notify.%s", l.StreamName, l.LeaseSubject, l.LeaseFilter)
}

func (l *LeaseConfig) SourceConsumerSubject() string {
	return fmt.Sprintf("%s.%s.notify.%s", l.StreamName, l.SourceSubject, l.SourceFilter)
}

func (l *LeaseConfig) LeaseMsgSubject(channel string, sequence string, msgId string, appendTokens ...string) string {
	tokens := []string{
		l.StreamName,
		l.LeaseSubject,
		channel,
		sequence,
		msgId,
	}

	return BuildSubject(tokens, appendTokens)
}

func (l *LeaseConfig) SourceMsgSubject(channel string, sequence string, msgId string) string {
	return fmt.Sprintf("%s.%s.%s.%s.%s", l.StreamName, l.SourceSubject, channel, sequence, msgId)
}

// RequestConsumerConfig returns a jetstream.ConsumerConfig for the requests
func (l *LeaseConfig) RequestConsumerConfig(appendTokens ...string) jetstream.ConsumerConfig {
	subject := l.RequestConsumerSubject(appendTokens...)
	durable := l.RequestDurability == Durable
	return l.consumerConfig("", durable, subject, l.RequestDeliverPolicy)
}

// LeaseConsumerConfig returns a jetstream.ConsumerConfig for the lease subject
func (l *LeaseConfig) LeaseConsumerConfig() jetstream.ConsumerConfig {
	subject := l.LeaseConsumerSubject()
	durable := l.LeaseDurability == Durable
	return l.consumerConfig("", durable, subject, l.LeaseDeliverPolicy)
}

// SourceConsumerConfig returns a jetstream.ConsumerConfig for the source subject
func (l *LeaseConfig) SourceConsumerConfig() jetstream.ConsumerConfig {
	subject := l.SourceConsumerSubject()
	durable := l.SourceDurability == Durable
	return l.consumerConfig(l.SourceConsumerName, durable, subject, l.SourceDeliverPolicy)
}

// consumerConfig creates a jetstream.ConsumerConfig with sensible defaults for use in a lease
//
// If name is empty, the subject will be normalised and used as the consumer name
// it will also be used as the durable name if durable = true
func (l *LeaseConfig) consumerConfig(name string, durable bool, subject string, deliverPolicy DeliverPolicy) jetstream.ConsumerConfig {
	if name == "" {
		nonAlphanumeric := regexp.MustCompile(`[^a-zA-Z0-9 ]+`)
		name = nonAlphanumeric.ReplaceAllString(subject, "")
	}

	jsDeliverPolicy := deliverPolicyMappings[deliverPolicy]

	conf := jetstream.ConsumerConfig{
		Name:          name,
		FilterSubject: subject,
		DeliverPolicy: jsDeliverPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    3,
	}

	if durable {
		conf.Durable = name
	}

	return conf
}
