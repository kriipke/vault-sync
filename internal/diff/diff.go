package diff

import (
	"fmt"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
	"gopkg.in/yaml.v3"
	"vault-sync/internal/vault"
)

type SecretDiff struct {
	Path     string
	Current  *vault.Secret
	Proposed *vault.Secret
	HasDiff  bool
	DiffText string
}

func CompareSecrets(current, proposed *vault.Secret) (*SecretDiff, error) {
	currentYAML, err := secretToYAML(current)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize current secret: %w", err)
	}

	proposedYAML, err := secretToYAML(proposed)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize proposed secret: %w", err)
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(currentYAML, proposedYAML, false)
	
	hasDiff := false
	for _, diff := range diffs {
		if diff.Type != diffmatchpatch.DiffEqual {
			hasDiff = true
			break
		}
	}

	var diffText string
	if hasDiff {
		diffText = formatUnifiedDiff(current.Path, currentYAML, proposedYAML)
	}

	return &SecretDiff{
		Path:     current.Path,
		Current:  current,
		Proposed: proposed,
		HasDiff:  hasDiff,
		DiffText: diffText,
	}, nil
}

func secretToYAML(secret *vault.Secret) (string, error) {
	if secret == nil {
		return "", nil
	}

	yamlData, err := yaml.Marshal(secret.Data)
	if err != nil {
		return "", err
	}

	return string(yamlData), nil
}

func formatUnifiedDiff(path, current, proposed string) string {
	currentLines := strings.Split(current, "\n")
	proposedLines := strings.Split(proposed, "\n")

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(current, proposed, false)
	
	var result strings.Builder
	result.WriteString(fmt.Sprintf("--- %s (current)\n", path))
	result.WriteString(fmt.Sprintf("+++ %s (proposed)\n", path))
	result.WriteString("@@ -1,")
	result.WriteString(fmt.Sprintf("%d", len(currentLines)))
	result.WriteString(" +1,")
	result.WriteString(fmt.Sprintf("%d", len(proposedLines)))
	result.WriteString(" @@\n")

	for _, diff := range diffs {
		lines := strings.Split(diff.Text, "\n")
		for i, line := range lines {
			if i == len(lines)-1 && line == "" {
				continue
			}
			
			switch diff.Type {
			case diffmatchpatch.DiffDelete:
				result.WriteString("-" + line + "\n")
			case diffmatchpatch.DiffInsert:
				result.WriteString("+" + line + "\n")
			case diffmatchpatch.DiffEqual:
				result.WriteString(" " + line + "\n")
			}
		}
	}

	return result.String()
}

func PrintDiff(diff *SecretDiff) {
	if !diff.HasDiff {
		fmt.Printf("No changes for secret: %s\n", diff.Path)
		return
	}

	fmt.Printf("Changes for secret: %s\n", diff.Path)
	fmt.Printf("%s\n", diff.DiffText)
}