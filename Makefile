SHELL := /bin/zsh

.PHONY: deps dev build test lint run docker-build frontend-deps frontend-build

deps:
	env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY -u http_proxy -u https_proxy -u all_proxy go mod tidy
	cd frontend && env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY -u http_proxy -u https_proxy -u all_proxy npm install

frontend-deps:
	cd frontend && env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY -u http_proxy -u https_proxy -u all_proxy npm install

dev:
	docker compose up --build

run:
	env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY -u http_proxy -u https_proxy -u all_proxy go run ./cmd/api

build:
	env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY -u http_proxy -u https_proxy -u all_proxy go build ./...
	cd frontend && env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY -u http_proxy -u https_proxy -u all_proxy npm run build

test:
	env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY -u http_proxy -u https_proxy -u all_proxy go test ./...
	cd frontend && env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY -u http_proxy -u https_proxy -u all_proxy npm run test

lint:
	env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY -u http_proxy -u https_proxy -u all_proxy go test ./...
	cd frontend && env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY -u http_proxy -u https_proxy -u all_proxy npm run lint

docker-build:
	docker build -f Dockerfile.api -t kubeaudit-revamp-api:latest .
	docker build -f frontend/Dockerfile -t kubeaudit-revamp-frontend:latest ./frontend

