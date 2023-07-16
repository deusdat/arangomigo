FROM golang:1.19

WORKDIR /opt/

RUN apt-get update && apt-get install -y wait-for-it

COPY . .
