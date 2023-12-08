package user

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/sub"
)

// StripeCustomerRequest represents the request body for creating a Stripe customer
type StripeCustomerRequest struct {
	Email string `json:"email"`
}

// CheckCustomerExists checks if a Stripe customer exists for a given email
func CheckCustomerExists(email string) (bool, error) {
  stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

  params := &stripe.CustomerListParams{}
  params.Filters.AddFilter("email", "", email)
  i := customer.List(params)

  // Check if any customer exists with the provided email
  for i.Next() {
    return true, nil
  }
  if err := i.Err(); err != nil {
    return false, err
  }

  return false, nil
}

// HandleCheckCustomerExists handles the request to check if a customer exists
func HandleCheckCustomerExists(c *gin.Context) {
  email := c.Param("email")

  exists, err := CheckCustomerExists(email)
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check if customer exists"})
    return
  }

  c.JSON(http.StatusOK, gin.H{"exists": exists})
}

// CheckSubscriptionStatus checks the subscription status of a Stripe customer
func CheckSubscriptionStatusByEmail(email string) (bool, error) {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	// List customers by email
	params := &stripe.CustomerListParams{}
	params.Filters.AddFilter("email", "", email)
	customers := customer.List(params)

	for customers.Next() {
		cust := customers.Customer()
    log.Printf("Checking customer: ID=%s, Email=%s\n", cust.ID, cust.Email)

		// List subscriptions for the customer
		subParams := &stripe.SubscriptionListParams{
			Customer: cust.ID,
		}
		subParams.AddExpand("data.default_payment_method")
		subs := sub.List(subParams)
    
		for subs.Next() {
			subscription := subs.Subscription()
      log.Printf("Subscription ID: %s, Status: %s\n", subscription.ID, subscription.Status)
			if subscription.Status == stripe.SubscriptionStatusActive {        
				return true, nil
			}
		}

		if err := subs.Err(); err != nil {
      log.Printf("Error listing subscriptions for customer %s: %v\n", cust.ID, err)
			return false, err
		}
	}

	if err := customers.Err(); err != nil {
    log.Printf("Error listing customers: %v\n", err)
		return false, err
	}

	return false, nil
}

func HandleSubscriptionStatus(c *gin.Context) {
	customerID := c.Param("customerID")

	isActive, err := CheckSubscriptionStatusByEmail(customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check subscription status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"isActive": isActive})
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

	url := supabaseURL + "/rest/v1/user?email=eq." + supabaseID

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
		log.Printf("Update Supabase Error: %s", err)
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

	params := &stripe.CustomerListParams{}
	params.Filters.AddFilter("email", "", req.Email)
	i := customer.List(params)
	for i.Next() {
		existingCustomer := i.Customer()
		c.JSON(http.StatusOK, existingCustomer)
		return
	}

	stripeCustomer, err := createStripeCustomer(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Stripe customer"})
		return
	}

	if err := UpdateSupabaseUser(req.Email, stripeCustomer.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Supabase user"})
		return
	}

	c.JSON(http.StatusOK, stripeCustomer)
}
