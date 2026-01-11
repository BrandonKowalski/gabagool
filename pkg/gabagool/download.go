package gabagool

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/internal"
	"github.com/veandco/go-sdl2/ttf"

	"github.com/veandco/go-sdl2/sdl"
)

type Download struct {
	URL         string
	Location    string
	DisplayName string
	Timeout     time.Duration
}

// DownloadError represents a failed download with its error.
type DownloadError struct {
	Download Download
	Error    error
}

// DownloadResult represents the result of the DownloadManager.
type DownloadResult struct {
	Completed []Download
	Failed    []DownloadError
}

type DownloadManagerOptions struct {
	AutoContinue       bool
	MaxConcurrent      int
	InsecureSkipVerify bool
}

type downloadJob struct {
	download       Download
	progress       float64
	totalSize      int64
	downloadedSize int64
	timeout        time.Duration
	isComplete     bool
	hasError       bool
	error          error
	cancelChan     chan struct{}

	lastSpeedUpdate time.Time
	lastSpeedBytes  int64
	currentSpeed    float64
}

type downloadManager struct {
	window             *internal.Window
	downloads          []Download
	downloadQueue      []*downloadJob
	activeJobs         []*downloadJob
	completedDownloads []Download
	failedDownloads    []Download
	errors             []error
	isAllComplete      bool
	maxConcurrent      int
	cancellationError  error

	progressBarWidth  int32
	progressBarHeight int32
	progressBarX      int32

	scrollOffset int32

	headers            map[string]string
	lastInputTime      time.Time
	inputDelay         time.Duration
	insecureSkipVerify bool

	showSpeed bool
}

func newDownloadManager(downloads []Download, headers map[string]string) *downloadManager {
	window := internal.GetWindow()

	responsiveBarWidth := window.GetWidth() * 3 / 4
	if responsiveBarWidth > 900 {
		responsiveBarWidth = 900
	}
	progressBarHeight := int32(40)
	progressBarX := (window.GetWidth() - responsiveBarWidth) / 2

	return &downloadManager{
		window:             window,
		downloads:          downloads,
		downloadQueue:      []*downloadJob{},
		activeJobs:         []*downloadJob{},
		completedDownloads: []Download{},
		failedDownloads:    []Download{},
		errors:             []error{},
		isAllComplete:      false,
		maxConcurrent:      3,
		headers:            headers,
		progressBarWidth:   responsiveBarWidth,
		progressBarHeight:  progressBarHeight,
		progressBarX:       progressBarX,
		scrollOffset:       0,
		lastInputTime:      time.Now(),
		inputDelay:         constants.DefaultInputDelay,
		showSpeed:          false,
	}
}

