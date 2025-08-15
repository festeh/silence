package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
	"github.com/revrost/go-openrouter"
	"silence-backend/logger"
)


type AIResponse struct {
	Message openrouter.ChatCompletionMessage `json:"message"`
	Error   string                            `json:"error,omitempty"`
}

func HandleAI(re *core.RequestEvent, openRouterAPIKey string) error {
	if re.Request.Method != http.MethodPost {
		return re.String(http.StatusMethodNotAllowed, "Method not allowed")
	}

	var chatReq openrouter.ChatCompletionRequest
	if err := json.NewDecoder(re.Request.Body).Decode(&chatReq); err != nil {
		logger.Error("Failed to decode AI request", "error", err)
		return re.JSON(http.StatusBadRequest, AIResponse{
			Error: "Invalid JSON request",
		})
	}

	if len(chatReq.Messages) == 0 {
		return re.JSON(http.StatusBadRequest, AIResponse{
			Error: "Messages are required",
		})
	}

	if chatReq.Model == "" {
		chatReq.Model = "meta-llama/llama-3.3-70b-instruct"
	}

	logger.Info("Processing AI request", "model", chatReq.Model, "message_count", len(chatReq.Messages), "tools_count", len(chatReq.Tools))

	client := openrouter.NewClient(openRouterAPIKey)

	response, err := client.CreateChatCompletion(context.Background(), chatReq)

	if err != nil {
		logger.Error("Failed to create chat completion", "error", err)
		return re.JSON(http.StatusInternalServerError, AIResponse{
			Error: "Failed to process AI request",
		})
	}

	if len(response.Choices) == 0 {
		logger.Error("No choices returned from OpenRouter")
		return re.JSON(http.StatusInternalServerError, AIResponse{
			Error: "No response from AI model",
		})
	}

	message := response.Choices[0].Message
	
	if len(message.ToolCalls) > 0 {
		logger.Info("AI request completed with tool calls", "tool_count", len(message.ToolCalls))
	} else {
		logger.Info("AI request completed successfully", "response_length", len(message.Content.Text))
	}

	return re.JSON(http.StatusOK, AIResponse{
		Message: message,
	})
}