all:
	GOARCH=amd64 GOOS=linux go build -o lambda-handler lambda.go
	zip function.zip lambda-handler
	aws lambda update-function-code --function-name ${LAMBDA_FUNCTION_NAME} --zip-file fileb://function.zip  | cat
