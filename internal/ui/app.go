package ui

import (
	"fmt"
	"image/color"
	"log"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"job-aggregator/internal/db"
	"job-aggregator/internal/scraper"
)

type State int

const (
	ScreenDashboard State = iota
	ScreenAddAccount
)

var listState widget.List

func init() {
	listState.Axis = layout.Vertical
}

// progressMsg is sent from the background scraping goroutine to the UI.
type progressMsg struct {
	Done    bool
	Message string
}

func Run() error {
	window := new(app.Window)
	window.Option(app.Title("Part-Time Job Aggregator"))

	theme := material.NewTheme()
	theme.Palette.Fg = color.NRGBA{R: 30, G: 30, B: 30, A: 255}

	var currentState = ScreenDashboard

	var startAllBtn widget.Clickable
	var addAccountNavBtn widget.Clickable
	var saveAccountBtn widget.Clickable
	var backBtn widget.Clickable

	var usernameInput widget.Editor
	usernameInput.SingleLine = true
	usernameInput.Submit = true

	// Loading state — driven by the background goroutine.
	var isLoading bool
	var progressText string
	progressCh := make(chan progressMsg, 32)

	// Category filter state.
	var selectedCategory string = ""
	var categoryDropdownOpen bool
	var categorySelectBtn widget.Clickable
	// Persistent clickable state for each dropdown option, keyed by category name.
	categoryOptionClicks := map[string]*widget.Clickable{}

	var ops op.Ops

	for {
		e := window.Event()
		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			// Drain progress messages without blocking.
		drain:
			for {
				select {
				case msg := <-progressCh:
					progressText = msg.Message
					if msg.Done {
						isLoading = false
						progressText = ""
					}
				default:
					break drain
				}
			}

			if startAllBtn.Clicked(gtx) && !isLoading {
				isLoading = true
				progressText = "Starting..."

				// Run scraping in a goroutine so the UI stays responsive.
				go func() {
					progressCh <- progressMsg{Message: "Clearing old jobs..."}
					db.ClearJobs()

					accounts := db.GetAccounts()
					enabled := 0
					for _, acc := range accounts {
						if acc.Enabled {
							enabled++
						}
					}
					progressCh <- progressMsg{Message: fmt.Sprintf("Fetching %d account(s)...", enabled)}

					for _, acc := range accounts {
						if !acc.Enabled {
							continue
						}
						progressCh <- progressMsg{
							Message: fmt.Sprintf("Fetching %s (%s)...", acc.Username, acc.Provider),
						}

						jobs, err := scraper.FetchJobs(acc.Provider, acc.Username)
						if err != nil {
							log.Printf("Error fetching for account %s: %v", acc.Username, err)
							progressCh <- progressMsg{
								Message: fmt.Sprintf("Error for %s: %v", acc.Username, err),
							}
							continue
						}

						log.Printf("Fetched %d jobs for account %s (%s)", len(jobs), acc.Username, acc.Provider)
						for _, job := range jobs {
							db.SaveJob(job.Title, job.URL, job.Category, job.Price, acc.Provider)
						}
						progressCh <- progressMsg{
							Message: fmt.Sprintf("%s: %d jobs", acc.Username, len(jobs)),
						}
					}

					progressCh <- progressMsg{Done: true}
				}()
			}

			if addAccountNavBtn.Clicked(gtx) {
				currentState = ScreenAddAccount
			}

			if backBtn.Clicked(gtx) {
				currentState = ScreenDashboard
			}

			if saveAccountBtn.Clicked(gtx) {
				if usernameInput.Text() != "" {
					db.AddAccount("Freelancer Việt", usernameInput.Text())
					usernameInput.SetText("")
					currentState = ScreenDashboard
				}
			}

			// Global Padding around the entire window content (16dp padding)
			layout.Inset{
				Top:    unit.Dp(16),
				Bottom: unit.Dp(16),
				Left:   unit.Dp(16),
				Right:  unit.Dp(16),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						title := material.H4(theme, "Job Aggregator")
						title.Color = color.NRGBA{R: 20, G: 80, B: 180, A: 255}
						return title.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if currentState == ScreenDashboard {
							return renderDashboard(
								gtx, theme,
								&startAllBtn, &addAccountNavBtn,
								isLoading, progressText,
								&selectedCategory,
								&categoryDropdownOpen,
								&categorySelectBtn,
								categoryOptionClicks,
							)
						}
						return renderAddAccount(gtx, theme, &usernameInput, &saveAccountBtn, &backBtn)
					}),
				)
			})

			e.Frame(gtx.Ops)
		}
	}
}