// DownloadManager manages and displays download progress.
// Returns ErrCancelled if the user cancels the downloads.
func DownloadManager(downloads []Download, headers map[string]string, opts DownloadManagerOptions) (*DownloadResult, error) {
	downloadManager := newDownloadManager(downloads, headers)

	if opts.MaxConcurrent > 0 {
		downloadManager.maxConcurrent = opts.MaxConcurrent
	}
	downloadManager.insecureSkipVerify = opts.InsecureSkipVerify

	result := DownloadResult{
		Completed: []Download{},
		Failed:    []DownloadError{},
	}
	cancelled := false

	if len(downloads) == 0 {
		return &result, nil
	}

	window := internal.GetWindow()
	renderer := window.Renderer
	processor := internal.GetInputProcessor()

	for _, download := range downloads {
		timeout := download.Timeout
		if timeout == 0 {
			timeout = 120 * time.Minute
		}

		job := &downloadJob{
			download:   download,
			timeout:    timeout,
			progress:   0,
			isComplete: false,
			hasError:   false,
			cancelChan: make(chan struct{}),
		}
		downloadManager.downloadQueue = append(downloadManager.downloadQueue, job)
	}

	downloadManager.startNextDownloads()

	downloadManager.render(renderer)
	renderer.Present()

	running := true
	var err error

	for running {
		if event := sdl.WaitEventTimeout(16); event != nil {
			switch event.(type) {
			case *sdl.QuitEvent:
				running = false
				err = sdl.GetError()
				downloadManager.cancelAllDownloads()
				cancelled = true

			case *sdl.KeyboardEvent, *sdl.ControllerButtonEvent, *sdl.ControllerAxisEvent, *sdl.JoyButtonEvent, *sdl.JoyAxisEvent, *sdl.JoyHatEvent:
				inputEvent := processor.ProcessSDLEvent(event.(sdl.Event))
				if inputEvent != nil && inputEvent.Pressed && downloadManager.isInputAllowed() {
					downloadManager.lastInputTime = time.Now()

					if downloadManager.isAllComplete {
						running = false
					} else if inputEvent.Button == constants.VirtualButtonY {
						downloadManager.cancelAllDownloads()
						cancelled = true
					} else if inputEvent.Button == constants.VirtualButtonX {
						downloadManager.showSpeed = !downloadManager.showSpeed
					}
				}
			}
		}

		downloadManager.updateJobStatus()

		if len(downloadManager.activeJobs) < downloadManager.maxConcurrent && len(downloadManager.downloadQueue) > 0 {
			downloadManager.startNextDownloads()
		}

		if len(downloadManager.activeJobs) == 0 && len(downloadManager.downloadQueue) == 0 && !downloadManager.isAllComplete {
			downloadManager.isAllComplete = true

			if opts.AutoContinue && len(downloadManager.failedDownloads) == 0 {
				running = false
			}
		}

		downloadManager.render(renderer)
		renderer.Present()
	}

	if err != nil {
		return nil, err
	}

	if cancelled {
		return nil, ErrCancelled
	}

	result.Completed = downloadManager.completedDownloads

	result.Failed = make([]DownloadError, len(downloadManager.failedDownloads))
	for i, download := range downloadManager.failedDownloads {
		var downloadErr error
		if i < len(downloadManager.errors) {
			downloadErr = downloadManager.errors[i]
		}
		result.Failed[i] = DownloadError{
			Download: download,
			Error:    downloadErr,
		}
	}

	return &result, nil
}

func (dm *downloadManager) isInputAllowed() bool {
	return time.Since(dm.lastInputTime) >= dm.inputDelay
}

func (dm *downloadManager) getAverageSpeed() float64 {
	if len(dm.activeJobs) == 0 {
		return 0
	}

	var totalSpeed float64
	activeCount := 0
	for _, job := range dm.activeJobs {
		if job.currentSpeed > 0 {
			totalSpeed += job.currentSpeed
			activeCount++
		}
	}

	if activeCount == 0 {
		return 0
	}

	return totalSpeed / float64(activeCount)
}

func (dm *downloadManager) startNextDownloads() {
	availableSlots := dm.maxConcurrent - len(dm.activeJobs)
	if availableSlots <= 0 {
		return
	}

	for i := 0; i < availableSlots && len(dm.downloadQueue) > 0; i++ {
		job := dm.downloadQueue[0]
		dm.downloadQueue = dm.downloadQueue[1:]
		dm.activeJobs = append(dm.activeJobs, job)

		go dm.downloadFile(job)
	}
}

func (dm *downloadManager) updateJobStatus() {
	var remaining []*downloadJob

	for _, job := range dm.activeJobs {
		if job.isComplete {
			dm.completedDownloads = append(dm.completedDownloads, job.download)
		} else if job.hasError {
			dm.failedDownloads = append(dm.failedDownloads, job.download)
			dm.errors = append(dm.errors, job.error)
		} else {
			remaining = append(remaining, job)
		}
	}

	dm.activeJobs = remaining
}

func (dm *downloadManager) cancelAllDownloads() {
	for _, job := range dm.activeJobs {
		close(job.cancelChan)
		if !job.isComplete && !job.hasError {
			job.hasError = true
			job.error = fmt.Errorf("download cancelled by user")
			dm.failedDownloads = append(dm.failedDownloads, job.download)
			dm.errors = append(dm.errors, job.error)
		}
	}

	for _, job := range dm.downloadQueue {
		job.hasError = true
		job.error = fmt.Errorf("download cancelled by user")
		dm.failedDownloads = append(dm.failedDownloads, job.download)
		dm.errors = append(dm.errors, job.error)
	}

	dm.activeJobs = []*downloadJob{}
	dm.downloadQueue = []*downloadJob{}
	dm.isAllComplete = true
}

