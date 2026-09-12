package iam

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	cachestore "github.com/avatar31/halmidi/internal/cache_store"
	"github.com/avatar31/halmidi/internal/core/s3common"
	"github.com/avatar31/halmidi/internal/logger"
	"github.com/avatar31/halmidi/utils"
)

const (
	defaultPolicyCacheTTL = 15 * time.Minute
)

type AuthzEngine struct {
	cache *cachestore.Cache
}

type policyStatement struct {
	// Action list in map format for O(1) lookup
	Actions    map[s3common.PolicyAction]struct{}
	Conditions policyStmtConditionMap
}

type authorizedActions struct {
	// Key: "resource" -> effect -> statement
	// Applicable for statements with specific resources
	// Ex: Deny s3:Delete* on arn:aws:s3:::prodbucket/myobject, Allow s3:Delete* on arn:aws:s3:::devBucket/myobject etc
	WithSpecificResourceStmt map[string]map[s3common.PolicyStatementEffect]*policyStatement

	// Key: effect -> statement (resource: nil)
	// Applicable for statements with actions where Resource is not applicable
	// Ex: ListBuckets, ListUsers etc.
	NoResourceStmt map[s3common.PolicyStatementEffect]*policyStatement

	// Key: effect -> statement (resource: *)
	// Applicable for statements with actions where Resource is applicable but set to wildcard
	// Ex: Allow s3:* on resource *, Deny iam:PassRole on resource *
	AllResourcesStmt map[s3common.PolicyStatementEffect]*policyStatement
}

type policyStmtConditionMap map[s3common.PolicyCondOp]map[string][]string

func (m policyStmtConditionMap) DeepCopy() policyStmtConditionMap {
	if m == nil {
		return nil
	}

	result := make(policyStmtConditionMap, len(m))

	for k1, inner := range m {
		newInner := make(map[string][]string, len(inner))

		for k2, slice := range inner {
			newSlice := make([]string, len(slice))
			copy(newSlice, slice)
			newInner[k2] = newSlice
		}

		result[k1] = newInner
	}

	return result
}

var (
	engine *AuthzEngine
	once   sync.Once
	sfg    = new(singleflight.Group)
)

func GetAuthzEngine() *AuthzEngine {
	once.Do(func() {
		engine = &AuthzEngine{cache: cachestore.GetCacheStore()}
	})

	return engine
}

func (e *AuthzEngine) Reset(ctx context.Context, userName string) {
	if e.cache != nil {
		e.cache.Delete(userName)
		logger.GetLogger(ctx).Debugf("Reset policy engine cache for user %s", userName)
	}
}

func (e *AuthzEngine) EvaluateRequest(req http.Request, action s3common.PolicyAction, resource string) error {
	if action == "" {
		return s3common.GetInvalidArgumentS3Error("", "action cannot be empty")
	}

	ctx := req.Context()
	log := logger.GetLogger(ctx)
	userName, ok := ctx.Value(utils.UserNameCtxKey).(string)
	if !ok {
		log.Error("UserName not found in context")
		return s3common.GetAccessDeniedS3Error("")
	}

	if e.cache != nil {
		cachedata, found := e.cache.Get(userName)
		if found {
			authzData, ok := cachedata.(authorizedActions)
			if ok {
				return e.checkAccess(ctx, authzData, req, action, resource)
			}
			// Cache data corrupted - delete and refetch
			e.cache.Delete(userName)
			log.Warningf("Cache data for user %s is corrupted, refetching policies", userName)
		}
	}

	log.Debugf("Cache miss for user %s, fetching policies", userName)

	_authzData, err, _ := sfg.Do(userName, func() (any, error) {
		startTime := time.Now()
		policies, err := e.getUserPolicies(ctx, userName)
		if err != nil {
			return nil, err
		}

		authzData := e.calculateAuthorizedActions(policies)
		duration := time.Since(startTime)
		if duration > 10*time.Millisecond {
			log.Warningf("Slow policy calculation: %v for user %s with %d policies", duration, userName, len(policies))
		}

		if e.cache != nil {
			e.cache.Set(userName, authzData, defaultPolicyCacheTTL)
		}

		return authzData, nil
	})

	if err != nil {
		return err
	}

	authzData, ok := _authzData.(authorizedActions)
	if !ok {
		log.Errorf("Unexpected type from singleflight for user %s: %T", userName, _authzData)
		return s3common.GetInternalErrorS3Error("")
	}

	return e.checkAccess(ctx, authzData, req, action, resource)
}

