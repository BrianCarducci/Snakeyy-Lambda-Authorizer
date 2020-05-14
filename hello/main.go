package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"

	"github.com/go-redis/redis/v7"
	uuid "github.com/satori/go.uuid"
)

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
var response events.APIGatewayProxyResponse

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	redisClient := redis.NewClient(&redis.Options{
		Addr: os.Getenv("ELASTICACHE_ENDPOINT"),
		Password: "",
		DB: 0,
	})

	incomingState := request.QueryStringParameters["state"]
	if incomingState == "" {
		newState := uuid.NewV4()
		err := redisClient.Set(newState.String(), newState.String(), time.Minute).Err()
		if err != nil {
			fmt.Println("Failed to add state to redis: " + err.Error())
				return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body: "Failed to add state to redis: " + err.Error(),
			}, nil
		}
		// Send state back to user as cookie and redirect user to log in to instagram
		instaAuthorizeReqData := url.Values{}
		instaAuthorizeReqData.Add("client_id", "270736967654851")
		instaAuthorizeReqData.Add("redirect_uri", "https://2afo5m8bll.execute-api.us-east-1.amazonaws.com/dev/hello/")
		instaAuthorizeReqData.Add("scope", "user_profile,user_media")
		instaAuthorizeReqData.Add("response_type", "code")
		instaAuthorizeReqData.Add("state", newState.String())
		return events.APIGatewayProxyResponse{
			StatusCode: 302,
			Headers: map[string]string{
				"Location": "https://api.instagram.com/oauth/authorize?" + instaAuthorizeReqData.Encode(),
				"Set-Cookie": "state="+newState.String()+";",
			},
		}, nil
	}
	// else, incoming state exists, so check with redis
	err := redisClient.Get(incomingState).Err()
	if err != nil {
		fmt.Println("State: " + incomingState + "Not found in redis. Err: " + err.Error())
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body: "State: " + incomingState + "Not found in redis. Err: " + err.Error(),
		}, nil
	}

	fmt.Println("Received body: ", request.Body)

	code := request.QueryStringParameters["code"]
	fmt.Println("code: " + code)

	if code == "" {
		fmt.Println("No authorization code supplied")
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body: "No authorization code supplied",
		}, nil
	}

	secret, err := getSecretByName("SNAKEYY_CLIENT_SECRET")
	if err != nil {
		fmt.Println("Failed to get secret: " + err.Error())
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body: "Failed to get secret: " + err.Error(),
		}, nil
	}

	form := url.Values{}
	form.Add("code", code)
	form.Add("client_id", "270736967654851")
	form.Add("client_secret", secret)
	form.Add("grant_type", "authorization_code")
	form.Add("redirect_uri", "https://2afo5m8bll.execute-api.us-east-1.amazonaws.com/dev/hello/")
	
	tokenReq, err := http.NewRequest("POST", "https://api.instagram.com/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		fmt.Println("Failed to assemble token request: " + err.Error())
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body: "Failed to assemble request: " + err.Error(),
		}, nil
	}
	tokenReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	tokenRes, err := http.DefaultClient.Do(tokenReq)
	if err != nil {
		fmt.Println("Error occurred in making token request: " + err.Error())
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body: "Error occurred in making token request: " + err.Error(),
		}, nil
	}

	tokenResBody, err := ioutil.ReadAll(tokenRes.Body)
	if err != nil {
		fmt.Println("Failed to read token response body: " + err.Error())
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body: "Failed to read token response body: " + err.Error(),
		}, nil
	}
	tokenResponse := TokenResponse{}
	if err := json.Unmarshal(tokenResBody, &tokenResponse); err != nil {
		fmt.Println("Failed to unmarshal token response: " + err.Error())
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body: "Failed to unmarshal token response: " + err.Error(),
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body: string(tokenResBody),
	}, nil

	// return events.APIGatewayProxyResponse{
	// 	StatusCode: 301,
	// 	Headers: map[string]string{
	// 		"Location": "https://snakeyy-5a5bb.firebaseapp.com",
	// 		"Set-Cookie": "token="+tokenResponse.AccessToken+";",
	// 	},
	// 	Body: string(tokenResBody),
	// }, nil
}

func main() {
	lambda.Start(Handler)
}

func getSecretByName(secretName string) (string, error) {
	region := "us-east-1"

	//Create a Secrets Manager client
	svc := secretsmanager.New(session.New(),
                                  aws.NewConfig().WithRegion(region))
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretName),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}

	// In this sample we only handle the specific exceptions for the 'GetSecretValue' API.
	// See https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html

	result, err := svc.GetSecretValue(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
				case secretsmanager.ErrCodeDecryptionFailure:
				// Secrets Manager can't decrypt the protected secret text using the provided KMS key.
				fmt.Println(secretsmanager.ErrCodeDecryptionFailure, aerr.Error())

				case secretsmanager.ErrCodeInternalServiceError:
				// An error occurred on the server side.
				fmt.Println(secretsmanager.ErrCodeInternalServiceError, aerr.Error())

				case secretsmanager.ErrCodeInvalidParameterException:
				// You provided an invalid value for a parameter.
				fmt.Println(secretsmanager.ErrCodeInvalidParameterException, aerr.Error())

				case secretsmanager.ErrCodeInvalidRequestException:
				// You provided a parameter value that is not valid for the current state of the resource.
				fmt.Println(secretsmanager.ErrCodeInvalidRequestException, aerr.Error())

				case secretsmanager.ErrCodeResourceNotFoundException:
				// We can't find the resource that you asked for.
				fmt.Println(secretsmanager.ErrCodeResourceNotFoundException, aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return "", err
	}

	var secretStringifiedMap string
	if result.SecretString != nil {
		secretStringifiedMap = *result.SecretString
	} else {
		return "", errors.New("The secret value is nil")
	}
	
	secretMap := map[string]string{}
	if err := json.Unmarshal([]byte(secretStringifiedMap), &secretMap); err != nil {
		return "", err
	}

	return secretMap[secretName], nil
}