package pdf

import (
	"context"
	"fmt"
	"html"
	"os"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// RenderPDF sobe um Chromium headless (seção 29) dentro do próprio container
// backend, carrega o HTML gerado por BuildDocument via file:// e devolve os
// bytes do PDF em A4, com cabeçalho/rodapé (seção 28). NoSandbox e as flags
// de GPU/dev-shm são necessárias para o Chromium rodar como root dentro de
// um container Linux comum, sem privilégios extra de kernel.
func RenderPDF(ctx context.Context, chromePath, documentHTML, headerTitle string) ([]byte, error) {
	tmpFile, err := os.CreateTemp("", "boraif-*.html")
	if err != nil {
		return nil, fmt.Errorf("creating temp html file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(documentHTML); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("writing temp html file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("closing temp html file: %w", err)
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath),
		chromedp.NoSandbox,
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()

	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, 60*time.Second)
	defer cancelTimeout()

	headerHTML := `<div style="font-size:8px; width:100%; text-align:center; color:#555;">` +
		html.EscapeString(headerTitle) + `</div>`
	footerHTML := `<div style="font-size:8px; width:100%; text-align:center; color:#555;">` +
		`Página <span class="pageNumber"></span> de <span class="totalPages"></span></div>`

	var pdfData []byte
	err = chromedp.Run(taskCtx,
		chromedp.Navigate("file://"+tmpPath),
		chromedp.WaitVisible(`body[data-math-ready="true"]`, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			data, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).
				WithPaperHeight(11.69).
				WithMarginTop(0.6).
				WithMarginBottom(0.6).
				WithMarginLeft(0.5).
				WithMarginRight(0.5).
				WithDisplayHeaderFooter(true).
				WithHeaderTemplate(headerHTML).
				WithFooterTemplate(footerHTML).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfData = data
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("running chromium: %w", err)
	}
	return pdfData, nil
}
