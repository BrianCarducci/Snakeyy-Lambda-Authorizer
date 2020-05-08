package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
var response events.APIGatewayProxyResponse

func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
	
	form := url.Values{}
	form.Add("code", code)
	form.Add("client_id", "270736967654851")
	form.Add("client_secret", os.Getenv("SNAKEYY_CLIENT_SECRET"))
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

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body: string(tokenResBody),
	}, nil

	// return events.APIGatewayProxyResponse{Body: request.Body, StatusCode: 200}, nil
}

func main() {
	lambda.Start(Handler)
}
