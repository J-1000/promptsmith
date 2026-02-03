import puppeteer from 'puppeteer';

async function takeScreenshot(url, outputPath) {
  const browser = await puppeteer.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox']
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1280, height: 800 });

  await page.goto(url, { waitUntil: 'networkidle0' });

  // Wait a bit for fonts to load
  await new Promise(resolve => setTimeout(resolve, 1000));

  await page.screenshot({ path: outputPath, fullPage: false });

  await browser.close();
  console.log(`Screenshot saved to ${outputPath}`);
}

const [,, url, output] = process.argv;

if (!url || !output) {
  console.error('Usage: node screenshot.js <url> <output.png>');
  process.exit(1);
}

takeScreenshot(url, output).catch(err => {
  console.error('Error:', err);
  process.exit(1);
});
