package apihandlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/avatar31/halmidi/internal/core/iam"
	"github.com/avatar31/halmidi/internal/core/s3common"
)

func CreateUserHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req iam.CreateUserRequest
	err := unmarshalRequestBody(c.Request, &req)
	if err != nil {
		SendErrorResp(c, err, nil)
		return
	}

	service := iam.NewUserService(ctx)
	user, err := service.CreateUser(ctx, req)
	if err != nil {
		SendErrorResp(c, err, nil)
		return
	}

	SendSuccessResp(c, http.StatusCreated, nil, User{
		Path:             user.Path,
		UserName:         user.Name,
		UserId:           user.Uuid,
		Arn:              user.Arn,
		CreateDate:       user.CreateDate,
		PasswordLastUsed: user.PasswordLastUsed,
	})
}

func CreateAccessKeyHandler(c *gin.Context) {
	ctx := c.Request.Context()
	userName := c.Param("userName")

	service, err := iam.NewUserAccessKeyService(ctx, userName)
	if err != nil {
		SendErrorResp(c, err, nil)
		return
	}

	accessKey, err := service.CreateAccessKey(ctx)
	if err != nil {
		SendErrorResp(c, err, nil)
		return
	}

	SendSuccessResp(c, http.StatusCreated, nil, accessKey)
}

func AttachPolicyHandler(c *gin.Context) {
	ctx := c.Request.Context()
	userName := c.Param("userName")

	var req AttachPolicyRequest
	err := unmarshalRequestBody(c.Request, &req)
	if err != nil {
		SendErrorResp(c, err, nil)
		return
	}

	entity, err := iam.NewUserService(ctx).GetUserEntity(ctx, userName)
	if err != nil {
		SendErrorResp(c, err, nil)
		return
	}

	if err := entity.AttachPolicyToUser(ctx, req.PolicyArn); err != nil {
		SendErrorResp(c, err, nil)
		return
	}

	SendSuccessResp(c, http.StatusOK, nil, nil)
}

func unmarshalRequestBody(r *http.Request, body any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(body); err != nil {
		return s3common.GetInvalidRequestS3Error("", err.Error())
	}

	if decoder.More() {
		return s3common.GetInvalidRequestS3Error("", "extra unexpected fields")
	}

	return nil
}
