package ui

import (
	"fmt"
	"image"
	"image/color"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"job-aggregator/internal/rest"
)

func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

type State int

const (
	ScreenDashboard State = iota
	ScreenAddAccount
	ScreenSettings
)

var listState widget.List

func init() {
	listState.Axis = layout.Vertical
}

// progressMsg is sent from a background goroutine to the UI.
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
	var settingsNavBtn widget.Clickable
	var saveAccountBtn widget.Clickable
	var backBtn widget.Clickable
	var saveSettingsBtn widget.Clickable
	var testSettingsBtn widget.Clickable

	var usernameInput widget.Editor
	usernameInput.SingleLine = true
	usernameInput.Submit = true

	// Settings form state.
	var apiUrlInput widget.Editor
	apiUrlInput.SingleLine = true
	apiUrlInput.Submit = true
	apiUrlInput.SetText(apiUrl()) // pre-fill with current value

	// Loading state — driven by background goroutines.
	var isLoading bool
	var progressText string
	progressCh := make(chan progressMsg, 32)

	// Category filter state.
	var selectedCategory string = ""
	var categoryDropdownOpen bool
	var categorySelectBtn widget.Clickable
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
					}
				default:
					break drain
				}
			}

			if startAllBtn.Clicked(gtx) && !isLoading {
				isLoading = true
				progressText = "Starting..."
				go func() {
					progressCh <- progressMsg{Message: "Clearing old jobs and crawling..."}
					if err := client().StartCrawlJobs(); err != nil {
						progressCh <- progressMsg{Message: "Error: " + err.Error()}
					} else {
						progressCh <- progressMsg{Message: "Done."}
					}
					progressCh <- progressMsg{Done: true}
				}()
			}

			if addAccountNavBtn.Clicked(gtx) {
				currentState = ScreenAddAccount
			}

			if settingsNavBtn.Clicked(gtx) {
				// Refresh the field with the current URL whenever we open Settings.
				apiUrlInput.SetText(apiUrl())
				currentState = ScreenSettings
			}

			if backBtn.Clicked(gtx) {
				currentState = ScreenDashboard
				progressText = ""
			}

			if saveAccountBtn.Clicked(gtx) {
				if usernameInput.Text() != "" {
					if err := client().AddAccount("Freelancer Việt", usernameInput.Text()); err != nil {
						progressText = "Error: " + err.Error()
					}
					usernameInput.SetText("")
					currentState = ScreenDashboard
				}
			}

			if saveSettingsBtn.Clicked(gtx) {
				raw := strings.TrimSpace(apiUrlInput.Text())
				if raw == "" {
					progressText = "Error: URL cannot be empty"
				} else if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
					progressText = "Error: URL must start with http:// or https://"
				} else {
					// Swap the client and probe the new endpoint.
					setApiUrl(strings.TrimRight(raw, "/"))
					isLoading = true
					progressText = "Testing connection..."
					go func() {
						start := time.Now()
						if err := client().Ping(); err != nil {
							progressCh <- progressMsg{Message: "Saved, but ping failed: " + err.Error()}
						} else {
							progressCh <- progressMsg{
								Message: fmt.Sprintf("Connected in %s", time.Since(start).Round(time.Millisecond)),
							}
						}
						progressCh <- progressMsg{Done: true}
					}()
				}
			}

			if testSettingsBtn.Clicked(gtx) {
				isLoading = true
				progressText = "Testing connection..."
				go func() {
					start := time.Now()
					if err := client().Ping(); err != nil {
						progressCh <- progressMsg{Message: "Ping failed: " + err.Error()}
					} else {
						progressCh <- progressMsg{
							Message: fmt.Sprintf("Pong in %s", time.Since(start).Round(time.Millisecond)),
						}
					}
					progressCh <- progressMsg{Done: true}
				}()
			}

			// Global padding.
			layout.Inset{
				Top:    unit.Dp(16),
				Bottom: unit.Dp(16),
				Left:   unit.Dp(16),
				Right:  unit.Dp(16),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						title := material.H4(theme, "Job Aggregator")
						title.Color = color.NRGBA{R: 20, G: 80, B: 180, A: 255}
						return title.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Spacer{Height: unit.Dp(4)}.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						endpoint := material.Body2(theme, "API: "+apiUrl())
						endpoint.Color = color.NRGBA{R: 120, G: 120, B: 120, A: 255}
						return endpoint.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						switch currentState {
						case ScreenDashboard:
							return renderDashboard(
								gtx, theme,
								&startAllBtn, &addAccountNavBtn, &settingsNavBtn,
								isLoading, progressText,
								&selectedCategory,
								&categoryDropdownOpen,
								&categorySelectBtn,
								categoryOptionClicks,
							)
						case ScreenAddAccount:
							return renderAddAccount(gtx, theme, &usernameInput, &saveAccountBtn, &backBtn)
						case ScreenSettings:
							return renderSettings(
								gtx, theme,
								&apiUrlInput,
								&saveSettingsBtn, &testSettingsBtn, &backBtn,
								isLoading, progressText,
							)
						}
						return layout.Dimensions{}
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
	startAllBtn, addAccountBtn, settingsBtn *widget.Clickable,
	isLoading bool,
	progressText string,
	selectedCategory *string,
	dropdownOpen *bool,
	selectBtn *widget.Clickable,
	optionClicks map[string]*widget.Clickable,
) layout.Dimensions {
	accounts := client().GetAccounts()
	allJobs := client().GetJobs()

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
	if *selectedCategory != "" && !seen[*selectedCategory] {
		*selectedCategory = ""
	}

	var jobs []rest.Job
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
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if isLoading {
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
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Width: unit.Dp(12)}.Layout(gtx)
				}),
				layout.Rigid(material.Button(theme, settingsBtn, "Settings").Layout),
			)
		}),

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if progressText == "" {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				prefix := ""
				if isLoading {
					prefix = "⏳ "
				}
				lbl := material.Body2(theme, prefix+progressText)
				lbl.Color = color.NRGBA{R: 200, G: 120, B: 0, A: 255}
				return lbl.Layout(gtx)
			})
		}),

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
		}),

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			info := fmt.Sprintf("Accounts: %d  |  Available Jobs Found: %d", len(accounts), len(jobs))
			lbl := material.Body1(theme, info)
			lbl.Color = color.NRGBA{R: 80, G: 80, B: 80, A: 255}
			return lbl.Layout(gtx)
		}),

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

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return renderTableHeader(gtx, theme)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(4)}.Layout(gtx)
		}),

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

