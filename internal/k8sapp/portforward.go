package k8sapp

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/klog/v2"
)

type PortForwarder struct {
	portForwarder *portforward.PortForwarder
	stopChan      chan struct{}
	readyChan     chan struct{}
	errChan       chan error
	localPort     int
}

func (p *PortForwarder) start() error {
	go func() {
		p.errChan <- p.portForwarder.ForwardPorts()
	}()

	readyTimer := time.NewTimer(5 * time.Second)

	select {
	case <-p.readyChan:
		readyTimer.Stop()

		ports, err := p.portForwarder.GetPorts()
		if err != nil {
			close(p.stopChan)
			return err
		}

		for _, port := range ports {
			p.localPort = int(port.Local)
			return nil
		}

		close(p.stopChan)
		return errors.New("No ports found for port forwarder")

	case <-readyTimer.C:
		close(p.stopChan)
		return errors.New("Port forward failed to become ready in time")
	}
}

type PortForward struct {
	cfg     *rest.Config
	hostIP  string
	streams genericclioptions.IOStreams
}

type LogErrorWriter struct {
	logger zerolog.Logger
}

func (l LogErrorWriter) Write(p []byte) (n int, err error) {
	l.logger.Error().Msg(string(p))

	return len(p), nil
}

func NewPortForward(cfg *rest.Config, logger zerolog.Logger) *PortForward {
	errLog := LogErrorWriter{logger: logger}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

	zerologr.NameFieldName = "logger"
	zerologr.NameSeparator = "/"
	zerologr.SetMaxV(1)

	zl := logger
	zl = zl.With().Caller().Timestamp().Logger()
	var log logr.Logger = zerologr.New(&zl)

	klog.LogToStderr(false)
	klog.SetLogger(log)
	pFwd := &PortForward{
		cfg:    cfg,
		hostIP: strings.TrimLeft(cfg.Host, "htps:/"),
		streams: genericiooptions.IOStreams{
			In:     os.Stdin,
			Out:    io.Discard,
			ErrOut: errLog,
		},
	}

	return pFwd
}

func (p *PortForward) ForwardPodPort(pod *corev1.Pod, podPort int) (int, func(), error) {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", pod.Namespace, pod.Name)
	podUrl := &url.URL{Scheme: "https", Path: path, Host: p.hostIP}
	portMapping := fmt.Sprintf("%d:%d", 0, podPort)

	pFwder, err := p.newPortForwarder(podUrl, portMapping)
	if err != nil {
		return 0, nil, err
	}

	err = pFwder.start()
	if err != nil {
		return 0, nil, err
	}

	go func() {
		select {
		case <-pFwder.stopChan:
			break
		case <-pFwder.errChan:
			close(pFwder.stopChan)
			break
		}
	}()

	close := func() {
		pFwder.stopChan <- struct{}{}
	}

	return pFwder.localPort, close, nil
}

func (p *PortForward) newPortForwarder(
	fwdUrl *url.URL,
	portMapping string,
) (*PortForwarder, error) {
	transport, upgrader, err := spdy.RoundTripperFor(p.cfg)
	if err != nil {
		return nil, err
	}

	// Create the dialer for the pod URL
	httpClient := &http.Client{Transport: transport}
	dialer := spdy.NewDialer(upgrader, httpClient, http.MethodPost, fwdUrl)

	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})
	errChan := make(chan error)

	// Create the portforwarder itself
	fw, err := portforward.New(
		dialer,
		[]string{portMapping},
		stopChan,
		readyChan,
		p.streams.Out,
		p.streams.ErrOut,
	)
	if err != nil {
		return nil, err
	}

	pFwder := &PortForwarder{
		portForwarder: fw,
		stopChan:      stopChan,
		readyChan:     readyChan,
		errChan:       errChan,
	}

	return pFwder, nil
}
