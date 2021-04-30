package dynamock

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// ToTable - method for set Table expectation
func (e *UpdateItemExpectation) ToTable(table string) *UpdateItemExpectation {
	e.table = &table
	return e
}

// WithKeys - method for set Keys expectation
func (e *UpdateItemExpectation) WithKeys(keys map[string]*dynamodb.AttributeValue) *UpdateItemExpectation {
	e.key = keys
	return e
}

// WithConditionExpression - method for setting a ConditionExpression expectation
func (e *UpdateItemExpectation) WithConditionExpression(expr *string) *UpdateItemExpectation {
	e.conditionExpression = expr
	return e
}

// WithExpressionAttributeNames - method for setting a ExpressionAttributeNames expectation
func (e *UpdateItemExpectation) WithExpressionAttributeNames(names map[string]*string) *UpdateItemExpectation {
	e.expressionAttributeNames = names
	return e
}

// WithExpressionAttributeValues - method for setting a ExpressionAttributeValues expectation
func (e *UpdateItemExpectation) WithExpressionAttributeValues(attrs map[string]*dynamodb.AttributeValue) *UpdateItemExpectation {
	e.expressionAttributeValues = attrs
	return e
}

// WithUpdateExpression - method for setting a UpdateExpression expectation
func (e *UpdateItemExpectation) WithUpdateExpression(expr *string) *UpdateItemExpectation {
	e.updateExpression = expr
	return e
}

func (e *UpdateItemExpectation) WithSetAttributeValueExpression(expr *string) *UpdateItemExpectation {
	e.setAttributeValueExpression = expr
	return e
}

// Updates - method for set Updates expectation
func (e *UpdateItemExpectation) Updates(attrs map[string]*dynamodb.AttributeValueUpdate) *UpdateItemExpectation {
	e.attributeUpdates = attrs
	return e
}

// WillReturns - method for set desired result
func (e *UpdateItemExpectation) WillReturns(res dynamodb.UpdateItemOutput) *UpdateItemExpectation {
	e.output = &res
	return e
}

// UpdateItem - this func will be invoked when test running matching expectation with actual input
func (e *MockDynamoDB) UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error) {
	if len(e.dynaMock.UpdateItemExpect) > 0 {
		x := e.dynaMock.UpdateItemExpect[0] //get first element of expectation

		if x.table != nil {
			if *x.table != *input.TableName {
				return &dynamodb.UpdateItemOutput{}, fmt.Errorf("Expect table %s but found table %s", *x.table, *input.TableName)
			}
		}

		if x.key != nil {
			if !reflect.DeepEqual(x.key, input.Key) {
				return &dynamodb.UpdateItemOutput{}, fmt.Errorf("Expect key %+v but found key %+v", x.key, input.Key)
			}
		}

		if x.attributeUpdates != nil {
			if !reflect.DeepEqual(x.attributeUpdates, input.AttributeUpdates) {
				return &dynamodb.UpdateItemOutput{}, fmt.Errorf("Expect key %+v but found key %+v", x.attributeUpdates, input.AttributeUpdates)
			}
		}

		if x.conditionExpression != nil {
			if !reflect.DeepEqual(x.conditionExpression, input.ConditionExpression) {
				return &dynamodb.UpdateItemOutput{}, fmt.Errorf("Expect key %+v but found key %+v", x.conditionExpression, input.ConditionExpression)
			}
		}

		if x.expressionAttributeNames != nil {
			if !reflect.DeepEqual(x.expressionAttributeNames, input.ExpressionAttributeNames) {
				return &dynamodb.UpdateItemOutput{}, fmt.Errorf("Expect key %+v but found key %+v", x.expressionAttributeNames, input.ExpressionAttributeNames)
			}
		}

		if x.expressionAttributeValues != nil {
			if !reflect.DeepEqual(x.expressionAttributeValues, input.ExpressionAttributeValues) {
				return &dynamodb.UpdateItemOutput{}, fmt.Errorf("Expect key %+v but found key %+v", x.expressionAttributeValues, input.ExpressionAttributeValues)
			}
		}

		if x.updateExpression != nil {
			if !reflect.DeepEqual(x.updateExpression, input.UpdateExpression) {
				return &dynamodb.UpdateItemOutput{}, fmt.Errorf("Expect key %+v but found key %+v", x.updateExpression, input.UpdateExpression)
			}
		}

		if x.setAttributeValueExpression != nil {

		}

		// delete first element of expectation
		e.dynaMock.UpdateItemExpect = append(e.dynaMock.UpdateItemExpect[:0], e.dynaMock.UpdateItemExpect[1:]...)

		return x.output, nil
	}

	return &dynamodb.UpdateItemOutput{}, fmt.Errorf("Update Item Expectation Not Found")
}

type parsedUpdateExpression struct {
	ADDExpressions    []addExpression
	DELETEExpressions []string
	REMOVEExpressions []string
	SETExpressions    []string
}

type operation string

const (
	ADD    operation = "ADD"
	DELETE operation = "DELETE"
	REMOVE operation = "REMOVE"
	SET    operation = "SET"
)

