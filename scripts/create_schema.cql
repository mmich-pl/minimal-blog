CREATE KEYSPACE log_storage WITH replication = {'class': 'SimpleStrategy', 'replication_factor': '1'}
                                   AND durable_writes = true;

USE log_storage;

CREATE TABLE logs
(
    timestamp  timestamp,
    level      text,
    message    text,
    attributes map<text, text>,
    PRIMARY KEY (timestamp)
);