func (dm *downloadManager) downloadFile(job *downloadJob) {
	url := job.download.URL
	filePath := job.download.Location

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		job.hasError = true
		job.error = err
		return
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		job.hasError = true
		job.error = err
		return
	}

	if dm.headers != nil {
		for k, v := range dm.headers {
			req.Header.Add(k, v)
		}
	}

	// Clone the default transport to preserve certifiable's root CA configuration
	transport := http.DefaultTransport.(*http.Transport).Clone()

	// Apply custom pooling and timeout settings
	transport.MaxIdleConns = 100
	transport.IdleConnTimeout = 90 * time.Second
	transport.MaxIdleConnsPerHost = 10

	// Apply InsecureSkipVerify if configured
	if dm.insecureSkipVerify {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	client := &http.Client{
		Timeout:   job.timeout,
		Transport: transport,
	}
	resp, err := client.Do(req)
	if err != nil {
		job.hasError = true
		job.error = err
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		job.hasError = true
		job.error = fmt.Errorf("bad status: %s", resp.Status)
		return
	}

	job.totalSize = resp.ContentLength

	out, err := os.Create(filePath)
	if err != nil {
		job.hasError = true
		job.error = err
		return
	}
	defer out.Close()

	job.lastSpeedUpdate = time.Now()
	job.lastSpeedBytes = 0
	job.currentSpeed = 0

	reader := &progressReader{
		reader: resp.Body,
		onProgress: func(bytesRead int64) {
			job.downloadedSize = bytesRead
			if job.totalSize > 0 {
				job.progress = float64(bytesRead) / float64(job.totalSize)
			}

			now := time.Now()
			elapsed := now.Sub(job.lastSpeedUpdate).Seconds()
			if elapsed >= 0.5 {
				bytesDiff := bytesRead - job.lastSpeedBytes
				job.currentSpeed = float64(bytesDiff) / elapsed
				job.lastSpeedUpdate = now
				job.lastSpeedBytes = bytesRead
			}
		},
		reportInterval: 1024,
	}

	done := make(chan error, 1)
	go func() {
		_, err := io.Copy(out, reader)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			job.hasError = true
			job.error = err
		} else {
			job.isComplete = true
		}
	case <-job.cancelChan:
		job.hasError = true
		job.error = fmt.Errorf("download canceled")
	}
}

func truncateFilename(filename string, maxWidth int32, font *ttf.Font) string {
	surface, _ := font.RenderUTF8Blended(filename, sdl.Color{R: 255, G: 255, B: 255, A: 255})
	if surface == nil {
		return filename
	}
	defer surface.Free()

	if surface.W <= maxWidth {
		return filename
	}

	ellipsis := "..."
	for len(filename) > 5 {
		filename = filename[:len(filename)-1]
		surface, _ := font.RenderUTF8Blended(filename+ellipsis, sdl.Color{R: 255, G: 255, B: 255, A: 255})
		if surface == nil {
			break
		}
		if surface.W <= maxWidth {
			surface.Free()
			return filename + ellipsis
		}
		surface.Free()
	}

	return filename + ellipsis
}

