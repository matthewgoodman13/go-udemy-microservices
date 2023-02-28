FROM alpine:latest

RUN mkdir /app

COPY frontendApp /app

CMD ["/app/frontendApp"]