.PHONY: docker-run
docker-run: vet lint
	@docker-compose up --build -d --remove-orphans

.PHONY: docker-up
docker-up:
	@docker-compose up -d

vet:  ## Run go vet
	go vet ./...

lint: ## Run go lint
	golangci-lint run
