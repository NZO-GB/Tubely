package main

import (
	"fmt"
	"net/http"
	"github.com/google/uuid"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
)

func validateUser(r *http.Request, jwtSecret string) (uuid.UUID, error) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		
		return uuid.Nil, fmt.Errorf("couldn't find JWT: %w", err)
	}

	userID, err := auth.ValidateJWT(token, jwtSecret)
	if err != nil {
		
		return uuid.Nil, fmt.Errorf("couldn't validate JWT: %w", err)
	}

	return userID, nil
}