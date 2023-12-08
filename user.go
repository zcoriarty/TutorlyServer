package user

import (

    "bytes"
    "encoding/json"
    "net/http"
    "os"

    "github.com/gin-gonic/gin"
    "github.com/stripe/stripe-go/v72"
    "github.com/stripe/stripe-go/v72/customer"
  
)

// StripeCustomerRequest represents the request body for creating a Stripe customer
type StripeCustomerRequest struct {
    Email string `json:"email"`
}

// createStripeCustomer handles the creation of a Stripe customer
func createStripeCustomer(email string) (*stripe.Customer, error) {
    stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

    params := &stripe.CustomerParams{
        Email: stripe.String(email),
    }

    return customer.New(params)
}

// updateSupabaseUser updates the user record in Supabase with the Stripe customer ID
func UpdateSupabaseUser(supabaseID, customerID string) error {
    supabaseURL := os.Getenv("SUPABASE_URL")
    supabaseKey := os.Getenv("SUPABASE_KEY")

    url := supabaseURL + "/rest/v1/user?id=eq." + supabaseID

    requestData := map[string]string{"customer_id": customerID}
    jsonValue, _ := json.Marshal(requestData)

    req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonValue))
    if err != nil {
        return err
    }

    req.Header.Set("apikey", supabaseKey)
    req.Header.Set("Authorization", "Bearer "+supabaseKey)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Prefer", "return=representation")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    return nil
}

// handleStripeCustomerCreation creates a Stripe customer and updates the Supabase user
func HandleStripeCustomerCreation(c *gin.Context) {
    var req StripeCustomerRequest
    if err := c.BindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
        return
    }

    stripeCustomer, err := createStripeCustomer(req.Email)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Stripe customer"})
        return
    }

    if err := updateSupabaseUser(req.Email, stripeCustomer.ID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Supabase user"})
        return
    }

    c.JSON(http.StatusOK, stripeCustomer)
}