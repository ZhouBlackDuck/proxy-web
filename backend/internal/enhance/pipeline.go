package enhance

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Pipeline orchestrates the config merge process
type Pipeline struct{}

func NewPipeline() *Pipeline {
	return &Pipeline{}
}

// BuildWithPorts builds config with port settings applied
func (p *Pipeline) BuildWithPorts(subYaml, overrideYaml, globalRules string, ports map[string]PortSetting) ([]byte, error) {
	// 1. Parse subscription config
	var subNode yaml.Node
	if err := yaml.Unmarshal([]byte(subYaml), &subNode); err != nil {
		return nil, fmt.Errorf("parse subscription yaml: %w", err)
	}

	// Ensure we have a mapping node
	configMap := getMappingNode(&subNode)
	if configMap == nil {
		return nil, fmt.Errorf("subscription yaml is not a mapping")
	}

	// 2. Apply override (shallow merge)
	if overrideYaml != "" {
		var overrideNode yaml.Node
		if err := yaml.Unmarshal([]byte(overrideYaml), &overrideNode); err != nil {
			return nil, fmt.Errorf("parse override yaml: %w", err)
		}
		overrideMap := getMappingNode(&overrideNode)
		if overrideMap != nil {
			configMap = shallowMerge(configMap, overrideMap)
		}
	}

	// 3. Prepend global rules
	if globalRules != "" {
		configMap = prependRules(configMap, globalRules)
	}

	// 4. Ensure defaults
	configMap = ensureDefaults(configMap)

	// 5. Apply port settings (remove disabled ports, set enabled ones)
	if ports != nil {
		configMap = applyPorts(configMap, ports)
	}

	// 6. Serialize back to yaml
	outNode := yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{configMap},
	}
	result, err := yaml.Marshal(&outNode)
	if err != nil {
		return nil, fmt.Errorf("serialize config: %w", err)
	}

	return result, nil
}

// Validate performs basic validation on the config yaml
func (p *Pipeline) Validate(yamlData []byte) []string {
	var errors []string

	var node yaml.Node
	if err := yaml.Unmarshal(yamlData, &node); err != nil {
		return []string{fmt.Sprintf("YAML syntax error: %v", err)}
	}

	m := getMappingNode(&node)
	if m == nil {
		return []string{"config is not a YAML mapping"}
	}

	// Check for proxies or proxy-providers
	hasProxies := hasKey(m, "proxies")
	hasProviders := hasKey(m, "proxy-providers")
	if !hasProxies && !hasProviders {
		errors = append(errors, "config has no proxies or proxy-providers defined")
	}

	// Check for rules
	if !hasKey(m, "rules") {
		errors = append(errors, "config has no rules defined")
	}

	// Check mixed-port or port
	hasMixedPort := hasKey(m, "mixed-port")
	hasPort := hasKey(m, "port")
	hasSocksPort := hasKey(m, "socks-port")
	if !hasMixedPort && !hasPort && !hasSocksPort {
		errors = append(errors, "config has no listening port configured (mixed-port/port/socks-port)")
	}

	return errors
}

// --- Internal helpers ---

// getMappingNode extracts the mapping node from a document or returns nil
func getMappingNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	if node.Kind == yaml.MappingNode {
		return node
	}
	return nil
}

// shallowMerge merges override into base. Override values take precedence.
// For mapping keys that exist in both, the override value replaces the base value.
// Special handling for "tun" key: deep merge.
func shallowMerge(base, override *yaml.Node) *yaml.Node {
	if base == nil || override == nil {
		return base
	}

	// Build a map of base keys for quick lookup
	baseKeys := make(map[string]int)
	for i := 0; i < len(base.Content); i += 2 {
		key := base.Content[i].Value
		baseKeys[key] = i
	}

	// Iterate override keys
	for i := 0; i < len(override.Content); i += 2 {
		overrideKey := override.Content[i]
		overrideVal := override.Content[i+1]

		if idx, exists := baseKeys[overrideKey.Value]; exists {
			if overrideKey.Value == "tun" {
				// Deep merge tun config
				baseTun := base.Content[idx+1]
				if baseTun.Kind == yaml.MappingNode && overrideVal.Kind == yaml.MappingNode {
					base.Content[idx+1] = shallowMerge(baseTun, overrideVal)
				} else {
					base.Content[idx+1] = overrideVal
				}
			} else {
				// Replace value
				base.Content[idx+1] = overrideVal
			}
		} else {
			// Add new key-value pair
			base.Content = append(base.Content, overrideKey, overrideVal)
			baseKeys[overrideKey.Value] = len(base.Content) - 2
		}
	}

	return base
}

