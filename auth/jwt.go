package auth

import (

  "os"
  
  "github.com/dgrijalva/jwt-go"
  "time"
  "net/http"
  "fmt"
)

var jwtKey = []byte(os.Getenv("JWT_SECRET_KEY"))

// Create a struct to read the username and password from the request body
type Credentials struct {
    Password string `json:"password"`
    Username string `json:"username"`
}

// Create a struct that will be encoded to a JWT
// Add other user fields you might need
type Claims struct {
    Username string `json:"username"`
    jwt.StandardClaims
}

// Create JWT token
func CreateToken(username string) (string, error) {
  expirationTime := time.Now().Add(5 * time.Minute)
  claims := &Claims{
    Username: username,
    StandardClaims: jwt.StandardClaims{
      ExpiresAt: expirationTime.Unix(),
    },
  }

  token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
  tokenString, err := token.SignedString(jwtKey)

  if err != nil {
    return "", err
  }

  return tokenString, nil
}

// Middleware to protect private routes
func IsAuthorized(endpoint func(http.ResponseWriter, *http.Request)) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

        if r.Header["Token"] != nil {

            token, err := jwt.ParseWithClaims(r.Header["Token"][0], &Claims{}, func(token *jwt.Token) (interface{}, error) {
                return jwtKey, nil
            })

            if err == nil {
                if token.Valid {
                    endpoint(w, r)
                }
            } else {
                fmt.Fprintf(w, err.Error())
            }

        } else {
            fmt.Fprintf(w, "Not Authorized")
        }
    })
}
