package command

import (
	"testing"

	"github.com/diogogmt/grafctl/pkg/grafsdk"
	"github.com/diogogmt/grafctl/pkg/simplejson"
	"github.com/stretchr/testify/assert"
)

func TestSanitizeTitle(t *testing.T) {
	client := &Client{
		Client:  &grafsdk.Client{},
		apiURL:  "http://localhost:3000",
		apiKey:  "test-key",
		verbose: true,
	}

	// Test basic sanitization
	assert.Equal(t, "my-dashboard", client.sanitizeTitle("My Dashboard"))
	assert.Equal(t, "cpu-usage", client.sanitizeTitle("CPU Usage"))
	assert.Equal(t, "api-requests-second", client.sanitizeTitle("API Requests/Second"))

	// Test special characters
	assert.Equal(t, "test-panel", client.sanitizeTitle("Test Panel!"))
	assert.Equal(t, "panel-123", client.sanitizeTitle("Panel 123"))
	assert.Equal(t, "panel-with-dashes", client.sanitizeTitle("Panel-with-dashes"))

	// Test edge cases
	assert.Equal(t, "untitled", client.sanitizeTitle(""))
	assert.Equal(t, "untitled", client.sanitizeTitle("   "))
	assert.Equal(t, "untitled", client.sanitizeTitle("!@#$%"))
	assert.Equal(t, "a", client.sanitizeTitle("A"))
}

func TestGetPanelTypePrefix(t *testing.T) {
	client := &Client{
		Client:  &grafsdk.Client{},
		apiURL:  "http://localhost:3000",
		apiKey:  "test-key",
		verbose: true,
	}

	// Test known panel types
	assert.Equal(t, "table", client.getPanelTypePrefix("table"))
	assert.Equal(t, "graph", client.getPanelTypePrefix("graph"))
	assert.Equal(t, "stat", client.getPanelTypePrefix("stat"))
	assert.Equal(t, "graph", client.getPanelTypePrefix("timeseries"))
	assert.Equal(t, "heatmap", client.getPanelTypePrefix("heatmap"))

	// Test unknown panel type
	assert.Equal(t, "panel", client.getPanelTypePrefix("panel"))
}

func TestGeneratePanelDescription(t *testing.T) {
	client := &Client{
		Client:  &grafsdk.Client{},
		apiURL:  "http://localhost:3000",
		apiKey:  "test-key",
		verbose: true,
	}

	// Test panel without row
	desc := client.generatePanelDescription("Business Metrics", "My Dashboard", "", "table", "CPU Usage")
	assert.Equal(t, "query=business-metrics/my-dashboard/table-cpu-usage", desc)

	// Test panel with row
	desc = client.generatePanelDescription("Business Metrics", "My Dashboard", "System Metrics", "graph", "Memory Usage")
	assert.Equal(t, "query=business-metrics/my-dashboard/system-metrics/graph-memory-usage", desc)

	// Test with special characters
	desc = client.generatePanelDescription("Business Metrics", "API Dashboard", "Request Stats", "stat", "Requests/Second!")
	assert.Equal(t, "query=business-metrics/api-dashboard/request-stats/stat-requests-second", desc)

	// Test with folder title
	desc = client.generatePanelDescription("Business Metrics", "My Dashboard", "", "table", "CPU Usage")
	assert.Equal(t, "query=business-metrics/my-dashboard/table-cpu-usage", desc)

	// Test with folder title and row
	desc = client.generatePanelDescription("Business Metrics", "My Dashboard", "System Metrics", "graph", "Memory Usage")
	assert.Equal(t, "query=business-metrics/my-dashboard/system-metrics/graph-memory-usage", desc)
}

func TestIsInvalidDescription(t *testing.T) {
	client := &Client{
		Client:  &grafsdk.Client{},
		apiURL:  "http://localhost:3000",
		apiKey:  "test-key",
		verbose: true,
	}

	// Test invalid descriptions
	assert.True(t, client.isInvalidDescription(""))
	assert.True(t, client.isInvalidDescription("query="))
	assert.True(t, client.isInvalidDescription("query=  "))
	assert.True(t, client.isInvalidDescription("some random text"))

	// Test valid descriptions
	assert.False(t, client.isInvalidDescription("query=valid/path"))
	assert.False(t, client.isInvalidDescription("query=queries/panel1\nquery=queries/panel2"))
}

func TestUpdatePanelDescription(t *testing.T) {
	client := &Client{
		Client:  &grafsdk.Client{},
		apiURL:  "http://localhost:3000",
		apiKey:  "test-key",
		verbose: true,
	}

	// Create a test panel
	panel := simplejson.New()
	panel.Set("type", "table")
	panel.Set("title", "Test Panel")
	panel.Set("description", "") // Empty description

	// Test updating invalid description
	updated, skipped, err := client.updatePanelDescription(panel, "Business Metrics", "My Dashboard", "", false, false)
	assert.NoError(t, err)
	assert.True(t, updated)
	assert.False(t, skipped)

	// Check that description was updated
	newDesc := panel.Get("description").MustString()
	assert.Equal(t, "query=business-metrics/my-dashboard/table-test-panel", newDesc)

	// Test dry run
	panel.Set("description", "") // Reset
	updated, skipped, err = client.updatePanelDescription(panel, "Business Metrics", "My Dashboard", "", false, true)
	assert.NoError(t, err)
	assert.True(t, updated)
	assert.False(t, skipped)

	// Check that description was NOT updated in dry run
	desc := panel.Get("description").MustString()
	assert.Equal(t, "", desc)

	// Test skipping valid description
	panel.Set("description", "query=valid/path")
	updated, skipped, err = client.updatePanelDescription(panel, "Business Metrics", "My Dashboard", "", false, false)
	assert.NoError(t, err)
	assert.False(t, updated)
	assert.True(t, skipped)

	// Test overwrite flag
	updated, skipped, err = client.updatePanelDescription(panel, "Business Metrics", "My Dashboard", "", true, false)
	assert.NoError(t, err)
	assert.True(t, updated)
	assert.False(t, skipped)
}
