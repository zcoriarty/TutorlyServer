package oai

import (
  "bytes"
  "encoding/json"
  "fmt"
  "io"
  "log"
  "mime/multipart"
  "net/http"
  "os"


  "github.com/gin-gonic/gin"

)

type ChatRequest struct {
  Model    string    `json:"model"`
  Messages []Message `json:"messages"`
}

type Message struct {
  Role    string `json:"role"`
  Content string `json:"content"`
}

type OpenAIResponse struct {
  ID      string   `json:"id"`
  Object  string   `json:"object"`
  Created int      `json:"created"`
  Model   string   `json:"model"`
  Choices []Choice `json:"choices"`
}

type Choice struct {
  Index        int            `json:"index"`
  Role         string         `json:"role"`
  Message      MessageContent `json:"message"`
  LogProbs     interface{}    `json:"logprobs"`
  FinishReason string         `json:"finish_reason"`
}

type MessageContent struct {
  Content string `json:"content"`
}

func createOpenAIRequest(prompt string, file multipart.File) (*ChatRequest, error) {
  var messages []Message

  if file != nil {
    buffer := new(bytes.Buffer)
    _, err := io.Copy(buffer, file)
    if err != nil {
      return nil, err
    }

    fileContent := buffer.String()
    messages = append(messages, Message{Role: "system", Content: fileContent})
  }

  messages = append(messages, Message{Role: "user", Content: prompt})

  return &ChatRequest{
    Model:    "gpt-3.5-turbo",
    Messages: messages,
  }, nil
}

func callOpenAI(apiKey string, chatRequest *ChatRequest) (*OpenAIResponse, error) {
  jsonData, err := json.Marshal(chatRequest)
  if err != nil {
    return nil, err
  }

  request, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
  if err != nil {
    return nil, err
  }

  request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
  request.Header.Set("Content-Type", "application/json")

  client := &http.Client{}
  response, err := client.Do(request)
  if err != nil {
    return nil, err
  }
  defer response.Body.Close()

  responseBody, _ := io.ReadAll(response.Body)
  // log.Printf("Raw OpenAI API response: %s", string(responseBody))

  // Reset the response body to its original state
  response.Body = io.NopCloser(bytes.NewBuffer(responseBody))

  var openAIResponse OpenAIResponse
  err = json.NewDecoder(response.Body).Decode(&openAIResponse)
  if err != nil {
    log.Printf("Error decoding OpenAI API response: %v", err)
    return nil, err
  }
  // log the response
  // log.Printf("OpenAI API Response: %+v", openAIResponse)
  // if len(openAIResponse.Choices) > 0 {
  // 	log.Printf("OpenAI API Response Choice Text: %s", openAIResponse.Choices[0].Message.Content)
  // }

  return &openAIResponse, nil
}

func HandleChatCompletion(c *gin.Context) {
  apiKey, exists := os.LookupEnv("TUTORLY_OPENAI_DEV")
  if !exists {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "OpenAI API key not found"})
    return
  }

  prompt := c.PostForm("prompt")
  fileHeader, _ := c.FormFile("file")
  var file multipart.File
  if fileHeader != nil {
    var err error
    file, err = fileHeader.Open()
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": "Error opening file"})
      return
    }
    defer file.Close()
  }

  chatRequest, err := createOpenAIRequest(prompt, file)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Error creating request"})
    return
  }

  openAIResponse, err := callOpenAI(apiKey, chatRequest)
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Error calling OpenAI API"})
    return
  }

  // log.Printf("Sending response back to frontend: %+v", openAIResponse)
  c.JSON(http.StatusOK, openAIResponse)

}