func renderDashboard(
	gtx layout.Context,
	theme *material.Theme,
	startAllBtn, addAccountBtn *widget.Clickable,
	isLoading bool,
	progressText string,
	selectedCategory *string,
	dropdownOpen *bool,
	selectBtn *widget.Clickable,
	optionClicks map[string]*widget.Clickable,
) layout.Dimensions {
	accounts := db.GetAccounts()
	allJobs := db.GetJobs()

	// Build the list of categories present in the data, with "All" first.
	categories := []string{"All"}
	seen := map[string]bool{"All": true}
	for _, j := range allJobs {
		if j.Category == "" {
			continue
		}
		if !seen[j.Category] {
			seen[j.Category] = true
			categories = append(categories, j.Category)
		}
	}

	// If the current selection is no longer valid, reset to "All".
	if *selectedCategory != "" && !seen[*selectedCategory] {
		*selectedCategory = ""
	}

	// Apply the filter.
	var jobs []db.Job
	if *selectedCategory == "" {
		jobs = allJobs
	} else {
		for _, j := range allJobs {
			if j.Category == *selectedCategory {
				jobs = append(jobs, j)
			}
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Action Buttons
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if isLoading {
						// Disabled-looking button when running.
						btn := material.Button(theme, startAllBtn, "Running...")
						btn.Background = color.NRGBA{R: 160, G: 160, B: 160, A: 255}
						return btn.Layout(gtx)
					}
					return material.Button(theme, startAllBtn, "Start All").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Width: unit.Dp(12)}.Layout(gtx)
				}),
				layout.Rigid(material.Button(theme, addAccountBtn, "Add Account").Layout),
			)
		}),

		// Progress / loading indicator
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !isLoading || progressText == "" {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body2(theme, "⏳ "+progressText)
				lbl.Color = color.NRGBA{R: 200, G: 120, B: 0, A: 255}
				return lbl.Layout(gtx)
			})
		}),

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
		}),

		// Status Summary Card
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			info := fmt.Sprintf("Accounts: %d  |  Available Jobs Found: %d", len(accounts), len(jobs))
			lbl := material.Body1(theme, info)
			lbl.Color = color.NRGBA{R: 80, G: 80, B: 80, A: 255}
			return lbl.Layout(gtx)
		}),

		// Category filter row (select box)
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body2(theme, "Filter by category:")
						return lbl.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Spacer{Width: unit.Dp(12)}.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return categorySelectBox(
							gtx, theme,
							categories,
							selectedCategory,
							dropdownOpen,
							selectBtn,
							optionClicks,
						)
					}),
				)
			})
		}),

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
		}),

		// Jobs Table Header
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return renderTableHeader(gtx, theme)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(4)}.Layout(gtx)
		}),

		// Jobs Table Rows List
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(jobs) == 0 {
				lbl := material.Body1(theme, "No jobs match the current filter.")
				lbl.Color = color.NRGBA{R: 120, G: 120, B: 120, A: 255}
				return lbl.Layout(gtx)
			}
			return material.List(theme, &listState).Layout(gtx, len(jobs), func(gtx layout.Context, i int) layout.Dimensions {
				return renderTableRow(gtx, theme, jobs[i], i%2 == 0)
			})
		}),
	)
}

