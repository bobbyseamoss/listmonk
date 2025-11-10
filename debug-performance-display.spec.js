const { test, expect } = require("@playwright/test");

test("Debug Email Performance Display Issue", async ({ page }) => {
  // Set up API response capture
  let apiResponse = null;
  let apiError = null;

  // Listen for the API call
  page.on("response", async (response) => {
    if (response.url().includes("/api/campaigns/performance/summary")) {
      try {
        apiResponse = await response.json();
        console.log("\n=== API RESPONSE ===");
        console.log("Status:", response.status());
        console.log("Data:", JSON.stringify(apiResponse, null, 2));
      } catch (e) {
        apiError = e;
        console.log("Error parsing API response:", e.message);
      }
    }
  });

  // Capture console messages
  const consoleMessages = [];
  page.on("console", msg => {
    const text = msg.text();
    consoleMessages.push({ type: msg.type(), text });
    console.log(`[${msg.type()}]`, text);
  });

  // Capture page errors
  const pageErrors = [];
  page.on("pageerror", error => {
    pageErrors.push(error.message);
    console.log("[PAGE ERROR]", error.message);
  });

  console.log("Navigating to login page...");
  await page.goto("https://list.bobbyseamoss.com/admin/campaigns", { timeout: 60000 });

  // Wait for login form
  console.log("Waiting for login form...");
  await page.waitForSelector("input[name=\"username\"]", { timeout: 10000 });

  // Fill in credentials
  console.log("Entering credentials...");
  await page.fill("input[name=\"username\"]", "adam");
  await page.fill("input[name=\"password\"]", "T@intshr3dd3r");

  // Submit login
  console.log("Submitting login...");
  await page.click("button[type=\"submit\"]");

  // Wait for navigation after login
  await page.waitForURL("**/admin/**", { timeout: 10000 });
  console.log("Login successful, current URL:", page.url());

  // Navigate to campaigns page if not already there
  if (!page.url().includes("/campaigns")) {
    console.log("Navigating to campaigns page...");
    await page.goto("https://list.bobbyseamoss.com/admin/campaigns");
  }

  // Wait for page to load
  await page.waitForLoadState("networkidle", { timeout: 30000 });
  console.log("Page loaded");

  // Wait a bit for any async operations
  await page.waitForTimeout(2000);

  // Check if Email Performance section exists
  console.log("\n=== CHECKING EMAIL PERFORMANCE SECTION ===");
  const performanceSection = await page.locator("text=Email Performance Last 30 Days").count();
  console.log("Email Performance section found:", performanceSection > 0);

  if (performanceSection > 0) {
    // Get the entire section content
    const sectionContent = await page.locator("text=Email Performance Last 30 Days").locator("..").locator("..").textContent();
    console.log("Section content:", sectionContent);

    // Try to find specific metrics
    const metrics = ["Open Rate", "Click Rate", "Bounce Rate", "Total Sent"];
    for (const metric of metrics) {
      const metricLocator = page.locator(`text=${metric}`);
      const count = await metricLocator.count();
      if (count > 0) {
        const metricContent = await metricLocator.locator("..").textContent();
        console.log(`${metric}:`, metricContent);
      }
    }
  }

  // Take a screenshot
  await page.screenshot({ path: "/home/adam/listmonk/performance-debug.png", fullPage: true });
  console.log("Screenshot saved to performance-debug.png");

  // Summary
  console.log("\n=== SUMMARY ===");
  console.log("API Response received:", apiResponse !== null);
  console.log("API Error:", apiError ? apiError.message : "None");
  console.log("Console errors:", consoleMessages.filter(m => m.type === "error").length);
  console.log("Page errors:", pageErrors.length);

  if (apiResponse) {
    console.log("\n=== API DATA STRUCTURE ===");
    console.log("Keys:", Object.keys(apiResponse));
    if (apiResponse.data) {
      console.log("Data keys:", Object.keys(apiResponse.data));
      console.log("Data values:", JSON.stringify(apiResponse.data, null, 2));
    }
  }

  if (consoleMessages.filter(m => m.type === "error").length > 0) {
    console.log("\n=== CONSOLE ERRORS ===");
    consoleMessages.filter(m => m.type === "error").forEach(m => {
      console.log(m.text);
    });
  }

  if (pageErrors.length > 0) {
    console.log("\n=== PAGE ERRORS ===");
    pageErrors.forEach(err => {
      console.log(err);
    });
  }
});
