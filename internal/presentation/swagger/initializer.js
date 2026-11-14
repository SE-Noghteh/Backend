window.onload = function () {
  function endpoint(url) {
    const parsed = new URL(url, window.location.href);
    return parsed.origin === window.location.origin ? parsed.pathname : "";
  }

  function bearer(value) {
    return "Bearer " + value.trim().replace(/^(Bearer\s+)+/i, "");
  }

  // Remove expired credentials before Swagger restores its saved authorization.
  // Decoding here is only for UI cleanup; the API still verifies the JWT.
  function removeExpiredAuthorization() {
    try {
      const saved = JSON.parse(window.localStorage.getItem("authorized") || "{}");
      if (!saved.BearerAuth) return;
      const value = bearer(saved.BearerAuth.value).slice(7);
      let expired = true;
      try {
        const payload = value.split(".")[1].replace(/-/g, "+").replace(/_/g, "/");
        const claims = JSON.parse(window.atob(payload));
        expired = !Number.isFinite(claims.exp) || claims.exp * 1000 <= Date.now();
      } catch (_) {
        // Malformed saved credentials must not be restored.
      }
      if (expired) {
        delete saved.BearerAuth;
        window.localStorage.setItem("authorized", JSON.stringify(saved));
      }
    } catch (_) {
      // Browser storage can be unavailable; keep Swagger usable in that case.
    }
  }

  function clearAuthorization() {
    window.ui.authActions.logout(["BearerAuth"]);
    window.ui.authActions.persistAuthorizationIfNeeded();
  }

  removeExpiredAuthorization();

  const tagOrder = ["Job seekers", "Parents", "Individual employers", "Legal employers", "Admin"];
  const operationOrder = [
    "get /job-seekers/signup-options",
    "post /job-seekers/signup",
    "post /job-seekers/verify-otp",
    "post /job-seekers/otp",
    "post /job-seekers/login",
    "post /job-seekers/forgot-password",
    "post /job-seekers/reset-password",
    "post /job-seekers/parent-approval/registered",
    "post /job-seekers/parent-approval/invite",
    "get /job-seekers/me",
    "get /job-seekers/documents/video",
    "post /job-seekers/phone-change",
    "post /job-seekers/phone-change/verify",
    "post /job-seekers/logout",
    "delete /job-seekers/me",
    "post /parents/invitations/validate",
    "post /parents/signup",
    "post /parents/verify-otp",
    "post /parents/otp",
    ...["individual-employers", "legal-employers"].flatMap(group => [
      "get /" + group + "/signup-options",
      "post /" + group + "/signup",
      "post /" + group + "/verify-otp",
      "post /" + group + "/otp",
      "post /" + group + "/login",
      "get /" + group + "/me",
      "get /" + group + "/documents/{kind}",
      "post /" + group + "/logout",
    ]),
    "post /admin/login",
    "get /admin/job-seekers",
    "get /admin/job-seekers/{id}",
    "get /admin/job-seekers/{id}/documents/video",
    "post /admin/job-seekers/{id}/review",
    "get /admin/parents",
    "get /admin/parents/{id}",
    "get /admin/parents/{id}/documents/{kind}",
    "post /admin/parents/{id}/review",
  ];
  function rank(list, key) {
    const index = list.indexOf(key);
    return index === -1 ? list.length : index;
  }

  window.ui = SwaggerUIBundle({
    url: "/swagger/doc.json",
    dom_id: "#swagger-ui",
    validatorUrl: null,
    persistAuthorization: true,
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
    plugins: [SwaggerUIBundle.plugins.DownloadUrl],
    layout: "StandaloneLayout",
    docExpansion: "list",
    deepLinking: true,
    defaultModelsExpandDepth: -1,
    tagsSorter: (a, b) => rank(tagOrder, a) - rank(tagOrder, b),
    operationsSorter: (a, b) =>
      rank(operationOrder, a.get("method").toLowerCase() + " " + a.get("path")) -
      rank(operationOrder, b.get("method").toLowerCase() + " " + b.get("path")),
    requestInterceptor: function (request) {
      if (/^\/v1\/(job-seekers|individual-employers|legal-employers|admin)\//.test(endpoint(request.url))) {
        for (const name of Object.keys(request.headers || {})) {
          if (name.toLowerCase() === "authorization" && request.headers[name]) {
            request.headers[name] = bearer(request.headers[name]);
          }
        }
      }
      return request;
    },
    responseInterceptor: function (response) {
      const path = endpoint(response.url);
      if (/^(\/v1\/(job-seekers|individual-employers|legal-employers|admin)\/login)$/.test(path) && response.status === 200) {
        const body = response.obj || JSON.parse(response.text);
        if (typeof body.access_token === "string" && body.access_token) {
          window.ui.preauthorizeApiKey("BearerAuth", bearer(body.access_token));
          window.ui.authActions.persistAuthorizationIfNeeded();
        }
      }
      if (/^(\/v1\/(job-seekers|individual-employers|legal-employers|admin)\/logout)$/.test(path) && response.status === 204) {
        clearAuthorization();
      }
      if (/^\/v1\/(job-seekers|individual-employers|legal-employers|admin)\/(me|logout|documents\/[^/]+)$/.test(path) && response.status === 401) {
        clearAuthorization();
      }
      return response;
    },
  });
};
