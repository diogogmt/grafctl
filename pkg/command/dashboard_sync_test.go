package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/diogogmt/grafctl/pkg/grafsdk"
	"github.com/diogogmt/grafctl/pkg/simplejson"
	"github.com/stretchr/testify/assert"
)

func TestQueryManagerGetByBaseAndRefId(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "grafctl-query-manager-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create queries directory
	queriesDir := filepath.Join(tempDir, "queries")
	err = os.MkdirAll(queriesDir, 0755)
	assert.NoError(t, err)

	// Create query files
	files := map[string]string{
		"panel1_a.sql":    "SELECT * FROM table_a",
		"panel1_b.promql": "up{job=\"test_b\"}",
		"panel1_c.sql":    "SELECT * FROM table_c",
		"panel2_a.promql": "up{job=\"test_2a\"}",
		"panel2_b.promql": "up{job=\"test_2b\"}",
	}

	for filename, content := range files {
		filePath := filepath.Join(queriesDir, filename)
		err = os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)
	}

	// Create query manager
	queryManager, err := NewQueryManager(queriesDir)
	assert.NoError(t, err)

	// Load all files
	for filename := range files {
		filePath := filepath.Join(queriesDir, filename)
		err = queryManager.Put(filePath)
		assert.NoError(t, err)
	}

	// Test GetByBaseAndRefId
	tests := []struct {
		baseName string
		refId    string
		expected string
		found    bool
	}{
		{"panel1", "A", "SELECT * FROM table_a", true},
		{"panel1", "B", "up{job=\"test_b\"}", true},
		{"panel1", "C", "SELECT * FROM table_c", true},
		{"panel2", "A", "up{job=\"test_2a\"}", true},
		{"panel2", "B", "up{job=\"test_2b\"}", true},
		{"panel1", "D", "", false}, // Not found
		{"panel3", "A", "", false}, // Not found
	}

	for _, test := range tests {
		query := queryManager.GetByBaseAndRefId(test.baseName, test.refId)
		if test.found {
			assert.NotNil(t, query, "Query should be found for base %s, refId %s", test.baseName, test.refId)
			assert.Equal(t, test.expected, query.Raw, "Query content mismatch for base %s, refId %s", test.baseName, test.refId)
		} else {
			assert.Nil(t, query, "Query should not be found for base %s, refId %s", test.baseName, test.refId)
		}
	}
}

func TestUpdatePanelTargetsMultiQuery(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "grafctl-update-panel-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create queries directory
	queriesDir := filepath.Join(tempDir, "queries")
	err = os.MkdirAll(queriesDir, 0755)
	assert.NoError(t, err)

	// Create query files
	files := map[string]string{
		"cpu_usage_f.promql": "avg by (mode)(irate(node_cpu_seconds_total{mode='idle',service=\"analyzer\"})) * 100",
		"cpu_usage_b.promql": "avg by (service)(irate(node_cpu_seconds_total{mode='user',service=\"analyzer\"})) * 100",
		"cpu_usage_a.promql": "avg by (service)(irate(node_cpu_seconds_total{mode=\"system\",service=\"analyzer\"})) * 100",
	}

	for filename, content := range files {
		filePath := filepath.Join(queriesDir, filename)
		err = os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)
	}

	// Create query manager
	queryManager, err := NewQueryManager(queriesDir)
	assert.NoError(t, err)

	// Load all files
	for filename := range files {
		filePath := filepath.Join(queriesDir, filename)
		err = queryManager.Put(filePath)
		assert.NoError(t, err)
	}

	// Create a mock client
	client := &Client{
		Client:  &grafsdk.Client{},
		apiURL:  "http://localhost:3000",
		apiKey:  "test-key",
		verbose: true,
	}

	// Create a panel with multiple targets
	panel := simplejson.New()
	panel.Set("type", "timeseries")
	panel.Set("title", "CPU Usage")
	panel.Set("description", "cpu_usage")

	datasource := map[string]interface{}{"type": "prometheus"}
	panel.Set("datasource", datasource)

	// Multiple targets with different refIds
	targets := []interface{}{
		map[string]interface{}{
			"expr":   "old_expr_1",
			"refId":  "F",
			"format": "time_series",
		},
		map[string]interface{}{
			"expr":   "old_expr_2",
			"refId":  "B",
			"format": "time_series",
		},
		map[string]interface{}{
			"expr":   "old_expr_3",
			"refId":  "A",
			"format": "time_series",
		},
	}
	panel.Set("targets", targets)

	// Update panel targets
	err = client.UpdatePanelTargets(queryManager, panel)
	assert.NoError(t, err)

	// Verify targets were updated
	updatedTargets := panel.Get("targets").MustArray()
	assert.Equal(t, 3, len(updatedTargets))

	// Check first target (refId F)
	target1 := simplejson.NewFromAny(updatedTargets[0])
	assert.Equal(t, "avg by (mode)(irate(node_cpu_seconds_total{mode='idle',service=\"analyzer\"})) * 100", target1.Get("expr").MustString())

	// Check second target (refId B)
	target2 := simplejson.NewFromAny(updatedTargets[1])
	assert.Equal(t, "avg by (service)(irate(node_cpu_seconds_total{mode='user',service=\"analyzer\"})) * 100", target2.Get("expr").MustString())

	// Check third target (refId A)
	target3 := simplejson.NewFromAny(updatedTargets[2])
	assert.Equal(t, "avg by (service)(irate(node_cpu_seconds_total{mode=\"system\",service=\"analyzer\"})) * 100", target3.Get("expr").MustString())
}

func TestUpdatePanelTargetsBackwardCompatibility(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "grafctl-update-panel-backward-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create queries directory
	queriesDir := filepath.Join(tempDir, "queries")
	err = os.MkdirAll(queriesDir, 0755)
	assert.NoError(t, err)

	// Create query file with old format
	filePath := filepath.Join(queriesDir, "old_panel.sql")
	err = os.WriteFile(filePath, []byte("SELECT * FROM old_table"), 0644)
	assert.NoError(t, err)

	// Create query manager
	queryManager, err := NewQueryManager(queriesDir)
	assert.NoError(t, err)

	// Load file
	err = queryManager.Put(filePath)
	assert.NoError(t, err)

	// Create a mock client
	client := &Client{
		Client:  &grafsdk.Client{},
		apiURL:  "http://localhost:3000",
		apiKey:  "test-key",
		verbose: true,
	}

	// Create a panel with old query= format
	panel := simplejson.New()
	panel.Set("type", "stat")
	panel.Set("title", "Old Panel")
	panel.Set("description", "query=old_panel")

	datasource := map[string]interface{}{"type": "postgres"}
	panel.Set("datasource", datasource)

	target := map[string]interface{}{"rawSql": "old_sql", "refId": "A"}
	targets := []interface{}{target}
	panel.Set("targets", targets)

	// Update panel targets
	err = client.UpdatePanelTargets(queryManager, panel)
	assert.NoError(t, err)

	// Verify target was updated
	updatedTargets := panel.Get("targets").MustArray()
	assert.Equal(t, 1, len(updatedTargets))

	target1 := simplejson.NewFromAny(updatedTargets[0])
	assert.Equal(t, "SELECT * FROM old_table", target1.Get("rawSql").MustString())
}

func TestUpdatePanelTargetsSingleTarget(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "grafctl-update-panel-single-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create queries directory
	queriesDir := filepath.Join(tempDir, "queries")
	err = os.MkdirAll(queriesDir, 0755)
	assert.NoError(t, err)

	// Create query file without refId suffix (single target case)
	filePath := filepath.Join(queriesDir, "single_panel.promql")
	err = os.WriteFile(filePath, []byte("up{job=\"single\"}"), 0644)
	assert.NoError(t, err)

	// Create query manager
	queryManager, err := NewQueryManager(queriesDir)
	assert.NoError(t, err)

	// Load file
	err = queryManager.Put(filePath)
	assert.NoError(t, err)

	// Create a mock client
	client := &Client{
		Client:  &grafsdk.Client{},
		apiURL:  "http://localhost:3000",
		apiKey:  "test-key",
		verbose: true,
	}

	// Create a panel with single target
	panel := simplejson.New()
	panel.Set("type", "stat")
	panel.Set("title", "Single Target Panel")
	panel.Set("description", "single_panel")

	datasource := map[string]interface{}{"type": "prometheus"}
	panel.Set("datasource", datasource)

	target := map[string]interface{}{"expr": "old_expr", "refId": "A"}
	targets := []interface{}{target}
	panel.Set("targets", targets)

	// Update panel targets
	err = client.UpdatePanelTargets(queryManager, panel)
	assert.NoError(t, err)

	// Verify target was updated
	updatedTargets := panel.Get("targets").MustArray()
	assert.Equal(t, 1, len(updatedTargets))

	target1 := simplejson.NewFromAny(updatedTargets[0])
	assert.Equal(t, "up{job=\"single\"}", target1.Get("expr").MustString())
}
