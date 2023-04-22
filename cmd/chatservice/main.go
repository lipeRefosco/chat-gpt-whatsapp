package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/lipeRefosco/chat-gpt-whatsapp/configs"
	"github.com/lipeRefosco/chat-gpt-whatsapp/internal/infra/repository"
	"github.com/lipeRefosco/chat-gpt-whatsapp/internal/infra/web"
	"github.com/lipeRefosco/chat-gpt-whatsapp/internal/infra/web/webserver"
	"github.com/lipeRefosco/chat-gpt-whatsapp/internal/usecase/chatcompletion"
	"github.com/sashabaranov/go-openai"
)

func main() {
	configs, err := configs.LoadConfig(".")
	if err != nil {
		panic(err)
	}

	conn, err := sql.Open(configs.DBDriver, fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true",
		configs.DBUser, configs.DBPassword, configs.DBHost, configs.DBPort, configs.DBName))
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	repo := repository.NewChatRepositoryMySQL(conn)
	client := openai.NewClient(configs.OpenAIApiKey)

	chatConfig := chatcompletion.ChatCompletionConfigInputDTO{
		Model:                configs.Model,
		ModelMaxTokens:       configs.ModelMaxTokens,
		Temperature:          float32(configs.Temperature),
		TopP:                 float32(configs.TopP),
		N:                    configs.N,
		Stop:                 configs.Stop,
		MaxTokens:            configs.MaxTokens,
		InitialSystemMessage: configs.InitialChatMessage,
	}

	// chatConfigStream := chatcompletion.ChatCompletionConfigInputDTO{
	// 	Model:                configs.Model,
	// 	ModelMaxTokens:       configs.ModelMaxTokens,
	// 	Temperature:          float32(configs.Temperature),
	// 	TopP:                 float32(configs.TopP),
	// 	N:                    configs.N,
	// 	Stop:                 configs.Stop,
	// 	MaxTokens:            configs.MaxTokens,
	// 	InitialSystemMessage: configs.InitialChatMessage,
	// }

	usecase := chatcompletion.NewChatCompletionUseCase(repo, client)

	// streamChannel := make(chan chatcompletionstream.ChatCompletionOutputDTO)
	// usecaseStream := chatcompletionstream.NewChatCompletionUSeCase(repo, client, streamChannel)

	webserver, err := webserver.NewWebServer(":" + configs.WebServerPort)
	if err != nil {
		panic(err)
	}
	webserverChatHandler := web.NewWebChatGPTHandler(*usecase, chatConfig, configs.AuthToken)
	webserver.AddHandler("/chat", webserverChatHandler.Handle)

	fmt.Println("Server running on port " + configs.WebServerPort)
	webserver.Start()
}
