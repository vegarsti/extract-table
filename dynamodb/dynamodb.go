package dynamodb

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func CreateTable(sess *session.Session) error {
	tableName := "Tables"
	billingMode := "PAY_PER_REQUEST"
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("Checksum"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("Checksum"),
				KeyType:       aws.String("HASH"),
			},
		},
		TableName:   aws.String(tableName),
		BillingMode: &billingMode,
	}
	svc := dynamodb.New(sess)
	if _, err := svc.CreateTable(input); err != nil && err.Error() != fmt.Sprintf("ResourceInUseException: Table already exists: %s", tableName) {
		return fmt.Errorf("create table: %w", err)
	}
	return nil
}

func PutTable(checksum string, table []byte) error {
	sess, err := session.NewSession()
	if err != nil {
		return fmt.Errorf("unable to create session: %w", err)
	}
	svc := dynamodb.New(sess)
	putInput := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"Checksum":  {S: &checksum},
			"JSONTable": {B: table},
		},
		TableName: aws.String("Tables"),
	}
	if _, err := svc.PutItem(putInput); err != nil {
		return fmt.Errorf("put item: %w", err)
	}
	return nil
}

func GetTable(checksum string) ([]byte, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("unable to create session: %w", err)
	}
	svc := dynamodb.New(sess)
	projection := "JSONTable"
	putInput := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"Checksum": {S: &checksum},
		},
		ProjectionExpression: &projection,
		TableName:            aws.String("Tables"),
	}
	output, err := svc.GetItem(putInput)
	if err != nil {
		return nil, fmt.Errorf("get item: %w", err)
	}
	table, ok := output.Item["JSONTable"]
	if !ok {
		return nil, nil
	}
	return table.B, nil
}

func VerifyAPIKey(key string) (bool, error) {
	sess, err := session.NewSession()
	if err != nil {
		return false, fmt.Errorf("unable to create session: %w", err)
	}
	svc := dynamodb.New(sess)
	projection := "taken"
	getInput := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"key": {S: &key},
		},
		ProjectionExpression: &projection,
		TableName:            aws.String("api-keys"),
	}
	output, err := svc.GetItem(getInput)
	if err != nil {
		return false, fmt.Errorf("get item: %w", err)
	}
	if _, ok := output.Item["taken"]; !ok {
		return false, nil
	}
	return true, nil
}