// prependRules adds global rules to the beginning of the rules array
func prependRules(config *yaml.Node, globalRules string) *yaml.Node {
	// Parse global rules
	ruleLines := parseRuleLines(globalRules)
	if len(ruleLines) == 0 {
		return config
	}

	// Find or create rules key in config
	rulesIdx := -1
	for i := 0; i < len(config.Content); i += 2 {
		if config.Content[i].Value == "rules" {
			rulesIdx = i + 1
			break
		}
	}

	// Create new rule nodes
	var newRuleNodes []*yaml.Node
	for _, rule := range ruleLines {
		newRuleNodes = append(newRuleNodes, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: rule,
		})
	}

	if rulesIdx >= 0 && config.Content[rulesIdx].Kind == yaml.SequenceNode {
		// Prepend to existing rules
		existingRules := config.Content[rulesIdx].Content
		config.Content[rulesIdx].Content = append(newRuleNodes, existingRules...)
	} else {
		// Create rules key
		rulesSeq := &yaml.Node{
			Kind:    yaml.SequenceNode,
			Tag:     "!!seq",
			Content: newRuleNodes,
		}
		config.Content = append(config.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "rules"},
			rulesSeq,
		)
	}

	return config
}

// parseRuleLines parses rule text into individual rule strings
func parseRuleLines(text string) []string {
	var rules []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rules = append(rules, line)
	}
	return rules
}

// ensureDefaults adds required fields if they're missing
func ensureDefaults(config *yaml.Node) *yaml.Node {
	// Ensure external-controller is set for WebUI to communicate
	if !hasKey(config, "external-controller") {
		config.Content = append(config.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "external-controller"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "127.0.0.1:9090"},
		)
	}

	// Ensure mode is set
	if !hasKey(config, "mode") {
		config.Content = append(config.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "mode"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "rule"},
		)
	}

	// Ensure log-level is set
	if !hasKey(config, "log-level") {
		config.Content = append(config.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "log-level"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "info"},
		)
	}

	return config
}

// hasKey checks if a mapping node contains a specific key
func hasKey(m *yaml.Node, key string) bool {
	if m == nil {
		return false
	}
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return true
		}
	}
	return false
}

// PortSetting defines a port's enabled state and value
type PortSetting struct {
	Enabled bool
	Port    int
}

// applyPorts manages port keys in the config:
// - enabled ports are set to their configured value
// - disabled ports are removed from the config entirely
func applyPorts(config *yaml.Node, ports map[string]PortSetting) *yaml.Node {
	if config == nil {
		return config
	}

	for key, setting := range ports {
		if setting.Enabled && setting.Port > 0 {
			// Set or update the port value
			if hasKey(config, key) {
				for i := 0; i < len(config.Content); i += 2 {
					if config.Content[i].Value == key {
						config.Content[i+1] = &yaml.Node{
							Kind:  yaml.ScalarNode,
							Tag:   "!!int",
							Value: fmt.Sprintf("%d", setting.Port),
						}
						break
					}
				}
			} else {
				config.Content = append(config.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
					&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: fmt.Sprintf("%d", setting.Port)},
				)
			}
		} else {
			// Remove the key entirely from config
			removeKey(config, key)
		}
	}

	return config
}

// removeKey removes a key-value pair from a mapping node
func removeKey(m *yaml.Node, key string) {
	if m == nil {
		return
	}
	newContent := make([]*yaml.Node, 0, len(m.Content))
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value != key {
			newContent = append(newContent, m.Content[i], m.Content[i+1])
		}
	}
	m.Content = newContent
}
