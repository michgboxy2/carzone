package login

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/michgboxy2/carzone/models"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var credentials models.Credentials

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		fmt.Println(err)
		http.Error(w, "Invalid Request Body", http.StatusBadRequest)
		return
	}

	valid := (credentials.UserName == "admin" && credentials.Password == "admin123")

	if !valid {
		http.Error(w, "Incorrect Username or password", http.StatusUnauthorized)
		return
	}

	tokenString, err := GenerateToken(credentials.UserName)

	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		log.Println("Error generating token: ", err)
		return
	}

	response := map[string]string{"token": tokenString}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func GenerateToken(userName string) (string, error) {
	expiration := time.Now().Add(24 * time.Hour)

	claims := &jwt.StandardClaims{
		ExpiresAt: expiration.Unix(),
		IssuedAt:  time.Now().Unix(),
		Subject:   userName,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte("some_value"))

	if err != nil {
		return "", err
	}

	return signedToken, nil
}
