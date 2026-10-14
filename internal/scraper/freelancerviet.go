package scraper

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
)

type FreelancerVietProvider struct {
	browser *rod.Browser
	mu      sync.Mutex // guards browser init
}

func init() {
	Register(&FreelancerVietProvider{})
}

func (f *FreelancerVietProvider) Name() string {
	return "Freelancer Việt"
}

const (
	freelancerVietBaseURL = "https://freelancerviet.vn"
	freelancerVietListURL = "https://freelancerviet.vn/viec-lam-freelance.html"
	maxPages              = 5
	concurrency           = 3 // number of concurrent browser pages

	// Job card container: two classes "card-body" and "p-10".
	jobCardSelector = ".card-body.p-10"
)

// ensureBrowser lazily starts Chrome and reuses it across calls.
// Safe for concurrent use.
func (f *FreelancerVietProvider) ensureBrowser() (*rod.Browser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.browser != nil {
		return f.browser, nil
	}

	headless := os.Getenv("HEADLESS") != "0"
	u := launcher.New().
		Headless(headless).
		Set("disable-blink-features", "AutomationControlled").
		Set("no-sandbox").
		MustLaunch()

	f.browser = rod.New().ControlURL(u).MustConnect()
	return f.browser, nil
}

// Close releases the browser. Call this from main on shutdown.
func (f *FreelancerVietProvider) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.browser != nil {
		_ = f.browser.Close()
		f.browser = nil
	}
}

// pageResult is the outcome of scraping one page.
type pageResult struct {
	page int
	jobs []JobItem
	err  error
}

// FetchJobs scrapes the job listing pages concurrently using a fixed-size
// worker pool. Pages are dispatched to workers as soon as they are known,
// and results are collected and merged while preserving page order.
func (f *FreelancerVietProvider) FetchJobs(username string) ([]JobItem, error) {
	fmt.Printf("[%s] Starting concurrent scrape (concurrency=%d) for account: %s\n",
		f.Name(), concurrency, username)

	browser, err := f.ensureBrowser()
	if err != nil {
		return nil, fmt.Errorf("failed to start browser: %w", err)
	}

	// Build the list of page URLs up-front. We don't know the real number of
	// pages in advance, so we optimistically schedule maxPages and stop as
	// soon as we detect an empty/error page. The first page is the list URL;
	// subsequent pages follow the "/page_N.html" pattern.
	pageURLs := make([]string, maxPages)
	pageURLs[0] = freelancerVietListURL
	for i := 1; i < maxPages; i++ {
		pageURLs[i] = fmt.Sprintf("%s/viec-lam-freelance/page_%d.html", freelancerVietBaseURL, i+1)
	}

	// Set up the worker pool.
	jobsCh := make(chan pageResult, maxPages)
	urlCh := make(chan struct {
		index int
		url   string
	}, maxPages)

	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range urlCh {
				doc, err := f.fetchDocument(browser, job.url)
				var pageJobs []JobItem
				if err == nil {
					pageJobs = f.parseJobCards(doc)

					if os.Getenv("DEBUG_DUMP_HTML") == "1" {
						if derr := dumpHTML(doc, fmt.Sprintf("doc_page_%d.html", job.index+1)); derr != nil {
							fmt.Printf("[%s] Warning: failed to dump page %d HTML: %v\n",
								f.Name(), job.index+1, derr)
						}
					}
				}
				jobsCh <- pageResult{page: job.index + 1, jobs: pageJobs, err: err}
			}
		}()
	}

	// Dispatch all page URLs, then close the channel to signal workers.
	go func() {
		for i, u := range pageURLs {
			urlCh <- struct {
				index int
				url   string
			}{index: i, url: u}
		}
		close(urlCh)
	}()

	// Close jobsCh when all workers are done.
	go func() {
		wg.Wait()
		close(jobsCh)
	}()

	// Collect results. Preserve page order for deterministic output.
	results := make([]pageResult, 0, maxPages)
	for r := range jobsCh {
		results = append(results, r)
	}

	// Sort results by page number.
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j-1].page > results[j].page; j-- {
			results[j-1], results[j] = results[j], results[j-1]
		}
	}

	// Merge results, deduping by title+URL. Stop at the first page that
	// errored or returned zero jobs (that's the natural end of pagination).
	var jobs []JobItem
	seen := make(map[string]bool)
	for _, r := range results {
		if r.err != nil {
			if r.page == 1 {
				return nil, fmt.Errorf("page 1 failed: %w", r.err)
			}
			fmt.Printf("[%s] Stopping merge at page %d: %v\n", f.Name(), r.page, r.err)
			break
		}

		if len(r.jobs) == 0 {
			fmt.Printf("[%s] Page %d returned 0 jobs — end of pagination\n", f.Name(), r.page)
			break
		}

		added := 0
		for _, job := range r.jobs {
			if job.Title == "" {
				continue
			}
			key := job.Title + "|" + job.URL
			if seen[key] {
				continue
			}
			seen[key] = true
			jobs = append(jobs, job)
			added++
		}
		fmt.Printf("[%s] Page %d: %d cards, %d new jobs\n", f.Name(), r.page, len(r.jobs), added)
	}

	fmt.Printf("[%s] Total jobs found: %d\n", f.Name(), len(jobs))
	return jobs, nil
}

