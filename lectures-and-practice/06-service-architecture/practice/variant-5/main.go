package main

import (
	"log"
	"net/http"

	"github.com/course-go-autumn-2026/practice6-variant5-comments/adapter"
	"github.com/course-go-autumn-2026/practice6-variant5-comments/handler"
	"github.com/course-go-autumn-2026/practice6-variant5-comments/repository"
	"github.com/course-go-autumn-2026/practice6-variant5-comments/service"
)

func main() {
	svc := service.NewCommentService(
		repository.NewMemoryCommentRepository(),
		repository.NewMemoryAuthorRepository(),
		adapter.HTTPSpamFilter{URL: "http://spam-filter.internal/check"},
		adapter.HTTPToxicityChecker{URL: "http://toxicity-check.internal/score"},
	)

	mux := http.NewServeMux()
	handler.NewCommentHandler(svc).Register(mux)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
