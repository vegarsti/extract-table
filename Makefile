all: lambda cli

clean:
	rm -f lambda cli

lambda: clean
	GOARCH=amd64 GOOS=linux go build -o main cmd/lambda/main.go
	zip function.zip main
	aws lambda update-function-code --function-name ${LAMBDA_FUNCTION_NAME} --zip-file fileb://function.zip  | cat

cli: clean
	go build -o cli cmd/cli/main.go
