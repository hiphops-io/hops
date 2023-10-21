package undistribute

import (
	"os"
	"path"
	"strings"

	"golang.org/x/sync/errgroup"
)

// StateMsg interface will usually be implemented as jetstream.Msg
type StateMsg interface {
	Data() []byte
	Subject() string
}

// UpFetchState updates the state of the event and returns the entire current state.
//
// The returned map is keyed by filename, which will be the msg ID if stored through this function.
// The msg ID is the last token of the subject, following the conventions of this package.
func UpFetchState(stateDir string, msg StateMsg) (string, map[string][]byte, error) {
	msgData := msg.Data()
	msgSubjectTokens := strings.Split(msg.Subject(), ".")
	msgId := msgSubjectTokens[len(msgSubjectTokens)-1]
	sequenceId := msgSubjectTokens[len(msgSubjectTokens)-2]
	sequenceDir := path.Join(stateDir, sequenceId)

	stateResult, err := fetchState(sequenceDir)
	if err != nil {
		return "", nil, err
	}

	stateResult[msgId] = msgData
	err = updateState(sequenceDir, msgId, msgData)

	return sequenceId, stateResult, nil
}

type stateFile struct {
	name string
	data []byte
}

func updateState(sequenceDir string, msgId string, msgData []byte) error {
	err := os.MkdirAll(sequenceDir, 0744)
	if err != nil {
		return err
	}

	err = os.WriteFile(path.Join(sequenceDir, msgId), msgData, 0644)
	if err != nil {
		return err
	}

	return nil
}

func fetchState(sequenceDir string) (map[string][]byte, error) {
	eg := new(errgroup.Group)
	stateResult := make(map[string][]byte)

	files, err := os.ReadDir(sequenceDir)
	if os.IsNotExist(err) {
		return stateResult, nil
	}
	if err != nil {
		return nil, err
	}

	data := make(chan stateFile, len(files))

	for _, file := range files {
		file := file
		if file.IsDir() {
			continue
		}

		eg.Go(func() error {
			return readFileData(data, path.Join(sequenceDir, file.Name()))
		})
	}

	err = eg.Wait()
	if err != nil {
		return nil, err
	}

	close(data)
	for state := range data {
		stateResult[state.name] = state.data
	}

	return stateResult, nil
}

func readFileData(data chan stateFile, filePath string) error {
	fileName := path.Base(filePath)
	contents, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	data <- stateFile{
		name: fileName,
		data: contents,
	}
	return nil
}
