package controller

import (
  "github.com/gin-gonic/gin"
  "net/http"
  "golang-idp-be/interfaces"
  "golang-idp-be/config"
  _ "os"
  _ "fmt"
  "io/ioutil"
  "encoding/json"
  "bytes"
)

func PostIdentitiesAuthenticate(c *gin.Context) {

  var input interfaces.PostIdentitiesAuthenticateRequest

  err := c.BindJSON(&input)

  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }

  client := &http.Client{}

  headers := map[string][]string{
    "Content-Type": []string{"application/json"},
    "Accept": []string{"application/json"},
  }

  req, err := http.NewRequest("GET", config.Hydra.LoginRequestUrl, nil)
  req.Header = headers

  q := req.URL.Query()
  q.Add("login_challenge", input.Challenge)
  req.URL.RawQuery = q.Encode()

  response, err := client.Do(req)

  responseData, err := ioutil.ReadAll(response.Body)

  var hydraLoginRequestResponse interfaces.HydraLoginRequestResponse
  json.Unmarshal(responseData, &hydraLoginRequestResponse)

  if hydraLoginRequestResponse.Skip {
    body, _ := json.Marshal(map[string]string{
      "subject": hydraLoginRequestResponse.Subject,
    })

    req, err = http.NewRequest("PUT", config.Hydra.LoginRequestAcceptUrl, bytes.NewBuffer(body))
    req.Header = headers

    q := req.URL.Query()
    q.Add("login_challenge", input.Challenge)
    req.URL.RawQuery = q.Encode()

    response, _ := client.Do(req)

    responseData, _ := ioutil.ReadAll(response.Body)

    var hydraLoginRequestAcceptResponse interfaces.HydraLoginRequestAcceptResponse
    json.Unmarshal(responseData, &hydraLoginRequestAcceptResponse)

    c.JSON(http.StatusOK, gin.H{
      "id": input.Id,
      "authenticated": true,
      "redirect_to": hydraLoginRequestAcceptResponse.RedirectTo,
    })

    return
  }


  if input.Id == "user-1" && input.Password == "1234" {

    // call hydra with accept login request
    body, _ := json.Marshal(map[string]string{
      "subject": input.Id,
    })

    req, err = http.NewRequest("PUT", config.Hydra.LoginRequestAcceptUrl, bytes.NewBuffer(body))
    req.Header = headers

    q := req.URL.Query()
    q.Add("login_challenge", input.Challenge)
    req.URL.RawQuery = q.Encode()

    response, _ := client.Do(req)

    responseData, _ := ioutil.ReadAll(response.Body)


    var hydraLoginRequestAcceptResponse interfaces.HydraLoginRequestAcceptResponse
    json.Unmarshal(responseData, &hydraLoginRequestAcceptResponse)


    c.JSON(http.StatusOK, gin.H{
      "id": input.Id,
      "authenticated": true,
      "redirect_to": hydraLoginRequestAcceptResponse.RedirectTo,
    })

    return
  }

  c.JSON(http.StatusOK, gin.H{
    "id": input.Id,
    "authenticated": false,
  })
}