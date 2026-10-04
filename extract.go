package main

import (
	"context"
	"log"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

type ExtractRequest struct {
	Url *string `json:"url" form:"url"`
}

type ExtractedReturn struct {
	Sources string
}

func buildExtracted(extractRequestUrl *string, serverOptions *ServerOptions) (*ExtractedReturn, error) {
	// build context options
	var opts []chromedp.ContextOption
	opts = append(opts, chromedp.WithLogf(log.Printf))
	opts = append(opts, chromedp.WithErrorf(log.Printf))

	if serverOptions.Debug {
		opts = append(opts, chromedp.WithDebugf(log.Printf))
	}

	allocatorContext, _ := chromedp.NewRemoteAllocator(context.Background(), "ws://"+serverOptions.ChromeUri)
	ctx, cancel := chromedp.NewContext(allocatorContext, opts...)

	defer cancel()

	var htmlContent string
	log.Printf("Here extracted 1")

	// Run tasks: navigate to URL, wait for body, and get outer HTML of <html>
	err := chromedp.Run(ctx,
		chromedp.Navigate(*extractRequestUrl),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var resolutionReason string

			// Execute the script and capture its return value into resolutionReason
			err := chromedp.Evaluate(`
				new Promise((resolve) => {
					const startTime = Date.now();
					let debugMsg = [];
					
					function logEvent(msg) {
						const elapsed = ((Date.now() - startTime) / 1000).toFixed(2) + "s";
						debugMsg.push("[" + elapsed + "] " + msg);
					}

					function findRootElement() {
						return document.querySelector('*[id*="root"], *[class*="root"]') || 
							Array.from(document.querySelectorAll('*')).find(el => el.tagName.toLowerCase().endsWith('-root')) || 
							document.body;
					}

					logEvent("Script execution started. Beginning Angular boot polling.");

					let checks = 0;
					const maxChecks = 35; 
					
					const bootInterval = setInterval(() => {
						checks++;
						const rootEl = findRootElement();
						const hasAngular = window.getAngularTestability && window.getAngularTestability(rootEl);

						if (hasAngular) {
							clearInterval(bootInterval);
							logEvent("Angular framework detected globally on root element: <" + rootEl.tagName.toLowerCase() + ">");
							executeAngularStrategy(rootEl);
							return;
						}

						if (checks >= maxChecks) {
							clearInterval(bootInterval);
							logEvent("Angular boot polling exceeded max checks. Routing to regular site fallback.");
							executeNonAngularStrategy();
						}
					}, 100);

					// 2. Advanced Angular Task-Queue Strategy
					function executeAngularStrategy(rootEl) {
						const testability = window.getAngularTestability(rootEl);
						
						if (testability.isStable()) {
							logEvent("Angular status: reportStable is true.");
							resolve(debugMsg.join(" | ") + " | SUCCESS: Instantly stable.");
							return;
						}

						logEvent("Angular status: reportStable is false. Inspecting Zone.js task queues.");

						// Fallback natural stability hook
						testability.whenStable(() => {
							logEvent("Angular framework naturally evented whenStable().");
							resolve(debugMsg.join(" | ") + " | SUCCESS: Natural stability.");
						});

						// Access modern Angular Zone task tracking natively
						let angularZone = window.Zone && window.Zone.current && window.Zone.current._zoneDelegate;
						
						// Poll the task queues directly. We wait until active HTTP requests/Promises clear out,
						// or until a hard maximum threshold is met.
						let queueSilenceTicks = 0;
						const queuePollInterval = setInterval(() => {
							let hasActiveRenderTasks = false;

							if (window.Zone && window.Zone.current) {
								const currentZone = window.Zone.current;
								// Retrieve raw task structures managed by Zone.js
								const hasMacro = currentZone._properties && currentZone._properties.macroTasks && currentZone._properties.macroTasks.length > 0;
								
								// Fallback to checking internal task queues if standard properties are hidden
								if (hasMacro) {
									hasActiveRenderTasks = true;
								}
							}

							// Supplement queue checking with structural DOM checks.
							// If the app is truly stuck on an infinite background loop, 
							// we make sure we give it at least 3.5 seconds total runtime before forcing a DOM silence check.
							const executionAge = Date.now() - startTime;
							if (!hasActiveRenderTasks && executionAge > 3500) {
								queueSilenceTicks++;
								if (queueSilenceTicks >= 15) { // 1000ms of complete structural and queue silence
									clearInterval(queuePollInterval);
									logEvent("Task queues cleared and DOM settled. Safe rendering window confirmed.");
									resolve(debugMsg.join(" | ") + " | SUCCESS: Queue drained execution.");
								}
							} else {
								queueSilenceTicks = 0; // Reset if tasks reappear or if we haven't hit minimum app runway time
							}
						}, 100);

						// Hard timeout safety net (Increased to 7.5s to ensure slow APIs finish emitting content)
						setTimeout(() => {
							clearInterval(queuePollInterval);
							logEvent("Hard threshold timeout ceiling reached.");
							resolve(debugMsg.join(" | ") + " | TIMEOUT: Forced hard cutoff boundary.");
						}, 7500);
					}

					// 3. Regular/Non-Angular Fallback Logic
					function executeNonAngularStrategy() {
						let staticTimer;
						const observer = new MutationObserver(() => {
							clearTimeout(staticTimer);
							staticTimer = setTimeout(() => { 
								observer.disconnect(); 
								logEvent("Standard Page: DOM silence window reached.");
								resolve(debugMsg.join(" | ") + " | SUCCESS: Non-Angular stabilization."); 
							}, 500);
						});
						observer.observe(document.body, { childList: true, subtree: true, attributes: true });
						
						setTimeout(() => { 
							observer.disconnect(); 
							logEvent("Standard Page: Safety cutoff reached.");
							resolve(debugMsg.join(" | ") + " | TIMEOUT: Non-Angular hard cutoff."); 
						}, 2500);
					}
				})`,
				&resolutionReason,
				func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
					return p.WithAwaitPromise(true)
				},
			).Do(ctx)

			if err != nil {
				log.Printf("Error executing evaluation: %v", err)
			} else {
				// Log why the execution completed
				log.Printf("Render Lifecycle Log: %s", resolutionReason)
			}
			return nil
		}),
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		log.Fatal(err)
	}

	return &ExtractedReturn{Sources: htmlContent}, nil
}
