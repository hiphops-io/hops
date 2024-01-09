// Creates/manages an in-process NATS server
//
// Currently this is used as part of the hops test suite, but in future it will be
// leveraged to enable user-side developers to run tests on Hiphops pipelines.
package nats

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// LocalServer is an in-process hiphops.io style NATS server instance
// created from a NATS config file.
type LocalServer struct {
	NatsServer *server.Server
	ServerOpts *server.Options
}

// NewLocalServer starts an in-process nats server from a config file
//
// LocalServer.Close() should be called when finished with the server
func NewLocalServer(natsConfigPath string, dataDir string, debug bool, logger server.Logger) (*LocalServer, error) {
	localNats := &LocalServer{}

	err := localNats.initServerOpts(natsConfigPath, dataDir)
	if err != nil {
		return nil, err
	}

	err = localNats.initServer(debug, logger)
	if err != nil {
		return nil, err
	}

	err = localNats.initJetstreamSetup()
	if err != nil {
		localNats.Close()
		return nil, err
	}

	return localNats, nil
}

func (l *LocalServer) AuthUrl(accountName string) (string, error) {
	user, err := l.User(accountName)
	if err != nil {
		return "", err
	}

	clientUrl := l.NatsServer.ClientURL()
	baseUrl, err := url.Parse(clientUrl)
	if err != nil {
		return "", err
	}

	baseUrl.User = url.UserPassword(user.Username, user.Password)

	return baseUrl.String(), nil
}

// Close shuts down the local nats server, waiting until shutdown is complete
func (l *LocalServer) Close() {
	l.NatsServer.Shutdown()
	l.NatsServer.WaitForShutdown()
}

// Connect establishes a client connection with the local nats server
func (l *LocalServer) Connect(accountName string) (*nats.Conn, error) {
	natsurl, err := l.AuthUrl(accountName)
	if err != nil {
		return nil, err
	}

	return nats.Connect(natsurl)
}

func (l *LocalServer) User(accountName string) (*server.User, error) {
	var user *server.User

	for _, u := range l.ServerOpts.Users {
		if accountName != "" && u.Account.Name == accountName {
			user = u
			break
		}

		// Effectively we default the account name to any non-HIPHOPS account
		if u.Account.Name != "HIPHOPS" {
			user = u
			break
		}
	}

	if user == nil {
		return nil, errors.New("Unable to find user in NATS config options")
	}

	return user, nil
}

// initJetstreamSetup creates the account's jetstream resources on the NATS server
func (l *LocalServer) initJetstreamSetup() error {
	ctx := context.Background()

	nc, err := l.Connect("")
	if nc != nil {
		defer nc.Drain()
	}
	if err != nil {
		return err
	}

	js, err := jetstream.New(nc)
	if err != nil {
		l.Close()
		return err
	}

	user, err := l.User("")
	if err != nil {
		l.Close()
		return err
	}

	// Create the account stream
	streamConf := jetstream.StreamConfig{
		Name: user.Account.Name,
		Subjects: []string{
			fmt.Sprintf("%s.>", user.Account.Name),
		},
		Discard:              jetstream.DiscardNew,
		DiscardNewPerSubject: true,
		MaxMsgsPerSubject:    1,
	}
	stream, err := js.CreateStream(ctx, streamConf)
	if err != nil {
		l.Close()
		return err
	}

	// Create the server consumer
	consumerConf := jetstream.ConsumerConfig{
		Name:          fmt.Sprintf("%s-%s", ConsumerBaseName(user.Account.Name, DefaultInterestTopic), ChannelNotify),
		FilterSubject: NotifyFilterSubject(user.Account.Name, DefaultInterestTopic),
		DeliverPolicy: jetstream.DeliverNewPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    3,
	}
	_, err = stream.CreateOrUpdateConsumer(ctx, consumerConf)
	if err != nil {
		l.Close()
		return err
	}

	// Create the request consumer
	requestConsumerConf := jetstream.ConsumerConfig{
		Name:          fmt.Sprintf("%s-%s", ConsumerBaseName(user.Account.Name, DefaultInterestTopic), ChannelRequest),
		FilterSubject: RequestFilterSubject(user.Account.Name, DefaultInterestTopic),
		DeliverPolicy: jetstream.DeliverNewPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    3,
	}
	_, err = stream.CreateOrUpdateConsumer(ctx, requestConsumerConf)
	if err != nil {
		l.Close()
		return err
	}

	return nil
}

func (l *LocalServer) initServer(debug bool, logger server.Logger) error {
	server, err := server.NewServer(l.ServerOpts)
	if err != nil {
		return err
	}

	server.SetLoggerV2(logger, debug, debug, debug)

	go server.Start()

	if !server.ReadyForConnections(5 * time.Second) {
		return errors.New("NATS server didn't become ready")
	}

	l.NatsServer = server
	return nil
}

func (l *LocalServer) initServerOpts(natsConfigPath string, dataDir string) error {
	opts, err := server.ProcessConfigFile(natsConfigPath)
	if err != nil {
		return err
	}

	if opts.StoreDir == "" {
		opts.StoreDir = dataDir
	}

	l.ServerOpts = opts
	return nil
}
