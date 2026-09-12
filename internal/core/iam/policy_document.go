package iam

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/avatar31/halmidi/internal/core/s3common"
)

var (
	policyDocRegex = regexp.MustCompile(s3common.PolicyDocumentRegex)
)

// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements.html
type PolicyDocument struct {
	Version   string       `json:"Version"`
	Id        string       `json:"Id,omitempty"`
	Statement []*Statement `json:"Statement,omitempty"`
}

type Statement struct {
	Sid         string                                   `json:"Sid,omitempty"`
	Effect      s3common.PolicyStatementEffect           `json:"Effect"`
	Action      any                                      `json:"Action,omitempty"`
	NotAction   any                                      `json:"NotAction,omitempty"`
	Resource    any                                      `json:"Resource,omitempty"`
	NotResource any                                      `json:"NotResource,omitempty"`
	Condition   map[s3common.PolicyCondOp]map[string]any `json:"Condition,omitempty"`

	// We are not supporting Prinicipal and NotPrincipal as it contains
	// AWS-account-IDs and ARNs which are not relevant in our case.
	// Principal    any `json:"Principal,omitempty"`
	// NotPrincipal any `json:"NotPrincipal,omitempty"`
}

func (p PolicyDocument) Validate() error {
	if p.Version == "" || p.Version != s3common.SUPPORTED_POLICY_VERSION {
		return s3common.GetMalformedPolicyDocumentS3Error("", "Unsupported policy version.")
	}

	for _, statement := range p.Statement {
		if err := statement.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (s Statement) Validate() error {
	if err := s.Effect.Validate(); err != nil {
		return err
	}

	if s.Action == nil && s.NotAction == nil {
		return s3common.GetMalformedPolicyDocumentS3Error("", "Either Action or NotAction must be specified.")
	}

	if s.Action != nil && s.NotAction != nil {
		return s3common.GetMalformedPolicyDocumentS3Error("", "Both Action and NotAction cannot be specified in Statement.")
	}

	if s.Action != nil {
		if err := s.ValidateActionOrNotAction(s.Action); err != nil {
			return err
		}
	}

	if s.NotAction != nil {
		if err := s.ValidateActionOrNotAction(s.NotAction); err != nil {
			return err
		}
	}

	if s.Resource != nil && s.NotResource != nil {
		return s3common.GetMalformedPolicyDocumentS3Error("", "Both Resource and NotResource cannot be specified in Statement.")
	}

	if s.Resource != nil {
		if err := s.ValidateResourceOrNotResource(s.Resource); err != nil {
			return err
		}
	}

	if s.NotResource != nil {
		if err := s.ValidateResourceOrNotResource(s.NotResource); err != nil {
			return err
		}
	}

	if s.Condition != nil {
		for condOp, condKeyMap := range s.Condition {
			if !s3common.IsValidIAMCondition(condOp) {
				return s3common.GetMalformedPolicyDocumentS3Error("", fmt.Sprintf("Unsupported condition operator '%s'.", condOp))
			}

			for key, val := range condKeyMap {
				if key == "" {
					return s3common.GetMalformedPolicyDocumentS3Error("", "Condition key cannot be empty.")
				}
				// Ensure values are string or []string/[]any
				switch v := val.(type) {
				case string, []string:
					// ok
				case []any:
					for _, item := range v {
						if _, ok := item.(string); !ok {
							return s3common.GetMalformedPolicyDocumentS3Error("", fmt.Sprintf("Condition value for key '%s' must be a string.", key))
						}
					}
				default:
					_ = v
					return s3common.GetMalformedPolicyDocumentS3Error("", fmt.Sprintf("Condition value for key '%s' must be a string or array of strings.", key))
				}
			}
		}
	}

	return nil
}

func (s Statement) ValidateActionOrNotAction(action any) error {
	actionVal := stringyfyAny(action)
	if action == "" || len(actionVal) == 0 {
		return s3common.GetMalformedPolicyDocumentS3Error("", "Action cannot be empty.")
	}

	for _, act := range actionVal {
		if err := s.validateAction(act); err != nil {
			return err
		}
	}

	return nil
}

func (s Statement) validateAction(action string) error {
	if action == s3common.AllActionsOrResourcePattern {
		return nil
	}

	if !strings.Contains(action, ":") {
		return s3common.GetMalformedPolicyDocumentS3Error("", "Action must be in the format 'service:action'.")
	}

	splits := strings.SplitN(action, ":", 2)
	if len(splits) != 2 || splits[0] == "" || splits[1] == "" {
		return s3common.GetMalformedPolicyDocumentS3Error("", "Action must be in the format 'service:action'.")
	}

	if splits[0] == "*" {
		return s3common.GetMalformedPolicyDocumentS3Error("", "Wildcard service prefix is not allowed in action.")
	}

	// TODO:P1: Should we accept policies of all services?
	if !slices.Contains(s3common.SUPPORTED_SERVICES, splits[0]) {
		return s3common.GetMalformedPolicyDocumentS3Error("", fmt.Sprintf("Unsupported service '%s' in action.", splits[0]))
	}

	return nil
}

func (s Statement) ValidateResourceOrNotResource(resource any) error {
	resourceVal := stringyfyAny(resource)
	if resource == "" || len(resourceVal) == 0 {
		return s3common.GetMalformedPolicyDocumentS3Error("", "Resource cannot be empty.")
	}

	for _, arn := range resourceVal {
		if arn != s3common.AllActionsOrResourcePattern && !s3common.IsValidARN(arn) {
			return s3common.GetMalformedPolicyDocumentS3Error("", "Invalid resource ARN.")
		}
	}

	return nil
}

func ParsePolicyDocument(policyDoc string) (*PolicyDocument, error) {
	decoder := json.NewDecoder(strings.NewReader(policyDoc))
	decoder.DisallowUnknownFields()

	var policy PolicyDocument
	if err := decoder.Decode(&policy); err != nil {
		return nil, s3common.GetMalformedPolicyDocumentS3Error("", err.Error())
	}

	return &policy, nil
}

func ValidatePolicyDocument(policyDoc string) error {
	if len(policyDoc) == 0 || !policyDocRegex.MatchString(policyDoc) {
		return s3common.GetMalformedPolicyDocumentS3Error("", "Invalid Policy document format.")
	}

	if len(policyDoc) > s3common.MAX_ALLOWED_POLICY_DOCUMENT_LENGTH {
		return s3common.GetMalformedPolicyDocumentS3Error("", "Policy document length exceeds the maximum allowed limit.")
	}

	policy, err := ParsePolicyDocument(policyDoc)
	if err != nil {
		return err
	}

	return policy.Validate()
}

func stringyfyAny(input any) []string {
	switch val := input.(type) {
	case string:
		return []string{val}
	case []string:
		return val
	case []any:
		result := make([]string, 0, len(val))
		for i := range val {
			strVal, ok := val[i].(string)
			if ok {
				result = append(result, strVal)
			}
		}
		return result
	default:
		return []string{}
	}
}
