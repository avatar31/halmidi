package iam

import (
    "fmt"
    "strings"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/avatar31/halmidi/internal/core/s3common"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func assertMalformedPolicy(t *testing.T, err error, msgContains string) {
    t.Helper()
    require.Error(t, err)
    s3Err, ok := err.(s3common.S3Error)
    require.True(t, ok, "expected S3Error, got %T", err)
    assert.Equal(t, s3common.MalformedPolicyDocument, s3Err.S3ErrorCode)
    if msgContains != "" {
        assert.Contains(t, s3Err.Message, msgContains)
    }
}

func validMinimalPolicy() string {
    return `{
        "Version": "2012-10-17",
        "Statement": [{
            "Effect": "Allow",
            "Action": "s3:GetObject",
            "Resource": "arn:aws:s3:::my-bucket/*"
        }]
    }`
}

// ---------------------------------------------------------------------------
// ValidatePolicyDocument
// ---------------------------------------------------------------------------

func TestValidatePolicyDocument(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        expectError bool
        errContains string
    }{
        {
            name:        "valid_minimal_policy",
            input:       validMinimalPolicy(),
            expectError: false,
        },
        {
            name:        "empty_string",
            input:       "",
            expectError: true,
            errContains: "Invalid Policy document format.",
        },
        {
            name:        "exceeds_max_length",
            input:       fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:GetObject","Resource":"arn:aws:s3:::b/*","Sid":"%s"}]}`, strings.Repeat("x", s3common.MAX_ALLOWED_POLICY_DOCUMENT_LENGTH)),
            expectError: true,
            errContains: "Policy document length exceeds",
        },
        {
            name:        "invalid_json",
            input:       `{"Version": "2012-10-17", "Statement": [}`,
            expectError: true,
        },
        {
            name:        "unknown_field_rejected",
            input:       `{"Version":"2012-10-17","UnknownField":"x","Statement":[{"Effect":"Allow","Action":"s3:GetObject","Resource":"arn:aws:s3:::b/*"}]}`,
            expectError: true,
        },
        {
            name: "valid_policy_array_action_array_resource",
            input: `{
                "Version": "2012-10-17",
                "Statement": [{
                    "Effect": "Allow",
                    "Action": ["s3:GetObject","s3:PutObject"],
                    "Resource": ["arn:aws:s3:::my-bucket/*","arn:aws:s3:::other-bucket/*"]
                }]
            }`,
            expectError: false,
        },
        {
            name: "valid_policy_star_action_star_resource",
            input: `{
                "Version": "2012-10-17",
                "Statement": [{"Effect": "Allow", "Action": "*", "Resource": "*"}]
            }`,
            expectError: false,
        },
        {
            name: "valid_policy_notaction_notresource",
            input: `{
                "Version": "2012-10-17",
                "Statement": [{
                    "Effect": "Deny",
                    "NotAction": "s3:DeleteObject",
                    "NotResource": "arn:aws:s3:::my-bucket/*"
                }]
            }`,
            expectError: false,
        },
        {
            name: "valid_iam_action_no_resource",
            input: `{
                "Version": "2012-10-17",
                "Statement": [{"Effect": "Allow", "Action": "iam:ListUsers"}]
            }`,
            expectError: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePolicyDocument(tt.input)
            if tt.expectError {
                assertMalformedPolicy(t, err, tt.errContains)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

// ---------------------------------------------------------------------------
// ParsePolicyDocument
// ---------------------------------------------------------------------------

func TestParsePolicyDocument(t *testing.T) {
    t.Run("valid_parses_correctly", func(t *testing.T) {
        policy, err := ParsePolicyDocument(validMinimalPolicy())
        require.NoError(t, err)
        assert.Equal(t, "2012-10-17", policy.Version)
        require.Len(t, policy.Statement, 1)
        assert.Equal(t, s3common.PolicyStatementEffectAllow, policy.Statement[0].Effect)
    })

    t.Run("invalid_json_returns_malformed_policy_error", func(t *testing.T) {
        _, err := ParsePolicyDocument(`{bad json}`)
        assertMalformedPolicy(t, err, "")
    })

    t.Run("unknown_field_returns_malformed_policy_error", func(t *testing.T) {
        _, err := ParsePolicyDocument(`{"Version":"2012-10-17","Unknown":"x","Statement":[]}`)
        assertMalformedPolicy(t, err, "")
    })

    t.Run("array_action_decoded_as_slice_any", func(t *testing.T) {
        // After JSON decode []string in policy is []interface{}, stringyfyAny must handle it
        doc := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["s3:GetObject","s3:PutObject"],"Resource":"arn:aws:s3:::b/*"}]}`
        policy, err := ParsePolicyDocument(doc)
        require.NoError(t, err)
        actions := stringyfyAny(policy.Statement[0].Action)
        assert.Equal(t, []string{"s3:GetObject", "s3:PutObject"}, actions)
    })
}

// ---------------------------------------------------------------------------
// PolicyDocument.Validate
// ---------------------------------------------------------------------------

func TestPolicyDocumentValidate(t *testing.T) {
    t.Run("empty_version_rejected", func(t *testing.T) {
        p := PolicyDocument{Version: "", Statement: []*Statement{}}
        assertMalformedPolicy(t, p.Validate(), "Unsupported policy version.")
    })

    t.Run("wrong_version_rejected", func(t *testing.T) {
        p := PolicyDocument{Version: "2008-10-17", Statement: []*Statement{}}
        assertMalformedPolicy(t, p.Validate(), "Unsupported policy version.")
    })

    t.Run("correct_version_accepted", func(t *testing.T) {
        p := PolicyDocument{
            Version: "2012-10-17",
            Statement: []*Statement{
                {Effect: s3common.PolicyStatementEffectAllow, Action: "s3:GetObject", Resource: "arn:aws:s3:::b/*"},
            },
        }
        assert.NoError(t, p.Validate())
    })

    t.Run("nil_statement_slice_accepted", func(t *testing.T) {
        // AWS allows policies with no statements
        p := PolicyDocument{Version: "2012-10-17"}
        assert.NoError(t, p.Validate())
    })

    t.Run("invalid_statement_propagates_error", func(t *testing.T) {
        p := PolicyDocument{
            Version: "2012-10-17",
            Statement: []*Statement{
                {Effect: "BadEffect", Action: "s3:GetObject", Resource: "arn:aws:s3:::b/*"},
            },
        }
        assertMalformedPolicy(t, p.Validate(), "")
    })
}

// ---------------------------------------------------------------------------
// Statement.Validate — Effect
// ---------------------------------------------------------------------------

func TestStatementValidate_Effect(t *testing.T) {
    tests := []struct {
        name        string
        effect      s3common.PolicyStatementEffect
        expectError bool
        errContains string
    }{
        {"allow_valid", s3common.PolicyStatementEffectAllow, false, ""},
        {"deny_valid", s3common.PolicyStatementEffectDeny, false, ""},
        {"empty_effect_invalid", "", true, ""},
        {"lowercase_allow_invalid", "allow", true, ""},
        {"lowercase_deny_invalid", "deny", true, ""},
        {"random_string_invalid", "Permit", true, ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            s := &Statement{
                Effect:   tt.effect,
                Action:   "s3:GetObject",
                Resource: "arn:aws:s3:::b/*",
            }
            err := s.Validate()
            if tt.expectError {
                assertMalformedPolicy(t, err, tt.errContains)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

// ---------------------------------------------------------------------------
// Statement.Validate — Action / NotAction mutual exclusion & presence
// ---------------------------------------------------------------------------

func TestStatementValidate_ActionNotAction(t *testing.T) {
    t.Run("both_nil_rejected", func(t *testing.T) {
        s := &Statement{Effect: s3common.PolicyStatementEffectAllow, Resource: "arn:aws:s3:::b/*"}
        assertMalformedPolicy(t, s.Validate(), "Either Action or NotAction must be specified.")
    })

    t.Run("both_set_rejected", func(t *testing.T) {
        s := &Statement{
            Effect:    s3common.PolicyStatementEffectAllow,
            Action:    "s3:GetObject",
            NotAction: "s3:PutObject",
            Resource:  "arn:aws:s3:::b/*",
        }
        assertMalformedPolicy(t, s.Validate(), "Both Action and NotAction cannot be specified")
    })

    t.Run("only_action_accepted", func(t *testing.T) {
        s := &Statement{Effect: s3common.PolicyStatementEffectAllow, Action: "s3:GetObject", Resource: "arn:aws:s3:::b/*"}
        assert.NoError(t, s.Validate())
    })

    t.Run("only_notaction_accepted", func(t *testing.T) {
        s := &Statement{Effect: s3common.PolicyStatementEffectDeny, NotAction: "s3:DeleteObject", Resource: "arn:aws:s3:::b/*"}
        assert.NoError(t, s.Validate())
    })
}

// ---------------------------------------------------------------------------
// Statement.Validate — Resource / NotResource mutual exclusion
// ---------------------------------------------------------------------------

func TestStatementValidate_ResourceNotResource(t *testing.T) {
    t.Run("both_resource_and_notresource_rejected", func(t *testing.T) {
        s := &Statement{
            Effect:      s3common.PolicyStatementEffectAllow,
            Action:      "s3:GetObject",
            Resource:    "arn:aws:s3:::b/*",
            NotResource: "arn:aws:s3:::other/*",
        }
        assertMalformedPolicy(t, s.Validate(), "Both Resource and NotResource cannot be specified")
    })

    t.Run("only_resource_accepted", func(t *testing.T) {
        s := &Statement{Effect: s3common.PolicyStatementEffectAllow, Action: "s3:GetObject", Resource: "arn:aws:s3:::b/*"}
        assert.NoError(t, s.Validate())
    })

    t.Run("only_notresource_accepted", func(t *testing.T) {
        s := &Statement{Effect: s3common.PolicyStatementEffectDeny, Action: "s3:GetObject", NotResource: "arn:aws:s3:::b/*"}
        assert.NoError(t, s.Validate())
    })

    t.Run("iam_action_without_resource_accepted", func(t *testing.T) {
        s := &Statement{Effect: s3common.PolicyStatementEffectAllow, Action: "iam:ListUsers"}
        assert.NoError(t, s.Validate())
    })
}

// ---------------------------------------------------------------------------
// Statement.validateAction
// ---------------------------------------------------------------------------

func TestStatementValidateAction(t *testing.T) {
    tests := []struct {
        name        string
        action      string
        expectError bool
        errContains string
    }{
        // Valid cases
        {"star_wildcard_accepted", "*", false, ""},
        {"s3_getobject", "s3:GetObject", false, ""},
        {"s3_star", "s3:*", false, ""},
        {"iam_list_users", "iam:ListUsers", false, ""},
        {"iam_star", "iam:*", false, ""},
        {"s3_get_wildcard_suffix", "s3:Get*", false, ""},
        {"s3_wildcard_prefix_object", "s3:*Object", false, ""},

        // Invalid cases
        {"no_colon_rejected", "s3GetObject", true, "format 'service:action'"},
        {"empty_service_rejected", ":GetObject", true, "format 'service:action'"},
        {"empty_action_part_rejected", "s3:", true, "format 'service:action'"},
        {"wildcard_service_prefix_rejected", "*:GetObject", true, "Wildcard service prefix"},
        {"unsupported_service_rejected", "ec2:DescribeInstances", true, "Unsupported service"},
        {"empty_string_rejected", "", true, ""},
    }

    s := &Statement{}
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := s.validateAction(tt.action)
            if tt.expectError {
                assertMalformedPolicy(t, err, tt.errContains)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

// ---------------------------------------------------------------------------
// Statement.ValidateActionOrNotAction
// ---------------------------------------------------------------------------

func TestValidateActionOrNotAction(t *testing.T) {
    s := &Statement{}

    t.Run("single_string_valid", func(t *testing.T) {
        assert.NoError(t, s.ValidateActionOrNotAction("s3:GetObject"))
    })

    t.Run("string_slice_valid", func(t *testing.T) {
        assert.NoError(t, s.ValidateActionOrNotAction([]string{"s3:GetObject", "s3:PutObject"}))
    })

    t.Run("any_slice_valid", func(t *testing.T) {
        // Simulates JSON-decoded array ([]interface{})
        assert.NoError(t, s.ValidateActionOrNotAction([]any{"s3:GetObject", "iam:ListUsers"}))
    })

    t.Run("empty_string_rejected", func(t *testing.T) {
        assertMalformedPolicy(t, s.ValidateActionOrNotAction(""), "Action cannot be empty.")
    })

    t.Run("empty_string_slice_rejected", func(t *testing.T) {
        assertMalformedPolicy(t, s.ValidateActionOrNotAction([]string{}), "Action cannot be empty.")
    })

    t.Run("empty_any_slice_rejected", func(t *testing.T) {
        assertMalformedPolicy(t, s.ValidateActionOrNotAction([]any{}), "Action cannot be empty.")
    })

    t.Run("any_slice_with_invalid_action_rejected", func(t *testing.T) {
        assertMalformedPolicy(t, s.ValidateActionOrNotAction([]any{"s3:GetObject", "invalid"}), "")
    })

    t.Run("any_slice_with_non_string_element_handled", func(t *testing.T) {
        // stringyfyAny skips non-string items; result slice will be shorter
        // but ValidateActionOrNotAction must not panic
        err := s.ValidateActionOrNotAction([]any{"s3:GetObject", 42})
        // 42 is silently skipped by stringyfyAny, only "s3:GetObject" is validated
        assert.NoError(t, err)
    })

    t.Run("unsupported_service_in_array_rejected", func(t *testing.T) {
        assertMalformedPolicy(t, s.ValidateActionOrNotAction([]string{"s3:GetObject", "ec2:Describe"}), "Unsupported service")
    })
}

// ---------------------------------------------------------------------------
// Statement.ValidateResourceOrNotResource
// ---------------------------------------------------------------------------

func TestValidateResourceOrNotResource(t *testing.T) {
    s := &Statement{}

    t.Run("star_wildcard_accepted", func(t *testing.T) {
        assert.NoError(t, s.ValidateResourceOrNotResource("*"))
    })

    t.Run("valid_s3_bucket_arn_accepted", func(t *testing.T) {
        assert.NoError(t, s.ValidateResourceOrNotResource("arn:aws:s3:::my-bucket/*"))
    })

    t.Run("valid_s3_object_arn_accepted", func(t *testing.T) {
        assert.NoError(t, s.ValidateResourceOrNotResource("arn:aws:s3:::my-bucket/key.txt"))
    })

    t.Run("valid_iam_user_arn_accepted", func(t *testing.T) {
        assert.NoError(t, s.ValidateResourceOrNotResource(
            fmt.Sprintf("arn:aws:iam::%s:user/alice", "halmidi")))
    })

    t.Run("array_of_valid_arns_accepted", func(t *testing.T) {
        assert.NoError(t, s.ValidateResourceOrNotResource(
            []string{"arn:aws:s3:::bucket-a/*", "arn:aws:s3:::bucket-b/*"}))
    })

    t.Run("any_slice_of_valid_arns_accepted", func(t *testing.T) {
        assert.NoError(t, s.ValidateResourceOrNotResource(
            []any{"arn:aws:s3:::bucket-a/*", "*"}))
    })

    t.Run("empty_string_rejected", func(t *testing.T) {
        assertMalformedPolicy(t, s.ValidateResourceOrNotResource(""), "Resource cannot be empty.")
    })

    t.Run("empty_slice_rejected", func(t *testing.T) {
        assertMalformedPolicy(t, s.ValidateResourceOrNotResource([]string{}), "Resource cannot be empty.")
    })

    t.Run("invalid_arn_rejected", func(t *testing.T) {
        assertMalformedPolicy(t, s.ValidateResourceOrNotResource("not-an-arn"), "Invalid resource ARN.")
    })

    t.Run("arn_exceeding_max_length_rejected", func(t *testing.T) {
        longARN := "arn:aws:s3:::" + strings.Repeat("a", s3common.MAX_ALLOWED_ARN_LENGTH)
        assertMalformedPolicy(t, s.ValidateResourceOrNotResource(longARN), "Invalid resource ARN.")
    })

    t.Run("unsupported_service_in_arn_rejected", func(t *testing.T) {
        assertMalformedPolicy(t, s.ValidateResourceOrNotResource("arn:aws:ec2:::instance/i-1234"), "Invalid resource ARN.")
    })

    t.Run("array_with_one_invalid_arn_rejected", func(t *testing.T) {
        assertMalformedPolicy(t, s.ValidateResourceOrNotResource(
            []string{"arn:aws:s3:::good-bucket/*", "not-an-arn"}), "Invalid resource ARN.")
    })
}

// ---------------------------------------------------------------------------
// Statement.Validate — Condition
// ---------------------------------------------------------------------------

func TestStatementValidate_Condition(t *testing.T) {
    validBase := func() *Statement {
        return &Statement{
            Effect:   s3common.PolicyStatementEffectAllow,
            Action:   "s3:GetObject",
            Resource: "arn:aws:s3:::b/*",
        }
    }

    t.Run("nil_condition_accepted", func(t *testing.T) {
        s := validBase()
        s.Condition = nil
        assert.NoError(t, s.Validate())
    })

    t.Run("valid_string_equals_condition_accepted", func(t *testing.T) {
        s := validBase()
        s.Condition = map[s3common.PolicyCondOp]map[string]any{
            s3common.CondOpStringEquals: {"s3:prefix": "home/"},
        }
        assert.NoError(t, s.Validate())
    })

    t.Run("valid_condition_value_as_any_slice_accepted", func(t *testing.T) {
        s := validBase()
        s.Condition = map[s3common.PolicyCondOp]map[string]any{
            s3common.CondOpStringEquals: {"s3:prefix": []any{"home/", "shared/"}},
        }
        assert.NoError(t, s.Validate())
    })

    t.Run("valid_condition_value_as_string_slice_accepted", func(t *testing.T) {
        s := validBase()
        s.Condition = map[s3common.PolicyCondOp]map[string]any{
            s3common.CondOpStringEquals: {"s3:prefix": []string{"home/", "shared/"}},
        }
        assert.NoError(t, s.Validate())
    })

    t.Run("unsupported_condition_operator_rejected", func(t *testing.T) {
        s := validBase()
        s.Condition = map[s3common.PolicyCondOp]map[string]any{
            "UnknownOperator": {"s3:prefix": "home/"},
        }
        assertMalformedPolicy(t, s.Validate(), "Unsupported condition operator")
    })

    t.Run("empty_condition_key_rejected", func(t *testing.T) {
        s := validBase()
        s.Condition = map[s3common.PolicyCondOp]map[string]any{
            s3common.CondOpStringEquals: {"": "home/"},
        }
        assertMalformedPolicy(t, s.Validate(), "Condition key cannot be empty.")
    })

    t.Run("non_string_value_in_any_slice_rejected", func(t *testing.T) {
        s := validBase()
        s.Condition = map[s3common.PolicyCondOp]map[string]any{
            s3common.CondOpStringEquals: {"s3:prefix": []any{"valid", 42}},
        }
        assertMalformedPolicy(t, s.Validate(), "must be a string")
    })

    t.Run("integer_condition_value_rejected", func(t *testing.T) {
        s := validBase()
        s.Condition = map[s3common.PolicyCondOp]map[string]any{
            s3common.CondOpNumericEquals: {"s3:max-keys": 100}, // int, not string
        }
        assertMalformedPolicy(t, s.Validate(), "must be a string or array of strings")
    })

    t.Run("bool_condition_value_rejected", func(t *testing.T) {
        s := validBase()
        s.Condition = map[s3common.PolicyCondOp]map[string]any{
            s3common.CondOpBool: {"aws:SecureTransport": true}, // bool, not string
        }
        assertMalformedPolicy(t, s.Validate(), "must be a string or array of strings")
    })

    t.Run("multiple_valid_condition_operators_accepted", func(t *testing.T) {
        s := validBase()
        s.Condition = map[s3common.PolicyCondOp]map[string]any{
            s3common.CondOpStringEquals: {"aws:UserAgent": "my-app"},
            s3common.CondOpBool:         {"aws:SecureTransport": "true"},
            s3common.CondOpIpAddress:    {"aws:SourceIp": "203.0.113.0/24"},
        }
        assert.NoError(t, s.Validate())
    })
}
