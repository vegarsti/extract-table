all:
	GOARCH=amd64 GOOS=linux go build main.go
	zip function.zip main
	aws lambda update-function-code --function-name ${LAMBDA_FUNCTION_NAME} --zip-file fileb://function.zip  | cat
