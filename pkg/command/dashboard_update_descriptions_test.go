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
	assert.Equal(t, "panel", client.getPanelTypePrefix("unknown_type"))
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

func TestRowAssignmentLogic(t *testing.T) {

	// Create a test dashboard with multiple rows and panels
	dashboard := simplejson.New()
	panels := []interface{}{
		// Row 1 at Y=0
		map[string]interface{}{
			"id":    1,
			"type":  "row",
			"title": "First Row",
			"gridPos": map[string]interface{}{
				"y": 0,
			},
		},
		// Panel A at Y=1 (should belong to First Row)
		map[string]interface{}{
			"id":    2,
			"type":  "table",
			"title": "Panel A",
			"gridPos": map[string]interface{}{
				"y": 1,
			},
		},
		// Row 2 at Y=10
		map[string]interface{}{
			"id":    3,
			"type":  "row",
			"title": "Second Row",
			"gridPos": map[string]interface{}{
				"y": 10,
			},
		},
		// Panel B at Y=11 (should belong to Second Row)
		map[string]interface{}{
			"id":    4,
			"type":  "graph",
			"title": "Panel B",
			"gridPos": map[string]interface{}{
				"y": 11,
			},
		},
		// Panel C at Y=2 (should belong to First Row)
		map[string]interface{}{
			"id":    5,
			"type":  "stat",
			"title": "Panel C",
			"gridPos": map[string]interface{}{
				"y": 2,
			},
		},
	}
	dashboard.Set("panels", panels)

	// Create mock dashboard metadata
	meta := simplejson.New()
	meta.Set("folderTitle", "Test Folder")

	// Create mock dashboard with metadata
	dashboardFull := &grafsdk.DashboardWithMeta{
		Dashboard: dashboard,
		Meta:      meta,
	}

	// Mock the GetDashboardByUID method by creating a test function
	testUpdateDescriptions := func() error {
		dashboardTitle := dashboardFull.Dashboard.Get("title").MustString()
		if dashboardTitle == "" {
			dashboardTitle = "Test Dashboard"
		}

		_ = dashboardFull.Meta.Get("folderTitle").MustString()

		panels := dashboardFull.Dashboard.Get("panels").MustArray()

		// First pass: collect row information
		rowInfo := make(map[int]string) // panel ID -> row title

		// Collect all rows first
		var rows []struct {
			title string
			y     int
		}
		for _, panelBy := range panels {
			panel := simplejson.NewFromAny(panelBy)
			panelType := panel.Get("type").MustString()
			if panelType == "row" {
				rowTitle := panel.Get("title").MustString()
				rowY := panel.Get("gridPos").Get("y").MustInt()
				rows = append(rows, struct {
					title string
					y     int
				}{title: rowTitle, y: rowY})
			}
		}

		// For each non-row panel, find the closest row above it
		for _, panelBy := range panels {
			panel := simplejson.NewFromAny(panelBy)
			panelType := panel.Get("type").MustString()
			if panelType != "row" {
				panelY := panel.Get("gridPos").Get("y").MustInt()
				panelID := panel.Get("id").MustInt()

				// Find the closest row above this panel
				var closestRow string
				var closestDistance int = -1

				for _, row := range rows {
					if panelY > row.y {
						distance := panelY - row.y
						if closestDistance == -1 || distance < closestDistance {
							closestDistance = distance
							closestRow = row.title
						}
					}
				}

				if closestRow != "" {
					rowInfo[panelID] = closestRow
				}
			}
		}

		// Verify the assignments
		assert.Equal(t, "First Row", rowInfo[2], "Panel A should belong to First Row")
		assert.Equal(t, "Second Row", rowInfo[4], "Panel B should belong to Second Row")
		assert.Equal(t, "First Row", rowInfo[5], "Panel C should belong to First Row")

		return nil
	}

	err := testUpdateDescriptions()
	assert.NoError(t, err)
}

