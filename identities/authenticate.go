package identities

import (
  "net/http"

  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "golang-idp-be/config"
  "golang-idp-be/environment"
  "golang-idp-be/gateway/idpbe"
  "golang-idp-be/gateway/hydra"
)

type AuthenticateRequest struct {
  Id              string            `json:"id"`
  Password        string            `json:"password"`
  Challenge       string            `json:"challenge" binding:"required"`
}

type AuthenticateResponse struct {
  Id              string            `json:"id"`
  Authenticated   bool              `json:"authenticated"`
}

const HydraSessionTimeout = 120 // 2m

func PostAuthenticate(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "route.logid": route.LogId,
      "component": "identities",
      "func": "PostAuthenticate",
    })

    log.Debug("Received authentication request")

    var input AuthenticateRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    // Create a new HTTP client to perform the request, to prevent serialization
    hydraClient := hydra.NewHydraClient(env.HydraConfig)

    hydraLoginResponse, err := hydra.GetLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.login"), hydraClient, input.Challenge)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    if hydraLoginResponse.Skip {
      hydraLoginAcceptRequest := hydra.HydraLoginAcceptRequest{
        Subject: hydraLoginResponse.Subject,
        Remember: true,
        RememberFor: HydraSessionTimeout, // This means auto logout in hydra after 30 seconds!
      }

      hydraLoginAcceptResponse := hydra.AcceptLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.loginAccept"), hydraClient, input.Challenge, hydraLoginAcceptRequest)

      log.Debug("id:"+input.Id+" authenticated:true redirect_to:"+hydraLoginAcceptResponse.RedirectTo)
      c.JSON(http.StatusOK, gin.H{
        "id": input.Id,
        "authenticated": true,
        "redirect_to": hydraLoginAcceptResponse.RedirectTo,
      })
      c.Abort()
      return
    }

    // Only challenge is required in the request, but no need to ask DB for empty id.
    if input.Id == "" {
      log.Debug("id:"+input.Id+" authenticated:false redirect_to:")
      c.JSON(http.StatusOK, gin.H{
        "id": input.Id,
        "authenticated": false,
      })
      c.Abort()
      return
    }

    identities, err := idpbe.FetchIdentitiesForSub(env.Driver, input.Id)
    if err != nil {
      log.Debug("id:"+input.Id+" authenticated:false redirect_to:")
      c.JSON(http.StatusOK, gin.H{
        "id": input.Id,
        "authenticated": false,
      })
      c.Abort()
      return;
    }

    if identities != nil {

      // FIXME: Fail if identities contains more than one. Hint: Missing a unique constraint in the db schema?
      identity := identities[0];

      valid, _ := idpbe.ValidatePassword(identity.Password, input.Password)
      if valid == true {
        hydraLoginAcceptRequest := hydra.HydraLoginAcceptRequest{
          Subject: identity.Id,
          Remember: true,
          RememberFor: HydraSessionTimeout, // This means auto logout in hydra after 30 seconds!
        }

        hydraLoginAcceptResponse := hydra.AcceptLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.loginAccept"), hydraClient, input.Challenge, hydraLoginAcceptRequest)

        log.Debug("id:"+identity.Id+" authenticated:true redirect_to:"+hydraLoginAcceptResponse.RedirectTo)
        c.JSON(http.StatusOK, gin.H{
          "id": identity.Id,
          "authenticated": true,
          "redirect_to": hydraLoginAcceptResponse.RedirectTo,
        })
        c.Abort()
        return
      }

    } else {
      log.Info("No identities found")
    }

    // Deny by default
    log.Debug("id:"+input.Id+" authenticated:false redirect_to:")
    c.JSON(http.StatusOK, gin.H{
      "id": input.Id,
      "authenticated": false,
    })
  }
  return gin.HandlerFunc(fn)
}
