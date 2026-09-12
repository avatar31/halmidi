package apihandlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/avatar31/halmidi/internal/core/iam"
	"github.com/avatar31/halmidi/internal/core/models"
)

func ListPolicyHandler(c *gin.Context) {
	ctx := c.Request.Context()
	service := iam.NewIAMPolicyService(ctx)

	policies, _, err := service.ListIAMPolicies(ctx, iam.ListPoliciesOptions{})
	if err != nil {
		SendErrorResp(c, err, nil)
		return
	}

	items := make([]any, 0, len(policies))
	for _, policy := range policies {
		items = append(items, convertPolicyModelToPolicyResp(0, policy))
	}

	SendSuccessResp(c, http.StatusOK, nil, ListResponse{Items: items})
}

func CreatePolicyHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req iam.CreatePolicyRequest
	err := unmarshalRequestBody(c.Request, &req)
	if err != nil {
		SendErrorResp(c, err, nil)
		return
	}

	service := iam.NewIAMPolicyService(ctx)
	policy, err := service.CreateIAMPolicy(ctx, req)
	if err != nil {
		SendErrorResp(c, err, nil)
		return
	}

	SendSuccessResp(c, http.StatusCreated, nil, convertPolicyModelToPolicyResp(0, policy))
}

func convertPolicyModelToPolicyResp(count int32, policy *models.IAMPolicy) Policy {
	return Policy{
		Arn:              policy.Arn,
		AttachmentCount:  count,
		CreateDate:       policy.CreateDate,
		DefaultVersionId: policy.DefaultVersionId,
		IsAttachable:     policy.IsAttachable,
		Path:             policy.Path,
		PolicyId:         policy.Uuid,
		PolicyName:       policy.Name,
		UpdateDate:       policy.UpdateDate,
	}
}
