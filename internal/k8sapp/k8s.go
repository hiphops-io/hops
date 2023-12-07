package k8sapp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hiphops-io/hops/nats"
	"github.com/hiphops-io/hops/worker"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	toolsWatch "k8s.io/client-go/tools/watch"
)

const (
	sidecarName          = "hiphops-sidecar"
	sidecarPort          = int32(8917)
	sidecarCpuLimit      = "250m"
	sidecarMemLimit      = "128Mi"
	responseEnvVarName   = "HIPHOPS_RESPONSE_SUBJECT"
	resultPathEnvVarName = "HIPHOPS_RESULT_PATH"
	resultVolumeName     = "hiphops"
	skipCleanupLabel     = "hiphops-io/skip_cleanup"
	runUUIDLabel         = "hiphops-io/uuid"
)

type (
	K8sHandler struct {
		clientset           *kubernetes.Clientset
		cfg                 *rest.Config
		logger              zerolog.Logger
		natsClient          *nats.Client
		portForward         *PortForward
		requiresPortForward bool
	}

	RunPodInput struct {
		Image       string            `json:"image" validate:"required"`
		Command     []string          `json:"command,omitempty"`
		Args        []string          `json:"args,omitempty"`
		Namespace   string            `json:"namespace,omitempty" validate:"omitempty,hostname_rfc1123"`
		InDir       string            `json:"in_dir,omitempty" validate:"omitempty,dirpath,min=1"`
		InFiles     map[string]string `json:"in_files,omitempty" validate:"omitempty,dive,keys,filepath,min=1,endkeys"`
		OutDir      string            `json:"out_dir,omitempty" validate:"omitempty,dirpath,min=1"`
		ResultPath  string            `json:"out_path,omitempty" validate:"omitempty,filepath,min=1"`
		CpuLimit    string            `json:"cpu,omitempty"`
		MemLimit    string            `json:"memory,omitempty"`
		SkipCleanup bool              `json:"skip_cleanup"`
	}
)

func NewK8sHandler(
	ctx context.Context,
	natsClient *nats.Client,
	kubeConfPath string,
	requiresPortForward bool,
	logger zerolog.Logger,
) (*K8sHandler, error) {
	k := &K8sHandler{
		natsClient:          natsClient,
		logger:              logger.With().Str("from", "k8s-handler").Logger(),
		requiresPortForward: requiresPortForward,
	}

	err := k.initKubeClient(kubeConfPath)
	if err != nil && errors.Is(err, clientcmd.ErrEmptyConfig) {
		k.logger.Info().Msg("No Kubernetes config provided or found. K8s worker will now stop.")
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	k.portForward = NewPortForward(k.cfg, logger)

	go k.watchPods(ctx)

	return k, nil
}

func (k *K8sHandler) AppName() string {
	return "k8s"
}

func (k *K8sHandler) Handlers() map[string]worker.Handler {
	handlers := map[string]worker.Handler{}
	handlers["run"] = k.RunPod
	return handlers
}

func (k *K8sHandler) RunPod(ctx context.Context, msg jetstream.Msg) error {
	runPodInput, err := k.parseMsgInput(msg)
	if err != nil {
		return fmt.Errorf("Invalid input: %w", err)
	}
	cpuLimit, err := resource.ParseQuantity(runPodInput.CpuLimit)
	if err != nil {
		return fmt.Errorf("Failed to parse CPU limit: %w", err)
	}
	memLimit, err := resource.ParseQuantity(runPodInput.MemLimit)
	if err != nil {
		return fmt.Errorf("Failed to parse memory limit: %w", err)
	}

	parsedMsg, err := nats.Parse(msg)
	if err != nil {
		return fmt.Errorf("Unable to parse response subject from message %s", msg.Subject())
	}

	pods := k.clientset.CoreV1().Pods(runPodInput.Namespace)
	id := uuid.NewString()

	labels := map[string]string{
		"hiphops-io/for":         "workflow-step",
		"hiphops-io/sequence_id": parsedMsg.SequenceId,
		runUUIDLabel:             id,
		skipCleanupLabel:         strconv.FormatBool(runPodInput.SkipCleanup),
	}

	podSpec := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "hiphops-run-",
			Namespace:    runPodInput.Namespace,
			Labels:       labels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  "main",
					Image: runPodInput.Image,
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							"cpu":    cpuLimit,
							"memory": memLimit,
						},
						Requests: corev1.ResourceList{
							"cpu":    cpuLimit,
							"memory": memLimit,
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "HIPHOPS_OUTPUT_DIR",
							Value: runPodInput.OutDir,
						},
						{
							Name:  "HIPHOPS_INPUT_DIR",
							Value: runPodInput.InDir,
						},
						{
							Name:  resultPathEnvVarName,
							Value: runPodInput.ResultPath,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resultVolumeName,
							MountPath: runPodInput.OutDir,
						},
					},
				},
				{
					Name:  sidecarName,
					Image: "halverneus/static-file-server:v1.8.10",
					Ports: []corev1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: sidecarPort,
						},
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							"cpu":    resource.MustParse(sidecarCpuLimit),
							"memory": resource.MustParse(sidecarMemLimit),
						},
						Requests: corev1.ResourceList{
							"cpu":    resource.MustParse(sidecarCpuLimit),
							"memory": resource.MustParse(sidecarMemLimit),
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  responseEnvVarName,
							Value: parsedMsg.ResponseSubject(),
						},
						{
							Name:  "PORT",
							Value: strconv.Itoa(int(sidecarPort)),
						},
						{
							Name:  "FOLDER",
							Value: "/output",
						},
						{
							Name:  "CORS",
							Value: "false",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resultVolumeName,
							MountPath: "/output",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: resultVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}

	if len(runPodInput.Command) > 0 {
		podSpec.Spec.Containers[0].Command = runPodInput.Command
	}
	if len(runPodInput.Args) > 0 {
		podSpec.Spec.Containers[0].Args = runPodInput.Args
	}

	if len(runPodInput.InFiles) > 0 {
		err = k.createPodFiles(ctx, podSpec, labels, runPodInput.InDir, runPodInput.InFiles)
		if err != nil {
			return err
		}
	}

	_, err = pods.Create(ctx, podSpec, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// createPodFiles creates configmaps from strings and mounts as files in the given pod
//
// This will modify the original pod object, mounting created configmaps as volumes
func (k *K8sHandler) createPodFiles(ctx context.Context, podSpec *corev1.Pod, labels map[string]string, dir string, files map[string]string) error {
	mountName := "hiphops-input"

	inputConfigMapSpec := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "hiphops-run-",
			Namespace:    podSpec.GetNamespace(),
			Labels:       labels,
		},
		Data: files,
	}

	cm := k.clientset.CoreV1().ConfigMaps(podSpec.Namespace)
	configMap, err := cm.Create(ctx, inputConfigMapSpec, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	volume := corev1.Volume{
		Name: mountName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMap.Name,
				},
			},
		},
	}

	volumeMount := corev1.VolumeMount{
		Name:      mountName,
		MountPath: dir,
	}

	podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, volume)
	podSpec.Spec.Containers[0].VolumeMounts = append(podSpec.Spec.Containers[0].VolumeMounts, volumeMount)

	return nil
}