func (dm *downloadManager) render(renderer *sdl.Renderer) {
	renderer.SetDrawColor(0, 0, 0, 255)
	renderer.Clear()

	font := internal.Fonts.SmallFont
	windowWidth := dm.window.GetWidth()
	windowHeight := dm.window.GetHeight()

	contentAreaStart := int32(20)
	contentAreaHeight := windowHeight - 20

	if len(dm.activeJobs) == 0 && dm.isAllComplete {
		var completeColor sdl.Color
		var completeText string

		var downloadText string
		if len(dm.downloads) > 1 {
			downloadText = "All Downloads"
		} else {
			downloadText = "Download"
		}

		if dm.failedDownloads != nil && len(dm.failedDownloads) > 0 {
			if dm.errors != nil && len(dm.errors) > 0 && dm.errors[0] != nil && dm.errors[0].Error() != "download cancelled by user" {
				completeText = fmt.Sprintf("%s Failed!", downloadText)
				completeColor = sdl.Color{R: 255, G: 0, B: 0, A: 255}
			} else {
				completeText = fmt.Sprintf("%s Canceled!", downloadText)
				completeColor = sdl.Color{R: 255, G: 0, B: 0, A: 255}
			}
		} else {
			completeText = fmt.Sprintf("%s Completed!", downloadText)
			completeColor = sdl.Color{R: 100, G: 255, B: 100, A: 255}
		}

		completeSurface, err := font.RenderUTF8Blended(completeText, completeColor)
		if err == nil && completeSurface != nil {
			completeTexture, err := renderer.CreateTextureFromSurface(completeSurface)
			if err == nil {
				centerY := (windowHeight - completeSurface.H) / 2
				completeRect := &sdl.Rect{
					X: (windowWidth - completeSurface.W) / 2,
					Y: centerY,
					W: completeSurface.W,
					H: completeSurface.H,
				}
				renderer.Copy(completeTexture, nil, completeRect)
				completeTexture.Destroy()
			}
			completeSurface.Free()
		}
	} else {
		maxFilenameSurface, _ := font.RenderUTF8Blended("Sample", sdl.Color{R: 255, G: 255, B: 255, A: 255})
		filenameHeight := int32(0)
		if maxFilenameSurface != nil {
			filenameHeight = maxFilenameSurface.H
			maxFilenameSurface.Free()
		}

		spacingBetweenFilenameAndBar := int32(5)
		spacingBetweenDownloads := int32(25)

		singleDownloadHeight := filenameHeight + spacingBetweenFilenameAndBar + dm.progressBarHeight

		// Only add space for individual speed display when there's a single download
		if dm.showSpeed && len(dm.activeJobs) == 1 {
			speedTextHeight := filenameHeight
			singleDownloadHeight += speedTextHeight + 5
		}

		averageSpeedHeight := int32(0)
		if dm.showSpeed && len(dm.activeJobs) > 1 {
			avgSpeed := dm.getAverageSpeed()
			if avgSpeed > 0 {
				avgSpeedMBps := avgSpeed / 1048576.0
				avgSpeedText := fmt.Sprintf("Average Speed: %.2f MB/s", avgSpeedMBps)
				avgSpeedSurface, err := font.RenderUTF8Blended(avgSpeedText, sdl.Color{R: 100, G: 200, B: 255, A: 255})
				if err == nil && avgSpeedSurface != nil {
					avgSpeedTexture, err := renderer.CreateTextureFromSurface(avgSpeedSurface)
					if err == nil {
						avgSpeedRect := &sdl.Rect{
							X: (windowWidth - avgSpeedSurface.W) / 2,
							Y: contentAreaStart,
							W: avgSpeedSurface.W,
							H: avgSpeedSurface.H,
						}
						renderer.Copy(avgSpeedTexture, nil, avgSpeedRect)
						avgSpeedTexture.Destroy()
						averageSpeedHeight = avgSpeedSurface.H + 15 // Add spacing
					}
					avgSpeedSurface.Free()
				}
			}
		}

		if len(dm.activeJobs) > 0 {
			// Check if we have no queued downloads and 1-3 active jobs
			hasNoQueue := len(dm.downloadQueue) == 0

			if hasNoQueue && len(dm.activeJobs) <= 3 {
				// Center 1-3 downloads vertically when no queue
				footerHeight := int32(80)
				availableHeight := contentAreaHeight - footerHeight - averageSpeedHeight

				totalHeight := int32(len(dm.activeJobs))*singleDownloadHeight + int32(len(dm.activeJobs)-1)*spacingBetweenDownloads
				startY := contentAreaStart + averageSpeedHeight + (availableHeight-totalHeight)/2
				if startY < contentAreaStart+averageSpeedHeight {
					startY = contentAreaStart + averageSpeedHeight + 10
				}

				for i, job := range dm.activeJobs {
					itemY := startY + int32(i)*(singleDownloadHeight+spacingBetweenDownloads)
					dm.renderDownloadItem(renderer, job, windowWidth, itemY, filenameHeight, spacingBetweenFilenameAndBar)
				}
			} else {
				dm.renderMultipleDownloads(renderer, windowWidth, contentAreaStart+averageSpeedHeight, contentAreaHeight-averageSpeedHeight, filenameHeight, spacingBetweenFilenameAndBar, spacingBetweenDownloads, singleDownloadHeight)
			}
		}
	}

	var footerHelpItems []FooterHelpItem
	if dm.isAllComplete {
		footerHelpItems = append(footerHelpItems, FooterHelpItem{ButtonName: "A", HelpText: "Close"})
	} else {
		helpText := "Cancel Download"
		if len(dm.downloads) > 1 {
			helpText = "Cancel All Downloads"
		}
		footerHelpItems = append(footerHelpItems, FooterHelpItem{ButtonName: "Y", HelpText: helpText})

		speedToggleText := "Show Speed"
		if dm.showSpeed {
			speedToggleText = "Hide Speed"
		}
		footerHelpItems = append(footerHelpItems, FooterHelpItem{ButtonName: "X", HelpText: speedToggleText})
	}

	renderFooter(renderer, internal.Fonts.SmallFont, footerHelpItems, 20, true, true)
}

