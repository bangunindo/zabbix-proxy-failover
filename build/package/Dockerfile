FROM golang:1.17 AS buildStage
COPY go.mod go.sum /source_code/
WORKDIR /source_code
RUN go mod download
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o /zabbix-proxy-failover -race github.com/bangunindo/zabbix-proxy-failover/cmd/zabbix-proxy-failover
RUN chmod +x /zabbix-proxy-failover


FROM debian

RUN apt-get update \
    && apt-get install -y ca-certificates \
    && update-ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=buildStage /zabbix-proxy-failover /zabbix-proxy-failover

ENTRYPOINT ["/zabbix-proxy-failover"]