// categorySelectBox renders a dropdown-style select box for category filtering.
// Clicking the button toggles the list of options; picking an option updates
// *selected and closes the dropdown.
func categorySelectBox(
	gtx layout.Context,
	theme *material.Theme,
	categories []string,
	selected *string,
	open *bool,
	btn *widget.Clickable,
	optionClicks map[string]*widget.Clickable,
) layout.Dimensions {
	// Ensure we have a Clickable for every category option.
	for _, c := range categories {
		if optionClicks[c] == nil {
			optionClicks[c] = &widget.Clickable{}
		}
	}

	// The label shown on the closed select box.
	display := *selected
	if display == "" {
		display = "All"
	}

	// Toggle open/close when the main button is clicked.
	if btn.Clicked(gtx) {
		*open = !*open
	}

	// If dropdown is closed, just render the button.
	if !*open {
		b := material.Button(theme, btn, display+"  ▼")
		b.Background = color.NRGBA{R: 230, G: 232, B: 238, A: 255}
		b.Color = color.NRGBA{R: 40, G: 40, B: 40, A: 255}
		return b.Layout(gtx)
	}

	// Open state: button on top, list below.
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			b := material.Button(theme, btn, display+"  ▲")
			b.Background = color.NRGBA{R: 20, G: 120, B: 220, A: 255}
			b.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			return b.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			// Draw the dropdown list with a background.
			return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Stack{}.Layout(gtx,
					layout.Expanded(func(gtx layout.Context) layout.Dimensions {
						// Background for the dropdown panel.
						defer clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops).Pop()
						paint.Fill(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
						return layout.Dimensions{Size: gtx.Constraints.Min}
					}),
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							func() []layout.FlexChild {
								children := make([]layout.FlexChild, 0, len(categories))
								for _, c := range categories {
									cat := c // capture
									clk := optionClicks[cat]
									children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										if clk.Clicked(gtx) {
											if cat == "All" {
												*selected = ""
											} else {
												*selected = cat
											}
											*open = false
										}
										isActive := (cat == "All" && *selected == "") || (cat == *selected)
										lbl := material.Body2(theme, cat)
										if isActive {
											lbl.Color = color.NRGBA{R: 20, G: 120, B: 220, A: 255}
											lbl.Font.Weight = 700
										}
										return layout.Inset{
											Top:    unit.Dp(6),
											Bottom: unit.Dp(6),
											Left:   unit.Dp(10),
											Right:  unit.Dp(10),
										}.Layout(gtx, lbl.Layout)
									}))
								}
								return children
							}()...,
						)
					}),
				)
			})
		}),
	)
}

func renderTableHeader(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, color.NRGBA{R: 235, G: 238, B: 245, A: 255})

	return layout.Inset{
		Top:    unit.Dp(8),
		Bottom: unit.Dp(8),
		Left:   unit.Dp(8),
		Right:  unit.Dp(8),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Flexed(0.2, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(theme, "Provider")
				lbl.Font.Weight = 700
				return lbl.Layout(gtx)
			}),
			layout.Flexed(0.45, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(theme, "Title")
				lbl.Font.Weight = 700
				return lbl.Layout(gtx)
			}),
			layout.Flexed(0.2, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(theme, "Category")
				lbl.Font.Weight = 700
				return lbl.Layout(gtx)
			}),
			layout.Flexed(0.15, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(theme, "Price")
				lbl.Font.Weight = 700
				return lbl.Layout(gtx)
			}),
		)
	})
}

func renderTableRow(gtx layout.Context, theme *material.Theme, job db.Job, isEven bool) layout.Dimensions {
	bgColor := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	if isEven {
		bgColor = color.NRGBA{R: 248, G: 249, B: 250, A: 255}
	}

	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, bgColor)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top:    unit.Dp(10),
				Bottom: unit.Dp(10),
				Left:   unit.Dp(8),
				Right:  unit.Dp(8),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(0.2, func(gtx layout.Context) layout.Dimensions {
						return material.Body2(theme, job.Provider).Layout(gtx)
					}),
					layout.Flexed(0.45, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body2(theme, job.Title)
						lbl.Color = color.NRGBA{R: 10, G: 90, B: 180, A: 255}
						return lbl.Layout(gtx)
					}),
					layout.Flexed(0.2, func(gtx layout.Context) layout.Dimensions {
						return material.Body2(theme, job.Category).Layout(gtx)
					}),
					layout.Flexed(0.15, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body2(theme, job.Price)
						lbl.Color = color.NRGBA{R: 0, G: 140, B: 60, A: 255}
						return lbl.Layout(gtx)
					}),
				)
			})
		}),
	)
}

func renderAddAccount(gtx layout.Context, theme *material.Theme, usernameInput *widget.Editor, saveBtn, backBtn *widget.Clickable) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.H6(theme, "Add Provider Account").Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(12)}.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Editor(theme, usernameInput, "Enter Account/Email").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(material.Button(theme, saveBtn, "Save").Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Width: unit.Dp(12)}.Layout(gtx)
				}),
				layout.Rigid(material.Button(theme, backBtn, "Back").Layout),
			)
		}),
	)
}
