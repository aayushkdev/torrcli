package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(addAccent)
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#526078", Dark: "#98A2BE"})
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#236A76", Dark: "#94C9D1"})
	focusStyle    = lipgloss.NewStyle().Bold(true).Foreground(addAccent)
	selectedStyle = lipgloss.NewStyle().Background(addFocusSurface).Foreground(addText)
	errorStyle    = lipgloss.NewStyle().Foreground(dangerColor)
	dialogStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(borderColor).Foreground(addText).Background(addSurface).Padding(1, 2)
)

var (
	addText         = lipgloss.AdaptiveColor{Light: "#26334D", Dark: "#CDD6F4"}
	addSurface      = lipgloss.AdaptiveColor{Light: "#EFF1F7", Dark: "#242436"}
	addAccent       = lipgloss.AdaptiveColor{Light: "#2456A6", Dark: "#89B4FA"}
	addSecondary    = lipgloss.AdaptiveColor{Light: "#526078", Dark: "#98A2BE"}
	addInputSurface = lipgloss.AdaptiveColor{Light: "#E1E6F0", Dark: "#181825"}
	addFocusSurface = lipgloss.AdaptiveColor{Light: "#D2DFF4", Dark: "#313244"}
	addDialogStyle  = dialogStyle.Foreground(addText).Background(addSurface).BorderForeground(addAccent)
	addLabelStyle   = lipgloss.NewStyle().Foreground(addText).Background(addSurface).Bold(true)
	addHintStyle    = lipgloss.NewStyle().Foreground(addSecondary).Background(addSurface)
)

var (
	dangerColor      = lipgloss.AdaptiveColor{Light: "#A42B49", Dark: "#F38BA8"}
	borderColor      = lipgloss.AdaptiveColor{Light: "#8994AB", Dark: "#626D8B"}
	separatorStyle   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#AFB8CB", Dark: "#454B65"})
	dialogTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(addText).Background(addSurface)
	dialogKeyStyle   = lipgloss.NewStyle().Bold(true).Foreground(addAccent).Background(addSurface)
)