func (e *AuthzEngine) getUserPolicies(ctx context.Context, userName string) ([]PolicyDocument, error) {
	log := logger.GetLogger(ctx)
	entity, err := NewUserService(ctx).GetUserEntity(ctx, userName)
	if err != nil {
		return nil, err
	}

	userPolicies, err := entity.ListUserPolicies(ctx, ListIAMResourceOptions{})
	if err != nil {
		return nil, err
	}

	userGroups, _, err := entity.ListGroupsForUser(ctx, ListIAMResourceOptions{})
	if err != nil {
		return nil, err
	}

	// Performance: Pre-allocate slice with estimated capacity
	docs := make([]PolicyDocument, 0, len(userPolicies)+len(userGroups)*2)

	for _, group := range userGroups {
		grpEntity, err := NewGroupService(ctx).GetGroupEntity(ctx, group.Name)
		if err != nil {
			log.WithError(err).Warningf("Failed to get group entity for group %s, skipping", group.Name)
			continue
		}

		groupPolicies, err := grpEntity.ListGroupPolicies(ctx, ListIAMResourceOptions{})
		if err != nil {
			return nil, err
		}
		userPolicies = append(userPolicies, groupPolicies...)
	}

	for _, pol := range userPolicies {
		policyDoc, err := ParsePolicyDocument(pol.DefaultPolicyDocument)
		if err != nil {
			log.WithError(err).Warningf("Failed to parse policy document %s for user %s, skipping",
				pol.Arn, userName)
			continue
		}
		docs = append(docs, *policyDoc)
	}

	return docs, nil
}

func (e *AuthzEngine) calculateAuthorizedActions(policies []PolicyDocument) authorizedActions {
	authz := authorizedActions{
		WithSpecificResourceStmt: map[string]map[s3common.PolicyStatementEffect]*policyStatement{},
		AllResourcesStmt: map[s3common.PolicyStatementEffect]*policyStatement{
			s3common.PolicyStatementEffectAllow: nil,
			s3common.PolicyStatementEffectDeny:  nil,
		},
		NoResourceStmt: map[s3common.PolicyStatementEffect]*policyStatement{
			s3common.PolicyStatementEffectAllow: nil,
			s3common.PolicyStatementEffectDeny:  nil,
		},
	}

	for i := range policies {
		for _, stmt := range policies[i].Statement {
			if stmt.Resource != nil {
				resources := stringyfyAny(stmt.Resource)
				if len(resources) == 0 {
					continue
				}

				allowedActions, notAllowedActions := e.calculateAllowedAndNotAllowedActions(stmt, resources)
				for _, resource := range resources {
					if resource == s3common.AllActionsOrResourcePattern {
						e.assignStmtEffectMap(stmt, authz.AllResourcesStmt, allowedActions, notAllowedActions)
						continue
					}

					if _, ok := authz.WithSpecificResourceStmt[resource]; !ok {
						authz.WithSpecificResourceStmt[resource] = map[s3common.PolicyStatementEffect]*policyStatement{
							stmt.Effect: {
								Actions:    map[s3common.PolicyAction]struct{}{},
								Conditions: policyStmtConditionMap{},
							},
						}
					}

					e.assignStmtEffectMap(stmt, authz.WithSpecificResourceStmt[resource], allowedActions, notAllowedActions)
				}
			}

			if stmt.NotResource != nil {
				notResources := stringyfyAny(stmt.NotResource)
				if len(notResources) == 0 {
					continue
				}

				allowedActions, notAllowedActions := e.calculateAllowedAndNotAllowedActions(stmt, notResources)
				for _, resource := range notResources {
					if resource == s3common.AllActionsOrResourcePattern {
						// {Effect: "Allow", Action: []string{"s3:Delete*"}, NotResource: "*"}
						// {Effect: "Deny", Action: []string{"s3:Get*"}, NotResource: "*"}
						if stmt.Effect == s3common.PolicyStatementEffectAllow {
							notAllowedActions = allowedActions
							allowedActions = e.negateActions(e.getServiceFilteredActions(notResources), notAllowedActions)
						}
						e.assignStmtEffectMap(stmt, authz.AllResourcesStmt, allowedActions, notAllowedActions)
						continue
					}

					if _, ok := authz.WithSpecificResourceStmt[resource]; !ok {
						authz.WithSpecificResourceStmt[resource] = map[s3common.PolicyStatementEffect]*policyStatement{
							stmt.Effect: {
								Actions:    map[s3common.PolicyAction]struct{}{},
								Conditions: policyStmtConditionMap{},
							},
						}
					}

					// {Effect: "Allow", Action: "Delete*", NotResource: "arn:aws:s3:::prodbucket/*"}
					// {Effect: "Deny", Action: "Delete*", NotResource: "arn:aws:s3:::devbucket/*"}
					// Store allowed actions in all-resources statement and
					// not-allowed actions in specific resource statement to simplify access check logic later
					e.assignStmtEffectMap(stmt, authz.AllResourcesStmt, allowedActions, notAllowedActions)
					e.assignStmtEffectMap(stmt, authz.WithSpecificResourceStmt[resource], notAllowedActions, allowedActions)
				}
			}

			if stmt.Resource == nil && stmt.NotResource == nil {
				if authz.NoResourceStmt[stmt.Effect] == nil {
					authz.NoResourceStmt[stmt.Effect] = &policyStatement{
						Actions:    map[s3common.PolicyAction]struct{}{},
						Conditions: policyStmtConditionMap{},
					}
				}

				actions := e.extractActionsFromStatement(stmt, []string{})
				for action := range actions {
					authz.NoResourceStmt[stmt.Effect].Actions[normalizeAction(action)] = struct{}{}
				}

				temp := authz.NoResourceStmt[stmt.Effect]
				temp.Conditions = e.mergeConditions(temp.Conditions, stmt.Condition)
				authz.NoResourceStmt[stmt.Effect] = temp
			}
		}
	}

	return authz
}