// renderSettings provides a form to change the REST API base URL at runtime.
func renderSettings(
	gtx layout.Context,
	theme *material.Theme,
	urlInput *widget.Editor,
	saveBtn, testBtn, backBtn *widget.Clickable,
	isLoading bool,
	progressText string,
) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.H6(theme, "Backend Settings").Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body2(theme, "REST API base URL (e.g. http://localhost:8080)")
			lbl.Color = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
			return lbl.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Editor(theme, urlInput, "http://localhost:8080").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if isLoading {
						btn := material.Button(theme, saveBtn, "Saving...")
						btn.Background = color.NRGBA{R: 160, G: 160, B: 160, A: 255}
						return btn.Layout(gtx)
					}
					return material.Button(theme, saveBtn, "Save").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Width: unit.Dp(12)}.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if isLoading {
						btn := material.Button(theme, testBtn, "Testing...")
						btn.Background = color.NRGBA{R: 160, G: 160, B: 160, A: 255}
						return btn.Layout(gtx)
					}
					return material.Button(theme, testBtn, "Test Connection").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Width: unit.Dp(12)}.Layout(gtx)
				}),
				layout.Rigid(material.Button(theme, backBtn, "Back").Layout),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if progressText == "" {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				prefix := ""
				if isLoading {
					prefix = "⏳ "
				}
				lbl := material.Body2(theme, prefix+progressText)
				lbl.Color = color.NRGBA{R: 200, G: 120, B: 0, A: 255}
				return lbl.Layout(gtx)
			})
		}),
	)
}

