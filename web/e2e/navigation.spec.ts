import { test, expect } from '@playwright/test'
import { installApiMocks } from './api-mocks'

test.describe('Navigation', () => {
  test.beforeEach(async ({ page }) => {
    await installApiMocks(page)
  })

  test('home page loads and shows prompts', async ({ page }) => {
    await page.goto('/')

    await expect(page).toHaveTitle('PromptSmith')
    await expect(page.getByRole('heading', { name: 'Prompts' })).toBeVisible()
    await expect(page.getByText('3 prompts tracked')).toBeVisible()
  })

  test('can navigate to prompt detail page', async ({ page }) => {
    await page.goto('/')

    await page.locator('a[href="/prompt/greeting"]').click()

    await expect(page.getByRole('heading', { name: 'greeting' })).toBeVisible()
    await expect(page.getByText('v1.0.2').first()).toBeVisible()
  })

  test('can navigate back to home from prompt page', async ({ page }) => {
    await page.goto('/prompt/greeting')

    await page.click('text=Prompts')

    await expect(page.getByRole('heading', { name: 'Prompts' })).toBeVisible()
  })

  test('logo navigates to home', async ({ page }) => {
    await page.goto('/prompt/greeting')

    await page.click('text=PromptSmith')

    await expect(page.getByRole('heading', { name: 'Prompts' })).toBeVisible()
  })
})