func (k *K8sHandler) parseMsgInput(msg jetstream.Msg) (RunPodInput, error) {
	defaultNamespace := "default"
	defaultInDir := "/input"
	defaultOutDir := "/output"
	defaultResultPath := "result.json"
	defaultTaskCpuLimit := sidecarCpuLimit
	defaultTaskMemLimit := sidecarMemLimit

	var runPodInput RunPodInput

	valid := validator.New()

	err := json.Unmarshal(msg.Data(), &runPodInput)
	if err != nil {
		return runPodInput, err
	}

	err = valid.Struct(runPodInput)
	if err != nil {
		return RunPodInput{}, err
	}

	if runPodInput.Namespace == "" {
		runPodInput.Namespace = defaultNamespace
	}
	if runPodInput.InDir == "" {
		runPodInput.InDir = defaultInDir
	}
	if runPodInput.OutDir == "" {
		runPodInput.OutDir = defaultOutDir
	}
	if runPodInput.ResultPath == "" {
		runPodInput.ResultPath = defaultResultPath
	}
	if runPodInput.CpuLimit == "" {
		runPodInput.CpuLimit = defaultTaskCpuLimit
	}
	if runPodInput.MemLimit == "" {
		runPodInput.MemLimit = defaultTaskMemLimit
	}

	return runPodInput, nil
}

