package iam

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/avatar31/halmidi/internal/core/s3common"
	testhelpers "github.com/avatar31/halmidi/test/unit/helpers"
	"github.com/avatar31/halmidi/utils"
)

func allResShouldBeNil(t *testing.T, authz *authorizedActions) {
	t.Helper()
	assert.Nil(t, authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow])
	assert.Nil(t, authz.AllResourcesStmt[s3common.PolicyStatementEffectDeny])
}

func noResShouldBeNil(t *testing.T, authz *authorizedActions) {
	t.Helper()
	assert.Nil(t, authz.NoResourceStmt[s3common.PolicyStatementEffectAllow])
	assert.Nil(t, authz.NoResourceStmt[s3common.PolicyStatementEffectDeny])
}

func bothAllAndNoResShouldBeNil(t *testing.T, authz *authorizedActions) {
	t.Helper()
	allResShouldBeNil(t, authz)
	noResShouldBeNil(t, authz)
}

// 1. Basic Authorization Tests
// Test Group 1.1: Simple Allow/Deny
func TestSimpleAllowDenyExactMatch(t *testing.T) {
	simpleAllowTests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		resource      string
		validateAuthz func(t *testing.T, authz *authorizedActions)
		expectedAllow bool
		description   string
	}{
		{
			name:     "exact_action_and_resource_match",
			policies: []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/my-file.txt"}}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/my-file.txt",
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res := "arn:aws:s3:::my-bucket/my-file.txt"
				bothAllAndNoResShouldBeNil(t, authz)
				assert.Nil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny])
				assert.NotNil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow])
				assert.Len(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow].Actions, 1)
				assert.Len(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow].Conditions, 0)
			},
			expectedAllow: true,
			description:   "Should allow when action and resource match exactly",
		},
		{
			name:          "action_mismatch",
			policies:      []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/my-file.txt"}}}}},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/my-file.txt",
			expectedAllow: false,
			description:   "Should deny when action doesn't match",
		},
		{
			name:          "resource_mismatch",
			policies:      []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/my-file.txt"}}}}},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/other-file.txt",
			expectedAllow: false,
			description:   "Should deny when resource doesn't match",
		},
		{
			name:     "multiple_actions_one_matches",
			policies: []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/my-file.txt"}}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res := "arn:aws:s3:::my-bucket/my-file.txt"
				bothAllAndNoResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 1)
				assert.Nil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny])
				assert.NotNil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow])
				assert.Len(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow].Actions, 3)
				assert.Len(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow].Conditions, 0)
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/my-file.txt",
			expectedAllow: true,
			description:   "Should allow when one of multiple actions matches",
		},
		{
			name:          "multiple_actions_none_matches",
			policies:      []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/my-file.txt"}}}}},
			action:        s3common.PolicyActionS3CreateBucket,
			expectedAllow: false,
			description:   "Should deny when none of actions matches",
		},
		{
			name:     "multiple_resources_one_matches",
			policies: []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/file1.txt", "arn:aws:s3:::my-bucket/file2.txt", "arn:aws:s3:::my-bucket/file3.txt"}}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res1, res2, res3 := "arn:aws:s3:::my-bucket/file1.txt", "arn:aws:s3:::my-bucket/file2.txt", "arn:aws:s3:::my-bucket/file3.txt"
				bothAllAndNoResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 3)

				for _, res := range []string{res1, res2, res3} {
					assert.Nil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny])
					assert.NotNil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow])
					assert.Len(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow].Actions, 1)
					assert.Len(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow].Conditions, 0)
				}
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file2.txt",
			expectedAllow: true,
			description:   "Should allow when one of multiple resources matches",
		},
		{
			name:     "multiple_resources_multiple_actions_one_action_one_resource_matches",
			policies: []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/file1.txt", "arn:aws:s3:::my-bucket/file2.txt", "arn:aws:s3:::my-bucket/file3.txt"}}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res1, res2, res3 := "arn:aws:s3:::my-bucket/file1.txt", "arn:aws:s3:::my-bucket/file2.txt", "arn:aws:s3:::my-bucket/file3.txt"
				bothAllAndNoResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 3)

				for _, res := range []string{res1, res2, res3} {
					assert.Nil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny])
					assert.NotNil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow])
					assert.Len(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow].Actions, 3)
					assert.Len(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow].Conditions, 0)
				}
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file2.txt",
			expectedAllow: true,
			description:   "Should allow when one of multiple resources matches",
		},
		{
			name:          "multiple_resources_multiple_actions_one_action_mathces_none_resource_matches",
			policies:      []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/file1.txt", "arn:aws:s3:::my-bucket/file2.txt", "arn:aws:s3:::my-bucket/file3.txt"}}}}},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file20.txt",
			expectedAllow: false,
			description:   "Should deny when none of multiple resources matches",
		},
		{
			name:          "multiple_resources_multiple_actions_none_action_mathces_one_resource_matches",
			policies:      []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/file1.txt", "arn:aws:s3:::my-bucket/file2.txt", "arn:aws:s3:::my-bucket/file3.txt"}}}}},
			action:        s3common.PolicyActionS3CreateBucket,
			resource:      "arn:aws:s3:::my-bucket/file2.txt",
			expectedAllow: false,
			description:   "Should deny when none of multiple actions matches",
		},
		{
			name:          "multiple_resources_multiple_actions_none_action_none_resource_matches",
			policies:      []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/file1.txt", "arn:aws:s3:::my-bucket/file2.txt", "arn:aws:s3:::my-bucket/file3.txt"}}}}},
			action:        s3common.PolicyActionS3CreateBucket,
			resource:      "arn:aws:s3:::my-bucket/file20.txt",
			expectedAllow: false,
			description:   "Should deny when none of multiple actions matches",
		},
		{
			name:          "multiple_resources_none_matches",
			policies:      []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/file1.txt", "arn:aws:s3:::my-bucket/file2.txt", "arn:aws:s3:::my-bucket/file3.txt"}}}}},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file20.txt",
			expectedAllow: false,
			description:   "Should deny when none of multiple resources matches",
		},
		{
			name:     "matches_list_action",
			policies: []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"iam:ListUsers"}}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 0)
				assert.Len(t, authz.NoResourceStmt[s3common.PolicyStatementEffectAllow].Actions, 1)
				assert.Len(t, authz.NoResourceStmt[s3common.PolicyStatementEffectAllow].Conditions, 0)
			},
			action:        s3common.PolicyActionIAMListUsers,
			expectedAllow: true,
			description:   "Should allow when action is list",
		},
		{
			name:          "not_matches_list_action",
			policies:      []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"iam:ListUsers"}}}}},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/my-file.txt",
			expectedAllow: false,
			description:   "Should deny when action is not list",
		},
		{
			name:     "no_policies_at_all",
			policies: []PolicyDocument{},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				bothAllAndNoResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 0)
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/my-file.txt",
			expectedAllow: false,
			description:   "No policies at all → implicit deny",
		},
		{
			name:     "empty_policy_document",
			policies: []PolicyDocument{{Version: "2012-10-17", Statement: []*Statement{}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				bothAllAndNoResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 0)
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/my-file.txt",
			expectedAllow: false,
			description:   "Policy document with no statements → implicit deny",
		},
		{
			name:          "no_policies_list_bucket",
			policies:      []PolicyDocument{{Version: "2012-10-17", Statement: []*Statement{}}},
			action:        s3common.PolicyActionS3ListBucket,
			resource:      "arn:aws:s3:::my-bucket",
			expectedAllow: false,
			description:   "No policies → implicit deny for ListBucket",
		},
	}

	for _, tt := range simpleAllowTests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}

			// Calculate authorized actions from the policy
			policies := tt.policies
			authz := engine.calculateAuthorizedActions(policies)

			if tt.validateAuthz != nil {
				tt.validateAuthz(t, &authz)
			}

			// Create a mock HTTP request
			req := testhelpers.NewMockRequest("GET", "/my-bucket/my-file.txt", nil, nil, nil, nil)

			// Call checkAccess
			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			// Verify the result
			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode, "Error should be AccessDenied")
			}
		})
	}

	simpleDenyTests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		resource      string
		validateAuthz func(t *testing.T, authz *authorizedActions)
		expectedAllow bool
		description   string
	}{
		{
			name:     "deny_only_exact_action_and_resource",
			policies: []PolicyDocument{{Statement: []*Statement{{Effect: "Deny", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/my-file.txt"}}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res := "arn:aws:s3:::my-bucket/my-file.txt"
				bothAllAndNoResShouldBeNil(t, authz)
				assert.Nil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow])
				assert.NotNil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny])
				assert.Len(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny].Actions, 1)
				assert.Len(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny].Conditions, 0)
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/my-file.txt",
			expectedAllow: false,
			description:   "Should deny when only a Deny statement matches exactly",
		},
		{
			name:          "deny_only_action_mismatch",
			policies:      []PolicyDocument{{Statement: []*Statement{{Effect: "Deny", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/my-file.txt"}}}}},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/my-file.txt",
			expectedAllow: false,
			description:   "Should deny (implicitly) when Deny action doesn't match and there is no Allow",
		},
		{
			name:          "deny_only_resource_mismatch",
			policies:      []PolicyDocument{{Statement: []*Statement{{Effect: "Deny", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/my-file.txt"}}}}},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/other-file.txt",
			expectedAllow: false,
			description:   "Should deny (implicitly) when Deny resource doesn't match and there is no Allow",
		},
		{
			name:     "deny_multiple_actions_one_matches",
			policies: []PolicyDocument{{Statement: []*Statement{{Effect: "Deny", Action: []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/my-file.txt"}}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res := "arn:aws:s3:::my-bucket/my-file.txt"
				bothAllAndNoResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 1)
				assert.Nil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow])
				assert.NotNil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny])
				assert.Len(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny].Actions, 3)
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/my-file.txt",
			expectedAllow: false,
			description:   "Should deny when one of multiple Deny actions matches",
		},
		{
			name:          "deny_multiple_actions_none_matches",
			policies:      []PolicyDocument{{Statement: []*Statement{{Effect: "Deny", Action: []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/my-file.txt"}}}}},
			action:        s3common.PolicyActionS3CreateBucket,
			resource:      "arn:aws:s3:::my-bucket/my-file.txt",
			expectedAllow: false,
			description:   "Should deny (implicitly) when none of the Deny actions matches and there is no Allow",
		},
		{
			name:     "deny_multiple_resources_one_matches",
			policies: []PolicyDocument{{Statement: []*Statement{{Effect: "Deny", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/file1.txt", "arn:aws:s3:::my-bucket/file2.txt", "arn:aws:s3:::my-bucket/file3.txt"}}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				bothAllAndNoResShouldBeNil(t, authz)

				res1, res2, res3 := "arn:aws:s3:::my-bucket/file1.txt", "arn:aws:s3:::my-bucket/file2.txt", "arn:aws:s3:::my-bucket/file3.txt"
				assert.Len(t, authz.WithSpecificResourceStmt, 3)
				for _, res := range []string{res1, res2, res3} {
					assert.Nil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow])
					assert.NotNil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny])
					assert.Len(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny].Actions, 1)
				}
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file2.txt",
			expectedAllow: false,
			description:   "Should deny when one of multiple Deny resources matches",
		},
		{
			name:          "deny_multiple_resources_none_matches",
			policies:      []PolicyDocument{{Statement: []*Statement{{Effect: "Deny", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/file1.txt", "arn:aws:s3:::my-bucket/file2.txt", "arn:aws:s3:::my-bucket/file3.txt"}}}}},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file20.txt",
			expectedAllow: false,
			description:   "Should deny (implicitly) when none of the Deny resources matches and there is no Allow",
		},
		{
			name: "deny_overrides_allow_same_action_same_resource",
			policies: []PolicyDocument{
				{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/my-file.txt"}}}},
				{Statement: []*Statement{{Effect: "Deny", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/my-file.txt"}}}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				bothAllAndNoResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 1)

				res := "arn:aws:s3:::my-bucket/my-file.txt"
				assert.NotNil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow])
				assert.Len(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow].Actions, 1)
				assert.NotNil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny])
				assert.Len(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny].Actions, 1)
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/my-file.txt",
			expectedAllow: false,
			description:   "Explicit Deny must override explicit Allow on same action+resource",
		},
		{
			name: "deny_one_action_allow_different_action_same_resource",
			policies: []PolicyDocument{
				{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/my-file.txt"}}}},
				{Statement: []*Statement{{Effect: "Deny", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/my-file.txt"}}}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				bothAllAndNoResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 1)

				res := "arn:aws:s3:::my-bucket/my-file.txt"
				assert.NotNil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow])
				assert.Len(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow].Actions, 3)
				assert.NotNil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny])
				assert.Len(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny].Actions, 1)
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/my-file.txt",
			expectedAllow: false,
			description:   "Deny on one action overrides Allow that covers the same action",
		},
		{
			name: "deny_one_action_allow_covers_different_actions",
			policies: []PolicyDocument{
				{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/my-file.txt"}}}},
				{Statement: []*Statement{{Effect: "Deny", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/my-file.txt"}}}},
			},
			action:        s3common.PolicyActionS3PutObject, // PutObject is only allowed, not denied
			resource:      "arn:aws:s3:::my-bucket/my-file.txt",
			expectedAllow: true,
			description:   "Allow on a different action should still work even when another action is denied",
		},
		{
			name: "deny_one_resource_allow_different_resource",
			policies: []PolicyDocument{
				{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/file1.txt"}}}},
				{Statement: []*Statement{{Effect: "Deny", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/file2.txt"}}}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file1.txt",
			expectedAllow: true,
			description:   "Deny on a different resource should not affect Allow on the requested resource",
		},
		{
			name: "deny_overrides_allow_across_multiple_resources",
			policies: []PolicyDocument{
				{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/file1.txt", "arn:aws:s3:::my-bucket/file2.txt", "arn:aws:s3:::my-bucket/file3.txt"}}}},
				{Statement: []*Statement{{Effect: "Deny", Action: []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/file1.txt", "arn:aws:s3:::my-bucket/file2.txt", "arn:aws:s3:::my-bucket/file3.txt"}}}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				for _, res := range []string{"arn:aws:s3:::my-bucket/file1.txt", "arn:aws:s3:::my-bucket/file2.txt", "arn:aws:s3:::my-bucket/file3.txt"} {
					assert.NotNil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow])
					assert.NotNil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny])
				}
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/file2.txt",
			expectedAllow: false,
			description:   "Explicit Deny must override explicit Allow across multiple resources",
		},
		{
			name:     "deny_only_list_action",
			policies: []PolicyDocument{{Statement: []*Statement{{Effect: "Deny", Action: []string{"iam:ListUsers"}}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				assert.Len(t, authz.WithSpecificResourceStmt, 0)
				allResShouldBeNil(t, authz)
				assert.Len(t, authz.NoResourceStmt[s3common.PolicyStatementEffectDeny].Actions, 1)
				assert.Len(t, authz.NoResourceStmt[s3common.PolicyStatementEffectDeny].Conditions, 0)
			},
			action:        s3common.PolicyActionIAMListUsers,
			expectedAllow: false,
			description:   "Should deny a no-resource (list) action when explicitly denied",
		},
		{
			name: "deny_list_action_overrides_allow_list_action",
			policies: []PolicyDocument{
				{Statement: []*Statement{{Effect: "Allow", Action: []string{"iam:ListUsers"}}}},
				{Statement: []*Statement{{Effect: "Deny", Action: []string{"iam:ListUsers"}}}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				assert.NotNil(t, authz.NoResourceStmt[s3common.PolicyStatementEffectAllow])
				assert.NotNil(t, authz.NoResourceStmt[s3common.PolicyStatementEffectDeny])
			},
			action:        s3common.PolicyActionIAMListUsers,
			expectedAllow: false,
			description:   "Explicit Deny on no-resource action must override explicit Allow",
		},
		{
			name: "deny_list_action_does_not_affect_other_actions",
			policies: []PolicyDocument{
				{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/my-file.txt"}}}},
				{Statement: []*Statement{{Effect: "Deny", Action: []string{"iam:ListUsers"}}}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/my-file.txt",
			expectedAllow: true,
			description:   "Deny on a no-resource action should not affect unrelated allowed actions",
		},
	}

	for _, tt := range simpleDenyTests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}

			policies := tt.policies
			authz := engine.calculateAuthorizedActions(policies)

			if tt.validateAuthz != nil {
				tt.validateAuthz(t, &authz)
			}

			req := testhelpers.NewMockRequest("GET", "/my-bucket/my-file.txt", nil, nil, nil, nil)

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode, "Error should be AccessDenied")
			}
		})
	}
}

// 2. AWS IAM Policy Precedence Tests
// Test Group 2.1: Explicit Deny > Explicit Allow
func TestPolicyPrecedence(t *testing.T) {
	tests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		resource      string
		validateAuthz func(t *testing.T, authz *authorizedActions)
		expectedAllow bool
		description   string
	}{
		// --- Explicit Deny overrides Explicit Allow ---
		{
			name: "explicit_deny_overrides_allow_global_resource",
			policies: []PolicyDocument{
				{Statement: []*Statement{{Effect: "Allow", Action: "*", Resource: "*"}}},
				{Statement: []*Statement{{Effect: "Deny", Action: []string{"s3:DeleteObject"}, Resource: "*"}}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 0)
				assert.NotNil(t, authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow])
				assert.NotNil(t, authz.AllResourcesStmt[s3common.PolicyStatementEffectDeny])
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::my-bucket/my-file.txt",
			expectedAllow: false,
			description:   "Explicit Deny on all-resources (*) must override Allow all on all-resources (*)",
		},
		{
			name: "explicit_deny_specific_resource_overrides_allow_global",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: "*", Resource: "*"},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"s3:PutObject"}, Resource: []string{"arn:aws:s3:::prodbucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				assert.NotNil(t, authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow])
				assert.Nil(t, authz.AllResourcesStmt[s3common.PolicyStatementEffectDeny])
				assert.Len(t, authz.WithSpecificResourceStmt, 1)
				denyStmt := authz.WithSpecificResourceStmt["arn:aws:s3:::prodbucket/*"][s3common.PolicyStatementEffectDeny]
				assert.NotNil(t, denyStmt)
				assert.Len(t, denyStmt.Actions, 1)
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::prodbucket/secret.txt",
			expectedAllow: false,
			description:   "Explicit Deny on specific resource must override Allow all on all-resources (*)",
		},
		{
			name: "explicit_deny_specific_resource_does_not_affect_other_resources",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: "*", Resource: "*"},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"s3:PutObject"}, Resource: []string{"arn:aws:s3:::prodbucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::devbucket/file.txt",
			expectedAllow: true,
			description:   "Explicit Deny on specific resource should not affect other resources allowed by global Allow",
		},
		{
			name: "allow_different_action_not_affected_by_deny",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: "*", Resource: "*"},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"s3:DeleteObject"}, Resource: "*"},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/my-file.txt",
			expectedAllow: true,
			description:   "Allow on a different action should not be affected by Deny on another action",
		},

		// --- Implicit Deny (no matching Allow) ---
		{
			name: "implicit_deny_no_matching_allow",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allResShouldBeNil(t, authz)
				noResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 1)
				allowStmt := authz.WithSpecificResourceStmt["arn:aws:s3:::my-bucket/*"][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Len(t, allowStmt.Actions, 1)
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::my-bucket/my-file.txt",
			expectedAllow: false,
			description:   "Should implicit deny when no Allow statement matches the requested action",
		},
		{
			name: "implicit_deny_no_matching_resource",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::other-bucket/my-file.txt",
			expectedAllow: false,
			description:   "Should implicit deny when no Allow statement matches the requested resource",
		},

		// --- Multiple policies, deny in one overrides allow in another ---
		{
			name: "deny_in_second_policy_overrides_allow_in_first",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject", "s3:PutObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"s3:PutObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allResShouldBeNil(t, authz)
				noResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				assert.Len(t, authz.WithSpecificResourceStmt, 1)
				assert.NotNil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow])
				assert.Len(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow].Actions, 2)
				assert.NotNil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny])
				assert.Len(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny].Actions, 1)
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "Deny in second policy must override Allow in first policy",
		},
		{
			name: "deny_in_first_policy_overrides_allow_in_second",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"s3:PutObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject", "s3:PutObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "Deny in first policy must override Allow in second policy (order should not matter)",
		},

		// --- Deny in same statement as part of multiple statements ---
		{
			name: "deny_in_same_policy_document_multiple_statements",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
					{Effect: "Deny", Action: []string{"s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allResShouldBeNil(t, authz)
				noResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				assert.Len(t, authz.WithSpecificResourceStmt, 1)
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Len(t, allowStmt.Actions, 3)
				denyStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny]
				assert.NotNil(t, denyStmt)
				assert.Len(t, denyStmt.Actions, 1)
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "Deny statement in same policy document must override Allow statement for same action",
		},
		{
			name: "allow_not_denied_action_in_same_policy_document",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
					{Effect: "Deny", Action: []string{"s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "Non-denied action in same policy document should still be allowed",
		},

		// --- Deny all via wildcard, then allow specific ---
		{
			name: "deny_all_then_allow_specific_action_should_still_deny",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: "*", Resource: "*"},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				assert.NotNil(t, authz.AllResourcesStmt[s3common.PolicyStatementEffectDeny])
				assert.Nil(t, authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow])
				res := "arn:aws:s3:::my-bucket/*"
				assert.Len(t, authz.WithSpecificResourceStmt, 1)
				assert.NotNil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow])
				assert.Nil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny])
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "Deny all (*) on all-resources (*) must override any specific Allow",
		},

		// --- Allow all via wildcard, deny nothing: should allow ---
		{
			name: "allow_all_no_deny_should_allow",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: "*", Resource: "*"},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 0)
				assert.NotNil(t, authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow])
				assert.Nil(t, authz.AllResourcesStmt[s3common.PolicyStatementEffectDeny])
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::any-bucket/any-file.txt",
			expectedAllow: true,
			description:   "Allow all (*) on all-resources (*) with no Deny should allow everything",
		},

		// --- IAM actions precedence ---
		{
			name: "iam_explicit_deny_overrides_allow",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"iam:*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"iam:DeleteUser"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 0)
				assert.NotNil(t, authz.NoResourceStmt[s3common.PolicyStatementEffectAllow])
				assert.NotNil(t, authz.NoResourceStmt[s3common.PolicyStatementEffectDeny])
				denyActions := authz.NoResourceStmt[s3common.PolicyStatementEffectDeny].Actions
				_, hasDenyDeleteUser := denyActions[normalizeAction(s3common.PolicyActionIAMDeleteUser)]
				assert.True(t, hasDenyDeleteUser)
			},
			action:        s3common.PolicyActionIAMDeleteUser,
			resource:      "",
			expectedAllow: false,
			description:   "Explicit Deny on IAM action must override Allow all IAM actions",
		},
		{
			name: "iam_allow_not_denied_action",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"iam:*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"iam:DeleteUser"}},
				}},
			},
			action:        s3common.PolicyActionIAMListUsers,
			resource:      "",
			expectedAllow: true,
			description:   "Non-denied IAM action should remain allowed when only another IAM action is denied",
		},

		// --- Cross-service precedence ---
		{
			name: "s3_deny_does_not_affect_iam_allow",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"iam:ListUsers"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				assert.NotNil(t, authz.NoResourceStmt[s3common.PolicyStatementEffectAllow])
				res := "arn:aws:s3:::my-bucket/*"
				denyStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny]
				assert.NotNil(t, denyStmt)
				_, hasDeny := denyStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObject)]
				assert.True(t, hasDeny)
			},
			action:        s3common.PolicyActionIAMListUsers,
			resource:      "",
			expectedAllow: true,
			description:   "Deny on S3 action should not affect Allow on IAM action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}

			authz := engine.calculateAuthorizedActions(tt.policies)
			if tt.validateAuthz != nil {
				tt.validateAuthz(t, &authz)
			}

			req := testhelpers.NewMockRequest("GET", "/my-bucket/my-file.txt", nil, nil, nil, nil)

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode, "Error should be AccessDenied")
			}
		})
	}
}

