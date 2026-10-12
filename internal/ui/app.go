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

	var ops op.Ops

	for {
		e := window.Event()
		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			if startAllBtn.Clicked(gtx) {
				log.Println("Cleaning up old jobs and starting job fetcher...")
				db.ClearJobs()

				accounts := db.GetAccounts()
				for _, acc := range accounts {
					if acc.Enabled {
						jobs, err := scraper.FetchJobs(acc.Provider, acc.Username)
						if err == nil {
							log.Printf("Fetched %d jobs for account %s (%s)", len(jobs), acc.Username, acc.Provider)
							for _, job := range jobs {
								db.SaveJob(job.Title, job.URL, job.Category, job.Price, acc.Provider)
							}
						} else {
							log.Printf("Error fetching for account %s: %v", acc.Username, err)
						}
					}
				}
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
							return renderDashboard(gtx, theme, &startAllBtn, &addAccountNavBtn)
						} else {
							return renderAddAccount(gtx, theme, &usernameInput, &saveAccountBtn, &backBtn)
						}
					}),
				)
			})

			e.Frame(gtx.Ops)
		}
	}
}

func renderDashboard(gtx layout.Context, theme *material.Theme, startAllBtn, addAccountBtn *widget.Clickable) layout.Dimensions {
	accounts := db.GetAccounts()
	jobs := db.GetJobs()

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Action Buttons
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(material.Button(theme, startAllBtn, "Start All").Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Width: unit.Dp(12)}.Layout(gtx)
				}),
				layout.Rigid(material.Button(theme, addAccountBtn, "Add Account").Layout),
			)
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
			return material.List(theme, &listState).Layout(gtx, len(jobs), func(gtx layout.Context, i int) layout.Dimensions {
				return renderTableRow(gtx, theme, jobs[i], i%2 == 0)
			})
		}),
	)
}

func renderTableHeader(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	// Table header background box
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
	// Alternating row background colors
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
