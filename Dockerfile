# syntax=docker/dockerfile:1

FROM golang:1.18.7-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY *.go ./

RUN go build -o /api

CMD "/api"
