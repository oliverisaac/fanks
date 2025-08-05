APP_NAME ?= $(shell basename ${PWD})

.PHONY: tailwind-dev
tailwind-dev:
	tailwindcss -i ./static/css/input.css -o ./static/css/style.css

.PHONY: tailwind-build
tailwind-build:
	tailwindcss -i ./static/css/input.css -o ./static/css/style.min.css --minify

.PHONY: templ-generate
templ-generate:
	templ generate

.PHONY: templ-watch
templ-watch:
	templ generate --watch

.PHONY: go-build
go-build:
	go build -o ./tmp/$(APP_NAME) ./cmd/$(APP_NAME)/
	
.PHONY: dev-build
dev-build: templ-generate tailwind-dev go-build

clean:
	rm docker-run

docker-run:
	echo "docker run --env-file .env --name ${APP_NAME} --rm -v $$(pwd)/tmp/:/data/ --env FANKS_DB_PATH="/data/${APP_NAME}.db" -i -p 8080:8080 oliverisaac/${APP_NAME}:latest" > ./docker-run
	chmod +x ./docker-run

.PHONY: docker-build
docker-build: templ-generate docker-run
	docker kill ${APP_NAME} || true
	docker build --build-arg VERSION_TAG="$$(git describe --tags HEAD)" -t oliverisaac/${APP_NAME}:latest .

cert.key:
	openssl genrsa -out cert.key 2048

cert.pem: cert.key
	openssl req -x509 -key cert.key -out cert.pem -sha256 -days 3650 -nodes -subj "/C=XX/ST=StateName/L=CityName/O=CompanyName/OU=CompanySectionName/CN=localhost"

.PHONY: certs
certs: cert.key cert.pem

.PHONY: dev certs
dev: docker-build
	air

.PHONY: build
build: templ-generate tailwind-build 
	go build -ldflags "-X main.Environment=production" -o ./bin/$(APP_NAME) ./cmd/$(APP_NAME)/

.PHONY: vet
vet:
	go vet ./...

.PHONY: staticcheck
staticcheck:
	staticcheck ./...

.PHONY: test
test:
	  go test -race -v -timeout 30s ./...

IMAGE_NAME = oliverisaac/${APP_NAME}:latest
release:
	./k8s-release.sh "${IMAGE_NAME}"

.PHONY: logs
logs:
	kubectl logs -f -n ${APP_NAME} deployments/${APP_NAME}
