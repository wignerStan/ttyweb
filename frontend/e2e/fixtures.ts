import { test as base, mergeTests } from '@playwright/test';
import { test as apiRequestTest } from '@seontechnologies/playwright-utils/api-request/fixtures';

// Compose base Playwright with api-request fixture.
// Provides: page, request, apiRequest, context, browser.
// Note: networkErrorMonitor was removed — it auto-fails on 4xx/5xx,
// conflicting with error-testing specs that intentionally verify error responses.
export const test = mergeTests(base, apiRequestTest);
export { expect } from '@playwright/test';
