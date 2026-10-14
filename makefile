.PHONY: build kill-app clean run

build:
	go build -o ./bin/job-aggregator-server ./cmd/server/
	go build -o ./bin/job-aggregator-ui ./cmd/app

kill-app:
	-pkill -f '/bin/job-aggregator-server'
	-pkill -f '/bin/job-aggregator-ui'

clean:
	rm -rf ./bin/job-aggregator-server ./bin/job-aggregator-ui

run: build kill-app
	./bin/job-aggregator-server &
	./bin/job-aggregator-ui &