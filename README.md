# Backend

Job seeker signup follows the existing clean architecture structure:

- `internal/domain/entities`: framework-independent job seeker, Jalali date, and enums.
- `internal/domain/ports`: repository, SMS, password hashing, and token contracts.
- `internal/application`: signup validation, OTP verification, login, and authentication through a unit of work.
- `internal/infrastructure`: GORM models/repositories, Ferzz SMS, bcrypt, and JWT implementations.
- `internal/presentation`: Gin controllers and Bearer authentication middleware. Wire connects implementations to interfaces.

Copy `.env.example` to `.env` when setting up a new environment, then configure PostgreSQL, `SMS_TOKEN`, and a random `JWT_SECRET` of at least 32 bytes. JWT configuration defaults to issuer `zedlancer`, audience `zedlancer-api`, and a one-hour access token lifetime (`JWT_ACCESS_TOKEN_TTL=1h`, configurable from `1m` to `24h`). The application refuses to start with an invalid JWT configuration. The local `.env` is ignored by Git. Run:

```sh
go run ./cmd/migrate
go run ./cmd/app
```

Migration creates `job_seekers`, `job_seeker_signup_otps`, `job_seeker_login_attempts`, and `revoked_access_tokens`. Startup also runs GORM AutoMigrate. Phone and national code have unique database indexes. Jalali birth dates are stored as `YYYY/MM/DD` text; they are validated against Jalali month lengths and leap years before persistence. Interests use a PostgreSQL JSONB array. Pending registrations are stored in JSONB alongside their OTP challenge and contain only the password hash.

API flow:

1. `GET /v1/job-seekers/signup-options` returns the ten Persian cities, ten interests, and allowed genders (`female`, `male`). Send these exact values back.
2. `POST /v1/job-seekers/signup` submits the whole profile and password. If validation passes, it stores a pending registration and automatically sends a six-digit OTP using `https://gateway.ferzz.ir/send`. It returns HTTP 202. Invalid input sends no SMS. No account exists until verification succeeds. `+989...` and `989...` mobile numbers are also accepted and normalized.

```json
{
  "first_name": "علی",
  "last_name": "رضایی",
  "national_code": "0013547895",
  "city": "تهران",
  "address": "خیابان آزادی، پلاک ۱۰",
  "birth_date": {"year": 1380, "month": 1, "day": 1},
  "gender": "male",
  "interests": ["برنامه نویسی", "طراحی"],
  "phone": "09123456789",
  "password": "choose-a-private-password"
}
```

3. `POST /v1/job-seekers/verify-otp` with `{"phone":"09123456789","otp":"123456"}` checks the received code. Successful verification atomically creates the account and consumes the challenge, returning HTTP 201 and public identity fields. Replace `123456` with the code actually received. Verification cannot change the stored profile or password.
4. `POST /v1/job-seekers/login` with `{"phone":"09123456789","password":"choose-a-private-password"}` returns HTTP 200 with `access_token`, `token_type: "Bearer"`, `expires_at`, and the public account identity. Only verified accounts can login.
5. `GET /v1/job-seekers/me` with header `Authorization: Bearer <access_token>` returns the authenticated account identity.
6. `POST /v1/job-seekers/logout` with the same header returns HTTP 204 and revokes that token. Repeating logout with the same unexpired token also returns 204. Other sessions remain active. Reusing a revoked token on a protected endpoint returns 401.

Use the applicant's own national code, which must pass the Iranian checksum. Passwords must be 8–72 bytes and are stored with bcrypt. All displayed profile fields are required, with at least one distinct interest. Birth dates must be real Jalali dates within the past 120 years.

OTP codes expire after five minutes and permit five failed verification attempts, including malformed codes. Pending registration expires after 30 minutes. To resend a code, `POST /v1/job-seekers/otp` with `{"phone":"09123456789"}`. Resend requires a pending validated registration and a 60-second wait. A new challenge replaces the old challenge. If the pending registration expires, submit signup again. Only salted OTP hashes are persisted. Phone-scoped PostgreSQL transaction locks serialize signup, resend, verification, and login requests.

Login returns the same generic error for unknown accounts and incorrect passwords, and uses a dummy bcrypt comparison for unknown phone numbers. Five failed attempts per normalized phone within a 15-minute window block further login attempts until that window ends. Failed attempts are committed to PostgreSQL; successful login clears them. JWTs use HS256 and include the subject, job seeker role, issuer, audience, issued-at, not-before, expiration, and a random token ID. Verification checks signature, algorithm, issuer, audience, time claims, role, and identity; authentication also checks that the account still exists and is verified. Logout persists token IDs and their expiration in PostgreSQL so revocation survives application restarts and applies to all instances. JWTs are never stored in the revocation table. Expired revocation rows can be removed after their `expires_at` time. There is no refresh endpoint; login again after expiry.

Error responses use `{"message":"..."}`: 400 for invalid input or missing pending registration, 401 for invalid OTP/credentials/access token, 409 for an existing phone or national code, 429 for resend cooldown or login throttling, and 503 for SMS delivery failure. Login token responses and `/me` use `Cache-Control: no-store`.

Run checks with `go test ./...` and `go vet ./...`. Unit tests use an in-memory repository and fake SMS sender; they send no real messages.

An opt-in PostgreSQL integration test verifies the full signup → verification → login → JWT authentication flow using a fake SMS sender and rolls back all test fixtures. Run from PowerShell: `$env:RUN_POSTGRES_INTEGRATION='1'; go test ./internal/infrastructure/persistence -run TestPostgresSignupVerificationAndLogin -count=1 -v`. It uses the database in `.env` and runs schema migrations.

To test in Swagger UI, start the server with `go run ./cmd/app` and open <http://localhost:8081/swagger/index.html> (use your `APP_PORT` if different). The UI and its assets are served by the application; no external Swagger account or CDN is needed.

Expand each endpoint and click **Try it out**, then **Execute**:

1. Fetch `/job-seekers/signup-options` to see allowed values.
2. Submit `/job-seekers/signup` with your details, phone, and password. This automatically sends a real SMS.
3. Submit `/job-seekers/verify-otp` with the same phone and received code. The code is never returned by the API.
4. Submit `/job-seekers/login` with the same phone and password. Swagger automatically attaches the returned token to subsequent protected requests.
5. Execute `/job-seekers/me`. Manual **Authorize** also accepts either the bare token or `Bearer <access_token>`.
6. Execute `/job-seekers/logout` to revoke the token. Swagger then clears its authorization. Swagger saves authorization in browser local storage and restores it after refresh or reopening the browser until the original JWT expires (one hour by default). Refreshing never extends expiration. Logout and a 401 from a protected endpoint clear saved authorization; expired or malformed tokens are removed before restoration.

Swagger sends requests to the current server and port automatically. The specification is available at `/swagger/doc.json`, with generated JSON/YAML files in `docs/`. After changing endpoint annotations or request/response schemas, regenerate with `go generate ./cmd/app` and restart the server.

Swagger authentication interceptor tests use Node built-in testing: `node --test internal/presentation/swagger/initializer.test.cjs`. The PostgreSQL integration test also checks HTTP login, authenticated requests, logout, replay rejection, and session isolation.
