// Package fundamentus is the FII fundamentals adapter: it fetches the Fundamentus FII
// results table in ONE request and parses price, dividend yield, P/VP, and segment for
// every listed FII (SPEC-006 FR-602, D4). It is a free, no-key source with no official
// API, so we scrape an HTML table — robustly (columns are located by header keyword) and
// defensively (a layout change or fetch failure degrades to ErrProviderUnavailable, never
// corrupting stored data; BR-602). HTTP/parsing live here only; the core stays pure (BR-601).
package fundamentus

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/platform/clock"
	"github.com/biel-ferreira/yield-forge/internal/platform/money"
)

const (
	resultsPath = "/fii_resultado.php"
	// maxBytes caps the response read; the full table (~1500 FIIs) is well under this.
	maxBytes  = 8 << 20 // 8 MiB
	userAgent = "YieldForge/1.0 (+https://github.com/biel-ferreira/yield-forge; market-data ingestion)"
)

// Adapter fetches FII fundamentals from a Fundamentus-compatible endpoint.
type Adapter struct {
	baseURL string
	client  *http.Client
	clock   clock.Clock
}

// New returns a Fundamentus adapter. timeout bounds each request (SPEC-006 FR-608); clk
// stamps ObservedAt/FetchedAt deterministically.
func New(baseURL string, timeout time.Duration, clk clock.Clock) *Adapter {
	return &Adapter{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
		clock:   clk,
	}
}

// FetchFIIQuotes fetches the full table once and returns quotes for the requested tickers.
// A ticker not present in the table is simply absent from the map (a miss, not an error).
// last-dividend is left empty here — it is enriched from Yahoo by the composite (D4).
func (a *Adapter) FetchFIIQuotes(ctx context.Context, tickers []marketdata.Ticker) (map[marketdata.Ticker]marketdata.FIIQuote, error) {
	want := make(map[string]marketdata.Ticker, len(tickers))
	for _, t := range tickers {
		want[t.String()] = t
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+resultsPath, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: fundamentus new request: %v", marketdata.ErrProviderUnavailable, err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: fundamentus request: %v", marketdata.ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: fundamentus status %d", marketdata.ErrProviderUnavailable, resp.StatusCode)
	}

	rows, err := parseTableRows(io.LimitReader(resp.Body, maxBytes))
	if err != nil || len(rows) < 2 {
		return nil, fmt.Errorf("%w: fundamentus parse: empty or unreadable table", marketdata.ErrProviderUnavailable)
	}
	cols := indexColumns(rows[0])
	if cols.ticker < 0 || cols.price < 0 {
		return nil, fmt.Errorf("%w: fundamentus parse: expected columns not found (layout changed?)", marketdata.ErrProviderUnavailable)
	}

	now := a.clock.Now().UTC()
	out := make(map[marketdata.Ticker]marketdata.FIIQuote)
	for _, row := range rows[1:] {
		q, t, ok := a.rowToQuote(row, cols, want, now)
		if ok {
			out[t] = q
		}
	}
	return out, nil
}

// rowToQuote builds a quote from a data row if its ticker is requested and parseable. A bad
// numeric field is tolerated (left zero / Other) so one odd cell never drops a whole row;
// only an unparseable ticker or price skips it.
func (a *Adapter) rowToQuote(row []string, cols columns, want map[string]marketdata.Ticker, now time.Time) (marketdata.FIIQuote, marketdata.Ticker, bool) {
	rawTicker := cell(row, cols.ticker)
	t, err := marketdata.ParseTicker(rawTicker)
	if err != nil {
		return marketdata.FIIQuote{}, marketdata.Ticker{}, false
	}
	if _, requested := want[t.String()]; !requested {
		return marketdata.FIIQuote{}, marketdata.Ticker{}, false
	}
	price, err := money.DecimalToMinor(cell(row, cols.price), 2)
	if err != nil {
		return marketdata.FIIQuote{}, marketdata.Ticker{}, false
	}

	q := marketdata.FIIQuote{
		Ticker:        t,
		PriceCentavos: price,
		Sector:        marketdata.ParseSector(cell(row, cols.sector)),
		Source:        "fundamentus",
		ObservedAt:    now,
		FetchedAt:     now,
	}
	if dy, err := money.DecimalToMinor(stripPercent(cell(row, cols.dividendYield)), 2); err == nil {
		q.DividendYieldBps = int(dy)
	}
	if pvp, err := money.DecimalToMinor(cell(row, cols.pvp), 4); err == nil {
		q.PVPBps = int(pvp)
	}
	return q, t, true
}

// columns holds the 0-based index of each field we read, or -1 when absent.
type columns struct {
	ticker, sector, price, dividendYield, pvp int
}

// indexColumns locates each field by a keyword in the header cells, so column reordering
// does not break parsing.
func indexColumns(header []string) columns {
	c := columns{ticker: -1, sector: -1, price: -1, dividendYield: -1, pvp: -1}
	for i, h := range header {
		h = strings.ToLower(strings.TrimSpace(h))
		switch {
		case c.ticker < 0 && strings.Contains(h, "papel"):
			c.ticker = i
		case c.sector < 0 && strings.Contains(h, "segmento"):
			c.sector = i
		case c.price < 0 && strings.Contains(h, "cota"): // "Cotação"
			c.price = i
		case c.dividendYield < 0 && strings.Contains(h, "dividend yield"):
			c.dividendYield = i
		case c.pvp < 0 && strings.Contains(h, "p/vp"):
			c.pvp = i
		}
	}
	return c
}

func cell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func stripPercent(s string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "%"))
}

// parseTableRows returns every <tr> as a slice of trimmed <td>/<th> cell texts.
func parseTableRows(r io.Reader) ([][]string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	var rows [][]string
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			var cells []string
			for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
				if ch.Type == html.ElementNode && (ch.Data == "td" || ch.Data == "th") {
					cells = append(cells, strings.TrimSpace(textOf(ch)))
				}
			}
			if len(cells) > 0 {
				rows = append(rows, cells)
			}
			return // a <tr> never nests another <tr>
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			visit(ch)
		}
	}
	visit(doc)
	return rows, nil
}

func textOf(n *html.Node) string {
	var sb strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			f(ch)
		}
	}
	f(n)
	return sb.String()
}
