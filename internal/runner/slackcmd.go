package runner

import (
	"errors"
	"fmt"
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

func BeginSlackCommandFlow(flow *markdown.Flow, hopsMsg *nats.HopsMsg, matchError error, t AccessTokenGetter) error {
	token, err := t()
	if err != nil {
		return fmt.Errorf("unable to fetch slack access token: %w", err)
	}
	api := slack.New(token)

	channelID := mapreader.Str(hopsMsg.Data, "channel_id")

	// TODO: Only do this message sending if we actually have a channel_id
	var text string
	if errors.Is(matchError, markdown.ErrCommandNotFound) {
		text = fmt.Sprintf("Sorry, `%s` didn't match any commands", hopsMsg.Data["text"].(string))
	} else if matchError != nil {
		text = fmt.Sprintf("An error occurred - this could be due to misconfiguration\n`%s`", matchError.Error())
	} else if flow == nil {
		text = "Command conditions not met - _Common causes are being restricted to specific channels or users_"
	}

	if text != "" {
		_, _, err = api.PostMessage(
			channelID,
			slack.MsgOptionText(text, false),
			slack.MsgOptionResponseURL(
				hopsMsg.Data["response_url"].(string),
				slack.ResponseTypeEphemeral,
			),
		)

		if err != nil {
			return err
		}

		return nil
	}

	// If we get here then we've got an actual command to present. Yay.
	blocks, err := CommandToSlackBlocks(flow.Command)
	if err != nil {
		// TODO: Do something with this error
		fmt.Println("Got an error creating the command form blocks:", err.Error())
		return nil
	}

	privateMeta, err := json.Marshal(CommandPrivateMeta{
		CommandAction: flow.ActionName(),
		ChannelID:     channelID,
	})

	if err != nil {
		// TODO: Do something with this error
		fmt.Println("unable to marshal private metadata")
		return nil
	}

	_, err = api.OpenView(
		hopsMsg.Data["trigger_id"].(string),
		slack.ModalViewRequest{
			Type:  slack.VTModal,
			Title: slack.NewTextBlockObject("plain_text", flow.ID, false, false),
			Blocks: slack.Blocks{
				BlockSet: blocks,
				// slack.NewTextBlockObject() // We want the rendered slack markdown here
				// Maybe with the option to hide it?
			},
			PrivateMetadata: string(privateMeta),
			CallbackID:      "user_command",
			Submit:          slack.NewTextBlockObject("plain_text", "Run", false, false),
			Close:           slack.NewTextBlockObject("plain_text", "Close", false, false),
		},
	)
	if err != nil {
		// TODO: Do something with this error
		// response_metadata.messages contains the actual error descriptions
		fmt.Println("Got an error opening view:", err.Error())
		return nil
	}

	return nil
}

func CommandToSlackBlocks(command markdown.Command) ([]slack.Block, error) {
	blocks := []slack.Block{}

	for _, p := range command {
		name, param := p.Param()

		switch param.Type {
		case "text", "string":
			blocks = append(blocks, ParamToTextInputBlock(name, param))
		case "bool":
			blocks = append(blocks, ParamToBooleanInputBlock(name, param))
		case "number":
			blocks = append(blocks, ParamToNumberInputBlock(name, param))
		default:
			return nil, fmt.Errorf("unable to parse param '%s' - unknown type '%s'", name, param.Type)
		}
	}

	// Add the submit button etc? - Need to ensure we're handling no params

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

func ParamInputBlock(name string, param markdown.Param, elem slack.BlockElement) slack.Block {
	return slack.InputBlock{
		BlockID:  name,
		Type:     slack.MBTInput,
		Label:    ParamBlockLabel(name),
		Element:  elem,
		Optional: !param.Required,
	}
}

func ParamToNumberInputBlock(name string, param markdown.Param) slack.Block {
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

	return ParamInputBlock(name, param, elem)
}

func ParamToBooleanInputBlock(name string, param markdown.Param) slack.Block {
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

	return ParamInputBlock(name, param, elem)
}

func ParamToTextInputBlock(name string, param markdown.Param) slack.Block {
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

	return ParamInputBlock(name, param, elem)
}