// 3. Wildcard Pattern Tests
// Test Group 3.1: Action Wildcards
func TestActionWildcards(t *testing.T) {
	tests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		resource      string
		validateAuthz func(t *testing.T, authz *authorizedActions)
		expectedAllow bool
		description   string
	}{
		// --- s3:* on specific resource ---
		{
			name: "s3_wildcard_all_actions_on_specific_resource_allows_get",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:*"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allResShouldBeNil(t, authz)
				noResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				assert.Len(t, authz.WithSpecificResourceStmt, 1)
				assert.Nil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny])
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Equal(t, len(s3common.AllS3SupportedActions), len(allowStmt.Actions))
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "s3:* on specific resource should allow any S3 action on matching resource",
		},
		{
			name: "s3_wildcard_all_actions_on_specific_resource_allows_delete",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:*"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "s3:* on specific resource should allow s3:DeleteObject on matching resource",
		},
		{
			name: "s3_wildcard_all_actions_on_specific_resource_denies_iam_action",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:*"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionIAMListUsers,
			resource:      "",
			expectedAllow: false,
			description:   "s3:* should not grant IAM actions (implicit deny)",
		},
		{
			name: "s3_wildcard_all_actions_on_specific_resource_denies_non_matching_resource",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:*"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::other-bucket/file.txt",
			expectedAllow: false,
			description:   "s3:* on specific resource should not allow access to a different resource",
		},

		// --- iam:* on no resource (IAM actions have no resource) ---
		{
			name:     "iam_wildcard_all_actions_no_resource_allows_list_users",
			policies: []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"iam:*"}, Resource: []string{"arn:aws:iam::123456789012:user/*"}}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				// iam:* with a specific resource pattern goes to WithSpecificResourceStmt
				allResShouldBeNil(t, authz)
				noResShouldBeNil(t, authz)
				res := "arn:aws:iam::123456789012:user/*"
				assert.Len(t, authz.WithSpecificResourceStmt, 1)
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Equal(t, len(s3common.AllIAMSupportedActions), len(allowStmt.Actions))
			},
			action:        s3common.PolicyActionIAMListUsers,
			resource:      "arn:aws:iam::123456789012:user/bob",
			expectedAllow: true,
			description:   "iam:* with specific IAM resource should allow any IAM action on matching resource",
		},
		{
			name: "iam_wildcard_no_resource_allows_delete_user",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"iam:*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 0)
				allowStmt := authz.NoResourceStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Equal(t, len(s3common.AllIAMSupportedActions), len(allowStmt.Actions))
			},
			action:        s3common.PolicyActionIAMDeleteUser,
			resource:      "",
			expectedAllow: true,
			description:   "iam:* with no resource should allow any IAM action",
		},
		{
			name: "iam_wildcard_no_resource_does_not_allow_s3_action",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"iam:*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "iam:* should not grant S3 actions (implicit deny)",
		},

		// --- Global wildcard * on * resource ---
		{
			name:     "global_wildcard_action_and_resource_allows_any_s3_action",
			policies: []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: "*", Resource: "*"}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 0)
				allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Equal(t, len(s3common.AllSupportedActions), len(allowStmt.Actions))
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::any-bucket/any-file.txt",
			expectedAllow: true,
			description:   "Action=* Resource=* should allow any S3 action on any resource",
		},
		{
			name:     "global_wildcard_action_and_resource_allows_any_iam_action",
			policies: []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: "*", Resource: "*"}}}},
			action:   s3common.PolicyActionIAMCreateUser,
			resource: "",
			// IAM actions with no resource: global * covers all, falls through to NoResourceStmt check
			// Since * means all actions, IAM actions are also included.
			// checkAllowStmt checks AllResourcesStmt first before NoResourceStmt.
			// AllResourcesStmt[Allow] has all actions including IAM — so IAM action with empty resource
			// should also be allowed via AllResourcesStmt.
			expectedAllow: true,
			description:   "Action=* Resource=* should allow any IAM action",
		},

		// --- s3:Get* prefix wildcard ---
		{
			name:     "s3_get_prefix_wildcard_allows_get_object",
			policies: []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:Get*"}, Resource: "*"}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 0)
				allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				// All s3:Get* actions should be in the allow list
				_, hasGetObject := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGetObject)
				_, hasGetBucketTagging := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetBucketTagging)]
				assert.True(t, hasGetBucketTagging)
				// s3:Put* should NOT be included
				_, hasPutObject := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3PutObject)]
				assert.False(t, hasPutObject)
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "s3:Get* should allow s3:GetObject",
		},
		{
			name:          "s3_get_prefix_wildcard_allows_get_bucket_tagging",
			policies:      []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:Get*"}, Resource: "*"}}}},
			action:        s3common.PolicyActionS3GetBucketTagging,
			resource:      "arn:aws:s3:::my-bucket",
			expectedAllow: true,
			description:   "s3:Get* should allow s3:GetBucketTagging",
		},
		{
			name:          "s3_get_prefix_wildcard_denies_put_object",
			policies:      []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:Get*"}, Resource: "*"}}}},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "s3:Get* should not allow s3:PutObject (implicit deny)",
		},
		{
			name:          "s3_get_prefix_wildcard_denies_delete_object",
			policies:      []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:Get*"}, Resource: "*"}}}},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "s3:Get* should not allow s3:DeleteObject (implicit deny)",
		},
		{
			name:          "s3_get_prefix_wildcard_denies_iam_action",
			policies:      []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: []string{"s3:Get*"}, Resource: "*"}}}},
			action:        s3common.PolicyActionIAMListUsers,
			resource:      "",
			expectedAllow: false,
			description:   "s3:Get* should not allow IAM actions (implicit deny)",
		},

		// --- s3:*Object suffix wildcard ---
		{
			name: "s3_object_suffix_wildcard_allows_get_object",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:*Object"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allResShouldBeNil(t, authz)
				noResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasGetObject := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGetObject)
				_, hasPutObject := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3PutObject)]
				assert.True(t, hasPutObject)
				_, hasDeleteObject := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObject)]
				assert.True(t, hasDeleteObject)
				// s3:ListBucket does NOT end with "Object"
				_, hasListBucket := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3ListBucket)]
				assert.False(t, hasListBucket)
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "s3:*Object should allow s3:GetObject",
		},
		{
			name: "s3_object_suffix_wildcard_allows_put_object",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:*Object"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "s3:*Object should allow s3:PutObject",
		},
		{
			name: "s3_object_suffix_wildcard_denies_list_bucket",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:*Object"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3ListBucket,
			resource:      "arn:aws:s3:::my-bucket",
			expectedAllow: false,
			description:   "s3:*Object should not allow s3:ListBucket (implicit deny)",
		},

		// --- Single char ? wildcard ---
		{
			name: "s3_single_char_wildcard_allows_matching_action",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					// s3:GetObject matches s3:G?tObject (? = 'e')
					{Effect: "Allow", Action: []string{"s3:G?tObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "s3:G?tObject with ? wildcard should match s3:GetObject",
		},
		{
			name: "s3_single_char_wildcard_denies_non_matching_action",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:G?tObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "s3:G?tObject should not match s3:PutObject (implicit deny)",
		},

		// --- Wildcard Deny action overrides wildcard Allow ---
		{
			name: "wildcard_deny_s3_delete_overrides_wildcard_allow_all",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:*"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"s3:Delete*"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allResShouldBeNil(t, authz)
				noResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				denyStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny]
				assert.NotNil(t, denyStmt)
				// s3:Delete* actions should be in deny
				_, hasDeleteObject := denyStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObject)]
				assert.True(t, hasDeleteObject)
				_, hasDeleteObjectVersion := denyStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObjectVersion)]
				assert.True(t, hasDeleteObjectVersion)
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "s3:Delete* Deny must override s3:* Allow",
		},
		{
			name: "wildcard_deny_s3_delete_does_not_affect_get",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:*"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"s3:Delete*"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "s3:Delete* Deny should not affect s3:GetObject which is still allowed by s3:*",
		},

		// --- Action wildcard is case-insensitive ---
		{
			name: "action_wildcard_case_insensitive_uppercase",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"S3:GETOBJECT"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "Action matching should be case-insensitive (uppercase action in policy)",
		},
		{
			name: "action_wildcard_case_insensitive_mixed",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GET*"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "Action wildcard s3:Get* should case-insensitively match s3:GetObject",
		},

		// --- Multiple wildcard action policies ---
		{
			name: "multiple_wildcard_actions_combined",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:Get*", "s3:Put*"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allResShouldBeNil(t, authz)
				noResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasGetObject := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGetObject)
				_, hasPutObject := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3PutObject)]
				assert.True(t, hasPutObject)
				_, hasDeleteObject := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObject)]
				assert.False(t, hasDeleteObject)
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "Multiple wildcard actions [s3:Get*, s3:Put*] combined should allow both Get and Put",
		},
		{
			name: "multiple_wildcard_actions_combined_denies_delete",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:Get*", "s3:Put*"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "Multiple wildcard actions [s3:Get*, s3:Put*] should not allow s3:DeleteObject (implicit deny)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}

			authz := engine.calculateAuthorizedActions(tt.policies)
			if tt.validateAuthz != nil {
				tt.validateAuthz(t, &authz)
			}

			req := testhelpers.NewMockRequest("GET", "/my-bucket/my-file.txt", nil, nil, nil, nil)

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode, "Error should be AccessDenied")
			}
		})
	}
}

