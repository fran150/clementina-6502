package ui

import (
	"testing"

	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
)

func TestNewOptionsWindow(t *testing.T) {
	// Create sample menu options
	menu := []*OptionsWindowMenuOption{
		{
			Key:            tcell.KeyF1,
			KeyName:        "F1",
			KeyDescription: "Test Option",
			SubMenu: []*OptionsWindowMenuOption{
				{
					Key:            tcell.KeyF2,
					KeyName:        "F2",
					KeyDescription: "Sub Option",
				},
			},
		},
	}

	window := NewOptionsWindow(menu)

	assert.NotNil(t, window)
	assert.NotNil(t, window.text)
	assert.Equal(t, menu, window.mainMenu)
	assert.Nil(t, window.active)

	// Test parent relationships are set correctly
	assert.Nil(t, menu[0].parent)
	assert.Equal(t, menu[0], menu[0].SubMenu[0].parent)
}

func TestOptionsWindow_ProcessKey(t *testing.T) {
	actionCalled := false
	backActionCalled := false

	menu := []*OptionsWindowMenuOption{
		{
			Key:            tcell.KeyF1,
			KeyName:        "F1",
			KeyDescription: "Test Option",
			Action: func(context *common.StepContext) {
				actionCalled = true
			},
			BackAction: func(context *common.StepContext) {
				backActionCalled = true
			},
			SubMenu: []*OptionsWindowMenuOption{
				{
					Rune:           'a',
					KeyName:        "a",
					KeyDescription: "Sub Option",
				},
			},
		},
	}

	window := NewOptionsWindow(menu)
	context := &common.StepContext{}

	// Test main menu action
	event := tcell.NewEventKey(tcell.KeyF1, 0, tcell.ModNone)
	window.ProcessKey(event, context)
	assert.True(t, actionCalled)
	assert.Equal(t, menu[0], window.active)

	// Test submenu action with rune
	event = tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)
	window.ProcessKey(event, context)

	// Test back action
	event = tcell.NewEventKey(tcell.KeyESC, 0, tcell.ModNone)
	window.ProcessKey(event, context)
	assert.True(t, backActionCalled)
	assert.Nil(t, window.active)

	// Unexpected Key
	event = tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)
	window.ProcessKey(event, context)
	assert.Nil(t, window.active)
}

func TestOptionsWindow_GetActiveOptions(t *testing.T) {
	menu := []*OptionsWindowMenuOption{
		{
			Key:            tcell.KeyF1,
			KeyName:        "F1",
			KeyDescription: "Test Option",
			SubMenu: []*OptionsWindowMenuOption{
				{
					Key:            tcell.KeyF2,
					KeyName:        "F2",
					KeyDescription: "Sub Option",
				},
			},
		},
	}

	window := NewOptionsWindow(menu)

	// Test main menu
	options := window.getActiveOptions()
	assert.Equal(t, menu, options)

	// Test submenu
	window.SetActiveMenu(menu[0])
	options = window.getActiveOptions()
	assert.Equal(t, menu[0].SubMenu, options)
}

func TestOptionsWindow_Clear(t *testing.T) {
	window := NewOptionsWindow(nil)
	window.Clear()
	// Verify the text view is cleared
	assert.Equal(t, "", window.text.GetText(false))
}

func TestOptionsWindow_GetDrawArea(t *testing.T) {
	window := NewOptionsWindow(nil)
	primitive := window.GetDrawArea()
	assert.NotNil(t, primitive)
	assert.Equal(t, window.text, primitive)
}

func TestOptionsWindow_Draw(t *testing.T) {
	// Test case 1: Main menu options
	t.Run("Draw main menu options", func(t *testing.T) {
		menu := []*OptionsWindowMenuOption{
			{
				Key:            tcell.KeyF1,
				KeyName:        "F1",
				KeyDescription: "Option 1",
			},
			{
				Key:            tcell.KeyF2,
				KeyName:        "F2",
				KeyDescription: "Option 2",
			},
		}

		window := NewOptionsWindow(menu)
		context := &common.StepContext{}

		window.Draw(context)

		text := window.text.GetText(true)
		assert.Contains(t, text, "F1 Option 1")
		assert.Contains(t, text, "F2 Option 2")
		assert.NotContains(t, text, "ESC Back")
	})

	// Test case 2: Submenu options with back option
	t.Run("Draw submenu options with back option", func(t *testing.T) {
		menu := []*OptionsWindowMenuOption{
			{
				Key:            tcell.KeyF1,
				KeyName:        "F1",
				KeyDescription: "Main Option",
				SubMenu: []*OptionsWindowMenuOption{
					{
						Key:            tcell.KeyF2,
						KeyName:        "F2",
						KeyDescription: "Sub Option",
					},
				},
			},
		}

		window := NewOptionsWindow(menu)
		window.SetActiveMenu(menu[0]) // Set active menu to show submenu
		context := &common.StepContext{}

		window.Draw(context)

		text := window.text.GetText(true)
		assert.Contains(t, text, "F2 Sub Option")
		assert.Contains(t, text, "ESC Back")
		assert.NotContains(t, text, "F1 Main Option")
	})

	// Test case 3: Empty menu
	t.Run("Draw empty menu", func(t *testing.T) {
		window := NewOptionsWindow(nil)
		context := &common.StepContext{}

		window.Draw(context)

		text := window.text.GetText(false)
		assert.Equal(t, "", text)
	})

	// Test case 4: Menu with rune keys
	t.Run("Draw menu with rune keys", func(t *testing.T) {
		menu := []*OptionsWindowMenuOption{
			{
				Rune:           'a',
				KeyName:        "a",
				KeyDescription: "Option A",
			},
			{
				Rune:           'b',
				KeyName:        "b",
				KeyDescription: "Option B",
			},
		}

		window := NewOptionsWindow(menu)
		context := &common.StepContext{}

		window.Draw(context)

		text := window.text.GetText(true)
		assert.Contains(t, text, "a Option A")
		assert.Contains(t, text, "b Option B")
	})

	// Test case 5: Clear and redraw
	t.Run("Clear and redraw", func(t *testing.T) {
		menu := []*OptionsWindowMenuOption{
			{
				Key:            tcell.KeyF1,
				KeyName:        "F1",
				KeyDescription: "Test Option",
			},
		}

		window := NewOptionsWindow(menu)
		context := &common.StepContext{}

		window.Draw(context)
		window.Clear()

		text := window.text.GetText(true)
		assert.Equal(t, "", text)

		window.Draw(context)
		text = window.text.GetText(true)
		assert.Contains(t, text, "F1 Test Option")
	})
}