func (e *AuthzEngine) assignStmtEffectMap(stmt *Statement, effectMap map[s3common.PolicyStatementEffect]*policyStatement,
	allowedActions, notAllowedActions map[s3common.PolicyAction]struct{}) {
	if allowedActions != nil {
		effect := s3common.PolicyStatementEffectAllow
		if effectMap[effect] == nil {
			effectMap[effect] = &policyStatement{
				Actions:    map[s3common.PolicyAction]struct{}{},
				Conditions: policyStmtConditionMap{},
			}
		}

		for action := range allowedActions {
			effectMap[effect].Actions[normalizeAction(action)] = struct{}{}
		}

		effectMap[effect].Conditions = e.mergeConditions(effectMap[effect].Conditions, stmt.Condition)
	}

	if notAllowedActions != nil {
		effect := s3common.PolicyStatementEffectDeny
		if effectMap[effect] == nil {
			effectMap[effect] = &policyStatement{
				Actions:    map[s3common.PolicyAction]struct{}{},
				Conditions: policyStmtConditionMap{},
			}
		}

		for action := range notAllowedActions {
			effectMap[effect].Actions[normalizeAction(action)] = struct{}{}
		}

		effectMap[effect].Conditions = e.mergeConditions(effectMap[effect].Conditions, stmt.Condition)
	}
}

func (e *AuthzEngine) mergeConditions(existing policyStmtConditionMap,
	new map[s3common.PolicyCondOp]map[string]any) policyStmtConditionMap {
	merged := existing.DeepCopy()

	for op := range new {
		if _, ok := merged[op]; !ok {
			merged[op] = make(map[string][]string)
		}
		for key := range new[op] {
			values := stringyfyAny(new[op][key])
			merged[op][key] = append(merged[op][key], values...)
		}
	}
	return merged
}

func (e *AuthzEngine) calculateAllowedAndNotAllowedActions(stmt *Statement, resources []string) (map[s3common.PolicyAction]struct{}, map[s3common.PolicyAction]struct{}) {
	// Case 1: Effect: Allow, Action: s3:GetObject, Resource: *
	// 		Store s3:GetObject in authz.AllResourcesStmt with Effect: Allow
	// Case 2: Effect: Allow, NotAction: s3:DeleteObject, Resource: *
	// 		Store (AllSupportedActions - DeleteObject) in authz.AllResourcesStmt with Effect: Allow
	// Case 3: Effect: Deny, Action: s3:DeleteObject, Resource: *
	// 		Store s3:DeleteObject in authz.AllResourcesStmt with Effect: Deny
	// Case 4: Effect: Deny, NotAction: s3:GetObject, Resource: *
	// 		Store (AllSupportedActions - GetObject) in authz.AllResourcesStmt with Effect: Deny
	// 			and Store GetObject in authz.AllResourcesStmt with Effect: Allow

	actions := e.extractActionsFromStatement(stmt, resources)

	// Effect: Allow
	if stmt.Effect == s3common.PolicyStatementEffectAllow {
		if stmt.Action != nil {
			// Case 1
			return actions, nil
		}

		// Case 2
		allowedActions := e.negateActions(e.getServiceFilteredActions(resources), actions)
		return allowedActions, nil
	}

	// Effect: Deny
	if stmt.Action != nil {
		// Case 3
		return nil, actions
	}

	// Case 4
	notAllowedActions := e.negateActions(e.getServiceFilteredActions(resources), actions)
	return actions, notAllowedActions
}

func (e *AuthzEngine) extractActionsFromStatement(stmt *Statement, resources []string) map[s3common.PolicyAction]struct{} {
	finalActions := make(map[s3common.PolicyAction]struct{})

	if stmt.Action != nil {
		actions := stringyfyAny(stmt.Action)
		for i := range actions {
			for action := range e.expandAction(actions[i], resources) {
				finalActions[action] = struct{}{}
			}
		}
	}

	if stmt.NotAction != nil {
		notActions := stringyfyAny(stmt.NotAction)
		for i := range notActions {
			for action := range e.expandAction(notActions[i], resources) {
				finalActions[action] = struct{}{}
			}
		}
	}

	return finalActions
}

func (e *AuthzEngine) expandAction(action string, resources []string) map[s3common.PolicyAction]struct{} {
	if strings.HasSuffix(action, ":*") {
		// Handle all operation for a service, e.g. "s3:*" or "iam:*"
		if action == s3common.AllS3ActionsPattern {
			actions := make(map[s3common.PolicyAction]struct{})
			for i := range s3common.AllS3SupportedActions {
				actions[s3common.AllS3SupportedActions[i]] = struct{}{}
			}
			return actions
		}

		if action == s3common.AllIAMActionsPattern {
			actions := make(map[s3common.PolicyAction]struct{})
			for i := range s3common.AllIAMSupportedActions {
				actions[s3common.AllIAMSupportedActions[i]] = struct{}{}
			}
			return actions
		}
	}

	serviceFilteredActions := e.getServiceFilteredActions(resources)
	if action == s3common.AllActionsOrResourcePattern {
		// Handle "*"
		actions := make(map[s3common.PolicyAction]struct{})
		for i := range serviceFilteredActions {
			actions[serviceFilteredActions[i]] = struct{}{}
		}
		return actions
	}

	if strings.ContainsAny(action, "*?") {
		// Handle wildcard in action, e.g. "s3:Get*" or "s3:Put*" etc.
		actionPattern := action
		matched := make(map[s3common.PolicyAction]struct{})
		for i := range serviceFilteredActions {
			if e.matchAction(actionPattern, string(serviceFilteredActions[i])) {
				matched[serviceFilteredActions[i]] = struct{}{}
			}
		}
		return matched
	}

	return map[s3common.PolicyAction]struct{}{s3common.PolicyAction(action): {}}
}

func (e *AuthzEngine) getServiceFilteredActions(resources []string) []s3common.PolicyAction {
	if len(resources) == 0 || slices.Contains(resources, s3common.AllActionsOrResourcePattern) {
		return s3common.AllSupportedActions
	}

	serviceFilteredActions := make([]s3common.PolicyAction, 0, len(s3common.AllSupportedActions))
	for i := range resources {
		if strings.HasPrefix(resources[i], "arn:aws:s3:::") {
			serviceFilteredActions = append(serviceFilteredActions, s3common.AllS3SupportedActions...)
			break
		}
	}

	for i := range resources {
		if strings.HasPrefix(resources[i], "arn:aws:iam::") {
			serviceFilteredActions = append(serviceFilteredActions, s3common.AllIAMSupportedActions...)
			break
		}
	}

	return serviceFilteredActions
}

// matchAction implements IAM Action matching
// Actions are case-insensitive
func (e *AuthzEngine) matchAction(pattern, action string) bool {
	return matchIAM(strings.ToLower(pattern), strings.ToLower(action))
}

// matchResource implements IAM Resource matching
// Resources are generally case-sensitive
func (e *AuthzEngine) matchResource(pattern, resource string) bool {
	return matchIAM(pattern, resource)
}

func (e *AuthzEngine) negateActions(allActions []s3common.PolicyAction,
	actions map[s3common.PolicyAction]struct{}) map[s3common.PolicyAction]struct{} {
	negated := make(map[s3common.PolicyAction]struct{})
	for i := range allActions {
		if _, ok := actions[allActions[i]]; !ok {
			negated[allActions[i]] = struct{}{}
		}
	}

	return negated
}

func (e *AuthzEngine) checkAccess(ctx context.Context, authz authorizedActions, req http.Request,
	action s3common.PolicyAction, resource string) error {
	normAction := normalizeAction(action)
	err := e.checkDenyStmt(ctx, authz, req, normAction, resource)
	if err != nil {
		return err
	}

	accessGranted := e.checkAllowStmt(ctx, authz, req, normAction, resource)
	if accessGranted {
		return nil
	}

	logger.GetLogger(ctx).Debugf("Access DENIED by implicit DENY: action=%s, resource=%s", action, resource)
	return s3common.GetAccessDeniedS3Error("")
}

// checkDenyStmt checks for explicit deny statements.
func (e *AuthzEngine) checkDenyStmt(ctx context.Context, authz authorizedActions, req http.Request,
	action s3common.PolicyAction, resource string) error {
	log := logger.GetLogger(ctx)

	// Check specific resource ALLOW/DENY statements (exact match)
	// Ex: Deny s3:DeleteObject on arn:aws:s3:::prodbucket/myobject,
	// 		Allow s3:DeleteObject on arn:aws:s3:::devbucket/myobject etc
	if stmts, ok := authz.WithSpecificResourceStmt[resource]; ok {
		if denyStmt := stmts[s3common.PolicyStatementEffectDeny]; denyStmt != nil {
			_, hasAction := denyStmt.Actions[action]
			if hasAction && e.evaluateConditions(denyStmt.Conditions, req) {
				log.Debugf("Access DENIED by specific resource (exact): action=%s, resource=%s", action, resource)
				return s3common.GetAccessDeniedS3Error("")
			}
		}
	}

	for resourcePattern, stmts := range authz.WithSpecificResourceStmt {
		if resourcePattern == resource {
			continue // Already checked exact match
		}

		if e.matchResource(resourcePattern, resource) {
			if denyStmt := stmts[s3common.PolicyStatementEffectDeny]; denyStmt != nil {
				_, hasAction := denyStmt.Actions[action]
				if hasAction && e.evaluateConditions(denyStmt.Conditions, req) {
					log.Debugf("Access DENIED by specific resource (wildcard): pattern=%s, action=%s, resource=%s",
						resourcePattern, action, resource)
					return s3common.GetAccessDeniedS3Error("")
				}
			}
		}
	}

	if denyStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectDeny]; denyStmt != nil {
		_, hasAction := denyStmt.Actions[action]
		if hasAction && e.evaluateConditions(denyStmt.Conditions, req) {
			log.Debugf("Access DENIED by all-resources (*): action=%s, resource=%s", action, resource)
			return s3common.GetAccessDeniedS3Error("")
		}
	}

	if denyStmt := authz.NoResourceStmt[s3common.PolicyStatementEffectDeny]; denyStmt != nil {
		_, hasAction := denyStmt.Actions[action]
		if hasAction && e.evaluateConditions(denyStmt.Conditions, req) {
			log.Debugf("Access DENIED by no-resources (*): action=%s, resource=%s", action, resource)
			return s3common.GetAccessDeniedS3Error("")
		}
	}

	return nil // No explicit deny found in any statements
}

// checkAllowStmt checks for explicit allow statements.
func (e *AuthzEngine) checkAllowStmt(ctx context.Context, authz authorizedActions, req http.Request,
	action s3common.PolicyAction, resource string) bool {
	log := logger.GetLogger(ctx)

	if stmts, ok := authz.WithSpecificResourceStmt[resource]; ok {
		if allowStmt := stmts[s3common.PolicyStatementEffectAllow]; allowStmt != nil {
			_, hasAction := allowStmt.Actions[action]
			if hasAction && e.evaluateConditions(allowStmt.Conditions, req) {
				log.Debugf("Access GRANTED by specific resource (exact): action=%s, resource=%s", action, resource)
				return true // Access granted!
			}
		}
	}

	for resourcePattern, stmts := range authz.WithSpecificResourceStmt {
		if resourcePattern == resource {
			continue // Already checked exact match
		}

		if e.matchResource(resourcePattern, resource) {
			if allowStmt := stmts[s3common.PolicyStatementEffectAllow]; allowStmt != nil {
				_, hasAction := allowStmt.Actions[action]
				if hasAction && e.evaluateConditions(allowStmt.Conditions, req) {
					log.Debugf("Access GRANTED by specific resource (wildcard): pattern=%s, action=%s, resource=%s",
						resourcePattern, action, resource)
					return true // Access granted!
				}
			}
		}
	}

	if allowStmt := authz.AllResourcesStmt[s3common.PolicyStatementEffectAllow]; allowStmt != nil {
		_, hasAction := allowStmt.Actions[action]
		if hasAction && e.evaluateConditions(allowStmt.Conditions, req) {
			log.Debugf("Access GRANTED by all-resources (*): action=%s, resource=%s", action, resource)
			return true // Access granted!
		}
	}

	if allowStmt := authz.NoResourceStmt[s3common.PolicyStatementEffectAllow]; allowStmt != nil {
		_, hasAction := allowStmt.Actions[action]
		if hasAction && e.evaluateConditions(allowStmt.Conditions, req) {
			log.Debugf("Access GRANTED by no-resources (*): action=%s, resource=%s", action, resource)
			return true // Access granted!
		}
	}

	return false // No explicit allow found in any statements
}

// evaluateConditions evaluates all conditions for a policy statement
// Returns true if all conditions pass (AND logic), false otherwise
func (e *AuthzEngine) evaluateConditions(conditions policyStmtConditionMap, req http.Request) bool {
	if len(conditions) == 0 {
		return true
	}

	for condOp, condMap := range conditions {
		for key, expectedValues := range condMap {
			// Get actual value from request
			actualValue := e.getConditionValueFromRequest(key, req)

			// Evaluate the condition
			if !e.evaluateSingleCondition(condOp, actualValue, expectedValues) {
				return false // Condition failed
			}
		}
	}

	return true
}

func (e *AuthzEngine) getConditionValueFromRequest(key string, req http.Request) any {
	// TODO: Are we covering all possible condition keys that S3 supports? Add more as needed
	switch key {
	case "aws:SourceIp":
		return e.getClientIP(req)
	case "aws:UserAgent":
		return req.Header.Get("User-Agent")
	case "aws:SecureTransport":
		if req.TLS != nil {
			return "true"
		}
		return "false"
	case "aws:CurrentTime":
		return e.getCurrentTime()
	case "aws:EpochTime":
		return e.getEpochTime()
	case "aws:Referer":
		return req.Header.Get("Referer")
	case "s3:prefix":
		return req.URL.Query().Get("prefix")
	case "s3:delimiter":
		return req.URL.Query().Get("delimiter")
	case "s3:max-keys":
		return req.URL.Query().Get("max-keys")
	case "s3:x-amz-acl":
		return req.Header.Get("x-amz-acl")
	case "s3:x-amz-grant-read", "s3:x-amz-grant-write", "s3:x-amz-grant-full-control":
		return req.Header.Get(key)
	case "s3:x-amz-server-side-encryption":
		return req.Header.Get("x-amz-server-side-encryption")
	case "s3:x-amz-storage-class":
		return req.Header.Get("x-amz-storage-class")
	case "aws:PrincipalArn":
		return getPrinicipalArn(req)
	default:
		if val := req.Header.Get(key); val != "" {
			return val
		}
		return req.URL.Query().Get(key)
	}
}

// evaluateSingleCondition evaluates a single condition operator
func (e *AuthzEngine) evaluateSingleCondition(condOp s3common.PolicyCondOp, actualValue any,
	expectedValues []string) bool {

	// Handle IfExists suffix
	hasIfExists := strings.HasSuffix(string(condOp), string(s3common.CondOpIfExistsSuffix))
	if hasIfExists {
		if actualValue == nil || actualValue == "" {
			return true // Condition passes if key doesn't exist and IfExists is set
		}
		condOp = s3common.PolicyCondOp(strings.TrimSuffix(string(condOp), string(s3common.CondOpIfExistsSuffix)))
	} else {
		if actualValue == nil {
			return false // Key must exist if IfExists is not set
		}
	}

	actualStr := e.anyToString(actualValue)

	// Handle ForAllValues and ForAnyValue prefixes
	if after, ok := strings.CutPrefix(string(condOp), string(s3common.CondOpForAllValuesPrefix)); ok {
		condOp = s3common.PolicyCondOp(after)
		return e.evaluateForAllValues(condOp, actualStr, expectedValues)
	}

	if after, ok := strings.CutPrefix(string(condOp), string(s3common.CondOpForAnyValuePrefix)); ok {
		condOp = s3common.PolicyCondOp(after)
		return e.evaluateForAnyValue(condOp, actualStr, expectedValues)
	}

	// Evaluate based on condition operator type
	// At least one expected value must match
	for _, expectedValue := range expectedValues {
		if e.evaluateConditionOperator(condOp, actualStr, expectedValue) {
			return true
		}
	}

	return false
}

// evaluateConditionOperator evaluates a condition based on operator type
func (e *AuthzEngine) evaluateConditionOperator(condOp s3common.PolicyCondOp, actual, expected string) bool {
	switch {
	case strings.HasPrefix(string(condOp), "String"):
		return e.evaluateStringCondition(condOp, actual, expected)
	case strings.HasPrefix(string(condOp), "Numeric"):
		return e.evaluateNumericCondition(condOp, actual, expected)
	case strings.HasPrefix(string(condOp), "Date"):
		return e.evaluateDateCondition(condOp, actual, expected)
	case condOp == s3common.CondOpBool:
		return e.evaluateBoolCondition(actual, expected)
	case condOp == s3common.CondOpBinaryEquals:
		return e.evaluateBinaryCondition(actual, expected)
	case condOp == s3common.CondOpIpAddress || condOp == s3common.CondOpNotIpAddress:
		return e.evaluateIpCondition(condOp, actual, expected)
	case strings.HasPrefix(string(condOp), "Arn"):
		return e.evaluateArnCondition(condOp, actual, expected)
	case condOp == s3common.CondOpNull:
		return e.evaluateNullCondition(actual, expected)
	default:
		return false
	}
}

// Helper methods for condition evaluation
func (e *AuthzEngine) evaluateStringCondition(condOp s3common.PolicyCondOp, actual, expected string) bool {
	switch condOp {
	case s3common.CondOpStringEquals:
		return actual == expected
	case s3common.CondOpStringNotEquals:
		return actual != expected
	case s3common.CondOpStringEqualsIgnoreCase:
		return strings.EqualFold(actual, expected)
	case s3common.CondOpStringNotEqualsIgnoreCase:
		return !strings.EqualFold(actual, expected)
	case s3common.CondOpStringLike:
		return e.matchResource(expected, actual)
	case s3common.CondOpStringNotLike:
		return !e.matchResource(expected, actual)
	default:
		return false
	}
}

func (e *AuthzEngine) evaluateNumericCondition(condOp s3common.PolicyCondOp, actual, expected string) bool {
	actualNum, err1 := e.stringToFloat64(actual)
	expectedNum, err2 := e.stringToFloat64(expected)
	if err1 != nil || err2 != nil {
		return false
	}

	switch condOp {
	case s3common.CondOpNumericEquals:
		return actualNum == expectedNum
	case s3common.CondOpNumericNotEquals:
		return actualNum != expectedNum
	case s3common.CondOpNumericLessThan:
		return actualNum < expectedNum
	case s3common.CondOpNumericLessThanEquals:
		return actualNum <= expectedNum
	case s3common.CondOpNumericGreaterThan:
		return actualNum > expectedNum
	case s3common.CondOpNumericGreaterThanEquals:
		return actualNum >= expectedNum
	default:
		return false
	}
}

func (e *AuthzEngine) evaluateDateCondition(condOp s3common.PolicyCondOp, actual, expected string) bool {
	actualTime, err1 := e.stringToTime(actual)
	expectedTime, err2 := e.stringToTime(expected)
	if err1 != nil || err2 != nil {
		return false
	}

	switch condOp {
	case s3common.CondOpDateEquals:
		return actualTime.Equal(expectedTime)
	case s3common.CondOpDateNotEquals:
		return !actualTime.Equal(expectedTime)
	case s3common.CondOpDateLessThan:
		return actualTime.Before(expectedTime)
	case s3common.CondOpDateLessThanEquals:
		return actualTime.Before(expectedTime) || actualTime.Equal(expectedTime)
	case s3common.CondOpDateGreaterThan:
		return actualTime.After(expectedTime)
	case s3common.CondOpDateGreaterThanEquals:
		return actualTime.After(expectedTime) || actualTime.Equal(expectedTime)
	default:
		return false
	}
}

func (e *AuthzEngine) evaluateBoolCondition(actual, expected string) bool {
	actualBool, err1 := e.stringToBool(actual)
	expectedBool, err2 := e.stringToBool(expected)
	return err1 == nil && err2 == nil && actualBool == expectedBool
}

func (e *AuthzEngine) evaluateBinaryCondition(actual, expected string) bool {
	return actual == expected
}

func (e *AuthzEngine) evaluateIpCondition(condOp s3common.PolicyCondOp, actual, expected string) bool {
	actualIP := e.parseIP(actual)
	if actualIP == nil {
		return false
	}

	_, ipNet, err := e.parseCIDR(expected)
	if err != nil {
		expectedIP := e.parseIP(expected)
		if expectedIP == nil {
			return false
		}
		result := actualIP.Equal(expectedIP)
		return condOp == s3common.CondOpIpAddress == result
	}

	result := ipNet.Contains(actualIP)
	return condOp == s3common.CondOpIpAddress == result
}

func (e *AuthzEngine) evaluateArnCondition(condOp s3common.PolicyCondOp, actual, expected string) bool {
	switch condOp {
	case s3common.CondOpArnEquals:
		return actual == expected
	case s3common.CondOpArnNotEquals:
		return actual != expected
	case s3common.CondOpArnLike:
		return e.matchResource(expected, actual)
	case s3common.CondOpArnNotLike:
		return !e.matchResource(expected, actual)
	default:
		return false
	}
}

func (e *AuthzEngine) evaluateNullCondition(actual, expected string) bool {
	expectedBool, err := e.stringToBool(expected)
	if err != nil {
		return false
	}
	isNull := actual == ""
	return isNull == expectedBool
}

func (e *AuthzEngine) evaluateForAllValues(condOp s3common.PolicyCondOp, actual string, expectedValues []string) bool {
	if actual == "" {
		return true // Empty set satisfies ForAllValues
	}

	// All actual values must match at least one expected value
	for actualVal := range strings.SplitSeq(actual, ",") {
		actualVal = strings.TrimSpace(actualVal)
		matched := false
		for _, expectedVal := range expectedValues {
			if e.evaluateConditionOperator(condOp, actualVal, expectedVal) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func (e *AuthzEngine) evaluateForAnyValue(condOp s3common.PolicyCondOp, actual string, expectedValues []string) bool {
	if actual == "" {
		return false
	}

	// At least one actual value must match at least one expected value
	for actualVal := range strings.SplitSeq(actual, ",") {
		actualVal = strings.TrimSpace(actualVal)
		for _, expectedVal := range expectedValues {
			if e.evaluateConditionOperator(condOp, actualVal, expectedVal) {
				return true
			}
		}
	}
	return false
}

// Type conversion helpers
func (e *AuthzEngine) anyToString(val any) string {
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

func (e *AuthzEngine) stringToFloat64(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func (e *AuthzEngine) stringToTime(s string) (time.Time, error) {
	// Try RFC3339 format
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Try epoch time
	if epoch, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(epoch, 0), nil
	}
	return time.Time{}, fmt.Errorf("cannot parse time")
}

func (e *AuthzEngine) stringToBool(s string) (bool, error) {
	return strconv.ParseBool(s)
}

func (e *AuthzEngine) parseIP(s string) net.IP {
	return net.ParseIP(s)
}

func (e *AuthzEngine) parseCIDR(s string) (net.IP, *net.IPNet, error) {
	return net.ParseCIDR(s)
}

func (e *AuthzEngine) getClientIP(req http.Request) string {
	ip, _, _ := net.SplitHostPort(req.RemoteAddr)
	return ip
}

func (e *AuthzEngine) getCurrentTime() string {
	return utils.ConvertTimeToString(time.Now())
}

func (e *AuthzEngine) getEpochTime() string {
	return fmt.Sprintf("%d", time.Now().Unix())
}

// matchIAM is the core wildcard matcher
// Supports '*' = any sequence of chars, '?' = single char
func matchIAM(pattern, str string) bool {
	p, s := 0, 0
	starIdx, match := -1, 0
	for s < len(str) {
		if p < len(pattern) && (pattern[p] == str[s] || pattern[p] == '?') {
			// character matches or single-character wildcard
			p++
			s++
		} else if p < len(pattern) && pattern[p] == '*' {
			// found a star, remember position and move pattern forward
			starIdx = p
			match = s
			p++
		} else if starIdx != -1 {
			// previously found star, backtrack
			p = starIdx + 1
			match++
			s = match
		} else {
			// no match possible
			return false
		}
	}

	// skip trailing stars in pattern
	for p < len(pattern) && pattern[p] == '*' {
		p++
	}

	return p == len(pattern)
}

func normalizeAction(action s3common.PolicyAction) s3common.PolicyAction {
	return s3common.PolicyAction(strings.ToLower(string(action)))
}

var getPrinicipalArn = func(req http.Request) string {
	ctx := req.Context()
	userName, _ := ctx.Value(utils.UserNameCtxKey).(string)
	if userName == "" {
		return ""
	}
	user, err := NewUserService(ctx).GetUserEntity(ctx, userName)
	if err != nil {
		return ""
	}

	return user.meta.Arn
}
