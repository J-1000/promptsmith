import { test, expect } from '@playwright/test'

test.describe('Prompt Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/prompt/greeting')
  })

  test('shows content tab by default', async ({ page }) => {
    await expect(page.getByText('greeting.prompt')).toBeVisible()
    await expect(page.getByText('You are a helpful assistant')).toBeVisible()
  })

  test('can switch to history tab', async ({ page }) => {
    await page.click('button:has-text("History")')

    await expect(page.getByText('Select two versions to compare')).toBeVisible()
    await expect(page.getByText('Add tone parameter for flexibility')).toBeVisible()
    await expect(page.getByText('Fix greeting for edge cases')).toBeVisible()
  })

  test('can select versions for comparison', async ({ page }) => {
    await page.click('button:has-text("History")')

    // Select first version
    await page.click('text=Add tone parameter for flexibility')
    // Select second version
    await page.click('text=Fix greeting for edge cases')

    // Diff button should now be enabled with version info
    const diffButton = page.getByRole('button', { name: /Diff/ })
    await expect(diffButton).toBeEnabled()
  })

  test('can view diff between versions', async ({ page }) => {
    await page.click('button:has-text("History")')

    await page.click('text=Add tone parameter for flexibility')
    await page.click('text=Fix greeting for edge cases')

    await page.click('button:has-text("Diff")')

    await expect(page.getByText(/Comparing/)).toBeVisible()
  })

  test('diff tab is disabled when less than 2 versions selected', async ({ page }) => {
    await page.click('button:has-text("History")')

    const diffButton = page.getByRole('button', { name: /Diff/ })
    await expect(diffButton).toBeDisabled()

    // Select only one version
    await page.click('text=Add tone parameter for flexibility')
    await expect(diffButton).toBeDisabled()
  })
})
