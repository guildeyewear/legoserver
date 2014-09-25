package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetUser(t *testing.T) {
	handler := http.HandlerFunc(handleUserReq)
	recorder := httptest.NewRecorder()
	url := fmt.Sprintf("http://localhost:3000/user")
	req, _ := http.NewRequest("GET", url, nil)
	handler.ServeHTTP(recorder, req)
	fmt.Println(recorder)
}
