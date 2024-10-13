FROM golang:1.23 as build
WORKDIR /app
COPY . .
RUN go build -o myapp
EXPOSE 8080
CMD ["/app/myapp"]
