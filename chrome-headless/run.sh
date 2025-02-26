#!/bin/bash

# Chrome no longer listens on 0.0.0.0 so we need a proxy to get to localhost
exec socat TCP4-LISTEN:1337,fork TCP4:127.0.0.1:1338 &
exec /usr/lib64/chromium-browser/headless_shell --no-sandbox --disable-hang-monitor --disable-features="site-per-process,Translate,BlinkGenPropertyTrees" --force-color-profile=srgb --password-store=basic --disable-popup-blocking --use-mock-keychain --safebrowsing-disable-auto-update --enable-automation --disable-sync --metrics-recording-only --disable-renderer-backgrounding --disable-prompt-on-repost --disable-ipc-flooding-protection --disable-gpu --disable-translate --disable-breakpad --disable-default-apps --disable-dev-shm-usage --disable-client-side-phishing-detection --disable-extensions --disable-background-networking --disable-sync --disable-background-timer-throttling --disable-backgrounding-occluded-windows --safebrowsing-disable-auto-update --metrics-recording-only --disable-default-apps --no-first-run --remote-debugging-port=1338
