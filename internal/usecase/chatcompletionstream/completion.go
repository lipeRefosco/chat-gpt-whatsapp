package chatcompletionstream

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/lipeRefosco/chat-gpt-whatsapp/internal/domain/entity"
	"github.com/lipeRefosco/chat-gpt-whatsapp/internal/domain/gateway"
	openai "github.com/sashabaranov/go-openai"
)

// DTO = é um objeto que só guarda valor,
// não tem comportamento e serve apenas para transitar em camadas
type ChatCompletionConfigInputDTO struct {
	Model                string
	ModelMaxTokens       int
	Tempeture            float32
	TopP                 float32
	N                    int
	Stop                 []string
	MaxTokens            int
	PresencePenalty      float32
	FrequencyPenalty     float32
	InitialSystemMessage string
}

type ChatCompletionInputDTO struct {
	ChatID      string
	UserID      string
	UserMessage string
	Config      ChatCompletionConfigInputDTO
}

type ChatCompletionOutputDTO struct {
	ChatID  string
	UserID  string
	Content string
}

type ChatCompletionUseCase struct {
	ChatGateway  gateway.ChatGateway
	OpenAiClient *openai.Client
	Stream       chan ChatCompletionOutputDTO
}

func NewChatCompletionUSeCase(chatGateway gateway.ChatGateway, openAiCLinet *openai.Client, stream chan ChatCompletionOutputDTO) *ChatCompletionUseCase {
	return &ChatCompletionUseCase{
		ChatGateway:  chatGateway,
		OpenAiClient: openAiCLinet,
		Stream:       stream,
	}
}

func (uc *ChatCompletionUseCase) Execute(ctx context.Context, input ChatCompletionInputDTO) (*ChatCompletionOutputDTO, error) {
	chat, err := uc.ChatGateway.FindChatByID(ctx, input.ChatID)
	if err != nil {
		if err.Error() == "chat not found" {
			// Create new chat (entity)
			chat, err := createNewChat(input)
			if err != nil {
				return nil, errors.New("error creating new chat: " + err.Error())
			}
			// Save on database
			err = uc.ChatGateway.CreateChat(ctx, chat)
			if err != nil {
				return nil, errors.New("error persisting new chat: " + err.Error())
			}
		} else {
			return nil, errors.New("error fetching existing chat: " + err.Error())
		}
	}
	userMessage, err := entity.NewMessage(entity.User, input.UserMessage, chat.Config.Model)
	if err != nil {
		return nil, errors.New("error createing new message: " + err.Error())
	}

	err = chat.AddMessage(userMessage)
	if err != nil {
		return nil, errors.New("error adding new message: " + err.Error())
	}

	messages := []openai.ChatCompletionMessage{}
	for _, msg := range chat.Messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	resp, err := uc.OpenAiClient.CreateChatCompletionStream(
		ctx,
		openai.ChatCompletionRequest{
			Model:            chat.Config.Model.Name,
			Messages:         messages,
			MaxTokens:        chat.Config.MaxTokens,
			Temperature:      chat.Config.Temperature,
			TopP:             chat.Config.TopP,
			PresencePenalty:  chat.Config.PresencePenalty,
			FrequencyPenalty: chat.Config.FrequecyPenalty,
			Stop:             chat.Config.Stop,
			Stream:           true,
		})
	if err != nil {
		return nil, errors.New("error creating chat completion: " + err.Error())
	}

	var fullRespose strings.Builder

	for {
		response, err := resp.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, errors.New("error streaming resposen: " + err.Error())
		}

		fullRespose.WriteString(response.Choices[0].Delta.Content)

		r := ChatCompletionOutputDTO{
			ChatID:  chat.ID,
			UserID:  input.UserID,
			Content: fullRespose.String(),
		}

		uc.Stream <- r
	}

	assistant, err := entity.NewMessage(entity.Assistant, fullRespose.String(), chat.Config.Model)
	if err != nil {
		return nil, errors.New("error creating assistant message: " + err.Error())
	}

	err = chat.AddMessage(assistant)
	if err != nil {
		return nil, errors.New("error adding new message: " + err.Error())
	}

	err = uc.ChatGateway.SaveChat(ctx, chat)
	if err != nil {
		return nil, errors.New("error saving chat: " + err.Error())
	}

	return &ChatCompletionOutputDTO{
		ChatID:  chat.ID,
		UserID:  input.UserID,
		Content: fullRespose.String(),
	}, nil
}

func createNewChat(input ChatCompletionInputDTO) (*entity.Chat, error) {
	model := entity.NewModel(input.Config.Model, input.Config.ModelMaxTokens)
	chatConfig := &entity.ChatConfig{
		Temperature:     input.Config.Tempeture,
		TopP:            input.Config.TopP,
		N:               input.Config.N,
		Stop:            input.Config.Stop,
		MaxTokens:       input.Config.MaxTokens,
		PresencePenalty: input.Config.PresencePenalty,
		FrequecyPenalty: input.Config.FrequencyPenalty,
		Model:           model,
	}

	initialMessage, err := entity.NewMessage(entity.System, input.Config.InitialSystemMessage, model)
	if err != nil {
		return nil, errors.New("error creating innitial message: " + err.Error())
	}

	chat, err := entity.NewChat(input.UserID, initialMessage, chatConfig)

	return chat, nil
}