type operationIndexTuple struct {
	Index     int
	Operation operation
}

type addExpression struct {
	path  string
	value string
}

func extractAddPathValuePairs(addExpr string) []addExpression {
	re := regexp.MustCompile(`ADD\s+((\S+\s+[\w:]+\s*,?\s*)+)`)
	subMatchRe := regexp.MustCompile(`(\S+)\s+([\w:]+)\s*,?\s*`)
	subMatches := re.FindStringSubmatch(addExpr)
	var result []addExpression
	if subMatches == nil {
		return result
	}

	pairMatches := subMatchRe.FindAllStringSubmatch(subMatches[1], -1)
	if pairMatches == nil {
		return result
	}
	for _, subMatch := range pairMatches {
		result = append(result, addExpression{subMatch[1], subMatch[2]})
	}
	return result
}

func parseUpdateExpression(updateExpression string) parsedUpdateExpression {
	addOp := operationIndexTuple{strings.Index(updateExpression, "ADD"), ADD}
	deleteOp := operationIndexTuple{strings.Index(updateExpression, "DELETE"), DELETE}
	removeOp := operationIndexTuple{strings.Index(updateExpression, "REMOVE"), REMOVE}
	setOp := operationIndexTuple{strings.Index(updateExpression, "SET"), SET}

	ops := []operationIndexTuple{
		addOp,
		deleteOp,
		removeOp,
		setOp,
	}
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].Index < ops[j].Index
	})

	result := parsedUpdateExpression{}
	for opIdx, op := range ops {
		if op.Index < 0 {
			// op.Index should be -1 for operations that are not present in an update expression
			continue
		}
		// get the substring for the operation
		var substr string
		if opIdx+1 < len(ops) {
			// We don't need to worry about the case where (opIdx+1).Index is -1, because we're iterating through a
			// ascending sorted array.
			substr = updateExpression[op.Index:ops[opIdx+1].Index]
		} else {
			substr = updateExpression[op.Index:]
		}
		// apply the operation specific parsing
		switch op.Operation {
		case ADD:
			result.ADDExpressions = extractAddPathValuePairs(substr)
		}
	}
	return result
}

// UpdateItemWithContext - this func will be invoked when test running matching expectation with actual input
func (e *MockDynamoDB) UpdateItemWithContext(ctx aws.Context, input *dynamodb.UpdateItemInput, opt ...request.Option) (*dynamodb.UpdateItemOutput, error) {
	if len(e.dynaMock.UpdateItemExpect) > 0 {
		x := e.dynaMock.UpdateItemExpect[0] //get first element of expectation

		if x.table != nil {
			if *x.table != *input.TableName {
				return &dynamodb.UpdateItemOutput{}, fmt.Errorf("Expect table %s but found table %s", *x.table, *input.TableName)
			}
		}

		if x.key != nil {
			if !reflect.DeepEqual(x.key, input.Key) {
				return &dynamodb.UpdateItemOutput{}, fmt.Errorf("Expect key %+v but found key %+v", x.key, input.Key)
			}
		}

		if x.attributeUpdates != nil {
			if !reflect.DeepEqual(x.attributeUpdates, input.AttributeUpdates) {
				return &dynamodb.UpdateItemOutput{}, fmt.Errorf("Expect key %+v but found key %+v", x.attributeUpdates, input.AttributeUpdates)
			}
		}

		if x.conditionExpression != nil {
			if !reflect.DeepEqual(x.conditionExpression, input.ConditionExpression) {
				return &dynamodb.UpdateItemOutput{}, fmt.Errorf("Expect key %+v but found key %+v", x.conditionExpression, input.ConditionExpression)
			}
		}

		if x.expressionAttributeNames != nil {
			if !reflect.DeepEqual(x.expressionAttributeNames, input.ExpressionAttributeNames) {
				return &dynamodb.UpdateItemOutput{}, fmt.Errorf("Expect key %+v but found key %+v", x.expressionAttributeNames, input.ExpressionAttributeNames)
			}
		}

		if x.expressionAttributeValues != nil {
			if !reflect.DeepEqual(x.expressionAttributeValues, input.ExpressionAttributeValues) {
				return &dynamodb.UpdateItemOutput{}, fmt.Errorf("Expect key %+v but found key %+v", x.expressionAttributeValues, input.ExpressionAttributeValues)
			}
		}

		if x.updateExpression != nil {
			if !reflect.DeepEqual(x.updateExpression, input.UpdateExpression) {
				return &dynamodb.UpdateItemOutput{}, fmt.Errorf("Expect key %+v but found key %+v", x.updateExpression, input.UpdateExpression)
			}
		}
		// delete first element of expectation
		e.dynaMock.UpdateItemExpect = append(e.dynaMock.UpdateItemExpect[:0], e.dynaMock.UpdateItemExpect[1:]...)

		return x.output, nil
	}

	return &dynamodb.UpdateItemOutput{}, fmt.Errorf("Update Item With Context Expectation Not Found")
}
