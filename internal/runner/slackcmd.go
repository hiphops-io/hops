package runner

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/goccy/go-json"
	"github.com/hiphops-io/hops/markdown"
	"github.com/hiphops-io/hops/nats"
	"github.com/manterfield/go-mapreader"
	"github.com/slack-go/slack"
)

type (
	AccessTokenGetter func() (string, error)

	CommandPrivateMeta struct {
		CommandAction string `json:"command_action"`
		ChannelID     string `json:"channel_id"`
	}
)

func SlackAccessTokenFunc(natsClient *nats.Client) AccessTokenGetter {
	var token string

	return func() (string, error) {
		if token != "" {
			return token, nil
		}

		reply, err := natsClient.NatsConn.Request("hiphops.slack.accesstoken", nil, time.Second*3)
		if err != nil {
			return "", err
		}

		json.Unmarshal(reply.Data, &token)

		return token, nil
	}
}

func SlackCommandRequest(flow *markdown.Flow, hopsMsg *nats.HopsMsg, matchError error, t AccessTokenGetter) error {
	token, err := t()
	if err != nil {
		return fmt.Errorf("unable to fetch slack access token: %w", err)
	}
	api := slack.New(token)

	responseURL, err := mapreader.StrErr(hopsMsg.Data, "response_url")
	if err != nil {
		return fmt.Errorf("unable to get response_url for command request: %w", err)
	}

	if errors.Is(matchError, markdown.ErrCommandNotFound) {
		sendErrorResponse(api, fmt.Sprintf("Sorry, `%s` didn't match any commands", mapreader.Str(hopsMsg.Data, "text")), responseURL)
		return err
	} else if matchError != nil {
		sendErrorResponse(api, fmt.Sprintf("An error occurred - this could be due to a misconfiguration\n`%s`", matchError.Error()), responseURL)
		return err
	} else if flow == nil {
		return sendErrorResponse(api, "Command conditions not met - _Perhaps the command is restricted to specific channels or users?_", responseURL)
	}

	// If we get here then we've got an actual command to present. Yay.
	blocks, err := CommandToSlackBlocks(flow.Command)
	if err != nil {
		sendErrorResponse(api, "command failed", responseURL)
		return fmt.Errorf("unable to render command as modal: %w", err)
	}

	privateMeta, err := json.Marshal(CommandPrivateMeta{
		CommandAction: flow.ActionName(),
		ChannelID:     mapreader.Str(hopsMsg.Data, "channel_id"),
	})

	if err != nil {
		sendErrorResponse(api, "command failed", responseURL)
		return fmt.Errorf("unable to marshal private metadata: %w", err)
	}

	_, err = api.OpenView(
		hopsMsg.Data["trigger_id"].(string),
		slack.ModalViewRequest{
			Type:  slack.VTModal,
			Title: slack.NewTextBlockObject("plain_text", flow.DisplayName(), false, false),
			Blocks: slack.Blocks{
				BlockSet: blocks,
				// slack.NewTextBlockObject() // We want the rendered slack markdown here
				// Maybe with the option to hide it?
			},
			PrivateMetadata: string(privateMeta),
			CallbackID:      "command",
			Submit:          slack.NewTextBlockObject("plain_text", "Run", false, false),
			Close:           slack.NewTextBlockObject("plain_text", "Close", false, false),
		},
	)
	if err != nil {
		sendErrorResponse(api, "command failed - unable to open form", responseURL)
		return fmt.Errorf("unable to open slack view: %w", err)
	}

	return nil
}

func SlackBlocksToCommandEvent(hopsMsg *nats.HopsMsg) error {
	if t := mapreader.Str(hopsMsg.Data, "type"); t != "view_submission" {
		return fmt.Errorf("unsupported slack interaction type for commands '%s'", t)
	}

	data, err := parseViewSubmissionCommand(hopsMsg.Data)
	if err != nil {
		return fmt.Errorf("unable to parse command from event: %w", err)
	}

	hopsMsg.Action = mapreader.Str(data, "hops.action")
	hopsMsg.Data = data

	return nil
}

func CommandToSlackBlocks(command markdown.Command) ([]slack.Block, error) {
	blocks := []slack.Block{}

	for _, p := range command {
		name, param := p.Param()
		displayName := p.DisplayName()

		switch param.Type {
		case "text", "string":
			blocks = append(blocks, ParamToTextInputBlock(name, displayName, param))
		case "bool":
			blocks = append(blocks, ParamToBooleanInputBlock(name, displayName, param))
		case "number":
			blocks = append(blocks, ParamToNumberInputBlock(name, displayName, param))
		default:
			return nil, fmt.Errorf("unable to parse param '%s' - unknown type '%s'", name, param.Type)
		}
	}

	return blocks, nil
}

