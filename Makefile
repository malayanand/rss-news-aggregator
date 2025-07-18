up:
	docker-compose up -d

down:
	docker-compose down

tidy:
	go mod tidy

start:
	# turn on “export all following vars”
	set -o allexport; \
	# source your .env file
	. ./.env; \
	# turn that behavior off again
	set +o allexport; \
	# now run your ingestor with just those vars in its env
	go run cmd/ingestor/main.go

