package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  "golang-idp-be/environment"
  "golang-idp-be/gateway/idpapi"
)

type DeleteVerificationRequest struct {
  Id               string `json:"id" binding:"required"`
  VerificationCode string `json:"verification_code" binding:"required"`
  RedirectTo       string `json:"redirect_to" binding:"required"`
}

type DeleteVerificationResponse struct {
  Id         string `json:"id" binding:"required"`
  Verified   bool   `json:"verifed" binding:"required"`
  RedirectTo string `json:"redirect_to" binding:"required"`
}

func PostDeleteVerification(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostRecoverVerification",
    })

    var input DeleteVerificationRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    denyResponse := DeleteVerificationResponse{
      Id: input.Id,
      Verified: false,
      RedirectTo: "",
    }

    identities, err := idpapi.FetchIdentitiesForSub(env.Driver, input.Id)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid id"})
      c.Abort();
      return
    }

    if identities == nil {
      log.WithFields(logrus.Fields{"id": input.Id}).Debug("Identity not found")
      c.JSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
      c.Abort();
      return
    }

    identity := identities[0];

    valid, err := idpapi.ValidatePassword(identity.OtpRecoverCode, input.VerificationCode)
    if err != nil {
      log.Debug(err.Error())
      log.WithFields(logrus.Fields{
        "id": denyResponse.Id,
        "verified": denyResponse.Verified,
        "redirect_to": denyResponse.RedirectTo,
      }).Debug("Delete verification rejected")
      c.JSON(http.StatusOK, denyResponse)
      c.Abort();
      return
    }

    if valid == true {

      n := idpapi.Identity{
        Id: identity.Id,
      }
      updatedIdentity, err := idpapi.DeleteIdentity(env.Driver, n)
      if err != nil {
        log.Debug(err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete identitiy failed"})
        c.Abort();
        return
      }

      acceptResponse := DeleteVerificationResponse{
        Id: updatedIdentity.Id,
        Verified: true,
        RedirectTo: input.RedirectTo,
      }
      log.WithFields(logrus.Fields{
        "id": acceptResponse.Id,
        "verified": acceptResponse.Verified,
        "redirect_to": acceptResponse.RedirectTo,
      }).Debug("Identity deleted")
      c.JSON(http.StatusOK, acceptResponse)
      return
    }

    // Deny by default
    log.WithFields(logrus.Fields{
      "id": denyResponse.Id,
      "verified": denyResponse.Verified,
      "redirect_to": denyResponse.RedirectTo,
    }).Debug("Delete verification rejected")
    c.JSON(http.StatusOK, denyResponse)
  }
  return gin.HandlerFunc(fn)
}