func ParamBlockLabel(name string) *slack.TextBlockObject {
	return &slack.TextBlockObject{
		Type:     slack.PlainTextType,
		Text:     name,
		Emoji:    false,
		Verbatim: false,
	}
}

func ParamInputBlock(name, displayName string, param markdown.Param, elem slack.BlockElement) slack.Block {
	return slack.InputBlock{
		BlockID:  name,
		Type:     slack.MBTInput,
		Label:    ParamBlockLabel(displayName),
		Element:  elem,
		Optional: !param.Required,
	}
}

func ParamToNumberInputBlock(name, displayName string, param markdown.Param) slack.Block {
	elem := slack.NumberInputBlockElement{
		Type:             slack.METNumber,
		ActionID:         name,
		Placeholder:      nil,
		IsDecimalAllowed: true,
	}

	if param.Default != nil {
		numberDefault := fmt.Sprintf("%v", param.Default)
		// Slack's number input decides that any InitialValue over (roughly) 14 digits
		// isn't an integer or float. Apparently numbers are finite after all.
		if len(numberDefault) <= 14 {
			elem.InitialValue = numberDefault
		}
	}

	return ParamInputBlock(name, displayName, param, elem)
}

func ParamToBooleanInputBlock(name, displayName string, param markdown.Param) slack.Block {
	trueOption := &slack.OptionBlockObject{
		Value: "true",
		Text: &slack.TextBlockObject{
			Type:     slack.PlainTextType,
			Text:     "true",
			Emoji:    false,
			Verbatim: false,
		},
	}

	falseOption := &slack.OptionBlockObject{
		Value: "false",
		Text: &slack.TextBlockObject{
			Type:     slack.PlainTextType,
			Text:     "false",
			Emoji:    false,
			Verbatim: false,
		},
	}

	elem := slack.RadioButtonsBlockElement{
		Type:     slack.METRadioButtons,
		ActionID: name,
		Options:  []*slack.OptionBlockObject{trueOption, falseOption},
	}

	defaultVal, ok := param.Default.(bool)
	if ok && defaultVal {
		elem.InitialOption = trueOption
	}
	if ok && !defaultVal {
		elem.InitialOption = falseOption
	}

	return ParamInputBlock(name, displayName, param, elem)
}

func ParamToTextInputBlock(name, displayName string, param markdown.Param) slack.Block {
	elem := slack.PlainTextInputBlockElement{
		Type:        slack.METPlainTextInput,
		Placeholder: nil,
		ActionID:    name,
		Multiline:   param.Type == "text",
	}

	defaultVal, ok := param.Default.(string)
	if ok {
		elem.InitialValue = defaultVal
	}

	return ParamInputBlock(name, displayName, param, elem)
}

func parseViewSubmissionCommand(payload map[string]any) (map[string]any, error) {
	commandPayload := map[string]any{
		"hops": mapreader.Map[any](payload, "hops"),
		"ctx":  payload,
	}

	p, err := mapreader.BytesErr(payload, "view.private_metadata")
	if err != nil {
		return nil, errors.New("unable to read required metadata for command")
	}

	privateMeta := CommandPrivateMeta{}
	if err := json.Unmarshal(p, &privateMeta); err != nil {
		return nil, fmt.Errorf("unable to parse required metadata for command: %w", err)
	}

	commandPayload["ctx"].(map[string]any)["channel_id"] = privateMeta.ChannelID
	commandPayload["hops"].(map[string]any)["action"] = privateMeta.CommandAction

	values, err := mapreader.MapErr[map[string]any](payload, "view.state.values")
	if err != nil {
		return nil, fmt.Errorf("unable to parse command payload: %w", err)
	}

	for k, v := range values {
		param, ok := v[k].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("unable to parse command param '%s'", k)
		}

		paramValue, err := parseViewInputValue(param)
		if err != nil {
			return nil, fmt.Errorf("unable to parse command param '%s' value ", k)
		}

		commandPayload[k] = paramValue
	}

	return commandPayload, nil
}

func parseViewInputValue(paramState map[string]any) (any, error) {
	inputType := mapreader.Str(paramState, "type")
	switch inputType {
	case "number_input":
		value := mapreader.Str(paramState, "value")
		if value == "" {
			return nil, nil
		}
		return strconv.ParseFloat(value, 64)
	case "radio_buttons":
		value := mapreader.Str(paramState, "selected_option.value")
		switch value {
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			return nil, nil
		}
	case "plain_text_input":
		return mapreader.Str(paramState, "value"), nil
	default:
		return nil, fmt.Errorf("unsupported input type '%s'", inputType)
	}
}

func sendErrorResponse(api *slack.Client, msg string, responseURL string) error {
	_, _, err := api.PostMessage(
		"",
		slack.MsgOptionText(msg, false),
		slack.MsgOptionResponseURL(
			responseURL,
			slack.ResponseTypeEphemeral,
		),
	)

	return err
}
