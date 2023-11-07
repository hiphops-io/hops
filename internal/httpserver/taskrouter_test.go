package httpserver

// TODO: Move these e2e/multi-package tests into main

// import (
// 	"bytes"
// 	"context"
// 	"fmt"
// 	"net/http"
// 	"net/http/httptest"
// 	"os"
// 	"testing"

// 	"github.com/goccy/go-json"
// 	"github.com/hiphops-io/hops/dsl"
// 	"github.com/hiphops-io/hops/logs"
// 	"github.com/nats-io/nats.go/jetstream"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"

// 	undist "github.com/hiphops-io/hops/undistribute"
// )

// var testHopsConfig string = `
// task say_hello {}

// task say_greeting {
// 	param greeting {
// 		type = "text"
// 		required = true
// 	}
// }
// `

// func TestTriggerTask(t *testing.T) {
// 	type testCase struct {
// 		name             string
// 		taskName         string
// 		payload          string
// 		expectStatusCode int
// 		expectResponse   taskRunResponse
// 	}

// 	tests := []testCase{
// 		{
// 			name:             "Trigger task with no params",
// 			taskName:         "say_hello",
// 			payload:          "{}",
// 			expectStatusCode: 200,
// 			expectResponse: taskRunResponse{
// 				Message: "OK",
// 			},
// 		},
// 		{
// 			name:             "Trigger non-existent task",
// 			taskName:         "say_goodbye",
// 			payload:          "{}",
// 			expectStatusCode: 404,
// 			expectResponse: taskRunResponse{
// 				Message: "Not found",
// 			},
// 		},
// 		{
// 			name:             "Trigger task with invalid inputs",
// 			taskName:         "say_greeting",
// 			payload:          "{}",
// 			expectStatusCode: 400,
// 			expectResponse: taskRunResponse{
// 				Message: "Invalid inputs for say_greeting",
// 				Errors: map[string][]string{
// 					"greeting": {
// 						dsl.InvalidRequired,
// 					},
// 				},
// 			},
// 		},
// 	}

// 	// Let's set up the hops config...
// 	logger := logs.NoOpLogger()
// 	taskHops, err := initTaskHops(testHopsConfig, t)
// 	require.NoError(t, err, "Test setup: Hops config should be valid")

// 	// ...Then the router and test server...
// 	taskRouter := TaskRouter(taskHops, lease, logger)
// 	testServer := httptest.NewServer(taskRouter)
// 	defer testServer.Close()

// 	// ...and set the endpoint once
// 	urlTmpl := testServer.URL + "/%s"

// 	for _, tc := range tests {
// 		t.Run(tc.name, func(t *testing.T) {
// 			// Prep the request
// 			url := fmt.Sprintf(urlTmpl, tc.taskName)
// 			request, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(tc.payload))
// 			require.NoError(t, err, "Request should be created")

// 			// Make the request
// 			response, err := http.DefaultClient.Do(request)
// 			require.NoError(t, err, "Request should be successful")
// 			defer response.Body.Close()

// 			// Decode the response to check expected values
// 			var runResponse taskRunResponse
// 			err = json.NewDecoder(response.Body).Decode(&runResponse)
// 			// Remove event ID for comparison and check separately
// 			eventId := runResponse.SequenceID
// 			runResponse.SequenceID = ""

// 			assert.NoError(t, err, "Body should be valid JSON")
// 			assert.Equal(t, tc.expectStatusCode, response.StatusCode, "Status code should match")
// 			assert.Equal(t, tc.expectResponse, runResponse, "Response body should match")
// 			if tc.expectStatusCode == http.StatusOK {
// 				assert.Regexp(t, "[0-9a-f]{40}", eventId, "Event ID must be a valid SHA1 hash")
// 			}
// 		})
// 	}
// }

// func initTaskHops(content string, t *testing.T) (*dsl.HopAST, error) {
// 	ctx := context.Background()
// 	hopsHcl := initTmpHopsFile(content, t)
// 	return dsl.ParseHopsTasks(ctx, hopsHcl)
// }

// func initTmpHopsFile(content string, t *testing.T) dsl.HclFiles {
// 	// TODO: This is a duplicate of the createTmpHopsFile func in the dsl package
// 	// we move to a sensible location and consolidate
// 	dir := t.TempDir()
// 	f, err := os.CreateTemp(dir, "*")
// 	require.NoError(t, err)

// 	f.WriteString(content)

// 	hclFile, _, err := dsl.ReadHopsFiles(f.Name())
// 	require.NoError(t, err)

// 	return hclFile, hash
// }

// type LeaseMock struct {
// 	calledWith []map[string]string
// }

// func (l *LeaseMock) Publish(ctx context.Context, channel undist.Channel, sequenceId string, msgId string, data []byte, appendTokens ...string) (*jetstream.PubAck, error) {
// 	return nil, nil
// }

// func (l *LeaseMock) PublishSource(ctx context.Context, channel undist.Channel, sequenceId string, msgId string, data []byte) (*jetstream.PubAck, error) {
// 	return nil, nil
// }
