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

.SILENT: