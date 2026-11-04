# Backend

Job seeker signup follows the existing clean architecture structure:

- `internal/domain/entities`: job seeker, pending signup OTP, and login attempt records.
- `internal/domain/enums`: separate city, interest, and gender types, allowed values, and membership validation.
- `internal/domain/valueobjects`: Jalali dates, access tokens, and authenticated principals.
- `internal/domain/errors`: framework-independent signup and authentication errors.
- `internal/domain/ports`: separate account, OTP, login attempt, and token revocation repository contracts, plus SMS, password hashing, and token interfaces.
- `internal/application`: signup validation, OTP verification, login, and authentication through a unit of work.
- `internal/infrastructure/persistence/models`: GORM models and table definitions, separate from queries.
- `internal/infrastructure/persistence/repositories/postgres`: focused repositories and explicit domain/model mapping. Other infrastructure packages implement Ferzz SMS, bcrypt, and JWT.
- `internal/presentation`: HTTP request/response schemas, Swagger examples, Gin controllers, shared error mapping, and Bearer authentication middleware. Controllers map transport requests into plain application inputs. Wire connects implementations to interfaces.

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

## Individual employers

The employer account is split into `employers` (shared `id` and `type`), `individual_employers` (one profile per employer ID), and `legal_employers` (the legal entity and its authorized representative). Both employer types have their own signup and authentication endpoints. Each employer type has its own pending signup table. Employer logins share phone-based throttling and transaction locks; job seeker challenges and lockouts remain separate.

The frontend owns its step navigation. The backend receives **one multipart signup submission** containing the complete individual profile, account credentials, and files. It validates and stores that submission, then sends an OTP automatically. A separate verification request checks the phone and code and creates the account. It does not save frontend profile stages.

| Endpoint | Purpose |
| --- | --- |
| `GET /v1/individual-employers/signup-options` | Existing cities/genders plus provinces and their cities |
| `POST /v1/individual-employers/otp` | Resend OTP only for an existing pending signup |
| `POST /v1/individual-employers/signup` | Receive complete multipart information and send OTP automatically; returns 202 |
| `POST /v1/individual-employers/verify-otp` | Verify phone/code and create the employer account; returns 201 |
| `POST /v1/individual-employers/login` | Phone/password login; returns an employer JWT |
| `GET /v1/individual-employers/me` | Authenticated employer identity and review status |
| `POST /v1/individual-employers/logout` | Revoke the current employer token |
| `GET /v1/individual-employers/documents/{kind}` | Private signed download for your own `identity`, `business-license`, or `video` |

In Swagger, expand signup, click **Try it out**, complete its form fields, choose the required files, and submit. Signup has no OTP field: successful submission sends the first SMS automatically. Then call `/individual-employers/verify-otp` with `{"phone":"09123456789","otp":"123456"}` using the actual received code. Do not upload files again. Login is rejected until verification succeeds. Login uses the existing JSON phone/password request format. Swagger automatically attaches and remembers the resulting JWT until its expiry. Swagger has one active authorization at a time; login with the account type whose endpoints you are testing.

Required profile fields: first/last/father name, valid national code, gender, Jalali `birth_year`/`birth_month`/`birth_day`, matching province/city, residence address, ten-digit postal code, phone, and password. Business name, business activity, website/social-page URL, email, and `has_legal_guardian` are optional. Passwords require 12–72 bytes including uppercase, lowercase, a number, and a symbol. The business license (`جواز کسب`) is optional and is saved as SQL NULL when absent.

`identity_document` is required: PDF, PNG, or JPEG, up to 5 MiB. Optional `business_license` uses the same limits. `verification_video` is required: MP4 or MOV, 30–60 seconds, up to 50 MiB. Uploaded content is inspected on the server; filenames and client Content-Type headers are not sufficient. Install `ffprobe` on PATH or set `FFPROBE_PATH` to its executable. ffmpeg is only needed to generate integration-test video fixtures.

All document bytes and the video are stored behind the `ports.Storage` interface using the S3 adapter. PostgreSQL stores the profile and object references/content types/sizes, not the binary files. Keys use random identifiers and do not contain names, phone numbers, or national codes. S3 uploads use no public-read ACL. Download URLs require authentication, are scoped to the requesting employer, expire after five minutes, and are returned with `Cache-Control: no-store`. Successful registration starts with `review_status=pending_review`; document review/approval APIs are not part of this implementation. Phone-verified employers may login while review is pending.

Configure the existing private bucket and credentials before testing real uploads:

```dotenv
S3_BUCKET=your-private-bucket
AWS_REGION=your-bucket-region
S3_ENDPOINT=
S3_USE_PATH_STYLE=false
FFPROBE_PATH=ffprobe
```

The AWS SDK uses its standard credential chain: IAM role, AWS profile, or `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` (plus `AWS_SESSION_TOKEN` for temporary credentials). No credentials are embedded in code or Swagger. For S3-compatible services, set their endpoint and enable path-style addressing if required. Allow `s3:PutObject`, `s3:GetObject`, and `s3:DeleteObject` on `arn:aws:s3:::your-private-bucket/individual-employers/*`. The bucket must already exist and its policy must keep objects private. Missing bucket configuration leaves other APIs available and returns 503 for employer uploads; it never falls back to public or local disk storage.

Pending information is stored in JSONB with a password hash and private file references; no employer account exists yet. OTPs expire after five minutes, allow five failed attempts, and have a 60-second resend cooldown. The pending submission is valid for 30 minutes. Resend requires an existing pending submission and does not re-upload files. After pending expiry, submit the information again. Verification, parent/profile creation, and OTP consumption run atomically within the unit of work. Failed attempts are committed. Verification uses the stored information and cannot change profile fields or the password. Partially uploaded objects are deleted on failed submission; replacing a pending submission removes its old objects only after the replacement succeeds. Cleanup errors are logged for operational follow-up. No raw OTP or password is persisted. The shared JWT implementation validates the account role, so an employer token cannot authenticate as a job seeker even if numeric IDs coincide.

`go test ./...` covers employer validation, nullable license handling, OTP expiry/lockout, password login, JWT role isolation, logout, upload cleanup, actual ffprobe inspection, and the S3 HTTP protocol against a local test server. For the real PostgreSQL multipart flow, run `$env:RUN_POSTGRES_INTEGRATION='1'; go test ./internal/infrastructure/persistence -count=1 -v` in PowerShell. These tests use fake SMS/storage and roll back database fixtures; they send no real messages or live S3 uploads.

## Legal employers

The legal employer flow uses the same clean architecture boundaries and shared OTP, password, JWT, media validation, and S3 interfaces as individual employers. GORM migrates `legal_employers` with a one-to-one `employers` foreign key, unique phone/company national identifier, and `legal_employer_signup_otps` for pending submissions.

| Endpoint | Purpose |
| --- | --- |
| `GET /v1/legal-employers/signup-options` | Legal entity types, representative positions, provinces/cities, genders, and the existing ten activities |
| `POST /v1/legal-employers/signup` | Submit the complete multipart form and automatically send OTP; returns 202 |
| `POST /v1/legal-employers/verify-otp` | JSON phone/otp; atomically create the legal profile and consume pending signup; returns 201 |
| `POST /v1/legal-employers/otp` | Resend OTP for a pending submission, after 60 seconds |
| `POST /v1/legal-employers/login` | JSON phone/password; returns a `legal_employer` JWT |
| `GET /v1/legal-employers/me` | Authenticated company, representative, contact information and review status |
| `POST /v1/legal-employers/logout` | Revoke the current token |
| `GET /v1/legal-employers/documents/{kind}` | Owner-only signed download for `establishment`, `identity`, `activity-license`, `authorization`, or `video` |

Required form fields match the organization and representative screenshots: company name, legal entity type, eleven-digit company national ID, numeric registration number, twelve-digit economic code, matching province/city, office address/postal code, representative first/last name, position, gender, valid personal national code, activities, phone, and password. `has_tax_file` is a boolean; email and website are optional. Select one to ten distinct values from the existing activities list. Send each activity under the repeated multipart key `activities` (Swagger supports this array). Company identifier format is validated locally; official organization and representative authority checks remain a document review process.

Required files are `establishment_document` (establishment notice or official gazette), `identity_document` (representative identity), and `verification_video`. Documents accept PDF/PNG/JPEG up to 5 MiB each; video accepts MP4/MOV, 30-60 seconds, up to 50 MiB. Set `requires_activity_license=true` and upload `activity_license` when the activity requires licensing. Representative position `attorney` requires `authorization_document` (power of attorney). Optional documents persist as SQL NULL when absent. The signup request is limited to 76 MiB. Terms acceptance, password confirmation, and frontend step navigation remain frontend responsibilities.

In Swagger, fetch signup-options, submit the legal signup form and files once, verify the received code using verify-otp, then login. Codes last five minutes, permit five failed attempts, and pending submissions last 30 minutes. Login issues the configured one-hour token by default. Swagger automatically authorizes and preserves it across refresh; logout revokes it. Legal, individual employer, and job seeker roles cannot access one another's protected routes. Phone verification permits login while `review_status` is `pending_review`; document approval is separate.

Legal uploads use private S3 keys under `legal-employers/`; allow the same PutObject/GetObject/DeleteObject permissions on that prefix as for `individual-employers/`. Existing S3 credentials and ffprobe configuration are reused. Failed submissions clean up new objects, while failed replacements preserve the old submission and its files. Verification uses stored information and never re-uploads files. Unit tests and the opt-in PostgreSQL integration flow use fake SMS/storage, real bcrypt/JWT and ffprobe, and roll back database fixtures.
