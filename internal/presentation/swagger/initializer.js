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
    requestInterceptor: function (request) {
      if (endpoint(request.url).startsWith("/v1/job-seekers/")) {
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
      if (path === "/v1/job-seekers/login" && response.status === 200) {
        const body = response.obj || JSON.parse(response.text);
        if (typeof body.access_token === "string" && body.access_token) {
          window.ui.preauthorizeApiKey("BearerAuth", bearer(body.access_token));
          window.ui.authActions.persistAuthorizationIfNeeded();
        }
      }
      if (path === "/v1/job-seekers/logout" && response.status === 204) {
        clearAuthorization();
      }
      if ((path === "/v1/job-seekers/me" || path === "/v1/job-seekers/logout") && response.status === 401) {
        clearAuthorization();
      }
      return response;
    },
  });
};