func (dm *downloadManager) renderMultipleDownloads(renderer *sdl.Renderer, windowWidth int32, contentAreaStart int32, contentAreaHeight int32, filenameHeight int32, spacingBetweenFilenameAndBar int32, spacingBetweenDownloads int32, singleDownloadHeight int32) {
	maxVisibleDownloads := 3

	remainingTextHeight := int32(0)
	totalRemaining := len(dm.activeJobs) - maxVisibleDownloads + len(dm.downloadQueue)
	if totalRemaining > 0 {
		remainingSurface, _ := internal.Fonts.SmallFont.RenderUTF8Blended("Sample", sdl.Color{R: 150, G: 150, B: 150, A: 255})
		if remainingSurface != nil {
			remainingTextHeight = remainingSurface.H + 15
			remainingSurface.Free()
		}
	}

	totalHeight := int32(maxVisibleDownloads)*singleDownloadHeight + int32(maxVisibleDownloads-1)*spacingBetweenDownloads + remainingTextHeight

	footerHeight := int32(80)
	availableHeight := contentAreaHeight - footerHeight

	startY := contentAreaStart + (availableHeight-totalHeight)/2
	if startY < contentAreaStart {
		startY = contentAreaStart + 10
	}

	renderCount := 0
	for _, job := range dm.activeJobs {
		if renderCount >= maxVisibleDownloads {
			break
		}

		itemY := startY + int32(renderCount)*(singleDownloadHeight+spacingBetweenDownloads)
		dm.renderDownloadItem(renderer, job, windowWidth, itemY, filenameHeight, spacingBetweenFilenameAndBar)
		renderCount++
	}

	if totalRemaining > 0 {
		remainingText := fmt.Sprintf("%d Additional Download%s Queued", totalRemaining, func() string {
			if totalRemaining == 1 {
				return ""
			}
			return "s"
		}())

		remainingSurface, err := internal.Fonts.SmallFont.RenderUTF8Blended(remainingText, sdl.Color{R: 150, G: 150, B: 150, A: 255})
		if err == nil && remainingSurface != nil {
			remainingTexture, err := renderer.CreateTextureFromSurface(remainingSurface)
			if err == nil {
				remainingY := startY + int32(maxVisibleDownloads)*(singleDownloadHeight+spacingBetweenDownloads) + 10
				remainingRect := &sdl.Rect{
					X: (windowWidth - remainingSurface.W) / 2,
					Y: remainingY,
					W: remainingSurface.W,
					H: remainingSurface.H,
				}
				renderer.Copy(remainingTexture, nil, remainingRect)
				remainingTexture.Destroy()
			}
			remainingSurface.Free()
		}
	}
}