// Test Group 3.2: Resource Wildcards
func TestResourceWildcards(t *testing.T) {
	tests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		resource      string
		validateAuthz func(t *testing.T, authz *authorizedActions)
		expectedAllow bool
		description   string
	}{
		// --- Global resource wildcard * ---
		{
			name: "global_resource_wildcard_allows_any_s3_resource",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: "*"},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 0)
				allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasGetObject := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGetObject)
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::any-bucket/any-file.txt",
			expectedAllow: true,
			description:   "Resource=* should allow the action on any S3 resource",
		},
		{
			name: "global_resource_wildcard_allows_different_bucket",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: "*"},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::completely-different-bucket/path/to/object.txt",
			expectedAllow: true,
			description:   "Resource=* should allow the action on any S3 bucket and key",
		},

		// --- Bucket-level wildcard (all objects in a bucket) ---
		{
			name: "bucket_prefix_wildcard_allows_matching_object",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allResShouldBeNil(t, authz)
				noResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				assert.Len(t, authz.WithSpecificResourceStmt, 1)
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Len(t, allowStmt.Actions, 1)
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/some/nested/path/file.txt",
			expectedAllow: true,
			description:   "arn:aws:s3:::my-bucket/* should allow access to any object in my-bucket",
		},
		{
			name: "bucket_prefix_wildcard_allows_root_object",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "arn:aws:s3:::my-bucket/* should allow access to a root-level object in my-bucket",
		},
		{
			name: "bucket_prefix_wildcard_denies_different_bucket",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::other-bucket/file.txt",
			expectedAllow: false,
			description:   "arn:aws:s3:::my-bucket/* should not allow access to a different bucket (implicit deny)",
		},

		// --- Path prefix wildcard (prefix within a bucket) ---
		{
			name: "path_prefix_wildcard_allows_matching_object",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/home/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allResShouldBeNil(t, authz)
				noResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/home/*"
				assert.Len(t, authz.WithSpecificResourceStmt, 1)
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/home/alice/document.pdf",
			expectedAllow: true,
			description:   "arn:aws:s3:::my-bucket/home/* should allow access to objects under home/ prefix",
		},
		{
			name: "path_prefix_wildcard_allows_deep_nested_path",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/home/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/home/alice/2024/january/report.csv",
			expectedAllow: true,
			description:   "arn:aws:s3:::my-bucket/home/* should allow access to deeply nested paths",
		},
		{
			name: "path_prefix_wildcard_denies_different_prefix",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/home/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/shared/file.txt",
			expectedAllow: false,
			description:   "arn:aws:s3:::my-bucket/home/* should not allow access to objects outside home/ prefix",
		},
		{
			name: "path_prefix_wildcard_denies_bucket_root",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/home/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "arn:aws:s3:::my-bucket/home/* should not allow access to root-level objects",
		},

		// --- Multiple resource wildcards in one statement ---
		{
			name: "multiple_resource_wildcards_allows_first_bucket_match",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{
						"arn:aws:s3:::bucket-a/*",
						"arn:aws:s3:::bucket-b/*",
					}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allResShouldBeNil(t, authz)
				noResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 2)
				for _, res := range []string{"arn:aws:s3:::bucket-a/*", "arn:aws:s3:::bucket-b/*"} {
					allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
					assert.NotNil(t, allowStmt)
					assert.Len(t, allowStmt.Actions, 1)
				}
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::bucket-a/file.txt",
			expectedAllow: true,
			description:   "Multiple resource wildcards: should allow matching the first bucket pattern",
		},
		{
			name: "multiple_resource_wildcards_allows_second_bucket_match",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{
						"arn:aws:s3:::bucket-a/*",
						"arn:aws:s3:::bucket-b/*",
					}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::bucket-b/file.txt",
			expectedAllow: true,
			description:   "Multiple resource wildcards: should allow matching the second bucket pattern",
		},
		{
			name: "multiple_resource_wildcards_denies_non_matching_bucket",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{
						"arn:aws:s3:::bucket-a/*",
						"arn:aws:s3:::bucket-b/*",
					}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::bucket-c/file.txt",
			expectedAllow: false,
			description:   "Multiple resource wildcards: should not allow a bucket not in the resource list (implicit deny)",
		},

		// --- Single char ? wildcard in resource ---
		{
			name: "resource_single_char_wildcard_allows_matching_bucket",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					// Matches my-bucket-a, my-bucket-b, my-bucket-1, etc.
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket-?/file.txt"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket-a/file.txt",
			expectedAllow: true,
			description:   "Resource ? wildcard should match a single character in bucket name",
		},
		{
			name: "resource_single_char_wildcard_deny_matching_bucket_with_nonmatching_file",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					// Matches my-bucket-a, my-bucket-b, my-bucket-1, etc.
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket-?/file.txt"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket-a/file1.txt",
			expectedAllow: false,
			description:   "Resource ? wildcard should not match if the rest of the resource string does not match (implicit deny)",
		},
		{
			name: "resource_single_char_wildcard_denies_multi_char_mismatch",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket-?/file.txt"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket-ab/file.txt",
			expectedAllow: false,
			description:   "Resource ? wildcard should not match more than one character (implicit deny)",
		},

		// --- Deny resource wildcard overrides specific Allow ---
		{
			name: "deny_bucket_wildcard_overrides_specific_allow",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/file.txt"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allResShouldBeNil(t, authz)
				noResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 2)
				allowRes := "arn:aws:s3:::my-bucket/file.txt"
				denyRes := "arn:aws:s3:::my-bucket/*"
				assert.NotNil(t, authz.WithSpecificResourceStmt[allowRes][s3common.PolicyStatementEffectAllow])
				assert.NotNil(t, authz.WithSpecificResourceStmt[denyRes][s3common.PolicyStatementEffectDeny])
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "Deny with bucket wildcard arn:aws:s3:::my-bucket/* must override specific Allow on arn:aws:s3:::my-bucket/file.txt",
		},
		{
			name: "deny_global_wildcard_overrides_specific_bucket_allow",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"s3:GetObject"}, Resource: "*"},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				assert.Nil(t, authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow])
				assert.NotNil(t, authz.AllResourcesStmt[s3common.PolicyStatementEffectDeny])
				res := "arn:aws:s3:::my-bucket/*"
				assert.NotNil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow])
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "Deny on Resource=* must override Allow on specific bucket wildcard",
		},

		// --- Allow wildcard resource does not bleed into unrelated bucket deny ---
		{
			name: "allow_wildcard_not_blocked_by_deny_on_different_bucket",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::dev-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::prod-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::dev-bucket/file.txt",
			expectedAllow: true,
			description:   "Deny on prod-bucket/* should not affect Allow on dev-bucket/*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}

			authz := engine.calculateAuthorizedActions(tt.policies)
			if tt.validateAuthz != nil {
				tt.validateAuthz(t, &authz)
			}

			req := testhelpers.NewMockRequest("GET", "/my-bucket/my-file.txt", nil, nil, nil, nil)

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode, "Error should be AccessDenied")
			}
		})
	}
}

// 4. NotAction and NotResource Tests
// Test Group 4.1: NotAction
func TestNotAction(t *testing.T) {
	tests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		resource      string
		validateAuthz func(t *testing.T, authz *authorizedActions)
		expectedAllow bool
		description   string
	}{
		// --- NotAction with Resource=* allows everything except the listed action ---
		{
			name: "notaction_excludes_delete_allows_get",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", NotAction: []string{"s3:DeleteObject"}, Resource: "*"},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 0)
				allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				// s3:DeleteObject must NOT be in the allow list
				_, hasDelete := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObject)]
				assert.False(t, hasDelete)
				// s3:GetObject must be in the allow list
				_, hasGet := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGet)
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "NotAction:[s3:DeleteObject] with Resource=* should allow s3:GetObject",
		},
		{
			name: "notaction_excludes_delete_denies_delete",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", NotAction: []string{"s3:DeleteObject"}, Resource: "*"},
				}},
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "NotAction:[s3:DeleteObject] with Resource=* should deny s3:DeleteObject (implicit deny)",
		},
		{
			name: "notaction_excludes_delete_allows_put",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", NotAction: []string{"s3:DeleteObject"}, Resource: "*"},
				}},
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "NotAction:[s3:DeleteObject] with Resource=* should allow s3:PutObject",
		},

		// --- NotAction with specific resource ---
		{
			name: "notaction_on_specific_resource_allows_non_excluded_action",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", NotAction: []string{"s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allResShouldBeNil(t, authz)
				noResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				assert.Len(t, authz.WithSpecificResourceStmt, 1)
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasDelete := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObject)]
				assert.False(t, hasDelete)
				_, hasGet := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGet)
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "NotAction:[s3:DeleteObject] on specific resource should allow s3:GetObject on matching resource",
		},
		{
			name: "notaction_on_specific_resource_denies_excluded_action",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", NotAction: []string{"s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "NotAction:[s3:DeleteObject] on specific resource should deny s3:DeleteObject (implicit deny)",
		},
		{
			name: "notaction_on_specific_resource_denies_non_matching_resource",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", NotAction: []string{"s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::other-bucket/file.txt",
			expectedAllow: false,
			description:   "NotAction on specific resource should deny access to a different resource (implicit deny)",
		},
		{
			name: "notaction_on_specific_resource_deny_all_excluding_get",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", NotAction: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allResShouldBeNil(t, authz)
				noResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				assert.Len(t, authz.WithSpecificResourceStmt, 1)
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasDelete := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObject)]
				assert.False(t, hasDelete)
				_, hasGet := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGet)
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "Deny NotAction:[s3:GetObject] with Resource=* shouldn't allow s3:PutObject",
		},

		// --- NotAction with multiple excluded actions ---
		{
			name: "notaction_multiple_excluded_actions_denies_each",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", NotAction: []string{"s3:DeleteObject", "s3:PutObject"}, Resource: "*"},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasDelete := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObject)]
				assert.False(t, hasDelete)
				_, hasPut := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3PutObject)]
				assert.False(t, hasPut)
				_, hasGet := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGet)
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "NotAction:[s3:DeleteObject, s3:PutObject] should deny s3:DeleteObject",
		},
		{
			name: "notaction_multiple_excluded_actions_denies_second_excluded",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", NotAction: []string{"s3:DeleteObject", "s3:PutObject"}, Resource: "*"},
				}},
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "NotAction:[s3:DeleteObject, s3:PutObject] should deny s3:PutObject",
		},
		{
			name: "notaction_multiple_excluded_actions_allows_non_excluded",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", NotAction: []string{"s3:DeleteObject", "s3:PutObject"}, Resource: "*"},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "NotAction:[s3:DeleteObject, s3:PutObject] should allow s3:GetObject",
		},

		// --- NotAction with wildcard exclusion ---
		{
			name: "notaction_wildcard_prefix_excludes_all_delete_actions",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", NotAction: []string{"s3:Delete*"}, Resource: "*"},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasDeleteObject := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObject)]
				assert.False(t, hasDeleteObject)
				_, hasDeleteObjectVersion := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObjectVersion)]
				assert.False(t, hasDeleteObjectVersion)
				_, hasGet := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGet)
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "NotAction:[s3:Delete*] should deny all s3:Delete* actions",
		},
		{
			name: "notaction_wildcard_prefix_allows_non_matching",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", NotAction: []string{"s3:Delete*"}, Resource: "*"},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "NotAction:[s3:Delete*] should allow s3:GetObject",
		},

		// --- Deny NotAction: deny everything except the listed action ---
		{
			name: "deny_notaction_denies_all_except_listed",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					// Allow all first so we have a baseline
					{Effect: "Allow", Action: "*", Resource: "*"},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					// Deny everything EXCEPT s3:GetObject
					{Effect: "Deny", NotAction: []string{"s3:GetObject"}, Resource: "*"},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				denyStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectDeny]
				assert.NotNil(t, denyStmt)
				// s3:GetObject must NOT be in deny (it is excluded via NotAction)
				_, hasGetInDeny := denyStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.False(t, hasGetInDeny)
				// s3:DeleteObject must be in deny
				_, hasDeleteInDeny := denyStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObject)]
				assert.True(t, hasDeleteInDeny)
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "Deny NotAction:[s3:GetObject] should deny everything except s3:GetObject",
		},
		{
			name: "deny_notaction_allows_the_excepted_action",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: "*", Resource: "*"},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", NotAction: []string{"s3:GetObject"}, Resource: "*"},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "Deny NotAction:[s3:GetObject] should NOT deny s3:GetObject (it is the exception)",
		},

		{
			name:     "notaction_delete_wildcard_allows_get",
			policies: []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", NotAction: []string{"s3:Delete*"}, Resource: "*"}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::any-bucket/file.txt",
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasGet := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGet)
				_, hasDeleteObject := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObject)]
				assert.False(t, hasDeleteObject)
			},
			expectedAllow: true,
			description:   "NotAction:[s3:Delete*] Resource=* should allow s3:GetObject",
		},
		{
			name:          "notaction_delete_wildcard_denies_delete",
			policies:      []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", NotAction: []string{"s3:Delete*"}, Resource: "*"}}}},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::any-bucket/file.txt",
			expectedAllow: false,
			description:   "NotAction:[s3:Delete*] Resource=* should deny s3:DeleteObject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}

			authz := engine.calculateAuthorizedActions(tt.policies)
			if tt.validateAuthz != nil {
				tt.validateAuthz(t, &authz)
			}

			req := testhelpers.NewMockRequest("GET", "/my-bucket/my-file.txt", nil, nil, nil, nil)

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode, "Error should be AccessDenied")
			}
		})
	}
}

// Test Group 4.2: NotResource
func TestNotResource(t *testing.T) {
	tests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		resource      string
		validateAuthz func(t *testing.T, authz *authorizedActions)
		expectedAllow bool
		description   string
	}{
		// --- NotResource: allow action on all resources EXCEPT the listed one ---
		{
			name: "notresource_excludes_prod_bucket_allows_dev_bucket",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: "*", NotResource: []string{"arn:aws:s3:::prodbucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				// NotResource is stored as negated effect on the excluded resource
				// plus an AllResources allow for the action
				noResShouldBeNil(t, authz)
				allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasGet := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGet)
				// The excluded resource should have a Deny entry for this action
				denyStmt := authz.WithSpecificResourceStmt["arn:aws:s3:::prodbucket/*"][s3common.PolicyStatementEffectDeny]
				assert.NotNil(t, denyStmt)
				_, hasGetInDeny := denyStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGetInDeny)
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::devbucket/file.txt",
			expectedAllow: true,
			description:   "NotResource:[prodbucket/*] should allow s3:GetObject on devbucket",
		},
		{
			name: "notresource_excludes_prod_bucket_denies_prod_bucket",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, NotResource: []string{"arn:aws:s3:::prodbucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::prodbucket/file.txt",
			expectedAllow: false,
			description:   "NotResource:[prodbucket/*] should deny s3:GetObject on prodbucket (the excluded resource)",
		},
		{
			name: "notresource_excludes_prod_bucket_allows_other_bucket",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, NotResource: []string{"arn:aws:s3:::prodbucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::staging-bucket/file.txt",
			expectedAllow: true,
			description:   "NotResource:[prodbucket/*] should allow s3:GetObject on any other bucket",
		},

		// --- NotResource with multiple excluded resources ---
		{
			name: "notresource_multiple_exclusions_denies_first_excluded",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:PutObject"}, NotResource: []string{
						"arn:aws:s3:::prodbucket/*",
						"arn:aws:s3:::backupbucket/*",
					}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasPut := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3PutObject)]
				assert.True(t, hasPut)
				for _, excludedRes := range []string{"arn:aws:s3:::prodbucket/*", "arn:aws:s3:::backupbucket/*"} {
					denyStmt := authz.WithSpecificResourceStmt[excludedRes][s3common.PolicyStatementEffectDeny]
					assert.NotNil(t, denyStmt, "expected deny entry for excluded resource %s", excludedRes)
					_, hasPutInDeny := denyStmt.Actions[normalizeAction(s3common.PolicyActionS3PutObject)]
					assert.True(t, hasPutInDeny)
				}
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::prodbucket/file.txt",
			expectedAllow: false,
			description:   "NotResource:[prodbucket/*, backupbucket/*] should deny s3:PutObject on prodbucket",
		},
		{
			name: "notresource_multiple_exclusions_denies_second_excluded",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:PutObject"}, NotResource: []string{
						"arn:aws:s3:::prodbucket/*",
						"arn:aws:s3:::backupbucket/*",
					}},
				}},
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::backupbucket/file.txt",
			expectedAllow: false,
			description:   "NotResource:[prodbucket/*, backupbucket/*] should deny s3:PutObject on backupbucket",
		},
		{
			name: "notresource_multiple_exclusions_allows_non_excluded",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:PutObject"}, NotResource: []string{
						"arn:aws:s3:::prodbucket/*",
						"arn:aws:s3:::backupbucket/*",
					}},
				}},
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::devbucket/file.txt",
			expectedAllow: true,
			description:   "NotResource:[prodbucket/*, backupbucket/*] should allow s3:PutObject on devbucket",
		},

		// --- Deny NotResource: deny on all resources except the listed one ---
		{
			name: "deny_notresource_denies_non_excluded_resource",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: "*", Resource: "*"},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					// Deny everything EXCEPT devbucket
					{Effect: "Deny", Action: []string{"s3:DeleteObject"}, NotResource: []string{"arn:aws:s3:::devbucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				// The global Allow is preserved
				allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				// The Deny NotResource should produce an AllResources deny
				// and a negated Allow (or no-deny) for the excluded resource
				denyStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectDeny]
				assert.NotNil(t, denyStmt)
				_, hasDeleteInDeny := denyStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObject)]
				assert.True(t, hasDeleteInDeny)
				// devbucket should NOT have a deny (it's the excluded resource)
				devDenyStmt := authz.WithSpecificResourceStmt["arn:aws:s3:::devbucket/*"][s3common.PolicyStatementEffectDeny]
				if devDenyStmt != nil {
					_, hasDeleteInDevDeny := devDenyStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObject)]
					assert.False(t, hasDeleteInDevDeny)
				}
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::prodbucket/file.txt",
			expectedAllow: false,
			description:   "Deny NotResource:[devbucket/*] should deny s3:DeleteObject on prodbucket",
		},
		{
			// TODO: P0: Fix Me
			name: "deny_notresource_allows_excluded_resource",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: "*", Resource: "*"},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"s3:DeleteObject"}, NotResource: []string{"arn:aws:s3:::devbucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::devbucket/file.txt",
			expectedAllow: true,
			description:   "Deny NotResource:[devbucket/*] should NOT deny s3:DeleteObject on devbucket (it is the exception)",
		},
		{
			name:     "notresource_allows_non_excluded_bucket",
			policies: []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: "*", NotResource: []string{"arn:aws:s3:::prodbucket/*"}}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::devbucket/file.txt",
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				denyStmt := authz.WithSpecificResourceStmt["arn:aws:s3:::prodbucket/*"][s3common.PolicyStatementEffectDeny]
				assert.NotNil(t, denyStmt)
			},
			expectedAllow: true,
			description:   "NotResource:[prodbucket/*] should allow s3:GetObject on devbucket",
		},
		{
			name:          "notresource_denies_excluded_bucket",
			policies:      []PolicyDocument{{Statement: []*Statement{{Effect: "Allow", Action: "*", NotResource: []string{"arn:aws:s3:::prodbucket/*"}}}}},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::prodbucket/file.txt",
			expectedAllow: false,
			description:   "NotResource:[prodbucket/*] should deny s3:GetObject on prodbucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}

			authz := engine.calculateAuthorizedActions(tt.policies)
			if tt.validateAuthz != nil {
				tt.validateAuthz(t, &authz)
			}

			req := testhelpers.NewMockRequest("GET", "/my-bucket/my-file.txt", nil, nil, nil, nil)

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode, "Error should be AccessDenied")
			}
		})
	}
}

// Test Group 4.3: NotAction + NotResource Combinations
func TestNotActionNotResourceCombinations(t *testing.T) {
	tests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		resource      string
		validateAuthz func(t *testing.T, authz *authorizedActions)
		expectedAllow bool
		description   string
	}{
		// --- Allow NotAction + NotResource: allow all actions except X on all resources except Y ---
		{
			name: "allow_notaction_notresource_allows_non_excluded_action_on_non_excluded_resource",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:      "Allow",
						NotAction:   []string{"s3:DeleteObject"},
						NotResource: []string{"arn:aws:s3:::prodbucket/*"},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				// s3:DeleteObject must not be in the allow list (excluded via NotAction)
				_, hasDelete := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObject)]
				assert.False(t, hasDelete)
				// s3:GetObject must be allowed
				_, hasGet := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGet)
				// prodbucket must have a deny for GetObject (excluded via NotResource)
				denyStmt := authz.WithSpecificResourceStmt["arn:aws:s3:::prodbucket/*"][s3common.PolicyStatementEffectDeny]
				assert.NotNil(t, denyStmt)
				_, hasGetInDeny := denyStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGetInDeny)
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::devbucket/file.txt",
			expectedAllow: true,
			description:   "Allow NotAction:[s3:DeleteObject] NotResource:[prodbucket/*] should allow s3:GetObject on devbucket",
		},
		{
			name: "allow_notaction_notresource_denies_excluded_action_on_non_excluded_resource",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:      "Allow",
						NotAction:   []string{"s3:DeleteObject"},
						NotResource: []string{"arn:aws:s3:::prodbucket/*"},
					},
				}},
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::devbucket/file.txt",
			expectedAllow: false,
			description:   "Allow NotAction:[s3:DeleteObject] NotResource:[prodbucket/*] should deny excluded action s3:DeleteObject even on devbucket",
		},
		{
			name: "allow_notaction_notresource_denies_non_excluded_action_on_excluded_resource",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:      "Allow",
						NotAction:   []string{"s3:DeleteObject"},
						NotResource: []string{"arn:aws:s3:::prodbucket/*"},
					},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::prodbucket/file.txt",
			expectedAllow: false,
			description:   "Allow NotAction:[s3:DeleteObject] NotResource:[prodbucket/*] should deny s3:GetObject on excluded resource prodbucket",
		},
		{
			name: "allow_notaction_notresource_denies_excluded_action_on_excluded_resource",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:      "Allow",
						NotAction:   []string{"s3:DeleteObject"},
						NotResource: []string{"arn:aws:s3:::prodbucket/*"},
					},
				}},
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::prodbucket/file.txt",
			expectedAllow: false,
			description:   "Allow NotAction:[s3:DeleteObject] NotResource:[prodbucket/*] should deny excluded action on excluded resource",
		},

		// --- Deny NotAction + NotResource: deny all actions except X on all resources except Y ---
		{
			name: "deny_notaction_notresource_denies_non_excluded_action_on_non_excluded_resource",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					// Baseline allow
					{Effect: "Allow", Action: "*", Resource: "*"},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					// Deny everything except s3:GetObject on everything except devbucket
					{
						Effect:      "Deny",
						NotAction:   []string{"s3:GetObject"},
						NotResource: []string{"arn:aws:s3:::devbucket/*"},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				denyStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectDeny]
				assert.NotNil(t, denyStmt)
				// s3:GetObject must NOT be in AllResources deny (excluded via NotAction)
				_, hasGetInDeny := denyStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.False(t, hasGetInDeny)
				// s3:DeleteObject must be in AllResources deny
				_, hasDeleteInDeny := denyStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObject)]
				assert.True(t, hasDeleteInDeny)
				// devbucket must NOT have a deny for s3:DeleteObject (excluded via NotResource)
				devDenyStmt := authz.WithSpecificResourceStmt["arn:aws:s3:::devbucket/*"][s3common.PolicyStatementEffectDeny]
				if devDenyStmt != nil {
					_, hasDeleteInDevDeny := devDenyStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObject)]
					assert.False(t, hasDeleteInDevDeny)
				}
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::prodbucket/file.txt",
			expectedAllow: false,
			description:   "Deny NotAction:[s3:GetObject] NotResource:[devbucket/*] should deny s3:DeleteObject on prodbucket",
		},
		{
			name: "deny_notaction_notresource_allows_excepted_action_on_non_excluded_resource",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: "*", Resource: "*"},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:      "Deny",
						NotAction:   []string{"s3:GetObject"},
						NotResource: []string{"arn:aws:s3:::devbucket/*"},
					},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::prodbucket/file.txt",
			expectedAllow: true,
			description:   "Deny NotAction:[s3:GetObject] NotResource:[devbucket/*] should NOT deny excepted action s3:GetObject on prodbucket",
		},
		{
			// TODO: P0: Fix Me
			name: "deny_notaction_notresource_allows_non_excluded_action_on_excepted_resource",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: "*", Resource: "*"},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:      "Deny",
						NotAction:   []string{"s3:GetObject"},
						NotResource: []string{"arn:aws:s3:::devbucket/*"},
					},
				}},
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::devbucket/file.txt",
			expectedAllow: true,
			description:   "Deny NotAction:[s3:GetObject] NotResource:[devbucket/*] should NOT deny s3:DeleteObject on excepted resource devbucket",
		},

		// --- NotAction + explicit Action interaction (cross-policy) ---
		{
			name: "notaction_policy_combined_with_explicit_allow_policy",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					// Explicitly allow only s3:GetObject
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					// NotAction:[s3:DeleteObject] on * means allow everything except Delete
					{Effect: "Allow", NotAction: []string{"s3:DeleteObject"}, Resource: "*"},
				}},
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "Combined explicit Allow s3:GetObject + NotAction:[s3:DeleteObject] Allow should allow s3:PutObject (covered by NotAction policy)",
		},
		{
			name: "notaction_policy_combined_with_explicit_allow_policy_denies_excluded",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", NotAction: []string{"s3:DeleteObject"}, Resource: "*"},
				}},
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "Combined explicit Allow s3:GetObject + NotAction:[s3:DeleteObject] Allow should deny s3:DeleteObject",
		},

		// --- NotResource + explicit Resource interaction (cross-policy) ---
		{
			name: "notresource_policy_combined_with_explicit_resource_policy_allows_non_excluded",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					// Allow s3:GetObject on specific bucket
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					// NotResource:[prodbucket/*] allows everything on non-prod
					{Effect: "Allow", Action: []string{"s3:PutObject"}, NotResource: []string{"arn:aws:s3:::prodbucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				// my-bucket should have GetObject allow
				myBucketAllow := authz.WithSpecificResourceStmt["arn:aws:s3:::my-bucket/*"][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, myBucketAllow)
				_, hasGet := myBucketAllow.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGet)
				// AllResourcesStmt should have PutObject allow from NotResource policy
				allResAllow := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allResAllow)
				_, hasPut := allResAllow.Actions[normalizeAction(s3common.PolicyActionS3PutObject)]
				assert.True(t, hasPut)
				// prodbucket should have PutObject deny from NotResource exclusion
				prodDeny := authz.WithSpecificResourceStmt["arn:aws:s3:::prodbucket/*"][s3common.PolicyStatementEffectDeny]
				assert.NotNil(t, prodDeny)
				_, hasPutInDeny := prodDeny.Actions[normalizeAction(s3common.PolicyActionS3PutObject)]
				assert.True(t, hasPutInDeny)
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "NotResource:[prodbucket/*] + explicit resource my-bucket: should allow s3:PutObject on my-bucket",
		},
		{
			name: "notresource_policy_combined_with_explicit_resource_policy_denies_excluded",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:PutObject"}, NotResource: []string{"arn:aws:s3:::prodbucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::prodbucket/file.txt",
			expectedAllow: false,
			description:   "NotResource:[prodbucket/*] should deny s3:PutObject on prodbucket even when another policy exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}

			authz := engine.calculateAuthorizedActions(tt.policies)
			if tt.validateAuthz != nil {
				tt.validateAuthz(t, &authz)
			}

			req := testhelpers.NewMockRequest("GET", "/my-bucket/my-file.txt", nil, nil, nil, nil)

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode, "Error should be AccessDenied")
			}
		})
	}
}

// 5. Condition Tests
// Test Group 5.1: String Conditions
func TestStringConditions(t *testing.T) {
	tests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		resource      string
		validateAuthz func(t *testing.T, authz *authorizedActions)
		buildRequest  func() *http.Request
		expectedAllow bool
		description   string
	}{
		// -------------------------------------------------------------------------
		// StringEquals — basic single value
		// -------------------------------------------------------------------------
		{
			name: "stringequals_prefix_matches_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals: {"s3:prefix": "home/"},
				},
			}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res := "arn:aws:s3:::my-bucket"
				bothAllAndNoResShouldBeNil(t, authz)
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Len(t, allowStmt.Actions, 1)
				assert.Len(t, allowStmt.Conditions, 1)
				assert.Len(t, allowStmt.Conditions[s3common.CondOpStringEquals]["s3:prefix"], 1)
				assert.Equal(t, "home/", allowStmt.Conditions[s3common.CondOpStringEquals]["s3:prefix"][0])
				assert.Nil(t, authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny])
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "home/"})
			},
			expectedAllow: true,
			description:   "StringEquals: s3:prefix=home/ should allow when request prefix matches exactly",
		},
		{
			name: "stringequals_prefix_mismatch_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals: {"s3:prefix": "home/"},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "shared/"})
			},
			expectedAllow: false,
			description:   "StringEquals: mismatched prefix should deny",
		},
		{
			name: "stringequals_prefix_missing_key_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals: {"s3:prefix": "home/"},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				// No prefix query param → key missing → StringEquals fails without IfExists
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, nil)
			},
			expectedAllow: false,
			description:   "StringEquals: missing condition key should deny (key must exist without IfExists)",
		},
		{
			name: "stringequals_empty_value_matches_empty_key",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals: {"s3:prefix": ""},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": ""})
			},
			expectedAllow: true,
			description:   "StringEquals: empty string condition value matches empty request value",
		},

		// -------------------------------------------------------------------------
		// StringEquals — multiple expected values (OR logic within one key)
		// -------------------------------------------------------------------------
		{
			name: "stringequals_multi_value_first_matches_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals: {"s3:prefix": []string{"home/", "shared/", "archive/"}},
				},
			}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res := "arn:aws:s3:::my-bucket"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Len(t, allowStmt.Conditions[s3common.CondOpStringEquals]["s3:prefix"], 3)
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "home/"})
			},
			expectedAllow: true,
			description:   "StringEquals multi-value: first value matches → allow (OR logic)",
		},
		{
			name: "stringequals_multi_value_second_matches_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals: {"s3:prefix": []string{"home/", "shared/", "archive/"}},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "shared/"})
			},
			expectedAllow: true,
			description:   "StringEquals multi-value: second value matches → allow (OR logic)",
		},
		{
			name: "stringequals_multi_value_none_matches_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals: {"s3:prefix": []string{"home/", "shared/", "archive/"}},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "tmp/"})
			},
			expectedAllow: false,
			description:   "StringEquals multi-value: no value matches → deny",
		},

		// -------------------------------------------------------------------------
		// StringNotEquals
		// -------------------------------------------------------------------------
		{
			name: "stringnotequals_different_value_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringNotEquals: {"s3:prefix": "restricted/"},
				},
			}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res := "arn:aws:s3:::my-bucket"
				bothAllAndNoResShouldBeNil(t, authz)
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Len(t, allowStmt.Conditions, 1)
				assert.Len(t, allowStmt.Conditions[s3common.CondOpStringNotEquals]["s3:prefix"], 1)
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "home/"})
			},
			expectedAllow: true,
			description:   "StringNotEquals: prefix != restricted/ → allow",
		},
		{
			name: "stringnotequals_matching_value_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringNotEquals: {"s3:prefix": "restricted/"},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "restricted/"})
			},
			expectedAllow: false,
			description:   "StringNotEquals: prefix == restricted/ → deny (actual equals the not-equals target)",
		},
		// StringNotEquals multi-value: evaluateSingleCondition returns true when actual != AT LEAST ONE
		// expected value. So "admin/" != "restricted/" is true → allow.
		{
			name: "stringnotequals_multi_value_satisfies_one_neq_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringNotEquals: {"s3:prefix": []string{"restricted/", "internal/"}},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "admin/"})
			},
			expectedAllow: true,
			description:   "StringNotEquals multi-value: admin/ != restricted/ (first expected) → allow",
		},

		// -------------------------------------------------------------------------
		// StringEqualsIgnoreCase
		// -------------------------------------------------------------------------
		{
			name: "stringequalsi_header_uppercase_matches_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:PutObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEqualsIgnoreCase: {"s3:x-amz-server-side-encryption": "aes256"},
				},
			}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res := "arn:aws:s3:::my-bucket/*"
				bothAllAndNoResShouldBeNil(t, authz)
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Contains(t, allowStmt.Conditions, s3common.CondOpStringEqualsIgnoreCase)
			},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.Header.Set("x-amz-server-side-encryption", "AES256")
				return req
			},
			expectedAllow: true,
			description:   "StringEqualsIgnoreCase: AES256 matches aes256 case-insensitively → allow",
		},
		{
			// TODO: P0: Fix Me
			name: "string_equals_ignore_case_condition_case_insensitive",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpStringEqualsIgnoreCase: {"s3:prefix": "Home/"},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"prefix": "HOME/"}, nil) // different case → should still match
			},
			expectedAllow: true,
			description:   "StringEqualsIgnoreCase must ignore case differences",
		},
		{
			name: "stringequalsi_header_different_value_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:PutObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEqualsIgnoreCase: {"s3:x-amz-server-side-encryption": "aes256"},
				},
			}}}},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.Header.Set("x-amz-server-side-encryption", "aws:kms")
				return req
			},
			expectedAllow: false,
			description:   "StringEqualsIgnoreCase: aws:kms does not match aes256 → deny",
		},

		// -------------------------------------------------------------------------
		// StringNotEqualsIgnoreCase
		// -------------------------------------------------------------------------
		{
			name: "stringnotequalsi_different_value_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:PutObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringNotEqualsIgnoreCase: {"s3:x-amz-storage-class": "GLACIER"},
				},
			}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Contains(t, allowStmt.Conditions, s3common.CondOpStringNotEqualsIgnoreCase)
			},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.Header.Set("x-amz-storage-class", "STANDARD")
				return req
			},
			expectedAllow: true,
			description:   "StringNotEqualsIgnoreCase: STANDARD != GLACIER (case-insensitive) → allow",
		},
		{
			name: "stringnotequalsi_same_value_case_insensitive_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:PutObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringNotEqualsIgnoreCase: {"s3:x-amz-storage-class": "GLACIER"},
				},
			}}}},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.Header.Set("x-amz-storage-class", "glacier") // lowercase same value
				return req
			},
			expectedAllow: false,
			description:   "StringNotEqualsIgnoreCase: glacier equals GLACIER case-insensitively → deny",
		},

		// -------------------------------------------------------------------------
		// StringLike
		// -------------------------------------------------------------------------
		{
			name: "stringlike_prefix_wildcard_matches_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringLike: {"aws:UserAgent": "aws-sdk-*"},
				},
			}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Contains(t, allowStmt.Conditions, s3common.CondOpStringLike)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.Header.Set("User-Agent", "aws-sdk-go/1.44.0")
				return req
			},
			expectedAllow: true,
			description:   "StringLike: aws-sdk-go/1.44.0 matches aws-sdk-* → allow",
		},
		{
			name: "stringlike_prefix_wildcard_no_match_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringLike: {"aws:UserAgent": "aws-sdk-*"},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.Header.Set("User-Agent", "curl/7.82.0")
				return req
			},
			expectedAllow: false,
			description:   "StringLike: curl/7.82.0 does not match aws-sdk-* → deny",
		},
		{
			name: "stringlike_single_char_wildcard_matches",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringLike: {"s3:prefix": "home/?/"},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "home/a/"})
			},
			expectedAllow: true,
			description:   "StringLike: home/a/ matches home/?/ (single-char wildcard) → allow",
		},
		{
			name: "stringlike_single_char_wildcard_multi_char_no_match_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringLike: {"s3:prefix": "home/?/"},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "home/alice/"})
			},
			expectedAllow: false,
			description:   "StringLike: home/alice/ does not match home/?/ (? must be single char) → deny",
		},
		{
			name: "stringlike_multi_pattern_first_matches_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringLike: {"s3:prefix": []string{"home/*", "shared/*"}},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "home/alice/docs"})
			},
			expectedAllow: true,
			description:   "StringLike multi-pattern: home/alice/docs matches home/* → allow",
		},
		{
			name: "stringlike_multi_pattern_none_matches_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringLike: {"s3:prefix": []string{"home/*", "shared/*"}},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "admin/secret"})
			},
			expectedAllow: false,
			description:   "StringLike multi-pattern: admin/secret matches neither home/* nor shared/* → deny",
		},

		// -------------------------------------------------------------------------
		// StringNotLike
		// -------------------------------------------------------------------------
		{
			name: "stringnotlike_no_match_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringNotLike: {"s3:prefix": "restricted/*"},
				},
			}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res := "arn:aws:s3:::my-bucket"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Contains(t, allowStmt.Conditions, s3common.CondOpStringNotLike)
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "home/alice"})
			},
			expectedAllow: true,
			description:   "StringNotLike: home/alice does not match restricted/* → allow",
		},
		{
			name: "stringnotlike_match_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringNotLike: {"s3:prefix": "restricted/*"},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "restricted/secret"})
			},
			expectedAllow: false,
			description:   "StringNotLike: restricted/secret matches restricted/* → condition fails → deny",
		},

		// -------------------------------------------------------------------------
		// Multiple condition keys (AND logic across keys in same operator)
		// -------------------------------------------------------------------------
		{
			name: "stringequals_two_keys_both_match_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals: {
						"s3:prefix":    "home/",
						"s3:delimiter": "/",
					},
				},
			}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res := "arn:aws:s3:::my-bucket"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Len(t, allowStmt.Conditions[s3common.CondOpStringEquals], 2)
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{
					"prefix":    "home/",
					"delimiter": "/",
				})
			},
			expectedAllow: true,
			description:   "StringEquals AND keys: prefix=home/ AND delimiter=/ both match → allow",
		},
		{
			name: "stringequals_two_keys_first_mismatch_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals: {
						"s3:prefix":    "home/",
						"s3:delimiter": "/",
					},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{
					"prefix":    "shared/",
					"delimiter": "/",
				})
			},
			expectedAllow: false,
			description:   "StringEquals AND keys: prefix=shared/ mismatch → deny (AND fails)",
		},
		{
			name: "stringequals_two_keys_second_mismatch_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals: {
						"s3:prefix":    "home/",
						"s3:delimiter": "/",
					},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{
					"prefix":    "home/",
					"delimiter": "|",
				})
			},
			expectedAllow: false,
			description:   "StringEquals AND keys: delimiter=| mismatch → deny (AND fails)",
		},

		// -------------------------------------------------------------------------
		// Multiple condition operators (AND logic across operators)
		// -------------------------------------------------------------------------
		{
			name: "two_operators_both_match_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:PutObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals:           {"s3:x-amz-storage-class": "STANDARD"},
					s3common.CondOpStringEqualsIgnoreCase: {"s3:x-amz-server-side-encryption": "aes256"},
				},
			}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Len(t, allowStmt.Conditions, 2)
				assert.Contains(t, allowStmt.Conditions, s3common.CondOpStringEquals)
				assert.Contains(t, allowStmt.Conditions, s3common.CondOpStringEqualsIgnoreCase)
			},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.Header.Set("x-amz-storage-class", "STANDARD")
				req.Header.Set("x-amz-server-side-encryption", "AES256")
				return req
			},
			expectedAllow: true,
			description:   "Two operators: StringEquals AND StringEqualsIgnoreCase both match → allow",
		},
		{
			name: "two_operators_first_fails_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:PutObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals:           {"s3:x-amz-storage-class": "STANDARD"},
					s3common.CondOpStringEqualsIgnoreCase: {"s3:x-amz-server-side-encryption": "aes256"},
				},
			}}}},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.Header.Set("x-amz-storage-class", "GLACIER") // wrong
				req.Header.Set("x-amz-server-side-encryption", "AES256")
				return req
			},
			expectedAllow: false,
			description:   "Two operators: StringEquals fails (wrong storage class) → deny (AND across operators)",
		},

		// -------------------------------------------------------------------------
		// Deny statement with string condition
		// -------------------------------------------------------------------------
		{
			name: "deny_stringequals_condition_matching_denies",
			policies: []PolicyDocument{
				{Statement: []*Statement{{
					Effect:   "Allow",
					Action:   []string{"s3:PutObject"},
					Resource: []string{"arn:aws:s3:::my-bucket/*"},
				}}},
				{Statement: []*Statement{{
					Effect:   "Deny",
					Action:   []string{"s3:PutObject"},
					Resource: []string{"arn:aws:s3:::my-bucket/*"},
					Condition: map[s3common.PolicyCondOp]map[string]any{
						s3common.CondOpStringEquals: {"s3:x-amz-storage-class": "GLACIER"},
					},
				}}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Len(t, allowStmt.Conditions, 0)
				denyStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny]
				assert.NotNil(t, denyStmt)
				assert.Len(t, denyStmt.Conditions, 1)
				assert.Contains(t, denyStmt.Conditions, s3common.CondOpStringEquals)
			},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.Header.Set("x-amz-storage-class", "GLACIER")
				return req
			},
			expectedAllow: false,
			description:   "Deny with StringEquals condition matching → deny fires",
		},
		{
			name: "deny_stringequals_condition_not_matching_allows",
			policies: []PolicyDocument{
				{Statement: []*Statement{{
					Effect:   "Allow",
					Action:   []string{"s3:PutObject"},
					Resource: []string{"arn:aws:s3:::my-bucket/*"},
				}}},
				{Statement: []*Statement{{
					Effect:   "Deny",
					Action:   []string{"s3:PutObject"},
					Resource: []string{"arn:aws:s3:::my-bucket/*"},
					Condition: map[s3common.PolicyCondOp]map[string]any{
						s3common.CondOpStringEquals: {"s3:x-amz-storage-class": "GLACIER"},
					},
				}}},
			},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.Header.Set("x-amz-storage-class", "STANDARD") // different → deny condition not triggered
				return req
			},
			expectedAllow: true,
			description:   "Deny with StringEquals condition not matching: deny not triggered → allow",
		},

		// -------------------------------------------------------------------------
		// ForAllValues:StringEquals
		// -------------------------------------------------------------------------
		{
			name: "forallvalues_stringequals_value_in_set_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.PolicyCondOp(s3common.CondOpForAllValuesPrefix + string(s3common.CondOpStringEquals)): {
						"s3:prefix": []string{"home/", "shared/"},
					},
				},
			}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res := "arn:aws:s3:::my-bucket"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				expectedKey := s3common.PolicyCondOp(s3common.CondOpForAllValuesPrefix + string(s3common.CondOpStringEquals))
				assert.Contains(t, allowStmt.Conditions, expectedKey)
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "home/"})
			},
			expectedAllow: true,
			description:   "ForAllValues:StringEquals: actual value home/ is in expected set {home/, shared/} → allow",
		},
		{
			name: "forallvalues_stringequals_value_not_in_set_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.PolicyCondOp(s3common.CondOpForAllValuesPrefix + string(s3common.CondOpStringEquals)): {
						"s3:prefix": []string{"home/", "shared/"},
					},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "admin/"})
			},
			expectedAllow: false,
			description:   "ForAllValues:StringEquals: actual value admin/ not in expected set → deny",
		},
		{
			name: "forallvalues_stringequals_empty_actual_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.PolicyCondOp(s3common.CondOpForAllValuesPrefix + string(s3common.CondOpStringEquals)): {
						"s3:prefix": []string{"home/", "shared/"},
					},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				// No prefix → empty string → ForAllValues vacuously true
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "ForAllValues:StringEquals: empty actual (key absent) → vacuously true → allow",
		},
		{
			name: "forallvalues_stringequals_multi_actual_all_in_set_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.PolicyCondOp(s3common.CondOpForAllValuesPrefix + string(s3common.CondOpStringEquals)): {
						"s3:prefix": []string{"home/", "shared/", "archive/"},
					},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				// Comma-separated multi-value as parsed by evaluateForAllValues
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "home/, shared/"})
			},
			expectedAllow: true,
			description:   "ForAllValues:StringEquals: both actual values in expected set → allow",
		},
		{
			name: "forallvalues_stringequals_multi_actual_one_not_in_set_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.PolicyCondOp(s3common.CondOpForAllValuesPrefix + string(s3common.CondOpStringEquals)): {
						"s3:prefix": []string{"home/", "shared/"},
					},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				// "home/" in set but "admin/" not → ForAllValues fails
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "home/, admin/"})
			},
			expectedAllow: false,
			description:   "ForAllValues:StringEquals: one actual value not in expected set → deny",
		},
		{
			name: "forallvalues_stringequals_multi_actual_multi_statement_all_in_set_allows",
			policies: []PolicyDocument{{Statement: []*Statement{
				{
					Effect:   "Allow",
					Action:   []string{"s3:ListBucket"},
					Resource: []string{"arn:aws:s3:::my-bucket"},
					Condition: map[s3common.PolicyCondOp]map[string]any{
						s3common.PolicyCondOp(s3common.CondOpForAllValuesPrefix + string(s3common.CondOpStringEquals)): {
							"s3:prefix": []string{"home/", "shared/"},
						},
					},
				},
				{
					Effect:   "Allow",
					Action:   []string{"s3:ListBucket"},
					Resource: []string{"arn:aws:s3:::my-bucket"},
					Condition: map[s3common.PolicyCondOp]map[string]any{
						s3common.PolicyCondOp(s3common.CondOpForAllValuesPrefix + string(s3common.CondOpStringEquals)): {
							"s3:prefix": []string{"archive/"},
						},
					},
				},
			}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				// Comma-separated multi-value as parsed by evaluateForAllValues
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "home/, archive/"})
			},
			expectedAllow: true,
			description:   "ForAllValues:StringEquals: all actual values in expected sets across multiple statements → allow",
		},

		// -------------------------------------------------------------------------
		// ForAnyValue:StringEquals
		// -------------------------------------------------------------------------
		{
			name: "foranyvalue_stringequals_one_value_matches_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.PolicyCondOp(s3common.CondOpForAnyValuePrefix + string(s3common.CondOpStringEquals)): {
						"s3:prefix": []string{"home/", "shared/"},
					},
				},
			}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res := "arn:aws:s3:::my-bucket"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				expectedKey := s3common.PolicyCondOp(s3common.CondOpForAnyValuePrefix + string(s3common.CondOpStringEquals))
				assert.Contains(t, allowStmt.Conditions, expectedKey)
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "home/"})
			},
			expectedAllow: true,
			description:   "ForAnyValue:StringEquals: actual value home/ in expected set → allow",
		},
		{
			name: "foranyvalue_stringequals_no_value_matches_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.PolicyCondOp(s3common.CondOpForAnyValuePrefix + string(s3common.CondOpStringEquals)): {
						"s3:prefix": []string{"home/", "shared/"},
					},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "admin/"})
			},
			expectedAllow: false,
			description:   "ForAnyValue:StringEquals: actual value admin/ not in expected set → deny",
		},
		{
			name: "foranyvalue_stringequals_empty_actual_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.PolicyCondOp(s3common.CondOpForAnyValuePrefix + string(s3common.CondOpStringEquals)): {
						"s3:prefix": []string{"home/", "shared/"},
					},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, nil)
			},
			expectedAllow: false,
			description:   "ForAnyValue:StringEquals: empty actual (key absent) → no match → deny",
		},
		{
			name: "foranyvalue_stringequals_multi_actual_one_matches_allows",
			policies: []PolicyDocument{{Statement: []*Statement{
				{
					Effect:   "Allow",
					Action:   []string{"s3:ListBucket"},
					Resource: []string{"arn:aws:s3:::my-bucket"},
					Condition: map[s3common.PolicyCondOp]map[string]any{
						s3common.PolicyCondOp(s3common.CondOpForAnyValuePrefix + string(s3common.CondOpStringEquals)): {
							"s3:prefix": []string{"home/"},
						},
					},
				},
				{
					Effect:   "Allow",
					Action:   []string{"s3:ListBucket"},
					Resource: []string{"arn:aws:s3:::my-bucket"},
					Condition: map[s3common.PolicyCondOp]map[string]any{
						s3common.PolicyCondOp(s3common.CondOpForAnyValuePrefix + string(s3common.CondOpStringEquals)): {
							"s3:prefix": []string{"shared/"},
						},
					},
				},
			}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				// "admin/" not in set but "home/" is → ForAnyValue passes
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "admin/, home/"})
			},
			expectedAllow: true,
			description:   "ForAnyValue:StringEquals: at least one actual value in expected set → allow",
		},

		// -------------------------------------------------------------------------
		// ForAllValues:StringLike  &  ForAnyValue:StringLike
		// -------------------------------------------------------------------------
		{
			name: "forallvalues_stringlike_all_match_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.PolicyCondOp(s3common.CondOpForAllValuesPrefix + string(s3common.CondOpStringLike)): {
						"s3:prefix": "home/*",
					},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "home/alice/docs"})
			},
			expectedAllow: true,
			description:   "ForAllValues:StringLike: home/alice/docs matches home/* → allow",
		},
		{
			name: "forallvalues_stringlike_no_match_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.PolicyCondOp(s3common.CondOpForAllValuesPrefix + string(s3common.CondOpStringLike)): {
						"s3:prefix": "home/*",
					},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "admin/secret"})
			},
			expectedAllow: false,
			description:   "ForAllValues:StringLike: admin/secret does not match home/* → deny",
		},
		{
			name: "foranyvalue_stringlike_one_value_matches_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.PolicyCondOp(s3common.CondOpForAnyValuePrefix + string(s3common.CondOpStringLike)): {
						"s3:prefix": []string{"home/*", "shared/*"},
					},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "shared/docs"})
			},
			expectedAllow: true,
			description:   "ForAnyValue:StringLike: shared/docs matches shared/* → allow",
		},
		{
			name: "foranyvalue_stringlike_no_value_matches_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.PolicyCondOp(s3common.CondOpForAnyValuePrefix + string(s3common.CondOpStringLike)): {
						"s3:prefix": []string{"home/*", "shared/*"},
					},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"prefix": "admin/secret"})
			},
			expectedAllow: false,
			description:   "ForAnyValue:StringLike: admin/secret matches neither home/* nor shared/* → deny",
		},

		// -------------------------------------------------------------------------
		// StringEqualsIfExists — IfExists modifier
		// -------------------------------------------------------------------------
		{
			name: "stringequals_ifexists_key_present_matches_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:PutObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals + s3common.CondOpIfExistsSuffix: {
						"s3:x-amz-storage-class": "STANDARD",
					},
				},
			}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				expectedKey := s3common.CondOpStringEquals + s3common.CondOpIfExistsSuffix
				assert.Contains(t, allowStmt.Conditions, expectedKey)
			},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.Header.Set("x-amz-storage-class", "STANDARD")
				return req
			},
			expectedAllow: true,
			description:   "StringEqualsIfExists: key present and value matches → allow",
		},
		{
			name: "stringequals_ifexists_key_present_mismatch_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:PutObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals + s3common.CondOpIfExistsSuffix: {
						"s3:x-amz-storage-class": "STANDARD",
					},
				},
			}}}},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.Header.Set("x-amz-storage-class", "GLACIER")
				return req
			},
			expectedAllow: false,
			description:   "StringEqualsIfExists: key present but value mismatches → deny",
		},
		{
			name: "stringequals_ifexists_key_absent_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:PutObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals + s3common.CondOpIfExistsSuffix: {
						"s3:x-amz-storage-class": "STANDARD",
					},
				},
			}}}},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// No x-amz-storage-class header → key absent → IfExists passes vacuously
				return testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "StringEqualsIfExists: key absent → condition passes vacuously → allow",
		},
		{
			name: "stringnotequals_ifexists_key_absent_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:PutObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringNotEquals + s3common.CondOpIfExistsSuffix: {
						"s3:x-amz-storage-class": "GLACIER",
					},
				},
			}}}},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "StringNotEqualsIfExists: key absent → condition passes vacuously → allow",
		},
		{
			name: "stringnotequals_ifexists_key_present_different_value_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:PutObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringNotEquals + s3common.CondOpIfExistsSuffix: {
						"s3:x-amz-storage-class": "GLACIER",
					},
				},
			}}}},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.Header.Set("x-amz-storage-class", "STANDARD")
				return req
			},
			expectedAllow: true,
			description:   "StringNotEqualsIfExists: key present and value != GLACIER → allow",
		},
		{
			name: "stringlike_ifexists_key_absent_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringLike + s3common.CondOpIfExistsSuffix: {
						"aws:UserAgent": "aws-sdk-*",
					},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "StringLikeIfExists: key absent → condition passes vacuously → allow",
		},
		{
			name: "stringlike_ifexists_key_present_matches_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringLike + s3common.CondOpIfExistsSuffix: {
						"aws:UserAgent": "aws-sdk-*",
					},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.Header.Set("User-Agent", "aws-sdk-go/1.44.0")
				return req
			},
			expectedAllow: true,
			description:   "StringLikeIfExists: key present and matches → allow",
		},
		{
			name: "stringlike_ifexists_key_present_no_match_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringLike + s3common.CondOpIfExistsSuffix: {
						"aws:UserAgent": "aws-sdk-*",
					},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.Header.Set("User-Agent", "curl/7.82.0")
				return req
			},
			expectedAllow: false,
			description:   "StringLikeIfExists: key present but no match → deny",
		},

		// -------------------------------------------------------------------------
		// Condition on AllResources (*) and NoResource (IAM) statements
		// -------------------------------------------------------------------------
		{
			name: "stringequals_condition_on_all_resources_matches_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: "*",
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals: {"aws:UserAgent": "internal-client"},
				},
			}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 0)
				allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Len(t, allowStmt.Conditions, 1)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::any-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("GET", "/any-bucket/file.txt", nil, nil, nil, nil)
				req.Header.Set("User-Agent", "internal-client")
				return req
			},
			expectedAllow: true,
			description:   "StringEquals on Resource=*: matching User-Agent → allow",
		},
		{
			name: "stringequals_condition_on_all_resources_mismatch_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: "*",
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals: {"aws:UserAgent": "internal-client"},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::any-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("GET", "/any-bucket/file.txt", nil, nil, nil, nil)
				req.Header.Set("User-Agent", "external-client")
				return req
			},
			expectedAllow: false,
			description:   "StringEquals on Resource=*: mismatched User-Agent → deny",
		},
		{
			name: "stringequals_condition_on_no_resource_matches_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect: "Allow",
				Action: []string{"iam:ListUsers"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals: {"aws:UserAgent": "internal-admin"},
				},
			}}}},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 0)
				allowStmt := authz.NoResourceStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.Len(t, allowStmt.Conditions, 1)
			},
			action:   s3common.PolicyActionIAMListUsers,
			resource: "",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("GET", "/iam/users", nil, nil, nil, nil)
				req.Header.Set("User-Agent", "internal-admin")
				return req
			},
			expectedAllow: true,
			description:   "StringEquals on NoResource IAM statement: matching → allow",
		},
		{
			name: "stringequals_condition_on_no_resource_mismatch_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect: "Allow",
				Action: []string{"iam:ListUsers"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals: {"aws:UserAgent": "internal-admin"},
				},
			}}}},
			action:   s3common.PolicyActionIAMListUsers,
			resource: "",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("GET", "/iam/users", nil, nil, nil, nil)
				req.Header.Set("User-Agent", "external-user")
				return req
			},
			expectedAllow: false,
			description:   "StringEquals on NoResource IAM statement: mismatch → deny",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}

			authz := engine.calculateAuthorizedActions(tt.policies)
			if tt.validateAuthz != nil {
				tt.validateAuthz(t, &authz)
			}

			req := tt.buildRequest()
			err := engine.checkAccess(req.Context(), authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode, "Error should be AccessDenied")
			}
		})
	}
}

// Test Group 5.2: Numeric Conditions
func TestNumericConditions(t *testing.T) {
	tests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		resource      string
		buildRequest  func() *http.Request
		validateAuthz func(t *testing.T, authz *authorizedActions)
		expectedAllow bool
		description   string
	}{
		// --- NumericEquals ---
		{
			name: "numericequals_max_keys_exact_match_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericEquals: {
								"s3:max-keys": "100",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				assert.Len(t, authz.WithSpecificResourceStmt, 1)
				res := "arn:aws:s3:::my-bucket"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasListBucket := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3ListBucket)]
				assert.True(t, hasListBucket)
				assert.NotNil(t, allowStmt.Conditions)
				condVals := allowStmt.Conditions[s3common.CondOpNumericEquals]["s3:max-keys"]
				assert.Equal(t, []string{"100"}, condVals)
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "100"})
			},
			expectedAllow: true,
			description:   "NumericEquals: max-keys=100 should allow when request max-keys matches exactly",
		},
		{
			name: "numericequals_max_keys_mismatch_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericEquals: {
								"s3:max-keys": "100",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "200"})
			},
			expectedAllow: false,
			description:   "NumericEquals: max-keys=100 should deny when request max-keys is 200",
		},
		{
			name: "numericequals_missing_key_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericEquals: {
								"s3:max-keys": "100",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				// No max-keys param
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, nil)
			},
			expectedAllow: false,
			description:   "NumericEquals: should deny when condition key is missing from request",
		},

		// --- NumericNotEquals ---
		{
			name: "numericnotequals_max_keys_different_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericNotEquals: {
								"s3:max-keys": "1000",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "100"})
			},
			expectedAllow: true,
			description:   "NumericNotEquals: should allow when max-keys does not equal 1000",
		},
		{
			name: "numericnotequals_max_keys_equal_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericNotEquals: {
								"s3:max-keys": "1000",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "1000"})
			},
			expectedAllow: false,
			description:   "NumericNotEquals: should deny when max-keys equals 1000",
		},

		// --- NumericLessThan ---
		{
			name: "numericlessthan_max_keys_within_limit_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericLessThan: {
								"s3:max-keys": "1000",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				condVals := allowStmt.Conditions[s3common.CondOpNumericLessThan]["s3:max-keys"]
				assert.Equal(t, []string{"1000"}, condVals)
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "500"})
			},
			expectedAllow: true,
			description:   "NumericLessThan: max-keys=500 should be allowed when limit is 1000",
		},
		{
			name: "numericlessthan_max_keys_at_limit_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericLessThan: {
								"s3:max-keys": "1000",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				// Equal to limit — NumericLessThan is strict
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "1000"})
			},
			expectedAllow: false,
			description:   "NumericLessThan: max-keys=1000 should deny when limit is 1000 (not strictly less)",
		},
		{
			name: "numericlessthan_max_keys_over_limit_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericLessThan: {
								"s3:max-keys": "1000",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "2000"})
			},
			expectedAllow: false,
			description:   "NumericLessThan: max-keys=2000 should deny when limit is 1000",
		},

		// --- NumericLessThanEquals ---
		{
			name: "numericlessthanequals_max_keys_at_limit_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericLessThanEquals: {
								"s3:max-keys": "1000",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				// Exactly at the limit — LessThanEquals should allow
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "1000"})
			},
			expectedAllow: true,
			description:   "NumericLessThanEquals: max-keys=1000 should allow when limit is 1000 (inclusive)",
		},
		{
			name: "numericlessthanequals_max_keys_below_limit_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericLessThanEquals: {
								"s3:max-keys": "1000",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "999"})
			},
			expectedAllow: true,
			description:   "NumericLessThanEquals: max-keys=999 should allow when limit is 1000",
		},
		{
			name: "numericlessthanequals_max_keys_over_limit_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericLessThanEquals: {
								"s3:max-keys": "1000",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "1001"})
			},
			expectedAllow: false,
			description:   "NumericLessThanEquals: max-keys=1001 should deny when limit is 1000",
		},

		// --- NumericGreaterThan ---
		{
			name: "numericgreaterthan_max_keys_above_threshold_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericGreaterThan: {
								"s3:max-keys": "10",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "50"})
			},
			expectedAllow: true,
			description:   "NumericGreaterThan: max-keys=50 should allow when threshold is 10",
		},
		{
			name: "numericgreaterthan_max_keys_at_threshold_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericGreaterThan: {
								"s3:max-keys": "10",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				// Equal to threshold — GreaterThan is strict
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "10"})
			},
			expectedAllow: false,
			description:   "NumericGreaterThan: max-keys=10 should deny when threshold is 10 (not strictly greater)",
		},
		{
			name: "numericgreaterthan_max_keys_below_threshold_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericGreaterThan: {
								"s3:max-keys": "10",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "5"})
			},
			expectedAllow: false,
			description:   "NumericGreaterThan: max-keys=5 should deny when threshold is 10",
		},

		// --- NumericGreaterThanEquals ---
		{
			name: "numericgreaterthanequals_max_keys_at_threshold_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericGreaterThanEquals: {
								"s3:max-keys": "10",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				// Exactly at threshold — GreaterThanEquals should allow
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "10"})
			},
			expectedAllow: true,
			description:   "NumericGreaterThanEquals: max-keys=10 should allow when threshold is 10 (inclusive)",
		},
		{
			name: "numericgreaterthanequals_max_keys_above_threshold_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericGreaterThanEquals: {
								"s3:max-keys": "10",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "11"})
			},
			expectedAllow: true,
			description:   "NumericGreaterThanEquals: max-keys=11 should allow when threshold is 10",
		},
		{
			name: "numericgreaterthanequals_max_keys_below_threshold_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericGreaterThanEquals: {
								"s3:max-keys": "10",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "9"})
			},
			expectedAllow: false,
			description:   "NumericGreaterThanEquals: max-keys=9 should deny when threshold is 10",
		},

		// --- Deny with numeric condition ---
		{
			name: "deny_numericgreaterthan_blocks_high_max_keys",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:ListBucket"}, Resource: []string{"arn:aws:s3:::my-bucket"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Deny",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericGreaterThan: {
								"s3:max-keys": "1000",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "2000"})
			},
			expectedAllow: false,
			description:   "Deny NumericGreaterThan: should deny s3:ListBucket when max-keys > 1000",
		},
		{
			name: "deny_numericgreaterthan_allows_when_condition_not_triggered",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:ListBucket"}, Resource: []string{"arn:aws:s3:::my-bucket"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Deny",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericGreaterThan: {
								"s3:max-keys": "1000",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				// max-keys <= 1000 so deny condition is not triggered
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "100"})
			},
			expectedAllow: true,
			description:   "Deny NumericGreaterThan: should allow when max-keys <= 1000 (deny condition not triggered)",
		},

		// --- Multiple numeric conditions (AND logic) ---
		{
			name: "multiple_numeric_conditions_all_match_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericGreaterThanEquals: {
								"s3:max-keys": "1",
							},
							s3common.CondOpNumericLessThanEquals: {
								"s3:max-keys": "1000",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				assert.NotNil(t, allowStmt.Conditions[s3common.CondOpNumericGreaterThanEquals])
				assert.NotNil(t, allowStmt.Conditions[s3common.CondOpNumericLessThanEquals])
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "500"})
			},
			expectedAllow: true,
			description:   "Multiple numeric conditions (AND): 1 <= max-keys <= 1000 should allow for max-keys=500",
		},
		{
			name: "multiple_numeric_conditions_one_fails_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericGreaterThanEquals: {
								"s3:max-keys": "1",
							},
							s3common.CondOpNumericLessThanEquals: {
								"s3:max-keys": "1000",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				// Fails the LessThanEquals condition
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "2000"})
			},
			expectedAllow: false,
			description:   "Multiple numeric conditions (AND): 1 <= max-keys <= 1000 should deny for max-keys=2000",
		},

		// --- Non-numeric value in condition key ---
		{
			name: "numericequals_non_numeric_value_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericEquals: {
								"s3:max-keys": "100",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				// Non-numeric value supplied — evaluateNumericCondition should fail gracefully
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "not-a-number"})
			},
			expectedAllow: false,
			description:   "NumericEquals: non-numeric actual value should deny (parse error treated as mismatch)",
		},

		// --- NumericGreaterThanEqualsIfExists ---
		{
			name: "numericgreaterthanequals_ifexists_key_absent_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpNumericGreaterThanEquals + s3common.CondOpIfExistsSuffix: {"s3:max-keys": "1"},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "NumericGreaterThanEqualsIfExists: key absent → condition passes vacuously → allow",
		},
		{
			name: "numericgreaterthanequals_ifexists_key_present_at_threshold_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpNumericGreaterThanEquals + s3common.CondOpIfExistsSuffix: {"s3:max-keys": "10"},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"max-keys": "10"})
			},
			expectedAllow: true,
			description:   "NumericGreaterThanEqualsIfExists: key present and value == threshold (inclusive) → allow",
		},

		// --- NumericNotEqualsIfExists ---
		{
			name: "numericnotequals_ifexists_key_absent_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpNumericNotEquals + s3common.CondOpIfExistsSuffix: {"s3:max-keys": "0"},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "NumericNotEqualsIfExists: key absent → condition passes vacuously → allow",
		},
		{
			name: "numericnotequals_ifexists_key_present_different_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpNumericNotEquals + s3common.CondOpIfExistsSuffix: {"s3:max-keys": "0"},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"max-keys": "100"})
			},
			expectedAllow: true,
			description:   "NumericNotEqualsIfExists: key present and value != 0 → allow",
		},
		{
			name: "numericnotequals_ifexists_key_present_same_value_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:ListBucket"},
				Resource: []string{"arn:aws:s3:::my-bucket"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpNumericNotEquals + s3common.CondOpIfExistsSuffix: {"s3:max-keys": "0"},
				},
			}}}},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, map[string]string{"max-keys": "0"})
			},
			expectedAllow: false,
			description:   "NumericNotEqualsIfExists: key present and value == 0 → deny",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}

			authz := engine.calculateAuthorizedActions(tt.policies)

			if tt.validateAuthz != nil {
				tt.validateAuthz(t, &authz)
			}

			req := tt.buildRequest()

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode, "Error should be AccessDenied")
			}
		})
	}
}

// Test Group 5.3: Date/Time Conditions
func TestDateTimeConditions(t *testing.T) {
	const (
		// Fixed reference timestamps used across tests
		pastTime    = "2020-01-01T00:00:00Z"
		presentTime = "2025-06-01T12:00:00Z"
		futureTime  = "2099-12-31T23:59:59Z"
	)

	tests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		resource      string
		buildRequest  func() *http.Request
		validateAuthz func(t *testing.T, authz *authorizedActions)
		expectedAllow bool
		description   string
	}{
		// --- DateEquals ---
		{
			name: "dateequals_different_timestamp_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpDateEquals: {
								"aws:CurrentTime": presentTime,
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasGet := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGet)
				condVals := allowStmt.Conditions[s3common.CondOpDateEquals]["aws:CurrentTime"]
				assert.Equal(t, []string{presentTime}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": pastTime}, nil)
			},
			expectedAllow: false,
			description:   "DateEquals: should deny when aws:CurrentTime does not match",
		},

		// --- DateNotEquals ---
		{
			name: "datenotequals_different_timestamp_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpDateNotEquals: {
								"aws:CurrentTime": pastTime,
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: true,
			description:   "DateNotEquals: should allow when aws:CurrentTime differs from the policy value",
		},

		// --- DateLessThan ---
		{
			name: "datelessthan_before_expiry_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpDateLessThan: {
								"aws:CurrentTime": futureTime,
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				condVals := allowStmt.Conditions[s3common.CondOpDateLessThan]["aws:CurrentTime"]
				assert.Equal(t, []string{futureTime}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// presentTime < futureTime → condition passes
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: true,
			description:   "DateLessThan: should allow when current time is before the expiry",
		},
		{
			name: "datelessthan_at_expiry_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpDateLessThan: {
								"aws:CurrentTime": presentTime,
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// Equal to expiry — DateLessThan is strict
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: false,
			description:   "DateLessThan: should deny when current time equals the expiry (not strictly less)",
		},
		{
			name: "datelessthan_after_expiry_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpDateLessThan: {
								"aws:CurrentTime": pastTime,
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// presentTime > pastTime → condition fails
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: false,
			description:   "DateLessThan: should deny when current time is after the expiry",
		},

		// --- DateLessThanEquals ---
		{
			name: "datelessthanequals_before_expiry_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpDateLessThanEquals: {
								"aws:CurrentTime": futureTime,
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: true,
			description:   "DateLessThanEquals: should allow when current time is before expiry",
		},
		{
			name: "datelessthanequals_after_expiry_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpDateLessThanEquals: {
								"aws:CurrentTime": pastTime,
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: false,
			description:   "DateLessThanEquals: should deny when current time is after the expiry",
		},

		// --- DateGreaterThan ---
		{
			name: "dategreaterthan_after_start_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpDateGreaterThan: {
								"aws:CurrentTime": pastTime,
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				condVals := allowStmt.Conditions[s3common.CondOpDateGreaterThan]["aws:CurrentTime"]
				assert.Equal(t, []string{pastTime}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// presentTime > pastTime → condition passes
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: true,
			description:   "DateGreaterThan: should allow when current time is after the start time",
		},
		{
			name: "dategreaterthan_before_start_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpDateGreaterThan: {
								"aws:CurrentTime": futureTime,
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// presentTime < futureTime → condition fails
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: false,
			description:   "DateGreaterThan: should deny when current time is before the start time",
		},

		// --- DateGreaterThanEquals ---
		{
			name: "dategreaterthanequals_at_start_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpDateGreaterThanEquals: {
								"aws:CurrentTime": presentTime,
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// Equal to start — DateGreaterThanEquals is inclusive
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: true,
			description:   "DateGreaterThanEquals: should allow when current time equals start time (inclusive)",
		},
		{
			name: "dategreaterthanequals_after_start_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpDateGreaterThanEquals: {
								"aws:CurrentTime": pastTime,
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: true,
			description:   "DateGreaterThanEquals: should allow when current time is after start time",
		},
		{
			name: "dategreaterthanequals_before_start_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpDateGreaterThanEquals: {
								"aws:CurrentTime": futureTime,
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: false,
			description:   "DateGreaterThanEquals: should deny when current time is before start time",
		},

		// --- Time window: DateGreaterThan + DateLessThan (AND logic) ---
		{
			name: "time_window_both_conditions_met_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpDateGreaterThan: {
								"aws:CurrentTime": pastTime,
							},
							s3common.CondOpDateLessThan: {
								"aws:CurrentTime": futureTime,
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				// Both condition operators must be stored
				assert.NotNil(t, allowStmt.Conditions[s3common.CondOpDateGreaterThan])
				assert.NotNil(t, allowStmt.Conditions[s3common.CondOpDateLessThan])
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// presentTime is within the window [pastTime, futureTime]
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: true,
			description:   "Date window (AND): pastTime < currentTime < futureTime should allow",
		},
		{
			name: "time_window_before_start_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpDateGreaterThan: {
								"aws:CurrentTime": utils.ConvertTimeToString(time.Now().Add(time.Hour)), // Set window start to 1 hour ago
							},
							s3common.CondOpDateLessThan: {
								"aws:CurrentTime": futureTime,
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// pastTime is before the window start presentTime
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": pastTime}, nil)
			},
			expectedAllow: false,
			description:   "Date window (AND): time before window start should deny",
		},
		{
			name: "time_window_after_end_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpDateGreaterThan: {
								"aws:CurrentTime": pastTime,
							},
							s3common.CondOpDateLessThan: {
								"aws:CurrentTime": utils.ConvertTimeToString(time.Now().Add(-time.Hour)), // Set window end to 1 hour ago
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// futureTime is after the window end presentTime
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": futureTime}, nil)
			},
			expectedAllow: false,
			description:   "Date window (AND): time after window end should deny",
		},

		// --- DateEqualsIfExists ---
		{
			name: "dateequals_ifexists_key_present_mismatch_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpDateEquals + s3common.CondOpIfExistsSuffix: {"aws:CurrentTime": presentTime},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": pastTime}, nil)
			},
			expectedAllow: false,
			description:   "DateEqualsIfExists: key present but value mismatches → deny",
		},

		// --- DateNotEqualsIfExists ---
		{
			name: "datenotequals_ifexists_key_absent_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpDateNotEquals + s3common.CondOpIfExistsSuffix: {"aws:CurrentTime": pastTime},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "DateNotEqualsIfExists: key absent → condition passes vacuously → allow",
		},
		{
			name: "datenotequals_ifexists_key_present_different_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpDateNotEquals + s3common.CondOpIfExistsSuffix: {"aws:CurrentTime": pastTime},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: true,
			description:   "DateNotEqualsIfExists: key present and value != expected → allow",
		},

		// --- DateLessThanIfExists ---
		{
			name: "datelessthan_ifexists_key_absent_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpDateLessThan + s3common.CondOpIfExistsSuffix: {"aws:CurrentTime": futureTime},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "DateLessThanIfExists: key absent → condition passes vacuously → allow",
		},
		{
			name: "datelessthan_ifexists_key_present_before_expiry_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpDateLessThan + s3common.CondOpIfExistsSuffix: {"aws:CurrentTime": futureTime},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: true,
			description:   "DateLessThanIfExists: key present and time < expiry → allow",
		},
		{
			name: "datelessthan_ifexists_key_present_after_expiry_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpDateLessThan + s3common.CondOpIfExistsSuffix: {"aws:CurrentTime": pastTime},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: false,
			description:   "DateLessThanIfExists: key present and time >= expiry → deny",
		},

		// --- DateLessThanEqualsIfExists ---
		{
			name: "datelessthanequals_ifexists_key_absent_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpDateLessThanEquals + s3common.CondOpIfExistsSuffix: {"aws:CurrentTime": futureTime},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "DateLessThanEqualsIfExists: key absent → condition passes vacuously → allow",
		},
		{
			name: "datelessthanequals_ifexists_key_present_after_expiry_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpDateLessThanEquals + s3common.CondOpIfExistsSuffix: {"aws:CurrentTime": pastTime},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: false,
			description:   "DateLessThanEqualsIfExists: key present and time > expiry → deny",
		},

		// --- DateGreaterThanIfExists ---
		{
			name: "dategreaterthan_ifexists_key_absent_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpDateGreaterThan + s3common.CondOpIfExistsSuffix: {"aws:CurrentTime": pastTime},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "DateGreaterThanIfExists: key absent → condition passes vacuously → allow",
		},
		{
			name: "dategreaterthan_ifexists_key_present_after_start_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpDateGreaterThan + s3common.CondOpIfExistsSuffix: {"aws:CurrentTime": pastTime},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: true,
			description:   "DateGreaterThanIfExists: key present and time > start → allow",
		},
		{
			name: "dategreaterthan_ifexists_key_present_before_start_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpDateGreaterThan + s3common.CondOpIfExistsSuffix: {"aws:CurrentTime": futureTime},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: false,
			description:   "DateGreaterThanIfExists: key present and time < start → deny",
		},

		// --- DateGreaterThanEqualsIfExists ---
		{
			name: "dategreaterthanequals_ifexists_key_absent_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpDateGreaterThanEquals + s3common.CondOpIfExistsSuffix: {"aws:CurrentTime": pastTime},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "DateGreaterThanEqualsIfExists: key absent → condition passes vacuously → allow",
		},
		{
			name: "dategreaterthanequals_ifexists_key_present_at_start_allows",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpDateGreaterThanEquals + s3common.CondOpIfExistsSuffix: {"aws:CurrentTime": presentTime},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: true,
			description:   "DateGreaterThanEqualsIfExists: key present and time == start (inclusive) → allow",
		},
		{
			name: "dategreaterthanequals_ifexists_key_present_before_start_denies",
			policies: []PolicyDocument{{Statement: []*Statement{{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpDateGreaterThanEquals + s3common.CondOpIfExistsSuffix: {"aws:CurrentTime": futureTime},
				},
			}}}},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil,
					map[string]string{"aws:CurrentTime": presentTime}, nil)
			},
			expectedAllow: false,
			description:   "DateGreaterThanEqualsIfExists: key present and time < start → deny",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}

			authz := engine.calculateAuthorizedActions(tt.policies)

			if tt.validateAuthz != nil {
				tt.validateAuthz(t, &authz)
			}

			req := tt.buildRequest()

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode, "Error should be AccessDenied")
			}
		})
	}
}

// Test Group 5.4: IP Address Conditions
func TestIPAddressConditions(t *testing.T) {
	tests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		resource      string
		buildRequest  func() *http.Request
		validateAuthz func(t *testing.T, authz *authorizedActions)
		expectedAllow bool
		description   string
	}{
		// --- IpAddress: exact IP ---
		{
			name: "ipaddress_exact_ip_matches_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpIpAddress: {
								"aws:SourceIp": "203.0.113.10",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasGet := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGet)
				condVals := allowStmt.Conditions[s3common.CondOpIpAddress]["aws:SourceIp"]
				assert.Equal(t, []string{"203.0.113.10"}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.RemoteAddr = "203.0.113.10:12345" // Simulate client IP in RemoteAddr as well
				return req
			},
			expectedAllow: true,
			description:   "IpAddress: exact IP match should allow",
		},
		{
			name: "ipaddress_exact_ip_mismatch_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpIpAddress: {
								"aws:SourceIp": "203.0.113.10",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.RemoteAddr = "203.0.113.99:12345" // Simulate client IP in RemoteAddr as well
				return req
			},
			expectedAllow: false,
			description:   "IpAddress: non-matching exact IP should deny",
		},

		// --- IpAddress: CIDR range ---
		{
			name: "ipaddress_cidr_range_matches_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpIpAddress: {
								"aws:SourceIp": "203.0.113.0/24",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				condVals := allowStmt.Conditions[s3common.CondOpIpAddress]["aws:SourceIp"]
				assert.Equal(t, []string{"203.0.113.0/24"}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// 203.0.113.42 is within 203.0.113.0/24
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.RemoteAddr = "203.0.113.42:12345" // Simulate client IP in RemoteAddr as well
				return req
			},
			expectedAllow: true,
			description:   "IpAddress: IP within CIDR range should allow",
		},
		{
			name: "ipaddress_cidr_range_outside_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpIpAddress: {
								"aws:SourceIp": "203.0.113.0/24",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// 10.0.0.1 is outside 203.0.113.0/24
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.RemoteAddr = "10.0.0.1:12345"
				return req
			},
			expectedAllow: false,
			description:   "IpAddress: IP outside CIDR range should deny",
		},
		{
			name: "ipaddress_cidr_boundary_ip_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpIpAddress: {
								"aws:SourceIp": "203.0.113.0/24",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// Network address itself is within range
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.RemoteAddr = "203.0.113.0:12345"
				return req
			},
			expectedAllow: true,
			description:   "IpAddress: network address of CIDR should be within range and allow",
		},

		// --- NotIpAddress ---
		{
			name: "notipaddress_ip_outside_blocked_range_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNotIpAddress: {
								"aws:SourceIp": "10.0.0.0/8",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				condVals := allowStmt.Conditions[s3common.CondOpNotIpAddress]["aws:SourceIp"]
				assert.Equal(t, []string{"10.0.0.0/8"}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// 203.0.113.10 is NOT in 10.0.0.0/8 → NotIpAddress passes
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.RemoteAddr = "203.0.113.10:12345"
				return req
			},
			expectedAllow: true,
			description:   "NotIpAddress: IP outside the blocked CIDR range should allow",
		},
		{
			name: "notipaddress_ip_inside_blocked_range_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNotIpAddress: {
								"aws:SourceIp": "10.0.0.0/8",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// 10.0.0.5 IS in 10.0.0.0/8 → NotIpAddress fails
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.RemoteAddr = "10.0.0.5:12345"
				return req
			},
			expectedAllow: false,
			description:   "NotIpAddress: IP inside the blocked CIDR range should deny",
		},

		// --- IpAddress missing / invalid ---
		{
			name: "ipaddress_missing_source_ip_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpIpAddress: {
								"aws:SourceIp": "203.0.113.0/24",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// No IP headers at all
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: false,
			description:   "IpAddress: missing source IP should deny (empty value cannot match CIDR)",
		},
		{
			name: "ipaddress_invalid_ip_format_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpIpAddress: {
								"aws:SourceIp": "203.0.113.0/24",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.RemoteAddr = "not-an-ip"
				return req
			},
			expectedAllow: false,
			description:   "IpAddress: non-parseable IP should deny",
		},

		// --- IpAddress: IfExists suffix ---
		{
			name: "ipaddress_ifexists_key_present_matching_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpIpAddress + s3common.CondOpIfExistsSuffix: {
								"aws:SourceIp": "203.0.113.0/24",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				condKey := s3common.CondOpIpAddress + s3common.CondOpIfExistsSuffix
				condVals := allowStmt.Conditions[condKey]["aws:SourceIp"]
				assert.Equal(t, []string{"203.0.113.0/24"}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// Key present and IP is within range → condition evaluates normally
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.RemoteAddr = "203.0.113.10:12345"
				return req
			},
			expectedAllow: true,
			description:   "IpAddressIfExists: key present and IP matches CIDR → allow",
		},
		{
			name: "ipaddress_ifexists_key_absent_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpIpAddress + s3common.CondOpIfExistsSuffix: {
								"aws:SourceIp": "203.0.113.0/24",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// No IP headers → key absent → IfExists passes automatically
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "IpAddressIfExists: key absent should pass condition and allow (IfExists semantics)",
		},
		{
			name: "ipaddress_ifexists_key_present_not_matching_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpIpAddress + s3common.CondOpIfExistsSuffix: {
								"aws:SourceIp": "203.0.113.0/24",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// Key present but IP is outside range → condition evaluates normally and fails
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.RemoteAddr = "10.0.0.1:12345"
				return req
			},
			expectedAllow: false,
			description:   "IpAddressIfExists: key present but IP not in CIDR → condition fails → deny",
		},

		// --- Deny with IpAddress condition ---
		{
			name: "deny_ipaddress_cidr_blocks_matching_ip",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Deny",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpIpAddress: {
								"aws:SourceIp": "10.0.0.0/8",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// IP is within the deny range
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.RemoteAddr = "10.1.2.3:12345"
				return req
			},
			expectedAllow: false,
			description:   "Deny IpAddress: IP within CIDR triggers deny override",
		},
		{
			name: "deny_ipaddress_cidr_allows_non_matching_ip",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Deny",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpIpAddress: {
								"aws:SourceIp": "10.0.0.0/8",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// IP is outside the deny range → deny condition not triggered
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.RemoteAddr = "203.0.113.10:12345"
				return req
			},
			expectedAllow: true,
			description:   "Deny IpAddress: IP outside CIDR does not trigger deny → allow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}

			authz := engine.calculateAuthorizedActions(tt.policies)

			if tt.validateAuthz != nil {
				tt.validateAuthz(t, &authz)
			}

			req := tt.buildRequest()

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode, "Error should be AccessDenied")
			}
		})
	}
}

// Test Group 5.5: Boolean Conditions
func TestBooleanConditions(t *testing.T) {
	tests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		resource      string
		buildRequest  func() *http.Request
		validateAuthz func(t *testing.T, authz *authorizedActions)
		expectedAllow bool
		description   string
	}{
		// --- Bool: secure transport ---
		{
			name: "bool_securetransport_true_tls_request_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpBool: {
								"aws:SecureTransport": "true",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasGet := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGet)
				condVals := allowStmt.Conditions[s3common.CondOpBool]["aws:SecureTransport"]
				assert.Equal(t, []string{"true"}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.TLS = &tls.ConnectionState{} // Non-nil TLS field indicates a TLS request
				return req
			},
			expectedAllow: true,
			description:   "Bool: aws:SecureTransport=true should allow when request indicates TLS",
		},
		{
			name: "bool_securetransport_true_non_tls_request_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpBool: {
								"aws:SecureTransport": "true",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// No TLS header → getConditionValueFromRequest returns "false"
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: false,
			description:   "Bool: aws:SecureTransport=true should deny when request is not TLS",
		},
		{
			name: "bool_securetransport_false_non_tls_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpBool: {
								"aws:SecureTransport": "false",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// No TLS → "false" → matches condition
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "Bool: aws:SecureTransport=false should allow when request is not TLS",
		},

		// --- Bool: IfExists suffix ---
		{
			name: "bool_ifexists_key_present_matching_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpBool + s3common.CondOpIfExistsSuffix: {
								"aws:SecureTransport": "true",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				condKey := s3common.CondOpBool + s3common.CondOpIfExistsSuffix
				condVals := allowStmt.Conditions[condKey]["aws:SecureTransport"]
				assert.Equal(t, []string{"true"}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.TLS = &tls.ConnectionState{} // Non-nil TLS field indicates a TLS request → key present and value is "true"
				return req
			},
			expectedAllow: true,
			description:   "BoolIfExists: key present and value matches → allow",
		},
		{
			name: "bool_ifexists_key_absent_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpBool + s3common.CondOpIfExistsSuffix: {
								// Use a custom key that won't be present
								"aws:CustomBoolKey": "true",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// aws:CustomBoolKey not set → key absent → IfExists passes
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "BoolIfExists: key absent should pass condition and allow (IfExists semantics)",
		},
		{
			name: "bool_ifexists_key_present_not_matching_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpBool + s3common.CondOpIfExistsSuffix: {
								"aws:SecureTransport": "true",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// No TLS → returns "false" → key present but value is "false", not "true"
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: false,
			description:   "BoolIfExists: key present but value does not match → deny",
		},

		// --- Deny with Bool condition ---
		{
			name: "deny_bool_non_tls_blocks_request",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Deny",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpBool: {
								"aws:SecureTransport": "false",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// No TLS → "false" → deny condition matches
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: false,
			description:   "Deny Bool: non-TLS request should be denied when Deny condition is aws:SecureTransport=false",
		},
		{
			name: "deny_bool_tls_request_not_blocked",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Deny",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpBool: {
								"aws:SecureTransport": "false",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
				req.TLS = &tls.ConnectionState{} // Non-nil TLS field indicates a TLS request
				return req
			},
			expectedAllow: true,
			description:   "Deny Bool: TLS request should not be blocked (deny condition is for false only)",
		},

		// --- ForAllValues:Bool ---
		{
			name: "forallvalues_bool_all_values_match_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							// ForAllValues: every value in the request set must match
							s3common.CondOpForAllValuesPrefix + s3common.CondOpBool: {
								"aws:SecureTransport": "false",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				condKey := s3common.PolicyCondOp(s3common.CondOpForAllValuesPrefix + string(s3common.CondOpBool))
				condVals := allowStmt.Conditions[condKey]["aws:SecureTransport"]
				assert.Equal(t, []string{"false"}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// Non-TLS → "false" → ForAllValues passes since single value matches
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "ForAllValues:Bool: single value matches expected → allow",
		},
		{
			name: "forallvalues_bool_value_not_matching_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpForAllValuesPrefix + s3common.CondOpBool: {
								"aws:SecureTransport": "true",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// Non-TLS → "false" → does not match "true"
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: false,
			description:   "ForAllValues:Bool: value does not match expected → deny",
		},

		// --- ForAnyValue:Bool ---
		{
			name: "foranyvalue_bool_at_least_one_matches_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpForAnyValuePrefix + s3common.CondOpBool: {
								// Accept either true or false (at least one must match)
								"aws:SecureTransport": "false",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				condKey := s3common.PolicyCondOp(s3common.CondOpForAnyValuePrefix + string(s3common.CondOpBool))
				condVals := allowStmt.Conditions[condKey]["aws:SecureTransport"]
				assert.Equal(t, []string{"false"}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// Non-TLS → "false" → at least one value matches
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "ForAnyValue:Bool: value matches at least one expected → allow",
		},
		{
			name: "foranyvalue_bool_none_matches_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpForAnyValuePrefix + s3common.CondOpBool: {
								"aws:SecureTransport": "true",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// Non-TLS → "false" → no match against "true"
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: false,
			description:   "ForAnyValue:Bool: no value matches any expected → deny",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}

			authz := engine.calculateAuthorizedActions(tt.policies)

			if tt.validateAuthz != nil {
				tt.validateAuthz(t, &authz)
			}

			req := tt.buildRequest()

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode, "Error should be AccessDenied")
			}
		})
	}
}

// Test Group 5.6: ARN Conditions
func TestARNConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test as this feature yet to implement")
	}

	tests := []struct {
		name          string
		policies      []PolicyDocument
		validateAuthz func(t *testing.T, authz *authorizedActions)
		action        s3common.PolicyAction
		resource      string
		buildRequest  func() *http.Request
		actualUserArn string
		expectedAllow bool
		description   string
	}{
		// --- ArnEquals ---
		{
			name: "arnequals_exact_arn_matches_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpArnEquals: {
								"aws:PrincipalArn": "arn:aws:iam::123456789012:user/alice",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasGet := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGet)
				condVals := allowStmt.Conditions[s3common.CondOpArnEquals]["aws:PrincipalArn"]
				assert.Equal(t, []string{"arn:aws:iam::123456789012:user/alice"}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "arn:aws:iam::123456789012:user/alice",
			expectedAllow: true,
			description:   "ArnEquals: exact ARN match should allow",
		},
		{
			name: "arnequals_different_arn_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpArnEquals: {
								"aws:PrincipalArn": "arn:aws:iam::123456789012:user/alice",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "arn:aws:iam::123456789012:user/bob",
			expectedAllow: false,
			description:   "ArnEquals: different ARN should deny",
		},

		// --- ArnNotEquals ---
		{
			name: "arnnotequals_different_arn_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpArnNotEquals: {
								"aws:PrincipalArn": "arn:aws:iam::123456789012:user/alice",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				condVals := allowStmt.Conditions[s3common.CondOpArnNotEquals]["aws:PrincipalArn"]
				assert.Equal(t, []string{"arn:aws:iam::123456789012:user/alice"}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "arn:aws:iam::123456789012:user/bob",
			expectedAllow: true,
			description:   "ArnNotEquals: different ARN should allow",
		},
		{
			name: "arnnotequals_same_arn_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpArnNotEquals: {
								"aws:PrincipalArn": "arn:aws:iam::123456789012:user/alice",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "arn:aws:iam::123456789012:user/alice",
			expectedAllow: false,
			description:   "ArnNotEquals: same ARN should deny",
		},

		// --- ArnLike: wildcard matching ---
		{
			name: "arnlike_wildcard_prefix_matches_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpArnLike: {
								"aws:PrincipalArn": "arn:aws:iam::123456789012:user/*",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				condVals := allowStmt.Conditions[s3common.CondOpArnLike]["aws:PrincipalArn"]
				assert.Equal(t, []string{"arn:aws:iam::123456789012:user/*"}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "arn:aws:iam::123456789012:user/alice",
			expectedAllow: true,
			description:   "ArnLike: wildcard user/* should match any specific user ARN",
		},
		{
			name: "arnlike_wildcard_no_match_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpArnLike: {
								"aws:PrincipalArn": "arn:aws:iam::123456789012:user/*",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// Role ARN should not match user/* pattern
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "arn:aws:iam::123456789012:role/my-role",
			expectedAllow: false,
			description:   "ArnLike: role ARN should not match user/* wildcard pattern",
		},
		{
			name: "arnlike_single_char_wildcard_matches_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpArnLike: {
								// Match any single-char username
								"aws:PrincipalArn": "arn:aws:iam::123456789012:user/?",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "arn:aws:iam::123456789012:user/a",
			expectedAllow: true,
			description:   "ArnLike: single char ? wildcard should match a single char username",
		},

		// --- ArnNotLike ---
		{
			name: "arnnotlike_non_matching_pattern_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpArnNotLike: {
								"aws:PrincipalArn": "arn:aws:iam::123456789012:role/*",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				condVals := allowStmt.Conditions[s3common.CondOpArnNotLike]["aws:PrincipalArn"]
				assert.Equal(t, []string{"arn:aws:iam::123456789012:role/*"}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// User ARN does not match role/* pattern → ArnNotLike passes
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "arn:aws:iam::123456789012:user/alice",
			expectedAllow: true,
			description:   "ArnNotLike: ARN not matching blocked pattern should allow",
		},
		{
			name: "arnnotlike_matching_pattern_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpArnNotLike: {
								"aws:PrincipalArn": "arn:aws:iam::123456789012:role/*",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// Role ARN matches role/* → ArnNotLike fails
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "arn:aws:iam::123456789012:role/my-role",
			expectedAllow: false,
			description:   "ArnNotLike: ARN matching the blocked pattern should deny",
		},

		// --- ARN IfExists suffix ---
		{
			name: "arnlike_ifexists_key_absent_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpArnLike + s3common.CondOpIfExistsSuffix: {
								"aws:PrincipalArn": "arn:aws:iam::123456789012:user/*",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				condKey := s3common.CondOpArnLike + s3common.CondOpIfExistsSuffix
				condVals := allowStmt.Conditions[condKey]["aws:PrincipalArn"]
				assert.Equal(t, []string{"arn:aws:iam::123456789012:user/*"}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// No PrincipalArn header → key absent → IfExists passes
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "",
			expectedAllow: true,
			description:   "ArnLikeIfExists: key absent should pass condition and allow",
		},
		{
			name: "arnlike_ifexists_key_present_matching_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpArnLike + s3common.CondOpIfExistsSuffix: {
								"aws:PrincipalArn": "arn:aws:iam::123456789012:user/*",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// Key present and matches pattern → condition passes
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "arn:aws:iam::123456789012:user/alice",
			expectedAllow: true,
			description:   "ArnLikeIfExists: key present and matches wildcard pattern → allow",
		},
		{
			name: "arnlike_ifexists_key_present_not_matching_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpArnLike + s3common.CondOpIfExistsSuffix: {
								"aws:PrincipalArn": "arn:aws:iam::123456789012:user/*",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// Key present but role ARN doesn't match user/* → condition fails
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "arn:aws:iam::123456789012:role/my-role",
			expectedAllow: false,
			description:   "ArnLikeIfExists: key present but does not match pattern → deny",
		},
		{
			name: "arnequals_ifexists_key_absent_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpArnEquals + s3common.CondOpIfExistsSuffix: {
								"aws:PrincipalArn": "arn:aws:iam::123456789012:user/alice",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// Key absent → IfExists passes
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "",
			expectedAllow: true,
			description:   "ArnEqualsIfExists: key absent should pass condition and allow",
		},

		// --- ForAllValues:ArnLike ---
		{
			name: "forallvalues_arnlike_all_arns_match_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpForAllValuesPrefix + s3common.CondOpArnLike: {
								"aws:PrincipalArn": "arn:aws:iam::123456789012:user/*",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				condKey := s3common.PolicyCondOp(s3common.CondOpForAllValuesPrefix + string(s3common.CondOpArnLike))
				condVals := allowStmt.Conditions[condKey]["aws:PrincipalArn"]
				assert.Equal(t, []string{"arn:aws:iam::123456789012:user/*"}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "arn:aws:iam::123456789012:user/alice",
			expectedAllow: true,
			description:   "ForAllValues:ArnLike: single matching ARN → allow",
		},
		{
			name: "forallvalues_arnlike_arn_not_matching_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpForAllValuesPrefix + s3common.CondOpArnLike: {
								"aws:PrincipalArn": "arn:aws:iam::123456789012:user/*",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "arn:aws:iam::123456789012:role/bad",
			expectedAllow: false,
			description:   "ForAllValues:ArnLike: ARN set not matching → deny",
		},

		// --- ForAnyValue:ArnLike ---
		{
			name: "foranyvalue_arnlike_at_least_one_matches_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpForAnyValuePrefix + s3common.CondOpArnLike: {
								"aws:PrincipalArn": "arn:aws:iam::123456789012:user/*",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				condKey := s3common.PolicyCondOp(s3common.CondOpForAnyValuePrefix + string(s3common.CondOpArnLike))
				condVals := allowStmt.Conditions[condKey]["aws:PrincipalArn"]
				assert.Equal(t, []string{"arn:aws:iam::123456789012:user/*"}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "arn:aws:iam::123456789012:user/alice",
			expectedAllow: true,
			description:   "ForAnyValue:ArnLike: User ARN matches user/* pattern → allow",
		},
		{
			name: "foranyvalue_arnlike_none_matching_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpForAnyValuePrefix + s3common.CondOpArnLike: {
								"aws:PrincipalArn": "arn:aws:iam::123456789012:user/*",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "arn:aws:iam::123456789012:role/r1",
			expectedAllow: false,
			description:   "ForAnyValue:ArnLike: User ARN does not match user/* pattern → deny",
		},

		// --- Deny with ARN condition ---
		{
			name: "deny_arnequals_blocks_specific_arn",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Deny",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpArnEquals: {
								"aws:PrincipalArn": "arn:aws:iam::123456789012:user/blocked-user",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "arn:aws:iam::123456789012:user/blocked-user",
			expectedAllow: false,
			description:   "Deny ArnEquals: blocked user ARN should be denied",
		},
		{
			name: "deny_arnequals_allows_other_arn",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Deny",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpArnEquals: {
								"aws:PrincipalArn": "arn:aws:iam::123456789012:user/blocked-user",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			actualUserArn: "arn:aws:iam::123456789012:user/alice",
			expectedAllow: true,
			description:   "Deny ArnEquals: non-blocked ARN should allow (deny condition not triggered)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalGetPrinicipalArn := getPrinicipalArn
			defer func() { getPrinicipalArn = originalGetPrinicipalArn }()
			getPrinicipalArn = func(req http.Request) string {
				return tt.actualUserArn
			}

			engine := &AuthzEngine{}
			authz := engine.calculateAuthorizedActions(tt.policies)
			if tt.validateAuthz != nil {
				tt.validateAuthz(t, &authz)
			}

			req := tt.buildRequest()

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode, "Error should be AccessDenied")
			}
		})
	}
}

// Test Group 5.7: Null Conditions
func TestNullConditions(t *testing.T) {
	tests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		resource      string
		buildRequest  func() *http.Request
		validateAuthz func(t *testing.T, authz *authorizedActions)
		expectedAllow bool
		description   string
	}{
		// --- Null: true (key must be absent) ---
		{
			name: "null_true_key_absent_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNull: {
								// Allow only when s3:prefix is NOT provided
								"s3:prefix": "true",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasGet := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, hasGet)
				condVals := allowStmt.Conditions[s3common.CondOpNull]["s3:prefix"]
				assert.Equal(t, []string{"true"}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// No prefix query param → key is absent → Null=true passes
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "Null=true: key absent should pass condition and allow",
		},
		{
			name: "null_true_key_present_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNull: {
								"s3:prefix": "true",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// prefix is present → key is not null → Null=true fails
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil,
					map[string]string{"prefix": "home/"})
			},
			expectedAllow: false,
			description:   "Null=true: key present should fail condition and deny",
		},

		// --- Null: false (key must be present) ---
		{
			name: "null_false_key_present_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNull: {
								// Allow only when s3:prefix IS provided
								"s3:prefix": "false",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasListBucket := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3ListBucket)]
				assert.True(t, hasListBucket)
				condVals := allowStmt.Conditions[s3common.CondOpNull]["s3:prefix"]
				assert.Equal(t, []string{"false"}, condVals)
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				// prefix is present → key is not null → Null=false passes
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"prefix": "home/"})
			},
			expectedAllow: true,
			description:   "Null=false: key present should pass condition and allow",
		},
		{
			name: "null_false_key_absent_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNull: {
								"s3:prefix": "false",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				// No prefix → key is null → Null=false fails
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil, nil)
			},
			expectedAllow: false,
			description:   "Null=false: key absent should fail condition and deny",
		},

		// --- Null with header-based key ---
		{
			name: "null_true_header_absent_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:PutObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNull: {
								// Allow only when no SSE header is present
								"s3:x-amz-server-side-encryption": "true",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// No SSE header → key is null → Null=true passes
				return testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "Null=true: SSE header absent should pass condition and allow",
		},
		{
			name: "null_true_header_present_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:PutObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNull: {
								"s3:x-amz-server-side-encryption": "true",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// SSE header present → key is not null → Null=true fails
				return testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt",
					map[string]string{"x-amz-server-side-encryption": "AES256"}, nil, nil, nil)
			},
			expectedAllow: false,
			description:   "Null=true: SSE header present should fail condition and deny",
		},
		{
			name: "null_false_header_present_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:PutObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNull: {
								// Require SSE header to be present
								"s3:x-amz-server-side-encryption": "false",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// SSE header present → not null → Null=false passes
				return testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt",
					map[string]string{"x-amz-server-side-encryption": "AES256"}, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "Null=false: SSE header present should pass condition and allow",
		},

		// --- Null combined with other conditions (AND) ---
		{
			name: "null_combined_with_string_condition_both_pass_allows",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:PutObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNull: {
								// SSE key must be present (not null)
								"s3:x-amz-server-side-encryption": "false",
							},
							s3common.CondOpStringEquals: {
								// AND SSE value must be AES256
								"s3:x-amz-server-side-encryption": "AES256",
							},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				// Both condition operators must be stored
				assert.NotNil(t, allowStmt.Conditions[s3common.CondOpNull])
				assert.NotNil(t, allowStmt.Conditions[s3common.CondOpStringEquals])
			},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// SSE present and value is AES256 → both conditions pass
				return testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt",
					map[string]string{"x-amz-server-side-encryption": "AES256"}, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "Null=false AND StringEquals: both conditions satisfied → allow",
		},
		{
			name: "null_combined_with_string_condition_null_fails_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:PutObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNull: {
								"s3:x-amz-server-side-encryption": "false",
							},
							s3common.CondOpStringEquals: {
								"s3:x-amz-server-side-encryption": "AES256",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// No SSE header → Null=false fails (key is null)
				return testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: false,
			description:   "Null=false AND StringEquals: SSE header absent fails Null condition → deny",
		},

		// --- Deny with Null condition ---
		{
			name: "deny_null_true_key_absent_triggers_deny",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:PutObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Deny",
						Action:   []string{"s3:PutObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNull: {
								// Deny when SSE header is absent
								"s3:x-amz-server-side-encryption": "true",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// No SSE header → key is null → Null=true passes → deny triggers
				return testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: false,
			description:   "Deny Null=true: SSE header absent triggers deny (enforce SSE on upload)",
		},
		{
			name: "deny_null_true_key_present_does_not_trigger_deny",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:PutObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Deny",
						Action:   []string{"s3:PutObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNull: {
								"s3:x-amz-server-side-encryption": "true",
							},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3PutObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// SSE header present → key is not null → Null=true fails → deny not triggered
				return testhelpers.NewMockRequest("PUT", "/my-bucket/file.txt",
					map[string]string{"x-amz-server-side-encryption": "AES256"}, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "Deny Null=true: SSE header present → deny condition fails → allow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}

			authz := engine.calculateAuthorizedActions(tt.policies)

			if tt.validateAuthz != nil {
				tt.validateAuthz(t, &authz)
			}

			req := tt.buildRequest()

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode, "Error should be AccessDenied")
			}
		})
	}
}

// Test Group 6.1: Specific Resource Statements
func TestSpecificResourceStatements(t *testing.T) {
	tests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		resource      string
		buildRequest  func() *http.Request
		validateAuthz func(t *testing.T, authz *authorizedActions)
		expectedAllow bool
		description   string
	}{
		// --- Exact resource match stored correctly in WithSpecificResourceStmt ---
		{
			name: "specific_resource_stored_in_correct_map_key",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/file.txt"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				bothAllAndNoResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/file.txt"
				assert.Contains(t, authz.WithSpecificResourceStmt, res)
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, ok := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, ok)
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "Specific resource must be stored under its exact ARN key in WithSpecificResourceStmt",
		},
		// --- Multiple specific resources each stored under their own key ---
		{
			name: "two_specific_resources_stored_as_separate_keys",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{
						"arn:aws:s3:::bucket-a/file.txt",
						"arn:aws:s3:::bucket-b/file.txt",
					}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				bothAllAndNoResShouldBeNil(t, authz)
				assert.Contains(t, authz.WithSpecificResourceStmt, "arn:aws:s3:::bucket-a/file.txt")
				assert.Contains(t, authz.WithSpecificResourceStmt, "arn:aws:s3:::bucket-b/file.txt")
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::bucket-a/file.txt",
			expectedAllow: true,
			description:   "Each specific resource in a statement must get its own map entry",
		},
		// --- Allow and Deny on same specific resource ---
		{
			name: "specific_resource_has_both_allow_and_deny_entries",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
					{Effect: "Deny", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				bothAllAndNoResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				denyStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectDeny]
				assert.NotNil(t, allowStmt)
				assert.NotNil(t, denyStmt)
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "Same specific resource with both Allow and Deny must have both effect entries; Deny wins",
		},
		// --- Wildcard resource pattern matching specific requested resource ---
		{
			name: "wildcard_resource_pattern_matches_specific_request",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:PutObject"}, Resource: []string{"arn:aws:s3:::my-bucket/uploads/*"}},
				}},
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/uploads/2024/report.pdf",
			expectedAllow: true,
			description:   "Wildcard resource pattern in WithSpecificResourceStmt must match a concrete requested resource",
		},
		// --- Two overlapping wildcard patterns, deny on narrower one ---
		{
			name: "narrower_deny_pattern_overrides_broader_allow_pattern",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
					{Effect: "Deny", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/private/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/private/secret.txt",
			expectedAllow: false,
			description:   "Narrower Deny pattern (private/*) must override broader Allow pattern (my-bucket/*)",
		},
		{
			name: "narrower_deny_pattern_does_not_affect_non_matching_path",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
					{Effect: "Deny", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/private/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/public/readme.txt",
			expectedAllow: true,
			description:   "Deny on private/* should not affect access to public/* (covered by broader Allow)",
		},
		// --- Conditions stored alongside actions in WithSpecificResourceStmt ---
		{
			name: "specific_resource_stmt_conditions_stored_correctly",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpStringEquals: {"s3:prefix": "home/"},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				bothAllAndNoResShouldBeNil(t, authz)
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				condVals := allowStmt.Conditions[s3common.CondOpStringEquals]["s3:prefix"]
				assert.Equal(t, []string{"home/"}, condVals)
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil,
					map[string]string{"prefix": "home/"})
			},
			expectedAllow: true,
			description:   "Conditions must be preserved when stored in WithSpecificResourceStmt",
		},
		// --- Merging actions from two Allow statements targeting same specific resource ---
		{
			name: "two_allow_stmts_same_resource_actions_merged",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
					{Effect: "Allow", Action: []string{"s3:PutObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				res := "arn:aws:s3:::my-bucket/*"
				allowStmt := authz.WithSpecificResourceStmt[res][s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasGet := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				_, hasPut := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3PutObject)]
				assert.True(t, hasGet, "GetObject must be merged into the allow statement")
				assert.True(t, hasPut, "PutObject must be merged into the allow statement")
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "Actions from two Allow statements targeting the same resource must be merged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}
			authz := engine.calculateAuthorizedActions(tt.policies)

			if tt.validateAuthz != nil {
				tt.validateAuthz(t, &authz)
			}

			var req *http.Request
			if tt.buildRequest != nil {
				req = tt.buildRequest()
			} else {
				req = testhelpers.NewMockRequest("GET", "/", nil, nil, nil, nil)
			}

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode)
			}
		})
	}
}

// Test Group 6.2: All Resources Statements
func TestAllResourcesStatements(t *testing.T) {
	tests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		buildRequest  func() *http.Request
		resource      string
		validateAuthz func(t *testing.T, authz *authorizedActions)
		expectedAllow bool
		description   string
	}{
		// --- Resource: * stored in AllResourcesStmt ---
		{
			name: "resource_star_stored_in_allresources_stmt",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: "*"},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				assert.Empty(t, authz.WithSpecificResourceStmt)
				allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, ok := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				assert.True(t, ok)
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::any-bucket/file.txt",
			expectedAllow: true,
			description:   "Resource=* must populate AllResourcesStmt, not WithSpecificResourceStmt",
		},
		// --- AllResourcesStmt Allow does not override specific resource Deny ---
		{
			name: "allresources_allow_does_not_override_specific_resource_deny",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: "*"},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::locked-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::locked-bucket/file.txt",
			expectedAllow: false,
			description:   "Specific resource Deny must override AllResourcesStmt Allow",
		},
		// --- AllResourcesStmt Deny blocks access even with specific resource Allow ---
		{
			name: "allresources_deny_blocks_specific_resource_allow",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::dev-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"s3:DeleteObject"}, Resource: "*"},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				denyStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectDeny]
				assert.NotNil(t, denyStmt)
				_, ok := denyStmt.Actions[normalizeAction(s3common.PolicyActionS3DeleteObject)]
				assert.True(t, ok)
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::dev-bucket/file.txt",
			expectedAllow: false,
			description:   "AllResourcesStmt Deny on Resource=* must block even a specific resource Allow",
		},
		// --- Actions from multiple Resource=* Allow statements are merged ---
		{
			name: "multiple_resource_star_allow_stmts_actions_merged",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: "*"},
					{Effect: "Allow", Action: []string{"s3:PutObject"}, Resource: "*"},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasGet := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3GetObject)]
				_, hasPut := allowStmt.Actions[normalizeAction(s3common.PolicyActionS3PutObject)]
				assert.True(t, hasGet)
				assert.True(t, hasPut)
			},
			action:        s3common.PolicyActionS3PutObject,
			resource:      "arn:aws:s3:::any-bucket/file.txt",
			expectedAllow: true,
			description:   "Actions from multiple Resource=* Allow statements must be merged in AllResourcesStmt",
		},
		// --- AllResourcesStmt with condition: condition must be evaluated ---
		{
			name: "allresources_allow_with_condition_condition_evaluated",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: "*",
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpStringEquals: {"aws:UserAgent": "my-app"},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::any-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/any-bucket/file.txt",
					map[string]string{"User-Agent": "other-app"}, nil, nil, nil)
			},
			expectedAllow: false,
			description:   "AllResourcesStmt Allow with condition must evaluate the condition (wrong UA → deny)",
		},
		// --- IAM Resource=* Allow grants IAM no-resource actions ---
		{
			name: "allresources_allow_iam_star_grants_iam_no_resource_actions",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"iam:*"}, Resource: "*"},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				noResShouldBeNil(t, authz)
				allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, ok := allowStmt.Actions[normalizeAction(s3common.PolicyActionIAMListUsers)]
				assert.True(t, ok)
			},
			action:        s3common.PolicyActionIAMListUsers,
			resource:      "",
			expectedAllow: true,
			description:   "iam:* on Resource=* stored in AllResourcesStmt must grant IAM no-resource actions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}
			authz := engine.calculateAuthorizedActions(tt.policies)

			if tt.validateAuthz != nil {
				tt.validateAuthz(t, &authz)
			}

			var req *http.Request
			if tt.buildRequest != nil {
				req = tt.buildRequest()
			} else {
				req = testhelpers.NewMockRequest("GET", "/", nil, nil, nil, nil)
			}

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode)
			}
		})
	}
}

// Test Group 6.3: No Resource Statements
func TestNoResourceStatements(t *testing.T) {
	tests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		resource      string
		buildRequest  func() *http.Request
		validateAuthz func(t *testing.T, authz *authorizedActions)
		expectedAllow bool
		description   string
	}{
		// --- IAM action without Resource field stored in NoResourceStmt ---
		{
			name: "iam_action_no_resource_stored_in_noResourceStmt",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"iam:ListUsers"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allResShouldBeNil(t, authz)
				assert.Empty(t, authz.WithSpecificResourceStmt)
				allowStmt := authz.NoResourceStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, ok := allowStmt.Actions[normalizeAction(s3common.PolicyActionIAMListUsers)]
				assert.True(t, ok)
			},
			action:        s3common.PolicyActionIAMListUsers,
			resource:      "",
			expectedAllow: true,
			description:   "IAM action with no Resource field must be stored in NoResourceStmt",
		},
		// --- Multiple no-resource IAM actions merged ---
		{
			name: "multiple_no_resource_iam_actions_merged",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"iam:ListUsers", "iam:GetUser"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allowStmt := authz.NoResourceStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				_, hasListUsers := allowStmt.Actions[normalizeAction(s3common.PolicyActionIAMListUsers)]
				_, hasGetUser := allowStmt.Actions[normalizeAction(s3common.PolicyActionIAMGetUser)]
				assert.True(t, hasListUsers)
				assert.True(t, hasGetUser)
			},
			action:        s3common.PolicyActionIAMGetUser,
			resource:      "",
			expectedAllow: true,
			description:   "Multiple IAM no-resource actions must all be stored in NoResourceStmt",
		},
		// --- NoResourceStmt Deny blocks no-resource Allow ---
		{
			name: "noresource_deny_blocks_noresource_allow",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"iam:DeleteUser"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"iam:DeleteUser"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				denyStmt := authz.NoResourceStmt[s3common.PolicyStatementEffectDeny]
				assert.NotNil(t, denyStmt)
				_, ok := denyStmt.Actions[normalizeAction(s3common.PolicyActionIAMDeleteUser)]
				assert.True(t, ok)
			},
			action:        s3common.PolicyActionIAMDeleteUser,
			resource:      "",
			expectedAllow: false,
			description:   "NoResourceStmt Deny must block NoResourceStmt Allow for the same action",
		},
		// --- NoResourceStmt does not affect S3 resource-based actions ---
		{
			name: "noresource_allow_does_not_grant_s3_action",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"iam:ListUsers"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "IAM no-resource Allow must not grant S3 resource-based actions (implicit deny)",
		},
		// --- NoResourceStmt with condition ---
		{
			name: "noresource_allow_with_condition_condition_evaluated",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect: "Allow",
						Action: []string{"iam:CreateUser"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpBool: {"aws:SecureTransport": "true"},
						},
					},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allowStmt := authz.NoResourceStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				condVals := allowStmt.Conditions[s3common.CondOpBool]["aws:SecureTransport"]
				assert.Equal(t, []string{"true"}, condVals)
			},
			action:   s3common.PolicyActionIAMCreateUser,
			resource: "",
			buildRequest: func() *http.Request {
				// Non-TLS → "false" → condition fails
				return testhelpers.NewMockRequest("POST", "/", nil, nil, nil, nil)
			},
			expectedAllow: false,
			description:   "NoResourceStmt Allow with Bool condition must evaluate condition (non-TLS → deny)",
		},
		// --- iam:* no-resource expands all IAM actions into NoResourceStmt ---
		{
			name: "iam_star_no_resource_expands_all_iam_actions",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"iam:*"}},
				}},
			},
			validateAuthz: func(t *testing.T, authz *authorizedActions) {
				allowStmt := authz.NoResourceStmt[s3common.PolicyStatementEffectAllow]
				assert.NotNil(t, allowStmt)
				for _, iamAction := range s3common.AllIAMSupportedActions {
					_, ok := allowStmt.Actions[normalizeAction(iamAction)]
					assert.True(t, ok, "Expected IAM action %s in NoResourceStmt Allow", iamAction)
				}
			},
			action:        s3common.PolicyActionIAMCreateAccessKey,
			resource:      "",
			expectedAllow: true,
			description:   "iam:* with no Resource must expand all supported IAM actions into NoResourceStmt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}
			authz := engine.calculateAuthorizedActions(tt.policies)

			if tt.validateAuthz != nil {
				tt.validateAuthz(t, &authz)
			}

			var req *http.Request
			if tt.buildRequest != nil {
				req = tt.buildRequest()
			} else {
				req = testhelpers.NewMockRequest("GET", "/", nil, nil, nil, nil)
			}

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode)
			}
		})
	}
}

// Test Group 7.1: Multiple Policies with Overlapping Statements
func TestMultiplePoliciesOverlappingStatements(t *testing.T) {
	tests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		buildRequest  func() *http.Request
		resource      string
		expectedAllow bool
		description   string
	}{
		// --- Three policies, cumulative Allow ---
		{
			name: "three_policies_cumulative_allow",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:PutObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:DeleteObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3DeleteObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "Actions from three separate Allow policies must be cumulative",
		},
		// --- Allow in policy 1, Deny in policy 3 (non-adjacent) ---
		{
			name: "allow_policy1_deny_policy3_non_adjacent",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:PutObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "Deny in non-adjacent third policy must still override Allow in first policy",
		},
		// --- Same action allowed by two policies on two different resources; only one resource requested ---
		{
			name: "same_action_two_policies_two_resources_only_first_requested",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::bucket-a/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::bucket-b/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::bucket-a/file.txt",
			expectedAllow: true,
			description:   "Same action allowed on two different resources → access to first resource is granted",
		},
		// --- Policy with no statements combined with a policy that has statements ---
		{
			name: "empty_policy_combined_with_allow_policy",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "Empty policy combined with Allow policy must still grant access",
		},
		// --- Overlapping Allow + Deny conditions: Deny condition matches, Allow condition also matches ---
		{
			name: "deny_condition_matches_overrides_allow_condition_matches",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpStringEquals: {"aws:UserAgent": "trusted-app"},
						},
					},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Deny",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpIpAddress: {"aws:SourceIp": "10.0.0.0/8"},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// Trusted app UA (Allow condition matches) but internal IP (Deny condition also matches)
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt",
					map[string]string{"User-Agent": "trusted-app"}, nil, nil, nil)
				req.RemoteAddr = "10.1.2.3:12345"
				return req
			},
			expectedAllow: false,
			description:   "When both Allow condition and Deny condition match, Deny must win",
		},
		// --- Allow condition matches, Deny condition does NOT match → allow ---
		{
			name: "allow_condition_matches_deny_condition_does_not_match",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpStringEquals: {"aws:UserAgent": "trusted-app"},
						},
					},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Deny",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpIpAddress: {"aws:SourceIp": "10.0.0.0/8"},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// Trusted app UA (Allow condition matches), public IP (Deny condition does NOT match)
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt",
					map[string]string{"User-Agent": "trusted-app"}, nil, nil, nil)
				req.RemoteAddr = "203.0.113.5:12345"
				return req
			},
			expectedAllow: true,
			description:   "Allow condition matches + Deny condition does not match → allow",
		},
		// --- Cross-policy conditions merged for same effect/resource ---
		{
			name: "cross_policy_conditions_merged_both_must_pass",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpStringEquals: {"aws:UserAgent": "my-app"},
						},
					},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:PutObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpBool: {"aws:SecureTransport": "true"},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", map[string]string{"User-Agent": "my-app"}, nil, nil, nil)
				req.TLS = &tls.ConnectionState{} // Simulate TLS connection for aws:SecureTransport condition
				return req
			},
			expectedAllow: true,
			description:   "Conditions from separate policies on same resource are stored per-statement; each evaluated independently",
		},
		// --- S3 + IAM policies coexist without interference ---
		{
			name: "s3_and_iam_policies_coexist",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"iam:ListUsers"}},
				}},
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Deny", Action: []string{"iam:DeleteUser"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "IAM-specific Deny must not interfere with unrelated S3 Allow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}
			authz := engine.calculateAuthorizedActions(tt.policies)

			var req *http.Request
			if tt.buildRequest != nil {
				req = tt.buildRequest()
			} else {
				req = testhelpers.NewMockRequest("GET", "/", nil, nil, nil, nil)
			}

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode)
			}
		})
	}
}

// Test Group 7.2: Large Scale
func TestLargeScalePolicies(t *testing.T) {
	t.Run("100_allow_policies_same_action_same_resource", func(t *testing.T) {
		// 100 identical Allow policies must not panic and must allow access
		policies := make([]PolicyDocument, 100)
		for i := range policies {
			policies[i] = PolicyDocument{
				Version: "2012-10-17",
				Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				},
			}
		}

		engine := &AuthzEngine{}
		authz := engine.calculateAuthorizedActions(policies)
		req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
		ctx := testhelpers.TestContext()

		err := engine.checkAccess(ctx, authz, *req, s3common.PolicyActionS3GetObject, "arn:aws:s3:::my-bucket/file.txt")
		assert.NoError(t, err, "100 identical Allow policies must allow access")
	})

	t.Run("large_number_of_distinct_specific_resources", func(t *testing.T) {
		// 200 distinct resources each with their own Allow
		policies := make([]PolicyDocument, 200)
		for i := range policies {
			policies[i] = PolicyDocument{
				Version: "2012-10-17",
				Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{fmt.Sprintf("arn:aws:s3:::bucket-%d/file.txt", i)},
					},
				},
			}
		}

		engine := &AuthzEngine{}
		authz := engine.calculateAuthorizedActions(policies)

		assert.Len(t, authz.WithSpecificResourceStmt, 200,
			"Each distinct resource must have its own entry in WithSpecificResourceStmt")

		req := testhelpers.NewMockRequest("GET", "/bucket-99/file.txt", nil, nil, nil, nil)
		ctx := testhelpers.TestContext()
		err := engine.checkAccess(ctx, authz, *req, s3common.PolicyActionS3GetObject, "arn:aws:s3:::bucket-99/file.txt")
		assert.NoError(t, err, "Access to one of 200 distinct resource entries must be allowed")
	})

	t.Run("all_s3_actions_explicitly_allowed_each_can_be_accessed", func(t *testing.T) {
		// One policy granting all S3 actions on *
		policies := []PolicyDocument{
			{Version: "2012-10-17", Statement: []*Statement{
				{Effect: "Allow", Action: []string{"s3:*"}, Resource: "*"},
			}},
		}

		engine := &AuthzEngine{}
		authz := engine.calculateAuthorizedActions(policies)

		req := testhelpers.NewMockRequest("GET", "/", nil, nil, nil, nil)
		ctx := testhelpers.TestContext()

		for _, action := range s3common.AllS3SupportedActions {
			err := engine.checkAccess(ctx, authz, *req, action, "arn:aws:s3:::any-bucket/any-file")
			assert.NoError(t, err, "s3:* must allow %s", action)
		}
	})

	t.Run("all_iam_actions_explicitly_denied_none_accessible", func(t *testing.T) {
		policies := []PolicyDocument{
			{Version: "2012-10-17", Statement: []*Statement{
				{Effect: "Allow", Action: []string{"iam:*"}},
			}},
			{Version: "2012-10-17", Statement: []*Statement{
				{Effect: "Deny", Action: []string{"iam:*"}},
			}},
		}

		engine := &AuthzEngine{}
		authz := engine.calculateAuthorizedActions(policies)

		req := testhelpers.NewMockRequest("GET", "/", nil, nil, nil, nil)
		ctx := testhelpers.TestContext()

		for _, action := range s3common.AllIAMSupportedActions {
			err := engine.checkAccess(ctx, authz, *req, action, "")
			assert.Error(t, err, "iam:* Deny must block %s", action)
		}
	})

	t.Run("many_conditions_all_must_pass_and_logic", func(t *testing.T) {
		// A single Allow statement with 3 conditions (AND logic): all must match
		policies := []PolicyDocument{
			{Version: "2012-10-17", Statement: []*Statement{
				{
					Effect:   "Allow",
					Action:   []string{"s3:GetObject"},
					Resource: []string{"arn:aws:s3:::my-bucket/*"},
					Condition: map[s3common.PolicyCondOp]map[string]any{
						s3common.CondOpStringEquals: {"aws:UserAgent": "secure-client"},
						s3common.CondOpBool:         {"aws:SecureTransport": "true"},
						s3common.CondOpIpAddress:    {"aws:SourceIp": "203.0.113.0/24"},
					},
				},
			}},
		}

		engine := &AuthzEngine{}
		authz := engine.calculateAuthorizedActions(policies)

		ctx := testhelpers.TestContext()

		// All three conditions satisfied
		req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt",
			map[string]string{"User-Agent": "secure-client"}, nil, nil, nil)
		req.TLS = &tls.ConnectionState{} // Simulate TLS connection for aws:SecureTransport condition
		req.RemoteAddr = "203.0.113.10:12345"
		err := engine.checkAccess(ctx, authz, *req, s3common.PolicyActionS3GetObject, "arn:aws:s3:::my-bucket/file.txt")
		assert.NoError(t, err, "All 3 conditions satisfied must allow")

		// One condition fails (wrong IP)
		req2 := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt",
			map[string]string{"User-Agent": "secure-client"}, nil, nil, nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req2.TLS = &tls.ConnectionState{} // Simulate TLS connection for aws:SecureTransport condition
		err2 := engine.checkAccess(ctx, authz, *req2, s3common.PolicyActionS3GetObject, "arn:aws:s3:::my-bucket/file.txt")
		assert.Error(t, err2, "One failing condition (wrong IP) must deny")
	})
}

// Test Group 8.1: Empty/Null Values
func TestEmptyNullValues(t *testing.T) {
	tests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		resource      string
		buildRequest  func() *http.Request
		expectedAllow bool
		description   string
	}{
		// --- Empty action string in checkAccess ---
		{
			name: "empty_action_returns_error",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: "*", Resource: "*"},
				}},
			},
			action:        "",
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "Empty action string must be rejected (implicit deny / invalid argument)",
		},
		// --- Empty resource string: no matching Allow → implicit deny ---
		{
			name: "empty_resource_no_match_implicit_deny",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "",
			expectedAllow: false,
			description:   "Empty resource string does not match specific resource Allow → implicit deny",
		},
		// --- Condition key present but with empty value: non-IfExists condition should fail ---
		{
			name: "condition_key_empty_value_non_ifexists_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpStringEquals: {"aws:UserAgent": "my-app"},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// User-Agent header is empty string (present key, empty value)
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt",
					map[string]string{"User-Agent": ""}, nil, nil, nil)
			},
			expectedAllow: false,
			description:   "StringEquals condition with empty actual value (empty UA) must deny",
		},
		// --- Condition key absent, IfExists: must pass ---
		{
			name: "condition_key_absent_ifexists_passes",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpStringEquals + s3common.CondOpIfExistsSuffix: {"aws:UserAgent": "my-app"},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				// No User-Agent header → key absent
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "IfExists condition with absent key must pass (IfExists semantics)",
		},
		// --- Numeric condition with non-numeric actual value: must fail ---
		{
			name: "numeric_condition_non_numeric_actual_value_denies",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::my-bucket"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpNumericLessThanEquals: {"s3:max-keys": "100"},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3ListBucket,
			resource: "arn:aws:s3:::my-bucket",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket", nil, nil, nil,
					map[string]string{"max-keys": "not-a-number"})
			},
			expectedAllow: false,
			description:   "NumericLessThanEquals with non-numeric actual value must fail the condition",
		},
		// --- Policy with zero statements does not panic ---
		{
			name: "zero_statements_policy_no_panic",
			policies: []PolicyDocument{
				{Version: "2012-10-17"},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: false,
			description:   "Policy with nil Statement slice must not panic and must implicit deny",
		},
		// --- Nil conditions map in statement: must not panic ---
		{
			name: "nil_conditions_map_no_panic",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}, Condition: nil},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "nil Condition map in statement must not panic and must be treated as no condition",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}
			authz := engine.calculateAuthorizedActions(tt.policies)

			var req *http.Request
			if tt.buildRequest != nil {
				req = tt.buildRequest()
			} else {
				req = testhelpers.NewMockRequest("GET", "/", nil, nil, nil, nil)
			}

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
			}
		})
	}
}

// Test Group 8.2: Special Characters
func TestSpecialCharacters(t *testing.T) {
	tests := []struct {
		name          string
		policies      []PolicyDocument
		action        s3common.PolicyAction
		resource      string
		buildRequest  func() *http.Request
		expectedAllow bool
		description   string
	}{
		// --- Resource ARN with spaces in key name ---
		{
			name: "resource_arn_with_slash_and_dots_in_key",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/path/to/file.v2.tar.gz"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/path/to/file.v2.tar.gz",
			expectedAllow: true,
			description:   "Resource ARN with slashes and dots in the key path must match exactly",
		},
		// --- Resource ARN with percent-encoded character (treated as literal) ---
		{
			name: "resource_arn_percent_encoded_no_match_to_decoded",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/file%20name.txt"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/file name.txt", // decoded form
			expectedAllow: false,
			description:   "Percent-encoded resource in policy must not match the decoded form (literal comparison)",
		},
		// --- Wildcard that contains special regex chars (treated as literals by matchIAM) ---
		{
			name: "resource_arn_with_dot_in_bucket_name_not_treated_as_regex",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					// The dot is a literal character in the ARN, not a regex wildcard
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my.bucket/file.txt"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::myXbucket/file.txt", // dot must NOT match arbitrary char
			expectedAllow: false,
			description:   "Dot in resource ARN must be treated as a literal, not a regex wildcard",
		},
		// --- Action with uppercase letters stored normalised (lowercase) ---
		{
			name: "action_uppercase_stored_normalised",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"S3:GETOBJECT"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject, // lowercase constant
			resource:      "arn:aws:s3:::my-bucket/file.txt",
			expectedAllow: true,
			description:   "Actions must be normalised to lowercase during storage and matching",
		},
		// --- Resource with unicode characters (exact match) ---
		{
			name: "resource_with_unicode_exact_match",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{Effect: "Allow", Action: []string{"s3:GetObject"}, Resource: []string{"arn:aws:s3:::my-bucket/résumé.pdf"}},
				}},
			},
			action:        s3common.PolicyActionS3GetObject,
			resource:      "arn:aws:s3:::my-bucket/résumé.pdf",
			expectedAllow: true,
			description:   "Unicode characters in resource ARN must match exactly",
		},
		// --- Wildcard * in condition value matches any string ---
		{
			name: "string_like_star_in_condition_value_matches_any",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpStringLike: {"aws:UserAgent": "*"},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt",
					map[string]string{"User-Agent": "anything-at-all"}, nil, nil, nil)
			},
			expectedAllow: true,
			description:   "StringLike condition value * must match any non-empty User-Agent string",
		},
		// --- Condition key with mixed case is matched case-sensitively ---
		{
			name: "condition_key_case_sensitive_no_match",
			policies: []PolicyDocument{
				{Version: "2012-10-17", Statement: []*Statement{
					{
						Effect:   "Allow",
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
						Condition: map[s3common.PolicyCondOp]map[string]any{
							s3common.CondOpStringEquals: {"aws:UserAgent": "my-app"},
						},
					},
				}},
			},
			action:   s3common.PolicyActionS3GetObject,
			resource: "arn:aws:s3:::my-bucket/file.txt",
			buildRequest: func() *http.Request {
				return testhelpers.NewMockRequest("GET", "/my-bucket/file.txt",
					map[string]string{"User-Agent": "MY-APP"}, nil, nil, nil) // wrong case
			},
			expectedAllow: false,
			description:   "StringEquals condition value comparison is case-sensitive (MY-APP ≠ my-app)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &AuthzEngine{}
			authz := engine.calculateAuthorizedActions(tt.policies)

			var req *http.Request
			if tt.buildRequest != nil {
				req = tt.buildRequest()
			} else {
				req = testhelpers.NewMockRequest("GET", "/", nil, nil, nil, nil)
			}

			ctx := testhelpers.TestContext()
			err := engine.checkAccess(ctx, authz, *req, tt.action, tt.resource)

			if tt.expectedAllow {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
				s3Err, _ := err.(s3common.S3Error)
				assert.Equal(t, s3common.AccessDenied, s3Err.S3ErrorCode)
			}
		})
	}
}

func BenchmarkCalculateAuthorizedActions(b *testing.B) {
	// Benchmark with 100 policies
	policies := make([]PolicyDocument, 100)
	for i := range policies {
		policies[i] = PolicyDocument{
			Version: "2012-10-17",
			Statement: []*Statement{
				{
					Effect:   "Allow",
					Action:   []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"},
					Resource: []string{fmt.Sprintf("arn:aws:s3:::bucket-%d/*", i)},
				},
			},
		}
	}
	engine := &AuthzEngine{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.calculateAuthorizedActions(policies)
	}
}

func BenchmarkCalculateAuthorizedActions_Parallel(b *testing.B) {
	policies := []PolicyDocument{
		{Version: "2012-10-17", Statement: []*Statement{
			{Effect: "Allow", Action: []string{"s3:*"}, Resource: "*"},
		}},
	}
	engine := &AuthzEngine{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			engine.calculateAuthorizedActions(policies)
		}
	})
}

func BenchmarkCheckAccess(b *testing.B) {
	policies := []PolicyDocument{
		{Version: "2012-10-17", Statement: []*Statement{
			{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::my-bucket/*"},
				Condition: map[s3common.PolicyCondOp]map[string]any{
					s3common.CondOpStringEquals: {"aws:UserAgent": "my-app"},
					s3common.CondOpBool:         {"aws:SecureTransport": "true"},
				},
			},
		}},
	}
	engine := &AuthzEngine{}
	authz := engine.calculateAuthorizedActions(policies)
	req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt",
		map[string]string{"User-Agent": "my-app"}, nil, nil, nil)
	ctx := testhelpers.TestContext()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.checkAccess(ctx, authz, *req,
			s3common.PolicyActionS3GetObject, "arn:aws:s3:::my-bucket/file.txt")
	}
}

func BenchmarkCheckAccess_Parallel(b *testing.B) {
	policies := []PolicyDocument{
		{Version: "2012-10-17", Statement: []*Statement{
			{Effect: "Allow", Action: []string{"s3:GetObject", "s3:PutObject"}, Resource: []string{"arn:aws:s3:::my-bucket/*"}},
			{Effect: "Deny", Action: []string{"s3:DeleteObject"}, Resource: "*"},
		}},
	}
	engine := &AuthzEngine{}
	authz := engine.calculateAuthorizedActions(policies)
	ctx := testhelpers.TestContext()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		req := testhelpers.NewMockRequest("GET", "/my-bucket/file.txt", nil, nil, nil, nil)
		for pb.Next() {
			_ = engine.checkAccess(ctx, authz, *req,
				s3common.PolicyActionS3GetObject, "arn:aws:s3:::my-bucket/file.txt")
		}
	})
}
