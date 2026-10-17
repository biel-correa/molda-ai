package main

import (
	"embed"
	"io/fs"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"molda-ai/internal/api"
)

//go:embed web/*
var embeddedFiles embed.FS

func main() {
	webFS, err := fs.Sub(embeddedFiles, "web")
	if err != nil {
		log.Fatal(err)
	}

	router := api.SetupRouter(webFS)

	// Automatic environment detection: AWS Lambda vs Local HTTP Server
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		adapter := ginadapter.NewV2(router)
		lambda.Start(adapter.ProxyWithContext)
	} else {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		log.Printf("MoldaAí running locally on http://localhost:%s\n", port)
		log.Fatal(router.Run(":" + port))
	}
}
