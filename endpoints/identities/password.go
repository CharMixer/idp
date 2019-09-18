package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  . "github.com/charmixer/idp/client"
)

func PutPassword(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostPassword",
    })

    var input IdentitiesPasswordRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    identity, exists, err := idp.FetchIdentityById(env.Driver, input.Id)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    if exists == true {

      valid, _ := idp.ValidatePassword(identity.Password, input.Password)
      if valid == true {
        // Nothing to change was the new password is same as current password
        c.JSON(http.StatusOK, IdentitiesPasswordResponse{ marshalIdentityToIdentityResponse(identity) })
        return
      }

      hashedPassword, err := idp.CreatePassword(input.Password)
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }

      updatedIdentity, err := idp.UpdatePassword(env.Driver, idp.Identity{
        Id: identity.Id,
        Password: hashedPassword,
      })
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }

      c.JSON(http.StatusOK, IdentitiesReadResponse{ marshalIdentityToIdentityResponse(updatedIdentity) })
      return
    }

    // Deny by default
    log.WithFields(logrus.Fields{"id": input.Id}).Info("Identity not found")
    c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
  }
  return gin.HandlerFunc(fn)
}