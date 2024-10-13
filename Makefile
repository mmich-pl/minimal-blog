up:
	docker-compose up -d
	while [ "$$(docker inspect -f '{{.State.Running}}' scylla-load-keyspace)" = "true" ]; do \
		sleep 2; \
	done
	volume_names=$$(docker inspect scylla-load-keyspace -f '{{range .Mounts}}{{println .Name}}{{end}}'); \
	docker rm -f scylla-load-keyspace > /dev/null 2>&1; \
	for volume_name in $$volume_names; do \
		docker volume rm $$volume_name > /dev/null 2>&1; \
	done

down:
	docker-compose down

GO_SERVER_CMD=go run server/main.go
NEXTJS_CMD=npm --prefix minimal-blog run dev
HEALTHCHECK_URL=http://localhost:8080/health
HEALTHCHECK_RETRY_INTERVAL=3
HEALTHCHECK_MAX_RETRIES=30

.PHONY: run
run: run-go-server run-nextjs

.PHONY: run-go-server
run-go-server:
	@echo "Starting Go server..."
	@$(GO_SERVER_CMD) &

.PHONY: run-nextjs
run-nextjs: wait-for-server
	@echo "Starting Next.js project..."
	@$(NEXTJS_CMD)

.PHONY: wait-for-server
wait-for-server:
	@echo "Waiting for Go server to be healthy at $(HEALTHCHECK_URL)..."
	@RETRY=0; \
	while ! curl -s $(HEALTHCHECK_URL) > /dev/null; do \
		if [ $$RETRY -ge $(HEALTHCHECK_MAX_RETRIES) ]; then \
			echo "Go server health check failed after $$RETRY retries."; \
			exit 1; \
		fi; \
		echo "Go server not healthy yet. Retrying in $(HEALTHCHECK_RETRY_INTERVAL) second..."; \
		sleep $(HEALTHCHECK_RETRY_INTERVAL); \
		RETRY=$$((RETRY+1)); \
	done
	@echo "Go server is healthy."

.SILENT: