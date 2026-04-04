package permission

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Manager manages permission rules from multiple sources
type Manager struct {
	mu       sync.RWMutex
	ruleSets map[Source]*RuleSet
	classifier Classifier
}

// NewManager creates a new permission manager
func NewManager() *Manager {
	return &Manager{
		ruleSets: make(map[Source]*RuleSet),
	}
}

// SetClassifier sets the auto-classifier
func (m *Manager) SetClassifier(classifier Classifier) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.classifier = classifier
}

// AddRule adds a rule to a source
func (m *Manager) AddRule(source Source, rule *Rule) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rule.Source = source
	if rule.ID == "" {
		rule.ID = generateRuleID()
	}
	rule.CreatedAt = time.Now()

	set, ok := m.ruleSets[source]
	if !ok {
		set = &RuleSet{Source: source}
		m.ruleSets[source] = set
	}

	set.Rules = append(set.Rules, rule)
}

// RemoveRule removes a rule by ID
func (m *Manager) RemoveRule(ruleID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, set := range m.ruleSets {
		for i, rule := range set.Rules {
			if rule.ID == ruleID {
				set.Rules = append(set.Rules[:i], set.Rules[i+1:]...)
				return true
			}
		}
	}

	return false
}

// ClearSessionRules clears all session-only rules
func (m *Manager) ClearSessionRules() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for source, set := range m.ruleSets {
		var filtered []*Rule
		for _, rule := range set.Rules {
			if !rule.SessionOnly {
				filtered = append(filtered, rule)
			}
		}
		set.Rules = filtered

		// Remove empty rule sets
		if len(set.Rules) == 0 {
			delete(m.ruleSets, source)
		}
	}
}

// GetRules returns all rules sorted by priority
func (m *Manager) GetRules() []*Rule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allRules []*Rule
	for _, set := range m.ruleSets {
		allRules = append(allRules, set.Rules...)
	}

	// Sort by source priority (highest first)
	sort.Slice(allRules, func(i, j int) bool {
		return allRules[i].Source.GetPriority() > allRules[j].Source.GetPriority()
	})

	return allRules
}

// GetRulesForTool returns rules that match a tool name
func (m *Manager) GetRulesForTool(toolName string) []*Rule {
	rules := m.GetRules()
	var matching []*Rule

	for _, rule := range rules {
		if rule.MatchesTool(toolName) {
			matching = append(matching, rule)
		}
	}

	return matching
}

// Check checks permission for a request
func (m *Manager) Check(req *PermissionRequest, ctx *PermissionContext) (*PermissionResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get matching rules sorted by priority
	rules := m.GetRulesForTool(req.ToolName)

	// Process rules in priority order
	for _, rule := range rules {
		if !m.ruleMatchesParameters(rule, req) {
			continue
		}

		decision := &Decision{
			Mode:   rule.Mode,
			Rule:   rule,
			Reason: rule.Reason,
		}

		switch rule.Mode {
		case ModeDeny:
			return &PermissionResult{
				Decision: decision,
				Denied:   true,
				Message:  rule.Reason,
			}, nil

		case ModeAccept:
			return &PermissionResult{
				Decision: decision,
				Allowed:  true,
			}, nil

		case ModeAsk:
			return &PermissionResult{
				Decision:          decision,
				RequiresUserInput: true,
				Ask:               true,
				Message:           rule.Reason,
			}, nil

		case ModeAuto:
			// Use classifier
			if m.classifier != nil {
				result, err := m.classifier.Classify(req, ctx)
				if err != nil {
					return nil, err
				}

				decision := &Decision{
					Mode:       result.Mode,
					IsAuto:     true,
					Confidence: result.Confidence,
					Reason:     result.Reason,
				}

				switch result.Mode {
				case ModeAccept:
					return &PermissionResult{
						Decision: decision,
						Allowed:  true,
					}, nil

				case ModeDeny:
					return &PermissionResult{
						Decision: decision,
						Denied:   true,
						Message:  result.Reason,
					}, nil

				default:
					return &PermissionResult{
						Decision:          decision,
						RequiresUserInput: true,
						Ask:               true,
						Message:           result.Reason,
					}, nil
				}
			}

			// No classifier, default to ask
			return &PermissionResult{
				Decision:          decision,
				RequiresUserInput: true,
				Ask:               true,
			}, nil
		}
	}

	// No matching rule, default to ask
	return &PermissionResult{
		RequiresUserInput: true,
		Ask:               true,
		Message:           fmt.Sprintf("No permission rule found for %s", req.ToolName),
	}, nil
}

// ruleMatchesParameters checks if a rule matches the request parameters
func (m *Manager) ruleMatchesParameters(rule *Rule, req *PermissionRequest) bool {
	if len(rule.Parameters) == 0 {
		return true
	}

	// Check each parameter
	for key, ruleValue := range rule.Parameters {
		reqValue, ok := req.Input[key]
		if !ok {
			return false
		}

		if !m.parameterMatches(ruleValue, reqValue) {
			return false
		}
	}

	return true
}

// parameterMatches checks if a rule parameter matches a request parameter
func (m *Manager) parameterMatches(ruleValue, reqValue interface{}) bool {
	switch rv := ruleValue.(type) {
	case string:
		if reqStr, ok := reqValue.(string); ok {
			return rv == reqStr || rv == "*"
		}
		return false

	case []string:
		if reqStr, ok := reqValue.(string); ok {
			for _, v := range rv {
				if v == reqStr || v == "*" {
					return true
				}
			}
			return false
		}
		return false

	case map[string]interface{}:
		if reqMap, ok := reqValue.(map[string]interface{}); ok {
			for k, v := range rv {
				if reqV, ok := reqMap[k]; !ok || !m.parameterMatches(v, reqV) {
					return false
				}
			}
			return true
		}
		return false

	default:
		return fmt.Sprintf("%v", ruleValue) == fmt.Sprintf("%v", reqValue)
	}
}

// DetectShadowedRules finds rules that are shadowed by higher priority rules
func (m *Manager) DetectShadowedRules() []ShadowedRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var shadowed []ShadowedRule

	// Get all rules sorted by priority
	rules := m.GetRules()

	// For each rule, check if it's shadowed
	for i, rule := range rules {
		for j := 0; j < i; j++ {
			higherRule := rules[j]

			// Check if higher rule shadows this rule
			if m.ruleShadows(higherRule, rule) {
				shadowed = append(shadowed, ShadowedRule{
					Rule:       rule,
					ShadowedBy: higherRule,
					Reason: fmt.Sprintf("Rule for %v (source: %s) is shadowed by rule for %v (source: %s)",
						rule.Tool, rule.Source,
						higherRule.Tool, higherRule.Source),
				})
				break
			}
		}
	}

	return shadowed
}

// ruleShadows checks if a higher priority rule shadows a lower priority one
func (m *Manager) ruleShadows(higher, lower *Rule) bool {
	// Higher priority source
	if higher.Source.GetPriority() <= lower.Source.GetPriority() {
		return false
	}

	// Same mode
	if higher.Mode != lower.Mode {
		return false
	}

	// Higher rule's tools are a superset of lower rule's tools
	higherTools := higher.GetTools()
	lowerTools := lower.GetTools()

	if len(higherTools) == 0 || (len(higherTools) == 1 && higherTools[0] == "*") {
		// Higher rule applies to all tools
		return true
	}

	// Check if all lower tools are covered by higher tools
	for _, lt := range lowerTools {
		covered := false
		for _, ht := range higherTools {
			if ht == "*" || ht == lt {
				covered = true
				break
			}
		}
		if !covered {
			return false
		}
	}

	return true
}

// generateRuleID generates a unique rule ID
func generateRuleID() string {
	return fmt.Sprintf("rule_%d", time.Now().UnixNano())
}
