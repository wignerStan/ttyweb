import { test as base, mergeTests } from '@playwright/test';
import { test as apiRequestTest } from '@seontechnologies/playwright-utils/api-request/fixtures';
import { test as networkErrorMonitorTest } from '@seontechnologies/playwright-utils/network-error-monitor/fixtures';

// Compose base Playwright with api-request and network-error-monitor fixtures.
// Provides: page, request, apiRequest, context, browser (and network-error-monitoring).
export const test = mergeTests(base, apiRequestTest, networkErrorMonitorTest);
export { expect } from '@playwright/test';
