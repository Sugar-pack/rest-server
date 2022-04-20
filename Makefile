.PHONY: docker-run
docker-run:
	@docker-compose up --build -d --remove-orphans

.PHONY: docker-build
docker-build: vet
	@docker-compose build

.PHONY: docker-up
docker-up:
	docker-compose up -d

vet:  ## Run go vet
	go vet ./...

lint: ## Run go lint
	golangci-lint run

test:
	go test -race -count 1 -cover ./...
