package commands

import (
	"sort"
	"strings"
	"sync"
)

// Registry manages all registered commands
type Registry struct {
	mu       sync.RWMutex
	commands map[string]*Command
	byCategory map[string][]*Command
}

// NewRegistry creates a new command registry
func NewRegistry() *Registry {
	return &Registry{
		commands:   make(map[string]*Command),
		byCategory: make(map[string][]*Command),
	}
}

// Register adds a command to the registry
func (r *Registry) Register(cmd *Command) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Register by name
	r.commands[cmd.Name] = cmd

	// Register by aliases
	for _, alias := range cmd.Aliases {
		r.commands[alias] = cmd
	}

	// Add to category index
	category := cmd.Category
	if category == "" {
		category = "Other"
	}
	r.byCategory[category] = append(r.byCategory[category], cmd)

	// Sort by SortOrder then by name
	sort.Slice(r.byCategory[category], func(i, j int) bool {
		if r.byCategory[category][i].SortOrder != r.byCategory[category][j].SortOrder {
			return r.byCategory[category][i].SortOrder < r.byCategory[category][j].SortOrder
		}
		return r.byCategory[category][i].Name < r.byCategory[category][j].Name
	})
}

// Unregister removes a command from the registry
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmd, ok := r.commands[name]
	if !ok {
		return
	}

	// Remove from name index
	delete(r.commands, name)

	// Remove from alias index
	for _, alias := range cmd.Aliases {
		delete(r.commands, alias)
	}

	// Remove from category index
	category := cmd.Category
	if category == "" {
		category = "Other"
	}
	for i, c := range r.byCategory[category] {
		if c.Name == cmd.Name {
			r.byCategory[category] = append(
				r.byCategory[category][:i],
				r.byCategory[category][i+1:]...,
			)
			break
		}
	}
}

// Get retrieves a command by name or alias
func (r *Registry) Get(name string) (*Command, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cmd, ok := r.commands[name]
	return cmd, ok
}

// GetAll returns all unique commands (by primary name)
func (r *Registry) GetAll() []*Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]bool)
	var result []*Command

	for _, cmd := range r.commands {
		if !seen[cmd.Name] {
			seen[cmd.Name] = true
			result = append(result, cmd)
		}
	}

	// Sort by name
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// GetByCategory returns commands grouped by category
func (r *Registry) GetByCategory() map[string][]*Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]*Command)
	for cat, cmds := range r.byCategory {
		result[cat] = append([]*Command{}, cmds...) // Copy slice
	}

	return result
}

// GetVisible returns commands that should be shown in help
func (r *Registry) GetVisible(availability CommandAvailability, isAuthenticated bool, isInternal bool) []*Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]bool)
	var result []*Command

	for _, cmd := range r.commands {
		if seen[cmd.Name] {
			continue
		}
		seen[cmd.Name] = true

		if cmd.IsHidden {
			continue
		}

		if !cmd.IsAvailable(availability, isAuthenticated, isInternal) {
			continue
		}

		result = append(result, cmd)
	}

	// Sort by category then by SortOrder
	sort.Slice(result, func(i, j int) bool {
		if result[i].Category != result[j].Category {
			return result[i].Category < result[j].Category
		}
		if result[i].SortOrder != result[j].SortOrder {
			return result[i].SortOrder < result[j].SortOrder
		}
		return result[i].Name < result[j].Name
	})

	return result
}

// Search finds commands matching a query
func (r *Registry) Search(query string) []*Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = strings.ToLower(query)
	seen := make(map[string]bool)
	var result []*Command

	for _, cmd := range r.commands {
		if seen[cmd.Name] {
			continue
		}
		seen[cmd.Name] = true

		// Search in name, aliases, and description
		if strings.Contains(strings.ToLower(cmd.Name), query) ||
			strings.Contains(strings.ToLower(cmd.Description), query) {
			result = append(result, cmd)
			continue
		}

		for _, alias := range cmd.Aliases {
			if strings.Contains(strings.ToLower(alias), query) {
				result = append(result, cmd)
				break
			}
		}
	}

	return result
}

// Parse parses a command input string
func (r *Registry) Parse(input string) (cmd *Command, args string, ok bool) {
	input = strings.TrimSpace(input)

	// Must start with /
	if !strings.HasPrefix(input, "/") {
		return nil, "", false
	}

	// Split into command and args
	parts := strings.SplitN(input[1:], " ", 2)
	name := parts[0]
	args = ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	cmd, ok = r.Get(name)
	return cmd, args, ok
}

// Complete returns commands that match a prefix for autocompletion
func (r *Registry) Complete(prefix string) []*Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !strings.HasPrefix(prefix, "/") {
		return nil
	}

	search := strings.ToLower(prefix[1:])
	seen := make(map[string]bool)
	var result []*Command

	for name, cmd := range r.commands {
		if seen[cmd.Name] {
			continue
		}

		if cmd.IsHidden {
			continue
		}

		if strings.HasPrefix(strings.ToLower(name), search) {
			seen[cmd.Name] = true
			result = append(result, cmd)
		}
	}

	// Sort by name
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// Count returns the total number of unique commands
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]bool)
	for _, cmd := range r.commands {
		seen[cmd.Name] = true
	}

	return len(seen)
}