func (dm *downloadManager) renderDownloadItem(renderer *sdl.Renderer, job *downloadJob, windowWidth int32, startY int32, filenameHeight int32, spacingBetweenFilenameAndBar int32) {
	font := internal.Fonts.SmallFont

	var displayText string
	if job.download.DisplayName != "" {
		displayText = job.download.DisplayName
	} else {
		displayText = filepath.Base(job.download.Location)
	}

	maxWidth := windowWidth * 3 / 4
	if maxWidth > 900 {
		maxWidth = 900
	}
	displayText = truncateFilename(displayText, maxWidth, font)

	filenameSurface, err := font.RenderUTF8Blended(displayText, sdl.Color{R: 255, G: 255, B: 255, A: 255})
	if err == nil && filenameSurface != nil {
		filenameTexture, err := renderer.CreateTextureFromSurface(filenameSurface)
		if err == nil {
			filenameRect := &sdl.Rect{
				X: (windowWidth - filenameSurface.W) / 2,
				Y: startY,
				W: filenameSurface.W,
				H: filenameSurface.H,
			}
			renderer.Copy(filenameTexture, nil, filenameRect)
			filenameTexture.Destroy()
		}
		filenameSurface.Free()
	}

	progressBarY := startY + filenameHeight + spacingBetweenFilenameAndBar

	progressBarBg := sdl.Rect{
		X: dm.progressBarX,
		Y: progressBarY,
		W: dm.progressBarWidth,
		H: dm.progressBarHeight,
	}

	progressWidth := int32(float64(dm.progressBarWidth) * job.progress)

	// Use smooth progress bar with anti-aliased rounded edges
	internal.DrawSmoothProgressBar(
		renderer,
		&progressBarBg,
		progressWidth,
		sdl.Color{R: 50, G: 50, B: 50, A: 255},
		sdl.Color{R: 100, G: 150, B: 255, A: 255},
	)

	percentText := fmt.Sprintf("%.0f%%", job.progress*100)
	if job.totalSize > 0 {
		downloadedMB := float64(job.downloadedSize) / 1048576.0
		totalMB := float64(job.totalSize) / 1048576.0
		percentText = fmt.Sprintf("%.0f%% (%.1fMB/%.1fMB)", job.progress*100, downloadedMB, totalMB)
	}

	percentSurface, err := font.RenderUTF8Blended(percentText, sdl.Color{R: 255, G: 255, B: 255, A: 255})
	if err == nil && percentSurface != nil {
		percentTexture, err := renderer.CreateTextureFromSurface(percentSurface)
		if err == nil {
			textX := dm.progressBarX + (dm.progressBarWidth-percentSurface.W)/2
			textY := progressBarY + (dm.progressBarHeight-percentSurface.H)/2

			percentRect := &sdl.Rect{
				X: textX,
				Y: textY,
				W: percentSurface.W,
				H: percentSurface.H,
			}
			renderer.Copy(percentTexture, nil, percentRect)
			percentTexture.Destroy()
		}
		percentSurface.Free()
	}

	// Only show individual speed for single downloads; use average speed for multiple
	if dm.showSpeed && job.currentSpeed > 0 && len(dm.activeJobs) == 1 {
		speedMBps := job.currentSpeed / 1048576.0
		speedText := fmt.Sprintf("%.2f MB/s", speedMBps)
		speedSurface, err := font.RenderUTF8Blended(speedText, sdl.Color{R: 150, G: 200, B: 255, A: 255})
		if err == nil && speedSurface != nil {
			speedTexture, err := renderer.CreateTextureFromSurface(speedSurface)
			if err == nil {
				speedX := dm.progressBarX + (dm.progressBarWidth-speedSurface.W)/2
				speedY := progressBarY + dm.progressBarHeight + 5 // 5px below the progress bar

				speedRect := &sdl.Rect{
					X: speedX,
					Y: speedY,
					W: speedSurface.W,
					H: speedSurface.H,
				}
				renderer.Copy(speedTexture, nil, speedRect)
				speedTexture.Destroy()
			}
			speedSurface.Free()
		}
	}
}

type progressReader struct {
	reader         io.Reader
	onProgress     func(bytesRead int64)
	bytesRead      int64
	lastReported   int64
	reportInterval int64
}

func (r *progressReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	r.bytesRead += int64(n)

	if r.bytesRead-r.lastReported >= r.reportInterval {
		if r.onProgress != nil {
			r.onProgress(r.bytesRead)
		}
		r.lastReported = r.bytesRead
	}

	if err != nil && r.onProgress != nil {
		r.onProgress(r.bytesRead)
	}

	return
}
