package httpserver

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog"

	"github.com/hiphops-io/hops/dsl"
	"github.com/hiphops-io/hops/nats"
)

type taskReader interface {
	ListTasks() []dsl.TaskAST
	GetTask(string) (dsl.TaskAST, error)
}

func TaskRouter(taskHops taskReader, natsClient NatsClient, logger zerolog.Logger) chi.Router {
	r := chi.NewRouter()
	controller := &taskController{
		logger:     logger,
		taskR:      taskHops,
		natsClient: natsClient,
	}
	r.Post("/{taskName}", controller.runTask)
	r.Get("/", controller.listTasks)

	return r
}

type taskRunResponse struct {
	Errors     map[string][]string `json:"errors"`
	Message    string              `json:"message"`
	SequenceID string              `json:"sequence_id"`
	statusCode int
}

type taskController struct {
	taskR      taskReader
	logger     zerolog.Logger
	natsClient NatsClient
}

func (c *taskController) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks := c.taskR.ListTasks()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (c *taskController) runTask(w http.ResponseWriter, r *http.Request) {
	runResponse := taskRunResponse{}

	taskName := chi.URLParam(r, "taskName")
	if taskName == "" {
		runResponse.statusCode = http.StatusBadRequest
		runResponse.Message = "Task name is required"
		c.writeTaskRunResponse(w, runResponse)
		return
	}

	var taskInput map[string]any
	err := json.NewDecoder(r.Body).Decode(&taskInput)
	if err != nil {
		runResponse.statusCode = http.StatusBadRequest
		runResponse.Message = "Unable to parse payload JSON"
		c.writeTaskRunResponse(w, runResponse)
		return
	}

	task, err := c.taskR.GetTask(taskName)
	if err != nil {
		runResponse.statusCode = http.StatusNotFound
		runResponse.Message = "Not found"
		c.writeTaskRunResponse(w, runResponse)
		return
	}

	// Validate the input
	validationMessages := task.ValidateInput(taskInput)
	if len(validationMessages) > 0 {
		runResponse.statusCode = http.StatusBadRequest
		runResponse.Message = fmt.Sprintf("Invalid inputs for %s", task.Name)
		runResponse.Errors = validationMessages
		c.writeTaskRunResponse(w, runResponse)
		return
	}

	// Build a source event
	sourceEvent, sequenceID, err := dsl.CreateSourceEvent(taskInput, "hiphops", "task", task.Name)
	if err != nil {
		runResponse.statusCode = http.StatusInternalServerError
		runResponse.Message = "Unable to create event"
		c.writeTaskRunResponse(w, runResponse)
		return
	}

	// Push the event message to the topic, including the hash as sequence ID and "event" as event ID
	_, err = c.natsClient.Publish(r.Context(), sourceEvent, nats.ChannelNotify, sequenceID, "event")
	if err != nil {
		runResponse.statusCode = http.StatusInternalServerError
		runResponse.Message = fmt.Sprintf("Unable to publish event: %s", err.Error())
		c.writeTaskRunResponse(w, runResponse)
		return
	}

	runResponse.statusCode = http.StatusOK
	runResponse.Message = "OK"
	runResponse.SequenceID = sequenceID
	c.writeTaskRunResponse(w, runResponse)
}

func (c *taskController) writeTaskRunResponse(w http.ResponseWriter, runResponse taskRunResponse) {
	// We only explicitly write non-200 status codes. This allows us to
	// properly convey failed encoding to end users without sending headers twice.
	isBadStatus := runResponse.statusCode != http.StatusOK

	w.Header().Set("Content-Type", "application/json")

	if isBadStatus {
		w.WriteHeader(runResponse.statusCode)
		c.logger.Error().Msg(runResponse.Message)
	}

	err := json.NewEncoder(w).Encode(runResponse)
	if err != nil {
		c.logger.Error().Err(err).Msg("Error encoding task response")

		// A bad status will already have been written, so we'll default to that
		if !isBadStatus {
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte(`{"message":"Internal server error"}`))
		return
	}
}
