package main

import (
  "log"
  "net/http"
  "os"
  "time"

  "tutorly/oai"
  "tutorly/user"
  "tutorly/auth"

	"github.com/gin-gonic/gin"
  "github.com/joho/godotenv"
  "github.com/gin-contrib/logger"
)

func main() {
  loadEnvironmentVariables()
  gin.SetMode(gin.ReleaseMode)
  router := setupRouter()

  s := &http.Server{
    Addr:           ":8080",
    Handler:        router,
    ReadTimeout:    10 * time.Second,
    WriteTimeout:   10 * time.Second,
    MaxHeaderBytes: 1 << 20,
  }
  log.Fatal(s.ListenAndServe())
}

func setupRouter() *gin.Engine {
    router := gin.New()

    router.Use(gin.Recovery())
    router.Use(logger.SetLogger())
    router.Use(CORSMiddleware())
    router.Use(auth.IsAuthorized())  // Apply JWT Middleware globally

    router.POST("/chat-completion", oai.HandleChatCompletion)
    router.POST("/create-stripe-customer", user.HandleStripeCustomerCreation)
    router.GET("/subscription-status/:customerID", user.HandleSubscriptionStatus)
    router.GET("/check-customer-exists/:email", user.HandleCheckCustomerExists)

    return router
}


func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func loadEnvironmentVariables() {
  if err := godotenv.Load(); err != nil {
    log.Println("No .env file found")
  }

  if stripeKey := os.Getenv("STRIPE_SECRET_KEY"); stripeKey == "" {
    log.Fatal("STRIPE_SECRET_KEY is not set")
  }
}
