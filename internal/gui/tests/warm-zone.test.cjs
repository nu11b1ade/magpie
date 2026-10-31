// Settings › Allowances' Time zone: daily warm-up HH:MM is read in the
// zone named here. This computer is the default and shows the machine's
// zone (the container's, in Docker). Picking Asia/Shanghai posts warmZone.
// The app's own menu, no <select>, a click doesn't scroll the page.
// en, zh, zh-TW, ja, de.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");
const { watchSettingsReads, settingsDrawn, reach } = require("./reader.cjs");

const assets = path.resolve(__dirname, "../assets");

function serve(lang, posts) {
  let settings = {
    theme: "light", lang, currency: "usd", web: true,
    codexWarmup: "", claudeWarmup: "", codexWarmAt: "", claudeWarmAt: "",
    warmZone: "", localZone: "Asia/Shanghai",
  };
  return async (route) => {
    const req = route.request(), url = new URL(req.url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:${JSON.stringify(lang)},theme:"light",web:true};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings });
    if (url.pathname === "/api/settings") {
      if (req.method() === "POST") {
        const body = req.postDataJSON();
        posts.push(body);
        settings = { ...settings, ...body, localZone: "Asia/Shanghai" };
      }
      return json(settings);
    }
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true } });
    if (url.pathname === "/api/groups") return json({ groups: [], models: [] });
    if (url.pathname === "/api/plugins") return json({ plugins: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await route.fulfill({ body: await fs.readFile(file), contentType }); } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

const W = {
  en: { zone: "Time zone", line: "Daily warm-up times, such as 06:00, are read in this zone", here: "This computer (Asia/Shanghai)", find: "Find a time zone" },
  zh: { zone: "时区", line: "每日定时预热的时间（如 06:00）按此时区读取", here: "本机（Asia/Shanghai）", find: "查找时区" },
  "zh-TW": { zone: "時區", line: "每日定時預熱的時間（如 06:00）按此時區讀取", here: "本機（Asia/Shanghai）", find: "尋找時區" },
  ja: { zone: "タイムゾーン", line: "毎日のウォームアップ時刻（06:00 など）はこのタイムゾーンで読まれます", here: "このコンピュータ（Asia/Shanghai）", find: "タイムゾーンを探す" },
  de: { zone: "Zeitzone", line: "Tägliche Vorwärm-Zeiten wie 06:00 werden in dieser Zeitzone gelesen", here: "Dieser Computer (Asia/Shanghai)", find: "Zeitzone suchen" },
};

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(`${engine} 440px: the time zone row stays inside the page`, async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    const page = await (await browser.newContext({ viewport: { width: 440, height: 800 }, locale: "en-US" })).newPage();
    page.setDefaultTimeout(8000);
    await page.route("**/*", serve("en", []));
    await watchSettingsReads(page);
    await page.goto("http://magpie.test/?view=settings&tab=usage");
    await settingsDrawn(page);
    const bad = await page.evaluate(() => {
      const de = document.documentElement, view = document.querySelector("#view-settings");
      const out = [];
      if (de.scrollWidth > de.clientWidth) out.push("page");
      if (view.scrollWidth > view.clientWidth) out.push("view");
      const row = document.querySelector("#warmZoneList .row"), rr = row.getBoundingClientRect();
      for (const e of row.querySelectorAll(".name, .sub, button")) {
        const b = e.getBoundingClientRect();
        if (b.right > rr.right + 0.5 || b.left < rr.left - 0.5) out.push(e.id || e.className);
      }
      return out;
    });
    assert.deepEqual(bad, []);
  });
  for (const lang of ["en", "zh", "zh-TW", "ja", "de"]) {
    const w = W[lang];
    test(`${engine} ${lang}: Time zone defaults to this computer and saves a pick`, async (t) => {
      const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
      t.after(() => browser.close());
      const page = await (await browser.newContext({ viewport: { width: 900, height: 800 }, locale: lang === "zh" ? "zh-CN" : lang === "zh-TW" ? "zh-TW" : lang === "ja" ? "ja-JP" : lang === "de" ? "de-DE" : "en-US" })).newPage();
      page.setDefaultTimeout(8000);
      page.setDefaultNavigationTimeout(60000);
      const errors = [], posts = [];
      page.on("pageerror", (e) => errors.push(e.message));
      await page.route("**/*", serve(lang, posts));
      await watchSettingsReads(page);
      await page.goto("http://magpie.test/?view=settings&tab=usage");
      await settingsDrawn(page);
      const btn = page.locator("#warmZonePick");
      await btn.waitFor();
      assert.equal(await page.locator("select").count(), 0);
      assert.equal((await btn.locator("span").innerText()).trim(), w.here);
      assert.equal((await page.locator("#warmZoneSub").innerText()).trim(), w.line);
      assert.equal(await btn.getAttribute("aria-label"), w.zone);
      const view = page.locator("#view-settings");
      await reach(page, btn);
      const top = await view.evaluate((e) => e.scrollTop);
      await btn.click();
      const menu = page.locator(".proto-menu");
      await menu.waitFor();
      assert.equal(await menu.locator(".sc-q").getAttribute("placeholder"), w.find);
      await menu.getByRole("menuitemradio", { name: "Asia/Shanghai", exact: true }).click();
      for (let i = 0; i < 50 && posts.length < 1; i++) await page.waitForTimeout(40);
      assert.equal(posts.length, 1);
      assert.equal(posts[0].warmZone, "Asia/Shanghai");
      await page.waitForFunction(() => document.querySelector("#warmZonePick span")?.textContent?.trim() === "Asia/Shanghai");
      assert.equal((await btn.locator("span").innerText()).trim(), "Asia/Shanghai");
      assert.equal(await view.evaluate((e) => e.scrollTop), top);
      assert.deepEqual(errors, []);
    });
  }
}