// --- Everything below is unchanged from the previous version ---
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

	display := *selected
	if display == "" {
		display = "All"
	}

	// Toggle open/close on the main button.
	if btn.Clicked(gtx) {
		*open = !*open
	}

	// --- Closed state ---
	if !*open {
		b := material.Button(theme, btn, display+"  ▼")
		b.Background = color.NRGBA{R: 230, G: 232, B: 238, A: 255}
		b.Color = color.NRGBA{R: 40, G: 40, B: 40, A: 255}
		return b.Layout(gtx)
	}

	// --- Open state: button + panel of options ---
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			b := material.Button(theme, btn, display+"  ▲")
			b.Background = color.NRGBA{R: 20, G: 120, B: 220, A: 255}
			b.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			return b.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				// Panel background.
				return layout.Stack{}.Layout(gtx,
					layout.Expanded(func(gtx layout.Context) layout.Dimensions {
						defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
						paint.Fill(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
						return layout.Dimensions{Size: gtx.Constraints.Min}
					}),
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						children := make([]layout.FlexChild, 0, len(categories))
						for _, c := range categories {
							cat := c
							clk := optionClicks[cat]
							children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								// Lay out the clickable FIRST so it registers
								// pointer events, then check Clicked().
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
								return clk.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{
										Top:    unit.Dp(6),
										Bottom: unit.Dp(6),
										Left:   unit.Dp(10),
										Right:  unit.Dp(10),
									}.Layout(gtx, lbl.Layout)
								})
							}))
						}
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
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
			layout.Flexed(0.55, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(theme, "Title / Description")
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

func renderTableRow(gtx layout.Context, theme *material.Theme, job rest.Job, isEven bool) layout.Dimensions {
	bgColor := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	if isEven {
		bgColor = color.NRGBA{R: 248, G: 249, B: 250, A: 255}
	}

	// One clickable per job — created fresh each render is fine because
	// Gio keys clicks by pointer identity only within a frame; for state
	// we need it to persist. Use a package-level map keyed by URL.
	clk := clickableFor(job.URL)

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

				// The whole right side (title + desc) is one clickable,
				// so clicking the title text opens the URL.
				if clk.Clicked(gtx) && job.URL != "" {
					openURL(job.URL)
				}

				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,

					// Provider
					layout.Flexed(0.15, func(gtx layout.Context) layout.Dimensions {
						return material.Body2(theme, job.Provider).Layout(gtx)
					}),

					// Title (hyperlink) + description stacked vertically
					layout.Flexed(0.55, func(gtx layout.Context) layout.Dimensions {
						return clk.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,

								// Title — styled like a link, with underline only under the text
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									lbl := material.Body2(theme, job.Title)
									lbl.Color = color.NRGBA{R: 10, G: 90, B: 180, A: 255}
									lbl.Font.Weight = 600
									// Measure the label first.
									dims := lbl.Layout(gtx)
									// Draw underline only under the text width.
									defer clip.Rect{
										Min: image.Point{X: 0, Y: dims.Size.Y - 1},
										Max: image.Point{X: dims.Size.X, Y: dims.Size.Y},
									}.Push(gtx.Ops).Pop()
									paint.Fill(gtx.Ops, color.NRGBA{R: 10, G: 90, B: 180, A: 255})
									return dims
								}),

								// Description
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									if job.Description == "" {
										return layout.Dimensions{}
									}
									return layout.Inset{Top: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										lbl := material.Body2(theme, job.Description)
										lbl.Color = color.NRGBA{R: 120, G: 120, B: 120, A: 255}
										return lbl.Layout(gtx)
									})
								}),
							)
						})
					}),

					// Category
					layout.Flexed(0.15, func(gtx layout.Context) layout.Dimensions {
						return material.Body2(theme, job.Category).Layout(gtx)
					}),

					// Price
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

// clickableFor returns a stable *widget.Clickable keyed by URL,
// so clicks survive across frames.
var (
	clickMu    sync.Mutex
	clickables = map[string]*widget.Clickable{}
)

func clickableFor(key string) *widget.Clickable {
	clickMu.Lock()
	defer clickMu.Unlock()
	c, ok := clickables[key]
	if !ok {
		c = &widget.Clickable{}
		clickables[key] = c
	}
	return c
}

func renderAddAccount(
	gtx layout.Context,
	theme *material.Theme,
	usernameInput *widget.Editor,
	saveBtn, backBtn *widget.Clickable,
) layout.Dimensions {
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
