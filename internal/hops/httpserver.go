package hops

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog"

	"github.com/hiphops-io/hops/dsl"
	"github.com/hiphops-io/hops/logs"
	"github.com/hiphops-io/hops/nats"
)

type (
	HTTPServer struct {
		hopsFiles      *dsl.HopsFiles
		hopsFileLoader *HopsFileLoader
		logger         zerolog.Logger
		mu             sync.RWMutex
		natsClient     *nats.Client
		server         *http.Server
		taskHops       *dsl.HopAST
		updatedAt      int64
	}

	taskRunResponse struct {
		Errors     map[string][]string `json:"errors"`
		Message    string              `json:"message"`
		SequenceID string              `json:"sequence_id"`
		statusCode int
	}
)

func NewHTTPServer(addr string, hopsFileLoader *HopsFileLoader, natsClient *nats.Client, logger zerolog.Logger) (*HTTPServer, error) {
	h := &HTTPServer{hopsFileLoader: hopsFileLoader, logger: logger, natsClient: natsClient}

	err := h.Reload(context.Background())
	if err != nil {
		return nil, err
	}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RedirectSlashes)
	r.Use(logs.AccessLogMiddleware(logger)) // TODO: Make logging less verbose for static/frontend requests
	r.Use(Healthcheck(natsClient, "/health"))
	// TODO: Make CORS configurable and lock down by default. As-is it could be
	// insecure for production/deployed use.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/updated-at", h.getUpdatedAt)

	// Serve the single page app for the console from the UI dir
	r.Mount("/console", ConsoleRouter(logger))

	// Serve the tasks API
	r.Route("/tasks", func(r chi.Router) {
		r.Post("/{taskName}", h.runTask)
		r.Get("/", h.listTasks)
	})

	// Serve the events API
	r.Mount("/events", EventRouter(natsClient, logger))

	h.server = &http.Server{
		Addr:    addr,
		Handler: r,
	}

	return h, nil
}

func (h *HTTPServer) Reload(ctx context.Context) error {
	hopsFiles, err := h.hopsFileLoader.Get()
	if err != nil {
		return err
	}
	// Serve the tasks API
	taskHops, err := dsl.ParseHopsTasks(ctx, hopsFiles)
	if err != nil {
		return err
	}

	h.mu.Lock()
	h.hopsFiles = hopsFiles
	h.taskHops = taskHops
	h.updatedAt = time.Now().UnixMicro()
	h.mu.Unlock()

	return nil
}

func (h *HTTPServer) Serve() error {
	h.logger.Info().Msgf("Console available on http://%s/console", h.server.Addr)
	return h.server.ListenAndServe()
}

func (h *HTTPServer) Shutdown(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

func (h *HTTPServer) getUpdatedAt(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	updatedAt := h.updatedAt
	h.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedAt)
}

func (h *HTTPServer) listTasks(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	tasks := h.taskHops.ListTasks()
	h.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (h *HTTPServer) runTask(w http.ResponseWriter, r *http.Request) {
	runResponse := taskRunResponse{}

	taskName := chi.URLParam(r, "taskName")
	if taskName == "" {
		runResponse.statusCode = http.StatusBadRequest
		runResponse.Message = "Task name is required"
		h.writeTaskRunResponse(w, runResponse)
		return
	}

	var taskInput map[string]any
	err := json.NewDecoder(r.Body).Decode(&taskInput)
	if err != nil {
		runResponse.statusCode = http.StatusBadRequest
		runResponse.Message = "Unable to parse payload JSON"
		h.writeTaskRunResponse(w, runResponse)
		return
	}

	h.mu.RLock()
	task, err := h.taskHops.GetTask(taskName)
	h.mu.RUnlock()

	if err != nil {
		runResponse.statusCode = http.StatusNotFound
		runResponse.Message = "Not found"
		h.writeTaskRunResponse(w, runResponse)
		return
	}

	// Validate the input
	validationMessages := task.ValidateInput(taskInput)
	if len(validationMessages) > 0 {
		runResponse.statusCode = http.StatusBadRequest
		runResponse.Message = fmt.Sprintf("Invalid inputs for %s", task.Name)
		runResponse.Errors = validationMessages
		h.writeTaskRunResponse(w, runResponse)
		return
	}

	// Build a source event
	sourceEvent, sequenceID, err := dsl.CreateSourceEvent(taskInput, "hiphops", "task", task.Name)
	if err != nil {
		runResponse.statusCode = http.StatusInternalServerError
		runResponse.Message = "Unable to create event"
		h.writeTaskRunResponse(w, runResponse)
		return
	}

	// Push the event message to the topic, including the hash as sequence ID and "event" as event ID
	_, _, err = h.natsClient.Publish(r.Context(), sourceEvent, nats.ChannelNotify, sequenceID, "event")
	if err != nil {
		runResponse.statusCode = http.StatusInternalServerError
		runResponse.Message = fmt.Sprintf("Unable to publish event: %s", err.Error())
		h.writeTaskRunResponse(w, runResponse)
		return
	}

	runResponse.statusCode = http.StatusOK
	runResponse.Message = "OK"
	runResponse.SequenceID = sequenceID
	h.writeTaskRunResponse(w, runResponse)
}

func (h *HTTPServer) writeTaskRunResponse(w http.ResponseWriter, runResponse taskRunResponse) {
	// We only explicitly write non-200 status codes. This allows us to
	// properly convey failed encoding to end users without sending headers twice.
	isBadStatus := runResponse.statusCode != http.StatusOK

	w.Header().Set("Content-Type", "application/json")

	if isBadStatus {
		w.WriteHeader(runResponse.statusCode)
		h.logger.Error().Msg(runResponse.Message)
	}

	err := json.NewEncoder(w).Encode(runResponse)
	if err != nil {
		h.logger.Error().Err(err).Msg("Error encoding task response")

		// A bad status will already have been written, so we'll default to that
		if !isBadStatus {
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte(`{"message":"Internal server error"}`))
		return
	}
}
