const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");
const { test } = require("node:test");

function setup(storage = new Map()) {
  let config;
  const authorizations = [];
  const logouts = [];
  let authorized = {};
  const ui = {
    preauthorizeApiKey(name, value) { authorized[name] = { value }; authorizations.push({ name, value }); },
    authActions: {
      logout(names) { for (const name of names) delete authorized[name]; logouts.push(Array.from(names)); },
      persistAuthorizationIfNeeded() { if (config.persistAuthorization) storage.set("authorized", JSON.stringify(authorized)); },
    },
  };
  function bundle(options) {
    config = options;
    if (config.persistAuthorization) {
      authorized = JSON.parse(storage.get("authorized") || "{}");
      for (const [name, auth] of Object.entries(authorized)) authorizations.push({ name, value: auth.value });
    }
    return ui;
  }
  bundle.presets = { apis: {} };
  bundle.plugins = { DownloadUrl: {} };
  const window = {
    location: { href: "http://localhost:8081/swagger/index.html", origin: "http://localhost:8081" },
    localStorage: { getItem(key) { return storage.get(key) ?? null; }, setItem(key, value) { storage.set(key, value); } },
    atob(value) { return Buffer.from(value, "base64").toString("binary"); },
  };
  vm.runInNewContext(fs.readFileSync(path.join(__dirname, "initializer.js"), "utf8"), {
    window, URL, SwaggerUIBundle: bundle, SwaggerUIStandalonePreset: {},
  });
  window.onload();
  return { config, authorizations, logouts, storage };
}

test("successful login automatically authorizes following requests", () => {
  const { config, authorizations } = setup();
  const response = { url: "http://localhost:8081/v1/job-seekers/login", status: 200, obj: { access_token: "test.jwt.token" } };
  assert.equal(config.responseInterceptor(response), response);
  assert.deepEqual(authorizations, [{ name: "BearerAuth", value: "Bearer test.jwt.token" }]);
  assert.equal(config.persistAuthorization, true);
});

test("manual authorization accepts bare tokens and avoids duplicate Bearer prefixes", () => {
  const { config } = setup();
  for (const value of ["test.jwt.token", "Bearer test.jwt.token", "bearer test.jwt.token", "Bearer Bearer test.jwt.token"]) {
    const request = { url: "/v1/job-seekers/me", headers: { authorization: value } };
    assert.equal(config.requestInterceptor(request), request);
    assert.equal(request.headers.authorization, "Bearer test.jwt.token");
  }
});

test("logout clears authorization only after the server confirms revocation", () => {
  const { config, logouts } = setup();
  config.responseInterceptor({ url: "/v1/job-seekers/logout", status: 500 });
  assert.equal(logouts.length, 0);
  config.responseInterceptor({ url: "/v1/job-seekers/logout", status: 204 });
  assert.deepEqual(logouts, [["BearerAuth"]]);
});

test("failed or foreign login responses cannot change authorization", () => {
  const { config, authorizations } = setup();
  config.responseInterceptor({ url: "/v1/job-seekers/login", status: 401 });
  config.responseInterceptor({ url: "https://example.com/v1/job-seekers/login", status: 200, obj: { access_token: "foreign" } });
  assert.equal(authorizations.length, 0);
});

test("login response text is supported when the parsed body is absent", () => {
  const { config, authorizations } = setup();
  config.responseInterceptor({ url: "/v1/job-seekers/login", status: 200, text: JSON.stringify({ access_token: "test.jwt.token" }) });
  assert.deepEqual(authorizations, [{ name: "BearerAuth", value: "Bearer test.jwt.token" }]);
});

function token(exp) {
  return "header." + Buffer.from(JSON.stringify({ exp })).toString("base64url") + ".signature";
}

test("login survives page reload without extending the original expiry", () => {
  const jwt = token(Math.floor(Date.now() / 1000) + 3600);
  const first = setup();
  first.config.responseInterceptor({ url: "/v1/job-seekers/login", status: 200, obj: { access_token: jwt } });
  const second = setup(first.storage);
  assert.deepEqual(second.authorizations, [{ name: "BearerAuth", value: "Bearer " + jwt }]);
});

test("expired or malformed credentials are removed before restoring a page", () => {
  for (const value of [token(Math.floor(Date.now() / 1000) - 1), token(undefined), "invalid-token"]) {
    const storage = new Map([["authorized", JSON.stringify({ BearerAuth: { value: "Bearer " + value }, OtherAuth: { value: "keep" } })]]);
    const state = setup(storage);
    assert.deepEqual(state.authorizations, [{ name: "OtherAuth", value: "keep" }]);
    assert.equal(JSON.parse(storage.get("authorized")).BearerAuth, undefined);
  }
});

test("logout clears saved credentials so refresh cannot restore them", () => {
  const jwt = token(Math.floor(Date.now() / 1000) + 3600);
  const first = setup();
  first.config.responseInterceptor({ url: "/v1/job-seekers/login", status: 200, obj: { access_token: jwt } });
  first.config.responseInterceptor({ url: "/v1/job-seekers/logout", status: 204 });
  assert.deepEqual(setup(first.storage).authorizations, []);
});

test("an unauthorized protected response clears revoked or expired credentials", () => {
  const state = setup();
  state.config.responseInterceptor({ url: "/v1/job-seekers/login", status: 200, obj: { access_token: token(Math.floor(Date.now() / 1000) + 3600) } });
  state.config.responseInterceptor({ url: "/v1/job-seekers/me", status: 401 });
  assert.deepEqual(setup(state.storage).authorizations, []);
});
