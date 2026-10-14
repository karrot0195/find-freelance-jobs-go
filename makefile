.PHONY: build kill-app clean run deploy

build:
	go build -o ./bin/job-aggregator-server ./cmd/server/
	go build -o ./bin/job-aggregator-ui ./cmd/app

build-apk:
	gogio -target android -o ./bin/job-aggregator.apk ./cmd/app/main.go

kill-app:
	-pkill -f '/bin/job-aggregator-server'
	-pkill -f '/bin/job-aggregator-ui'

clean:
	rm -rf ./bin/job-aggregator-server ./bin/job-aggregator-ui

run: build kill-app
	APP_PORT=":8080" API_KEY=secret API_SECRET_KEY=secret ./bin/job-aggregator-server &
	./bin/job-aggregator-ui &

deploy: clean build
	@./.server/$(word 2,$(MAKECMDGOALS)).sh

# Catch-all to prevent make from treating the app name as a target
%:
	@: