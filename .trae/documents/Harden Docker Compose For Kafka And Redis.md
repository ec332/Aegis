## Goals
- Run Kafka (KRaft) and Redis as containers with persistence and healthchecks
- Secure local access while keeping services functional
- Update service dependencies to wait for healthy Kafka/Redis
- Provide verification commands

## Compose Changes
- Kafka (single broker, KRaft):
  - Image: bitnami/kafka:latest
  - Volumes: `kafka-data:/bitnami/kafka`
  - Environment:
    - `KAFKA_ENABLE_KRAFT=yes`
    - `KAFKA_CFG_NODE_ID=1`
    - `KAFKA_CFG_PROCESS_ROLES=broker,controller`
    - `KAFKA_CFG_CONTROLLER_LISTENER_NAMES=CONTROLLER`
    - `KAFKA_CFG_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093`
    - `KAFKA_CFG_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092`
    - `KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=1@kafka:9093`
    - `KAFKA_CFG_OFFSETS_TOPIC_REPLICATION_FACTOR=1`
    - `KAFKA_CFG_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1`
    - `KAFKA_CFG_TRANSACTION_STATE_LOG_MIN_ISR=1`
    - `KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE=true`
  - Ports: bind to localhost only `127.0.0.1:9092:9092` (optional; omit to keep internal only)
  - Healthcheck: `kafka-topics.sh --bootstrap-server kafka:9092 --list`

- Redis:
  - Image: redis:7-alpine
  - Command: `redis-server --requirepass ${REDIS_PASSWORD} --appendonly yes`
  - Volumes: `redis-data:/data`
  - Ports: bind to localhost `127.0.0.1:6379:6379` (optional)
  - Healthcheck: `redis-cli -a ${REDIS_PASSWORD} ping` expecting `PONG`

- Services depends_on:
  - For API Gateway, Market, Wallet, Settlement, Transaction:
    - `depends_on:` set Kafka to `service_started` (or healthy if healthcheck available)
    - `depends_on:` set Redis to `service_healthy`

## Service Configuration
- API Gateway and services keep using internal hostnames:
  - Kafka bootstrap: `kafka:9092`
  - Redis URL: `redis://:${REDIS_PASSWORD}@redis:6379`
- Compose `.env` add `REDIS_PASSWORD` (non-secret dev value) and ensure not committed for production secrets

## Security Measures
- Bind ports to `127.0.0.1` to avoid external exposure
- Set Redis password; avoid logging credentials
- Keep Kafka PLAINTEXT for local dev; do not expose publicly
- Use Docker network defaults for internal name resolution

## Verification
- Kafka:
  - `docker exec -it aegis-kafka-1 bash -lc 'kafka-topics.sh --bootstrap-server kafka:9092 --list'`
  - Produce/consume with `kcat -b kafka:9092 -L` (if `kcat` available)
- Redis:
  - `docker exec -it aegis-redis-1 redis-cli -a $REDIS_PASSWORD ping` → `PONG`
  - `docker exec -it aegis-redis-1 redis-cli -a $REDIS_PASSWORD set test 123 && redis-cli -a $REDIS_PASSWORD get test`
- Services:
  - `curl -i http://localhost:8080/health`
  - `grpcurl -plaintext localhost:50051 list` (Market)
  - `curl -i http://localhost:5555/health` (Transaction)

## Documentation
- Update README with Kafka/Redis container notes, security bindings, env (`REDIS_PASSWORD`), and verification commands

## Acceptance Criteria
- Kafka and Redis containers run with persistence and healthchecks
- Services start only after Kafka/Redis ready
- No external exposure beyond localhost bindings
- README documents setup and verification clearly