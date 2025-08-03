package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
	"github.com/revrost/go-openrouter"
	"silence-backend/logger"
)

type MessageRequest struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AIRequest struct {
	Messages []MessageRequest `json:"messages"`
	Model    string           `json:"model,omitempty"`
}

type AIResponse struct {
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

func HandleAI(re *core.RequestEvent, openRouterAPIKey string) error {
	if re.Request.Method != http.MethodPost {
		return re.String(http.StatusMethodNotAllowed, "Method not allowed")
	}

	var req AIRequest
	if err := json.NewDecoder(re.Request.Body).Decode(&req); err != nil {
		logger.Error("Failed to decode AI request", "error", err)
		return re.JSON(http.StatusBadRequest, AIResponse{
			Error: "Invalid JSON request",
		})
	}

	if len(req.Messages) == 0 {
		return re.JSON(http.StatusBadRequest, AIResponse{
			Error: "Messages are required",
		})
	}

	model := req.Model
	if model == "" {
		model = "meta-llama/llama-3.3-70b-instruct"
	}

	logger.Info("Processing AI request", "model", model, "message_count", len(req.Messages))

	client := openrouter.NewClient(openRouterAPIKey)

	// Convert request messages to OpenRouter format
	var messages []openrouter.ChatCompletionMessage
	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			messages = append(messages, openrouter.SystemMessage(msg.Content))
		case "user":
			messages = append(messages, openrouter.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, openrouter.AssistantMessage(msg.Content))
		default:
			messages = append(messages, openrouter.UserMessage(msg.Content))
		}
	}

	response, err := client.CreateChatCompletion(context.Background(), openrouter.ChatCompletionRequest{
		Model:    model,
		Messages: messages,
	})

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

	content := response.Choices[0].Message.Content.Text
	logger.Info("AI request completed successfully", "response_length", len(content))

	return re.JSON(http.StatusOK, AIResponse{
		Content: content,
	})
}