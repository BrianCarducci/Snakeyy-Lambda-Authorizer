package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

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

	code := request.PathParameters["code"]
	fmt.Println("code: " + code)

	if len(code) == 0 {
		fmt.Println("No authorization code supplied")
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body: "No authorization code supplied",
		}, nil
	}
	
	values := url.Values{}
	values.Add("code", code)
	values.Add("CLIENT_ID", "270736967654851")
	values.Add("CLIENT_SECRET", os.Getenv("SNAKEYY_CLIENT_SECRET"))
	values.Add("grant_type", "authorization_code")
	values.Add("redirect_uri", "https://2afo5m8bll.execute-api.us-east-1.amazonaws.com/dev/hello")
	
	tokenReq, err := http.NewRequest("POST", "https://instagram.com/oauth/access_token?" + values.Encode(), nil)
	if err != nil {
		fmt.Println("Failed to assemble token request: " + err.Error())
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body: "Failed to assemble request: " + err.Error(),
		}, nil
	}

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

// Handler is our lambda handler invoked by the `lambda.Start` function call
// func Handler(ctx context.Context) (Response, error) {
// 	var buf bytes.Buffer

// 	body, err := json.Marshal(map[string]interface{}{
// 		"message": "Go Serverless v1.0! Your function executed successfully!",
// 	})
// 	if err != nil {
// 		return Response{StatusCode: 404}, err
// 	}
// 	json.HTMLEscape(&buf, body)

// 	resp := Response{
// 		StatusCode:      200,
// 		IsBase64Encoded: false,
// 		Body:            buf.String(),
// 		Headers: map[string]string{
// 			"Content-Type":           "application/json",
// 			"X-MyCompany-Func-Reply": "hello-handler",
// 		},
// 	}

// 	return resp, nil
// }

// func main() {
// 	lambda.Start(Handler)
// }