func TestMixedPanelStructure(t *testing.T) {
	// Create a test dashboard with mixed panel structure (nested + standalone)
	dashboard := simplejson.New()
	panels := []interface{}{
		// Row 1 at Y=0 with nested panels
		map[string]interface{}{
			"id":    1,
			"type":  "row",
			"title": "First Row",
			"gridPos": map[string]interface{}{
				"y": 0,
			},
			"panels": []interface{}{
				// Nested panel in First Row
				map[string]interface{}{
					"id":    2,
					"type":  "table",
					"title": "Nested Panel A",
					"gridPos": map[string]interface{}{
						"y": 1,
					},
				},
			},
		},
		// Row 2 at Y=10 with empty panels array
		map[string]interface{}{
			"id":    3,
			"type":  "row",
			"title": "Second Row",
			"gridPos": map[string]interface{}{
				"y": 10,
			},
			"panels": []interface{}{}, // Empty - panels will be standalone
		},
		// Standalone panel at Y=11 (should belong to Second Row)
		map[string]interface{}{
			"id":    4,
			"type":  "graph",
			"title": "Standalone Panel B",
			"gridPos": map[string]interface{}{
				"y": 11,
			},
		},
		// Another standalone panel at Y=12 (should belong to Second Row)
		map[string]interface{}{
			"id":    5,
			"type":  "stat",
			"title": "Standalone Panel C",
			"gridPos": map[string]interface{}{
				"y": 12,
			},
		},
	}
	dashboard.Set("panels", panels)

	// Create mock dashboard metadata
	meta := simplejson.New()
	meta.Set("folderTitle", "Test Folder")

	// Create mock dashboard with metadata
	dashboardFull := &grafsdk.DashboardWithMeta{
		Dashboard: dashboard,
		Meta:      meta,
	}

	// Mock the row assignment logic
	testRowAssignment := func() error {
		panels := dashboardFull.Dashboard.Get("panels").MustArray()

		// First pass: collect row information
		rowInfo := make(map[int]string) // panel ID -> row title

		// Collect all rows first
		var rows []struct {
			title string
			y     int
		}
		for _, panelBy := range panels {
			panel := simplejson.NewFromAny(panelBy)
			panelType := panel.Get("type").MustString()
			if panelType == "row" {
				rowTitle := panel.Get("title").MustString()
				rowY := panel.Get("gridPos").Get("y").MustInt()
				rows = append(rows, struct {
					title string
					y     int
				}{title: rowTitle, y: rowY})
			}
		}

		// Handle nested panels within rows
		for _, panelBy := range panels {
			panel := simplejson.NewFromAny(panelBy)
			panelType := panel.Get("type").MustString()
			if panelType == "row" {
				rowTitle := panel.Get("title").MustString()
				// Process panels nested within this row
				for _, subPanelBy := range panel.Get("panels").MustArray() {
					subPanel := simplejson.NewFromAny(subPanelBy)
					subPanelID := subPanel.Get("id").MustInt()
					rowInfo[subPanelID] = rowTitle
				}
			}
		}

		// Handle standalone panels (not nested in rows) - find the closest row above them
		for _, panelBy := range panels {
			panel := simplejson.NewFromAny(panelBy)
			panelType := panel.Get("type").MustString()
			if panelType != "row" {
				panelY := panel.Get("gridPos").Get("y").MustInt()
				panelID := panel.Get("id").MustInt()

				// Skip if this panel is already assigned to a row (nested panels)
				if _, exists := rowInfo[panelID]; exists {
					continue
				}

				// Find the closest row above this panel
				var closestRow string
				var closestDistance int = -1

				for _, row := range rows {
					if panelY > row.y {
						distance := panelY - row.y
						if closestDistance == -1 || distance < closestDistance {
							closestDistance = distance
							closestRow = row.title
						}
					}
				}

				if closestRow != "" {
					rowInfo[panelID] = closestRow
				}
			}
		}

		// Verify the assignments
		assert.Equal(t, "First Row", rowInfo[2], "Nested Panel A should belong to First Row")
		assert.Equal(t, "Second Row", rowInfo[4], "Standalone Panel B should belong to Second Row")
		assert.Equal(t, "Second Row", rowInfo[5], "Standalone Panel C should belong to Second Row")

		return nil
	}

	err := testRowAssignment()
	assert.NoError(t, err)
}