func (k *K8sHandler) watchPods(ctx context.Context) {
	timeOut := int64(60)

	listOpts := metav1.ListOptions{
		TimeoutSeconds: &timeOut,
		LabelSelector:  "hiphops-io/for=workflow-step",
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		return k.clientset.CoreV1().Pods("").Watch(ctx, listOpts)
	}

	watcher, _ := toolsWatch.NewRetryWatcher("1", &cache.ListWatch{WatchFunc: watchFunc})

	for event := range watcher.ResultChan() {
		if event.Type == watch.Deleted {
			// Skip processing deleted pods, not much we can do with them as we
			// won't have details of where to send a response.
			continue
		}

		pod := event.Object.(*corev1.Pod)

		responseSubject, err := k.getPodEnvVar(pod, sidecarName, responseEnvVarName)
		if err != nil {
			// We can't respond as we don't know the subject, so we'll just kill this and log
			k.logger.Error().Err(err).Msg("Unable to process pod as cannot determine response subject")
			err := k.deletePod(ctx, pod)
			if err != nil {
				k.logger.Error().Err(err).Msg("Unable to delete unprocessable pod")
			}
		}

		startedAt := pod.CreationTimestamp.Time
		result, done, err := k.processWatchedPod(ctx, pod)

		if done {
			var deleteErr error

			err, sent := k.natsClient.PublishResult(ctx, startedAt, result, err, responseSubject)
			if err != nil {
				k.logger.Error().Err(err).Msgf("Error sending response for pod %s", pod.Name)
			} else {
				// Only attempt deletion if sending the response succeeded to avoid data loss.
				deleteErr = k.deletePod(ctx, pod)
			}

			if sent && err != nil {
				k.logger.Info().Err(err).Msgf("K8s pod run failed")
			}

			if deleteErr != nil {
				k.logger.Error().Err(err).Msgf("Error deleting pod %s", pod.Name)
			}
		}
	}
}

// processWatchedPod handles a pod event from the watcher, which relates to a hiphops task
//
// This will gather results and status
func (k *K8sHandler) processWatchedPod(ctx context.Context, pod *corev1.Pod) (interface{}, bool, error) {
	var result interface{}
	var err error
	beginWorkTimeout := 2 * time.Minute
	done := false

	switch pod.Status.Phase {
	// PodRunning is the expected state, as a healthy pod will continue running until we fetch results
	case corev1.PodRunning:
		// We inspect each container in the pod to understand the outcome and gather results
		result, done, err = k.processWatchedPodContainers(ctx, pod, beginWorkTimeout)

	case corev1.PodFailed:
		err = errors.New("Task pod failed")
		done = true

	case corev1.PodSucceeded:
		// Pod success is a failure state, as the sidecar should keep the pod running
		// indefinitely (until we manually gather results and delete)
		err = errors.New("Task pod completed before results were gathered")
		done = true

	case corev1.PodPending:
		// If it's pending and within deadline, we leave it alone to work.
		// Otherwise kill it. Most often long pending states are due to incorrect
		// config or missing images.
		if time.Now().Sub(pod.CreationTimestamp.Time) > beginWorkTimeout {
			err = errors.New("Task pod failed to start in time")
			done = true
		}

	default:
		k.logger.Info().Msgf("Skipping pod '%s' in unknown phase %s", pod.Name, pod.Status.Phase)
	}

	return result, done, err
}

// processWatchedPodContainers checks a running pod's containers to determine
// if it's in a final state, fetching the results if so.
//
// The return value dictates if a pod is in a final state and should be deleted
func (k *K8sHandler) processWatchedPodContainers(
	ctx context.Context,
	pod *corev1.Pod,
	beginWorkTimeout time.Duration,
) (interface{}, bool, error) {
	var result interface{}
	var err error
	done := false
	sidecarRunning := false

	for _, container := range pod.Status.ContainerStatuses {
		if container.Name == sidecarName && container.State.Running == nil {
			// Can't do anything with this as there's no sidecar to grab results
			err = errors.New("Hiphops sidecar has stopped unexpectedly")
			done = true
			break
		}
		// No further processing required for the sidecar
		if container.Name == sidecarName {
			sidecarRunning = true
			continue
		}

		// Any still running containers should be allowed to complete
		if container.State.Running != nil {
			done = false
			break
		}

		// Kill pods that have waited too long, otherwise leave them to run
		if container.State.Waiting != nil {
			if time.Now().Sub(pod.CreationTimestamp.Time) > beginWorkTimeout {
				err = errors.New("Task pod failed to start work before deadline")
				done = true
				break
			} else {
				done = false
				break
			}
		}

		if container.State.Terminated != nil {
			// All other states set final and immediately exit the loop,
			// so we can just set this to true and it will be respected as the final
			// outcome if all other pods are terminated.
			done = true
		}
	}

	if done && sidecarRunning {
		result, err = k.getPodResult(ctx, pod)
	}

	return result, done, err
}