// fetchDocument drives a real headless browser, waits for React to finish
// rendering the job list, then pulls the DOM directly via Eval (bypassing
// page.HTML(), which can lag behind the live DOM for React apps).
//
// Each call creates its own page, so it is safe to call concurrently.
func (f *FreelancerVietProvider) fetchDocument(browser *rod.Browser, targetURL string) (*html.Node, error) {
	page, err := stealth.Page(browser)
	if err != nil {
		return nil, fmt.Errorf("failed to create stealth page: %w", err)
	}
	defer page.Close()

	page = page.Timeout(60 * time.Second)

	if err := page.Navigate(targetURL); err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("wait load failed: %w", err)
	}

	_ = page.WaitRequestIdle(500*time.Millisecond, nil, nil, nil)

	if err := page.WaitElementsMoreThan(jobCardSelector, 0); err != nil {
		return nil, fmt.Errorf("job cards never appeared: %w", err)
	}

	htmlStr, err := page.Eval(`() => {
		const el = document.getElementById('project-list');
		return el ? el.outerHTML : document.body.outerHTML;
	}`)
	if err != nil {
		return nil, fmt.Errorf("eval outerHTML failed: %w", err)
	}

	raw := htmlStr.Value.Str()

	if os.Getenv("DEBUG_DUMP_HTML") == "1" {
		fmt.Printf("[%s] Eval HTML length = %d, card-body p-10 occurrences = %d\n",
			f.Name(), len(raw), strings.Count(raw, "card-body p-10"))
	}

	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	return doc, nil
}

// dumpHTML serializes an *html.Node to a file for debugging.
func dumpHTML(doc *html.Node, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return html.Render(f, doc)
}

// ---------- Parsing helpers ----------

func (f *FreelancerVietProvider) parseJobCards(doc *html.Node) []JobItem {
	var jobs []JobItem
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && hasClass(n, "card-body") && hasClass(n, "p-10") {
			job := parseCardBody(n)
			if job.Title != "" {
				jobs = append(jobs, job)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)
	return jobs
}

// parseCardBody extracts fields from a single job card.
//
// Card structure (as rendered):
//
//	<div class="card-body p-10">
//	  <div class="row">
//	    <div class="col-md-9">
//	      <h3><a href="...">Title</a></h3>
//	      <ul><li>Category</li>...</ul>
//	    </div>
//	    <div class="col-md-3">
//	      <p class="font-size-24 ...">Price</p>
//	    </div>
//	  </div>
//	</div>
//
// All lookups use recursive search because the fields are nested inside
// wrapper divs, not direct children of the card container.
func parseCardBody(card *html.Node) JobItem {
	var item JobItem

	// 1) Title & URL
	if h3 := findChildTagRecursive(card, "h3"); h3 != nil {
		if a := findChildTagRecursive(h3, "a"); a != nil {
			item.Title = strings.TrimSpace(extractText(a))
			for _, attr := range a.Attr {
				if attr.Key == "href" {
					href := strings.TrimSpace(attr.Val)
					if href != "" {
						if strings.HasPrefix(href, "http") {
							item.URL = href
						} else {
							item.URL = freelancerVietBaseURL + href
						}
					}
					break
				}
			}
		}
	}

	// 2) Category
	if ul := findChildTagRecursive(card, "ul"); ul != nil {
		if li := findChildTagRecursive(ul, "li"); li != nil {
			item.Category = strings.TrimSpace(extractText(li))
		}
	}

	// 3) Price
	var findPrice func(*html.Node)
	findPrice = func(n *html.Node) {
		if item.Price != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "p" && hasClass(n, "font-size-24") {
			item.Price = strings.TrimSpace(extractText(n))
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findPrice(c)
		}
	}
	findPrice(card)

	// 4) Description: the FIRST <p> that is NOT the price paragraph.
	var findDesc func(*html.Node) bool
	findDesc = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "p" && !hasClass(n, "font-size-24") {
			txt := strings.TrimSpace(extractText(n))
			if txt != "" {
				item.Description = shorten(txt, 100)
				return true
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if findDesc(c) {
				return true
			}
		}
		return false
	}
	findDesc(card)

	return item
}

// shorten trims to at most n words and appends "..." if truncated.
func shorten(s string, maxWords int) string {
	s = strings.Join(strings.Fields(s), " ") // collapse whitespace
	words := strings.Fields(s)
	if len(words) <= maxWords {
		return s
	}
	return strings.Join(words[:maxWords], " ") + " ..."
}
func hasClass(n *html.Node, className string) bool {
	if n == nil {
		return false
	}
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			for _, cls := range strings.Fields(attr.Val) {
				if cls == className {
					return true
				}
			}
		}
	}
	return false
}

// findChildTag returns the first DIRECT child element with the given tag.
func findChildTag(n *html.Node, tag string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			return c
		}
	}
	return nil
}

// findChildTagRecursive returns the first DESCENDANT element with the given tag.
func findChildTagRecursive(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findChildTagRecursive(c, tag); found != nil {
			return found
		}
	}
	return nil
}

// findByClass returns the first element (depth-first) that has the given class.
func findByClass(n *html.Node, className string) *html.Node {
	if n.Type == html.ElementNode && hasClass(n, className) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findByClass(c, className); found != nil {
			return found
		}
	}
	return nil
}

func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(extractText(c))
	}
	return sb.String()
}
