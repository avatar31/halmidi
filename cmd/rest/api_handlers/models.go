package apihandlers

import "github.com/avatar31/halmidi/internal/core/s3common"

type User struct {
	Path             string `json:"path"`
	UserName         string `json:"userName"`
	UserId           string `json:"userId"`
	Arn              string `json:"arn"`
	CreateDate       string `json:"createDate"`
	PasswordLastUsed string `json:"passwordLastUsed,omitempty"`
}

type Policy struct {
	PolicyName                    string `json:"policyName"`
	DefaultVersionId              string `json:"defaultVersionId"`
	PolicyId                      string `json:"policyId"`
	Path                          string `json:"path"`
	Arn                           string `json:"arn"`
	AttachmentCount               int32  `json:"attachmentCount"`
	CreateDate                    string `json:"createDate"`
	UpdateDate                    string `json:"updateDate"`
	IsAttachable                  bool   `json:"isAttachable"`
	PermissionsBoundaryUsageCount int32  `json:"permissionsBoundaryUsageCount"`
}

type ListResponse struct {
	Items []any `json:"items"`
}

type AttachPolicyRequest struct {
	PolicyArn string `json:"policyArn"`
}

type ErrorResp struct {
	Message   string               `json:"message"`
	Code      s3common.S3ErrorCode `json:"code"`
	RequestID string               `json:"requestId"`
}