func (k *K8sHandler) deletePod(ctx context.Context, pod *corev1.Pod) error {
	// Check if we should skip cleanup
	skipCleanup, err := k.getPodLabel(pod, skipCleanupLabel)
	if err != nil || skipCleanup == "true" {
		k.logger.Debug().Msgf("skip_cleanup is set for pod: %s. Skipping delete", pod.Name)
		return nil
	}

	deletePropagation := metav1.DeletePropagationForeground
	deleteOpts := metav1.DeleteOptions{
		PropagationPolicy: &deletePropagation,
	}

	err = k.clientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, deleteOpts)
	if err != nil && apiErrors.IsNotFound(err) {
		err = nil
	}
	if err != nil {
		return err
	}

	// Now cleanup any configmaps
	runUUID, err := k.getPodLabel(pod, runUUIDLabel)
	if err != nil {
		return err
	}

	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", runUUIDLabel, runUUID),
	}
	err = k.clientset.CoreV1().ConfigMaps(pod.Namespace).DeleteCollection(ctx, deleteOpts, listOpts)
	if err != nil && apiErrors.IsNotFound(err) {
		err = nil
	}

	return err
}

func (k *K8sHandler) getPodResult(ctx context.Context, pod *corev1.Pod) (interface{}, error) {
	outputPath, err := k.getPodEnvVar(pod, "main", resultPathEnvVarName)
	if err != nil {
		return nil, err
	}

	url, close, err := k.getPodUrl(pod, outputPath)
	if close != nil {
		defer close()
	}
	if err != nil {
		return nil, err
	}

	// Create an HTTP client that retries a few times
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 4
	retryClient.RetryWaitMax = 10 * time.Second
	retryClient.Logger = nil // TODO: Wrap zerolog such that we can use it here
	client := retryClient.StandardClient()

	resp, err := client.Get(url)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		k.logger.Debug().Err(err).Msgf("Failed to get result for pod %s", pod.Name)
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusNotFound:
		// Not found isn't an error state, it just means there's no output.
		// If output was expected but not present, that's left to the logic in
		// the user's pipelines to handle
		return "", nil

	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		fileExtension := strings.ToLower(filepath.Ext(outputPath))
		if fileExtension == ".json" {
			return k.parseResultJSON(body)
		}

		return string(body), nil
	default:
		return nil, fmt.Errorf("Unable to fetch result from task pod, status code: %d", resp.StatusCode)
	}
}

func (k *K8sHandler) getPodUrl(pod *corev1.Pod, outputPath string) (string, func(), error) {
	var podUrl string

	if k.requiresPortForward {
		localPort, close, err := k.portForward.ForwardPodPort(pod, 8917)
		if err != nil {
			return "", close, err
		}

		podUrl = fmt.Sprintf("http://%s:%d/%s", "localhost", localPort, outputPath)
	} else {
		podUrl = fmt.Sprintf("http://%s:%d/%s", pod.Status.PodIP, sidecarPort, outputPath)
	}

	return podUrl, nil, nil
}

func (k *K8sHandler) getPodEnvVar(pod *corev1.Pod, containerName string, varName string) (string, error) {
	var container corev1.Container
	var varValue string

	// Find the sidecar container
	for _, c := range pod.Spec.Containers {
		if c.Name != containerName {
			continue
		}

		container = c
	}

	if container.Name == "" {
		return "", fmt.Errorf("No container found with name '%s'", containerName)
	}

	// Find the env var for hiphops response subject
	for _, envVar := range container.Env {
		if envVar.Name != varName {
			continue
		}

		varValue = envVar.Value
	}

	if varValue == "" {
		return "", fmt.Errorf("No env var found with name %s", varName)
	}

	return varValue, nil
}

func (k *K8sHandler) getPodLabel(pod *corev1.Pod, label string) (string, error) {
	for key, v := range pod.Labels {
		if key != label {
			continue
		}

		return v, nil
	}

	return "", fmt.Errorf("Label '%s' not found on pod", label)
}

// initKubeConfig creates a kubernetes client from a kube config
func (k *K8sHandler) initKubeClient(kubeConfPath string) error {
	var cfg *rest.Config

	// If not explicitly set, attempt to auto-load the kube config file
	if kubeConfPath == "" {
		k.logger.Info().Msg("Automatically detecting kube config as no path provided")

		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

		config, err := kubeConfig.ClientConfig()
		if err != nil {
			return err
		}
		cfg = config

	} else {
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfPath)
		if err != nil {
			return err
		}
		cfg = config
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	k.cfg = cfg
	k.clientset = clientset

	return nil
}

func (k *K8sHandler) parseResultJSON(body []byte) (interface{}, error) {
	var resultData map[string]interface{}

	err := json.Unmarshal(body, &resultData)
	if err != nil {
		return nil, err
	}

	return resultData, nil
}

var disallowedConfigMapKey = regexp.MustCompile(`[^-._a-zA-Z0-9]+`)

func cleanConfigMapKey(key string) (string, error) {
	clean := disallowedConfigMapKey.ReplaceAllString(key, "")

	if clean == "" {
		return "", fmt.Errorf("Cannot create valid config map key from: %s", key)
	}

	return clean, nil
}
