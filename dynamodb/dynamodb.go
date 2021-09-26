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

func PutTable(sess *session.Session, checksum string, table []byte) error {
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

func GetTable(sess *session.Session, checksum string) ([]byte, error) {
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
