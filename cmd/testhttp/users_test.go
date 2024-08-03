package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateUser(t *testing.T) {
	type UserBody struct {
		Email     string `json:"email"`
		Firstname string `json:"firstname"`
		Lastname  string `json:"lastname"`
		Password  string `json:"password"`
		Username  string `json:"username"`
	}

	u := UserBody{
		Email:     "johndoe@example.com",
		Firstname: "Johnaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Lastname:  "Doe",
		Password:  "password123",
		Username:  "johndoe",
	}

	b, err := json.Marshal(u)
	if err != nil {
		require.Nilf(t, err, "failed to marshal user: %v", err)
		return
	}

	req, err := http.NewRequest("POST", "http://localhost:8080/users", bytes.NewBuffer(b))
	if err != nil {
		require.Nilf(t, err, "failed to create request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		require.Nilf(t, err, "failed to send request: %+v", err)
		return
	}
	defer resp.Body.Close()

	t.Logf("response Status: %s", resp.Status)
	t.Logf("response Headers: %s", resp.Header)
	body, _ := io.ReadAll(resp.Body)
	t.Logf("response Body: %s", body)

}
