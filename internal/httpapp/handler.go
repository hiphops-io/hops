package httpapp

// import (
// 	"context"
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"strings"
// 	"time"

// 	"github.com/go-playground/validator/v10"
// 	"github.com/goccy/go-json"
// 	"github.com/hashicorp/go-retryablehttp"
// 	"github.com/nats-io/nats.go/jetstream"
// 	"github.com/rs/zerolog"

// 	"github.com/hiphops-io/hops/nats"
// 	"github.com/hiphops-io/hops/worker"
// )

// type (
// 	DoInput struct {
// 		JSON    interface{}       `json:"json"`
// 		Method  string            `json:"method" validate:"required,oneof=GET HEAD POST PUT DELETE OPTIONS PATCH"`
// 		Params  map[string]string `json:"params"`
// 		Data    []byte            `json:"data"`
// 		Retries int               `json:"retries" validate:"gte=0,lte=3"`
// 		Headers map[string]string `json:"headers"`
// 		URL     string            `json:"url" validate:"required,http_url"`
// 	}

// 	doJob struct {
// 		msg             jetstream.Msg
// 		startedAt       time.Time
// 		responseSubject string
// 	}

// 	HTTPHandler struct {
// 		logger     zerolog.Logger
// 		natsClient *nats.Client
// 		doJobCh    chan doJob
// 	}
// )

// // NewDoInput creates a DoInput object from a json object given as bytes
// func NewDoInput(data []byte) (*DoInput, error) {
// 	doInput := &DoInput{}

// 	// Unmarshal the payload
// 	err := json.Unmarshal(data, &doInput)
// 	if err != nil {
// 		return doInput, fmt.Errorf("Unable to parse input: %w", err)
// 	}
// 	if doInput.Method == "" {
// 		doInput.Method = "GET"
// 	}
// 	doInput.Method = strings.ToUpper(doInput.Method)

// 	// Run field validation
// 	valid := validator.New()
// 	err = valid.Struct(doInput)
// 	if err != nil {
// 		return doInput, fmt.Errorf("Invalid input: %w", err)
// 	}

// 	return doInput, nil
// }

// func (d *DoInput) HTTPRequest() (*retryablehttp.Request, error) {
// 	var body interface{}
// 	switch {
// 	case d.JSON != nil:
// 		jsonBody, err := json.Marshal(d.JSON)
// 		if err != nil {
// 			return nil, fmt.Errorf("Invalid JSON payload: %w", err)
// 		}
// 		body = jsonBody
// 	case d.Data != nil:
// 		body = []byte(d.Data)
// 	default:
// 		body = nil
// 	}

// 	request, err := retryablehttp.NewRequest(d.Method, d.URL, body)
// 	if err != nil {
// 		return nil, fmt.Errorf("Preparing request failed: %w", err)
// 	}

// 	q := request.URL.Query()
// 	for k, v := range d.Params {
// 		q.Add(k, v)
// 	}
// 	request.URL.RawQuery = q.Encode()

// 	for k, v := range d.Headers {
// 		request.Header.Add(k, v)
// 	}

// 	return request, nil
// }

// func NewHTTPHandler(ctx context.Context, natsClient *nats.Client, logger zerolog.Logger) (*HTTPHandler, error) {
// 	h := &HTTPHandler{
// 		natsClient: natsClient,
// 		logger:     logger,
// 		doJobCh:    make(chan doJob, 100),
// 	}

// 	go h.doWorker(ctx, h.doJobCh)

// 	return h, nil
// }

// func (h *HTTPHandler) AppName() string {
// 	return "http"
// }

// func (h *HTTPHandler) Do(ctx context.Context, msg jetstream.Msg) error {
// 	startedAt := time.Now()

// 	parsedMsg, err := nats.Parse(msg)
// 	if err != nil {
// 		return fmt.Errorf("Unable to parse response subject from message %s", msg.Subject())
// 	}

// 	job := doJob{
// 		msg:             msg,
// 		startedAt:       startedAt,
// 		responseSubject: parsedMsg.ResponseSubject(),
// 	}

// 	h.doJobCh <- job

// 	return nil
// }

// func (h *HTTPHandler) Handlers() map[string]worker.Handler {
// 	handlers := map[string]worker.Handler{}
// 	handlers["do"] = h.Do
// 	return handlers
// }

// func (h *HTTPHandler) doWorker(ctx context.Context, jobs chan doJob) {
// 	for do := range jobs {
// 		var result interface{}
// 		resultMsg, err := h.handleDoJob(do)
// 		if resultMsg != nil {
// 			result = *resultMsg
// 		}

// 		err, _ = h.natsClient.PublishResult(ctx, do.startedAt, result, err, do.responseSubject)
// 		if err != nil {
// 			h.logger.Error().Err(err).Msgf("Unable to publish result to: %s", do.responseSubject)
// 		}
// 	}
// }

// func (h *HTTPHandler) handleDoJob(do doJob) (*nats.ResultMsg, error) {
// 	startedAt := do.startedAt
// 	msg := do.msg

// 	doInput, err := NewDoInput(msg.Data())
// 	if err != nil {
// 		return nil, err
// 	}

// 	request, err := doInput.HTTPRequest()
// 	if err != nil {
// 		return nil, err
// 	}

// 	httpC := httpClient(doInput.Retries)

// 	resp, err := httpC.Do(request)
// 	if resp != nil {
// 		defer resp.Body.Close()
// 	}
// 	if err != nil {
// 		return nil, fmt.Errorf("Request failed: %w", err)
// 	}

// 	return createResult(startedAt, resp)
// }

// func createResult(startedAt time.Time, response *http.Response) (*nats.ResultMsg, error) {
// 	resultMsg := nats.NewResultMsg(startedAt, nil, nil)

// 	responseBody, err := io.ReadAll(response.Body)
// 	if err != nil {
// 		return nil, fmt.Errorf("Unable to read response: %w", err)
// 	}

// 	var jsonData interface{}
// 	if isJsonContent(response) {
// 		err := json.Unmarshal(responseBody, &jsonData)
// 		if err != nil {
// 			jsonData = nil
// 		}
// 	}

// 	headers := map[string]string{}
// 	for k, v := range response.Header {
// 		headers[k] = strings.Join(v, ", ")
// 	}

// 	resultMsg.Headers = headers
// 	resultMsg.Body = string(responseBody)
// 	resultMsg.JSON = jsonData
// 	resultMsg.StatusCode = response.StatusCode
// 	resultMsg.URL = response.Request.URL.String()

// 	return &resultMsg, nil
// }

// // httpClient creates a new retryable http client with default config
// func httpClient(retries int) *retryablehttp.Client {
// 	httpC := retryablehttp.NewClient()
// 	httpC.RetryMax = retries
// 	httpC.HTTPClient.Timeout = time.Second * 10
// 	httpC.Logger = nil

// 	return httpC
// }

// func isJsonContent(response *http.Response) bool {
// 	contentType := response.Header.Get("Content-Type")
// 	if contentType == "" {
// 		return false
// 	}
// 	mediaType := strings.TrimSpace(strings.Split(contentType, ";")[0])

// 	if strings.ToLower(mediaType) == "application/json" {
// 		return true
// 	}

// 	return false
// }
