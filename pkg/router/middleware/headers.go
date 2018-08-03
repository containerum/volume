package middleware

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/textproto"

	volErrors "git.containerum.net/ch/volume-manager/pkg/errors"
	"github.com/containerum/cherry/adaptors/gonic"
	"github.com/containerum/kube-client/pkg/model"
	headers "github.com/containerum/utils/httputil"
	"github.com/gin-gonic/gin"
	"github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

var (
	ErrInvalidUserRole               = errors.New("invalid user role")
	ErrUnableDecodeUserHeaderData    = errors.New("decode user header data failed")
	ErrUnableUnmarshalUserHeaderData = errors.New("unmarshal user header data failed")
)

type UserHeaderDataMap map[string]model.UserHeaderData

//ParseUserHeaderData decodes headers for substitutions
func ParseUserHeaderData(str string) (UserHeaderDataMap, error) {
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		log.WithError(err).WithField("Value", str).Warn(ErrUnableDecodeUserHeaderData)
		return nil, ErrUnableDecodeUserHeaderData
	}
	var userData []model.UserHeaderData
	err = jsoniter.Unmarshal(data, &userData)
	if err != nil {
		log.WithError(err).WithField("Value", string(data)).Warn(ErrUnableUnmarshalUserHeaderData)
		return nil, ErrUnableUnmarshalUserHeaderData
	}
	result := UserHeaderDataMap{}
	for _, v := range userData {
		result[v.ID] = v
	}
	return result, nil
}

func RequiredUserHeaders() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		log.WithField("Headers", ctx.Request.Header).Debug("Header list")
		notFoundHeaders := requireHeaders(ctx, headers.UserRoleXHeader)
		if len(notFoundHeaders) > 0 {
			gonic.Gonic(volErrors.ErrRequiredHeadersNotProvided().AddDetails(notFoundHeaders...), ctx)
			return
		}
		/* Check User-Role and User-Namespace, X-User-Volume */
		role := GetHeader(ctx, headers.UserRoleXHeader)
		if isUser, err := checkIsUserRole(role); err != nil {
			log.WithField("Value", role).WithError(err).Warn("Check User-Role Error")
			gonic.Gonic(volErrors.ErrRequestValidationFailed().AddDetailF("invalid role %s", role), ctx)
		} else {
			//User-Role: user, check User-Namespace, X-User-Volume
			if isUser {
				notFoundHeaders := requireHeaders(ctx,
					headers.UserRoleXHeader,
					headers.UserNamespacesXHeader,
					headers.UserVolumesXHeader,
					headers.UserIDXHeader,
				)
				if len(notFoundHeaders) > 0 {
					gonic.Gonic(volErrors.ErrRequiredHeadersNotProvided().AddDetails(notFoundHeaders...), ctx)
					return
				}
				userNs, errNs := checkUserNamespace(GetHeader(ctx, headers.UserNamespacesXHeader))
				if errNs != nil {
					log.WithField("Value", GetHeader(ctx, headers.UserNamespacesXHeader)).WithError(errNs).Warn("Check User-Namespace header Error")
					gonic.Gonic(volErrors.ErrRequestValidationFailed().AddDetails(fmt.Sprintf("%v: %v", headers.UserNamespacesXHeader, errNs)), ctx)
					return
				}
				ctx.Set(UserNamespaces, userNs)
			}
		}
		ctx.Set(UserRole, GetHeader(ctx, headers.UserRoleXHeader))
	}
}

func checkIsUserRole(userRole string) (bool, error) {
	switch userRole {
	case "", RoleAdmin:
		return false, nil
	case RoleUser:
		return true, nil
	}
	return false, ErrInvalidUserRole
}

func checkUserNamespace(userNamespace string) (UserHeaderDataMap, error) {
	return ParseUserHeaderData(userNamespace)
}

func requireHeaders(ctx *gin.Context, headers ...string) (notFoundHeaders []string) {
	for _, v := range headers {
		if GetHeader(ctx, v) == "" {
			notFoundHeaders = append(notFoundHeaders, v)
		}
	}
	return
}

func GetHeader(ctx *gin.Context, header string) string {
	return ctx.GetHeader(textproto.CanonicalMIMEHeaderKey(header))
}
