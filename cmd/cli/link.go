package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/cli/browser"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type (
	LinkServer struct {
		server *echo.Echo
	}

	CredsRequest struct {
		Creds string `json:"creds" validate:"required"`
	}

	EchoValidator struct {
		validator *validator.Validate
	}

	LinkCmd struct {
		Dir string `arg:"positional" default:"." help:"path to Hiphops dir - defaults to current directory"`
		// TODO: Accept an optional output path argument to write the creds to
	}
)

func (l *LinkCmd) Run() error {
	fmt.Println("Linking to hiphops.io")

	creds, err := l.getCreds()
	if err != nil {
		return err
	}

	// TODO:
	// Restart things if required after writing creds - might be easier to have
	// them restart on their own when files change

	credsPath := filepath.Join(l.Dir, ".hiphops", "hiphops-io.creds")
	os.WriteFile(credsPath, []byte(creds), 0644)

	return nil
}

func (l *LinkCmd) getCreds() (string, error) {
	urlStr := "https://stage-app.hiphops.io/"
	// Note: We should move this into a config struct at the top level for cmds
	if value, ok := os.LookupEnv("HIPHOPS_WEB_URL"); ok {
		urlStr = value
	}

	hiphopsURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	link, err := NewLinkServer()
	if err != nil {
		return "", err
	}

	return link.LinkAccount(hiphopsURL)
}

func NewLinkServer() (*LinkServer, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, err
	}

	e := echo.New()
	e.HideBanner = true
	e.Validator = NewEchoValidator()
	e.Listener = listener
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:*", "http://0.0.0.0:*", "https://*.hiphops.io"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{echo.HeaderAccept, echo.HeaderAuthorization, echo.HeaderContentType, echo.HeaderXCSRFToken},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	return &LinkServer{server: e}, nil
}

// LinkAccount starts a server listening on an ephemeral endpoint for credentials, shutting the server down when done
//
// Note: The server will listen for at most 1 minute before timing out and shutting down, returning empty creds
func (l *LinkServer) LinkAccount(hiphopsURL *url.URL) (string, error) {
	ctx := context.Background()
	oneTimePath := fmt.Sprintf("/creds/%s", uuid.NewString())
	doneChan := make(chan struct{})
	var creds string

	l.server.POST(oneTimePath, func(c echo.Context) (err error) {
		// Single use endpoint, so we always signal done after any request
		defer func() {
			doneChan <- struct{}{}
		}()

		credsReq := new(CredsRequest)
		if err = c.Bind(credsReq); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if err = c.Validate(credsReq); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		creds = credsReq.Creds

		return c.JSON(http.StatusOK, struct{}{})
	})

	// TODO: Check if I actually need the address here, given it's present on listener
	go l.server.Start("0.0.0.0:0")
	defer l.server.Shutdown(ctx)

	signinURL, err := signinURLString(hiphopsURL, l.server.Listener.Addr().(*net.TCPAddr).Port, oneTimePath)
	if err != nil {
		return "", err
	}

	// Open Hiphops.io in the browser for the user to sign in and get creds.
	err = browser.OpenURL(signinURL)
	if err != nil {
		return "", err
	}

	select {
	case <-doneChan:
		var err error = nil
		if creds == "" {
			err = errors.New("received empty credentials")
		}

		return creds, err
	case <-time.After(1 * time.Minute):
		return "", errors.New("timed out waiting for link response")
	}
}

func NewEchoValidator() *EchoValidator {
	return &EchoValidator{
		validator: validator.New(),
	}
}

func (ev *EchoValidator) Validate(i interface{}) error {
	if err := ev.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

// signinURLString constructs the sign in URL and params for the user to open in a browser
func signinURLString(hiphopsURL *url.URL, listenPort int, listenPath string) (string, error) {
	signinURL := hiphopsURL.JoinPath("link")

	callbackURL, err := url.Parse(fmt.Sprintf("http://localhost:%d", listenPort))
	if err != nil {
		return "", err
	}

	params := url.Values{
		"callback": []string{callbackURL.JoinPath(listenPath).String()},
	}

	return fmt.Sprintf("%s?%s", signinURL.String(), params.Encode()), nil
}
