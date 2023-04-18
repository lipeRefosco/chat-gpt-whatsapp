package entity

import (
	"errors"

	"github.com/google/uuid"
)

type Status int64

const (
	Ended  Status = 0
	Active Status = 1
)

type ChatConfig struct {
	Model           *Model
	Temperature     float32  // 0.0 to 1.0 - lower is more precise and higher is more creative
	TopP            float32  // 0.0. to 1.0 - to a low value, like 0.1 model will be very conservative in
	N               int      //Number of messages to generate
	Stop            []string // list of tokens to stop on
	MaxTokens       int      // number of tokens to generate
	PresencePenalty float32  // -2.0 to 2.0 - number between -2.9 and 2.0. Positive values penalize new tokens
	FrequecyPenalty float32  // -2.0 to 2.0 - number between -2.9 and 2.0. Positive values penalize new tokens
}

type Chat struct {
	ID                   string
	UserID               string
	InitialSystemMessage *Message
	Messages             []*Message
	ErasedMessages       []*Message
	Status               Status
	TokenUsage           int
	Config               *ChatConfig
}

func (c *Chat) Validate() error {
	if c.UserID == "" {
		return errors.New("user id is empty")
	}

	if c.Status != Active && c.Status != Ended {
		return errors.New("invalid status")
	}

	if c.Config.Temperature < 0 || c.Config.Temperature > 2 {
		return errors.New("invalid tempeture")
	}

	// ... more validations for config
	return nil
}

func NewChat(userID string, initialSystemMessage *Message, chatConfig *ChatConfig) (*Chat, error) {
	chat := &Chat{
		ID:                   uuid.New().String(),
		UserID:               userID,
		InitialSystemMessage: initialSystemMessage,
		Status:               Active,
		Config:               chatConfig,
		TokenUsage:           0,
	}

	chat.AddMessage(initialSystemMessage)

	if err := chat.Validate(); err != nil {
		return nil, err
	}

	return chat, nil
}

func (c *Chat) AddMessage(m *Message) error {
	if c.Status == Ended {
		return errors.New("chat is ended. no more messages allowed")
	}

	for {
		if c.Config.Model.GetMaxTokens() >= m.GetQtdTokens()+c.TokenUsage {
			c.Messages = append(c.Messages, m)
			c.RefreshTokenUsage()
			break
		}
		c.ErasedMessages = append(c.ErasedMessages, c.Messages[0]) // Guarda a mensagem mais antiga
		c.Messages = c.Messages[1:]                                // Remove a mensagem mais antiga da lista de mensagens
		c.RefreshTokenUsage()
	}

	return nil
}

func (c *Chat) GetMessages() []*Message {
	return c.Messages
}

func (c *Chat) CountMessages() int {
	return len(c.Messages)
}

func (c *Chat) End() {
	c.Status = Ended
}

func (c *Chat) RefreshTokenUsage() {
	c.TokenUsage = 0
	for m := range c.Messages {
		c.TokenUsage += c.Messages[m].GetQtdTokens()
	}
}
