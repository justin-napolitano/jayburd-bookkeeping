import { test, expect } from 'playwright/test';

// Noncredential synthetic identifier; no fixture can authenticate with Google.
const key = 'AIza' + 'A'.repeat(35);
test('configuration is memory-only and the entry has no finance routes', async ({page, context}) => {
  const external: string[] = [];
  await context.route('**/*', route => {
    if (new URL(route.request().url()).origin !== 'http://127.0.0.1:39091') { external.push(route.request().url()); return route.abort(); }
    return route.continue();
  });
  await page.goto('/');
  await expect(page.getByRole('heading', {name: 'Connect development sign-in'})).toBeVisible();
  await expect(page.getByLabel('Password', {exact: true})).toHaveCount(0);
  await page.getByLabel('Public Firebase web API key').fill('invalid');
  await page.getByRole('button', {name: 'Continue to sign in'}).click();
  await expect(page.getByRole('alert')).toBeVisible();
  await page.getByLabel('Public Firebase web API key').fill(key);
  await page.getByRole('button', {name: 'Continue to sign in'}).click();
  await expect(page.getByLabel('Password', {exact: true})).toBeVisible();
  expect(external).toEqual([]);
  expect(await page.evaluate(() => [localStorage.length, sessionStorage.length])).toEqual([0, 0]);
  expect((await page.request.get('/v1/entities')).status()).toBe(404);
  await page.reload();
  await expect(page.getByRole('heading', {name: 'Connect development sign-in'})).toBeVisible();
});

test('production login requests verification explicitly against intercepted synthetic provider responses', async ({page, context}) => {
  let sends = 0;
  const prohibited: string[] = [];
  const token = [btoa('{}'), btoa(JSON.stringify({sub:'synthetic-user',iat:Math.floor(Date.now()/1000),exp:Math.floor(Date.now()/1000)+3600})), 'synthetic'].join('.');
  await context.route('**/*', route => {
    const url = new URL(route.request().url());
    if (url.origin === 'http://127.0.0.1:39091') {
      if (url.pathname.startsWith('/v1/')) prohibited.push(url.pathname);
      return route.continue();
    }
    if (url.hostname === 'identitytoolkit.googleapis.com') {
      const headers = {'access-control-allow-origin':'http://127.0.0.1:39091', 'access-control-allow-headers':'content-type,x-client-version,x-firebase-gmpid,x-firebase-client', 'content-type':'application/json'};
      if (route.request().method() === 'OPTIONS') return route.fulfill({status:204,headers});
      if (url.pathname.endsWith('accounts:signInWithPassword')) return route.fulfill({headers,json:{localId:'synthetic-user',email:'fixture@example.invalid',idToken:token,refreshToken:'synthetic-refresh',expiresIn:'3600'}});
      if (url.pathname.endsWith('accounts:lookup')) return route.fulfill({headers,json:{users:[{localId:'synthetic-user',email:'fixture@example.invalid',emailVerified:false,providerUserInfo:[{providerId:'password',email:'fixture@example.invalid'}]}]}});
      if (url.pathname.endsWith('accounts:sendOobCode')) { sends++; return route.fulfill({headers,json:{email:'fixture@example.invalid'}}); }
    }
    prohibited.push(url.origin + url.pathname); return route.abort();
  });
  await page.goto('/');
  await page.getByLabel('Public Firebase web API key').fill(key);
  await page.getByRole('button', {name:'Continue to sign in'}).click();
  await page.getByLabel('Email', {exact:true}).fill('fixture@example.invalid');
  await page.getByLabel('Password', {exact:true}).fill('synthetic-password');
  await page.getByRole('button', {name:'Sign in',exact:true}).click();
  await expect(page.getByRole('heading', {name:'Verify your email'})).toBeVisible();
  expect(sends).toBe(0);
  await page.getByRole('button', {name:'Send verification email'}).click();
  await expect(page.getByRole('status')).toContainText('Verification email requested');
  expect(sends).toBe(1);
  await page.getByRole('button', {name:'Cancel and sign out'}).click();
  await expect(page.getByLabel('Password', {exact:true})).toHaveValue('');
  expect(prohibited).toEqual([]);
  expect(await page.evaluate(() => [localStorage.length,sessionStorage.length])).toEqual([0,0]);
});
