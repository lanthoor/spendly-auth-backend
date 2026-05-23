.PHONY: run test tidy lint deploy

run:
	PORT=8080 go run ./cmd/main.go

test:
	go test ./... -v

tidy:
	go mod tidy

lint:
	golangci-lint run ./...

deploy:
	gcloud functions deploy VerifyIntegrity \
		--gen2 \
		--runtime=go123 \
		--region=us-central1 \
		--source=. \
		--entry-point=VerifyIntegrity \
		--trigger-http \
		--allow-unauthenticated \
		--memory=256Mi \
		--timeout=10s \
		--min-instances=0 \
		--max-instances=1 \
		--service-account=spendly-integrity@$$(gcloud config get-value project).iam.gserviceaccount.com
