GOOS=windows  GOARCH=amd64  go build -o builds/arangomigo-amd64.exe     cmd/arangomigo/main.go
GOOS=windows  GOARCH=386    go build -o builds/arangomigo-386.exe       cmd/arangomigo/main.go
GOOS=darwin   GOARCH=amd64  go build -o builds/arangomigo-amd64-darwin  cmd/arangomigo/main.go
GOOS=linux    GOARCH=amd64  go build -o builds/arangomigo-amd64-linux   cmd/arangomigo/main.go
GOOS=linux    GOARCH=386    go build -o builds/arangomigo-386-linux     cmd/arangomigo/main.go