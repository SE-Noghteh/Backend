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

1. `GET /v1/job-seekers/signup-options` returns the Persian city names, ten interests, and allowed genders (`female`, `male`). Use `GET /v1/provinces` for all 31 provinces and `GET /v1/cities?province=...` to list cities for the selected province. Send the exact names returned by the location endpoints.
2. `POST /v1/job-seekers/signup` accepts one `multipart/form-data` submission with the whole profile, password, and required `identity_document` and `verification_video`. Use `GET /v1/provinces` and `GET /v1/cities?province=...` for matching `province` and `city`, and send `postal_code` as a 10-digit string. Use `birth_year`, `birth_month`, and `birth_day` for the Jalali birth date, and repeat `interests` for each selected interest. JSON signup bodies are no longer accepted. The first page of the birth certificate must be a PNG or JPEG up to 5 MiB. The video must be MP4/MOV/WebM, 1-20 seconds, up to 50 MiB. The complete request is limited to 56 MiB. Configure S3 and ffprobe before signup. After validation and private upload, the API stores a pending registration and sends a six-digit OTP using `https://gateway.ferzz.ir/send`, returning HTTP 202. No account exists until verification succeeds. `+989...` and `989...` mobile numbers are normalized.

```sh
curl -X POST http://localhost:8081/v1/job-seekers/signup \
  -F 'first_name=Test' -F 'last_name=User' \
  -F 'national_code=0013547895' \
  -F 'province=تهران' -F 'city=تهران' -F 'address=Your address' \
  -F 'postal_code=0123456789' \
  -F 'birth_year=1388' -F 'birth_month=1' -F 'birth_day=1' \
  -F 'gender=male' -F 'interests=INTEREST_FROM_SIGNUP_OPTIONS' \
  -F 'phone=09123456789' -F 'password=choose-a-private-password' \
  -F 'identity_document=@birth-certificate.png' -F 'verification_video=@verification.mp4'
```

3. `POST /v1/job-seekers/verify-otp` with `{"phone":"09123456789","otp":"123456"}` checks the received code. Successful verification atomically creates the account and consumes the challenge, returning HTTP 201 and public identity fields. Replace `123456` with the code actually received. Verification cannot change the stored profile or password.
4. `POST /v1/job-seekers/login` with `{"phone":"09123456789","password":"choose-a-private-password"}` returns HTTP 200 with `access_token`, `token_type: "Bearer"`, `expires_at`, and the public account identity. Only verified accounts can login.
5. `GET /v1/job-seekers/me` with header `Authorization: Bearer <access_token>` returns the authenticated account identity.
6. `POST /v1/job-seekers/logout` with the same header returns HTTP 204 and revokes that token. Repeating logout with the same unexpired token also returns 204. Other sessions remain active. Reusing a revoked token on a protected endpoint returns 401.
7. `GET /v1/job-seekers/documents/video` with the same authorization returns a private download link for the authenticated job seeker's video, valid for five minutes and returned with `Cache-Control: no-store`.

Job seeker documents use random private keys under `job-seekers/` in the existing S3 bucket. Include that prefix in your bucket credentials' PutObject/GetObject/DeleteObject permissions. PostgreSQL stores the document metadata in the `identity_document` and `verification_video` JSONB columns. Pending signup also stores both references, and OTP resend reuses both files. The new columns are nullable for older accounts. Previously submitted, unexpired pending registrations can still verify under the existing OTP rules. Failed submissions clean up new uploads. Replacing a pending signup or deleting an account cleans up its files after commit. Admins can obtain a five-minute download link for the birth certificate from `GET /v1/admin/job-seekers/{id}/documents/identity`.

The shared location catalog uses the [end-of-1404 Iranian administrative divisions workbook](https://github.com/sajaddp/list-of-cities-in-Iran/blob/ace21bb251f96ced350bbc01541acffd71121feb/offical/list.xlsx). It contains 31 provinces and 1,458 province–city choices. Names come from province (`Coderec=1`) and city (`Coderec=5`) rows, with Persian ی and ک normalized. Numbered subdivisions are omitted when a single corresponding city exists in the same county. The API retains names shared by multiple provinces in each province's list.

Use the applicant's own national code, which must pass the Iranian checksum. Passwords must be 8â€“72 bytes and are stored with bcrypt. All displayed profile fields are required, with at least one distinct interest. Birth dates must be real Jalali dates. Job seekers must be between 15 and 22 years old inclusive, based on their full birth date. Signup returns HTTP 400 before uploading files or sending an OTP when the applicant is outside this range: `{"message":"invalid signup: job seeker is under age: minimum age is 15 years"}` for applicants under 15, or `{"message":"invalid signup: job seeker is over age: maximum age is 22 years"}` for applicants over 22.

OTP codes expire after five minutes and permit five failed verification attempts, including malformed codes. Pending registration expires after 30 minutes. To resend a code, `POST /v1/job-seekers/otp` with `{"phone":"09123456789"}`. Resend requires a pending validated registration and a 60-second wait. A new challenge replaces the old challenge. If the pending registration expires, submit signup again. Only salted OTP hashes are persisted. Phone-scoped PostgreSQL transaction locks serialize signup, resend, verification, and login requests.

Login returns the same generic error for unknown accounts and incorrect passwords, and uses a dummy bcrypt comparison for unknown phone numbers. Five failed attempts per normalized phone within a 15-minute window block further login attempts until that window ends. Failed attempts are committed to PostgreSQL; successful login clears them. JWTs use HS256 and include the subject, job seeker role, issuer, audience, issued-at, not-before, expiration, and a random token ID. Verification checks signature, algorithm, issuer, audience, time claims, role, and identity; authentication also checks that the account still exists and is verified. Logout persists token IDs and their expiration in PostgreSQL so revocation survives application restarts and applies to all instances. JWTs are never stored in the revocation table. Expired revocation rows can be removed after their `expires_at` time. There is no refresh endpoint; login again after expiry.

Error responses use `{"message":"..."}`: 400 for invalid input or missing pending registration, 401 for invalid OTP/credentials/access token, 409 for an existing phone or national code, 429 for resend cooldown or login throttling, and 503 for SMS delivery failure. Login token responses and `/me` use `Cache-Control: no-store`.

Run checks with `go test ./...` and `go vet ./...`. Unit tests use an in-memory repository and fake SMS sender; they send no real messages.

All account types share the SMS sender. For OTPs other than password-reset codes, when the provider accepts a send, the application logs `OTP sent` with the destination and plaintext `code`, including resends. View these entries with `docker compose logs -f api`. Failed sends do not log the code; PostgreSQL still stores only salted OTP hashes.

An opt-in PostgreSQL integration test verifies the full signup â†’ verification â†’ login â†’ JWT authentication flow using a fake SMS sender and rolls back all test fixtures. Run from PowerShell: `$env:RUN_POSTGRES_INTEGRATION='1'; go test ./internal/infrastructure/persistence -run TestPostgresSignupVerificationAndLogin -count=1 -v`. It uses the database in `.env` and runs schema migrations.

To test in Swagger UI, start the server with `go run ./cmd/app` and open <http://localhost:8081/swagger/index.html> (use your `APP_PORT` if different). The UI and its assets are served by the application; no external Swagger account or CDN is needed.

Swagger groups APIs in this order: Job seekers, Parents, Individual employers, Legal employers. Endpoints within each group follow the signup, verification, login and account-management workflow.

Expand each endpoint and click **Try it out**, then **Execute**:

1. Fetch `/job-seekers/signup-options` to see allowed values.
2. Submit `/job-seekers/signup` with your details, phone, and password. This automatically sends a real SMS.
3. Submit `/job-seekers/verify-otp` with the same phone and received code. The code is never returned by the API.
4. Submit `/job-seekers/login` with the same phone and password. Swagger automatically attaches the returned token to subsequent protected requests.
5. Execute `/job-seekers/me`. Manual **Authorize** also accepts either the bare token or `Bearer <access_token>`.
6. Execute `/job-seekers/logout` to revoke the token. Swagger then clears its authorization. Swagger saves authorization in browser local storage and restores it after refresh or reopening the browser until the original JWT expires (one hour by default). Refreshing never extends expiration. Logout and a 401 from a protected endpoint clear saved authorization; expired or malformed tokens are removed before restoration.

Swagger sends requests to the current server and port automatically. The specification is available at `/swagger/doc.json`, with generated JSON/YAML files in `docs/`. After changing endpoint annotations or request/response schemas, regenerate with `go generate ./cmd/app` and restart the server.

Swagger authentication interceptor tests use Node built-in testing: `node --test internal/presentation/swagger/initializer.test.cjs`. The PostgreSQL integration test also checks HTTP login, authenticated requests, logout, replay rejection, and session isolation.

## Job seeker parent approval

After signup OTP verification, `signup_status` remains `pending` until the required approvals are complete. Parent connection alone controls the parent-approval flow: `parent_connected` is derived from the stored `parent_id`, with no internal stage or completion marker. Once a parent is connected, both parent approval APIs reject further requests with HTTP 409 and `{"message":"parent already connected"}`. Ages 15-17 must connect a parent; ages 18-22 can leave it disconnected. Verified job seekers can log in while pending.
1. For an already registered parent, call `POST /v1/job-seekers/parent-approval/registered` with `{"phone":"09129876543"}`. This creates a seven-day request with `connection_status: "pending"`. The signed-in parent must approve it before the connection takes effect. Check the result through `GET /v1/job-seekers/parent-approval/request`. For a new parent, call `POST /v1/job-seekers/parent-approval/invite` with `first_name`, `last_name`, and `phone`. This attempts to text a seven-day invitation code. The response includes `invitation_sms_sent` and the code for manual sharing if delivery fails. Wait 60 seconds before issuing another invitation to the same phone. An existing verified parent can approve requests from multiple children.
2. The parent first calls `POST /v1/parents/invitations/validate` with `{"code":"RETURNED_INVITATION_CODE"}`. It returns the saved first/last name, phone, child's first name and expiry. Invalid, replaced, expired or consumed codes return HTTP 400. Code validation does not consume it; the same code is required for signup. OTP verification uses the saved phone number and OTP instead.
3. Submit `POST /v1/parents/signup` as multipart form data: `code`, `relationship` (`father`, `mother`, or `guardian`), `password` (8-72 bytes), `consent_accepted=true`, `identity_document` (PNG/JPEG, up to 5 MiB) and `consent_video` (MP4/MOV/WebM, 1-20 seconds, up to 50 MiB). The complete request is limited to 56 MiB. Name and phone come only from the invitation and cannot be changed through this form. The documents are private. The API returns HTTP 202 after sending an OTP to the saved parent's phone. No parent account exists yet.
4. Call `POST /v1/parents/verify-otp` with `{"phone":"09129876543","otp":"123456"}`, using the OTP actually received by SMS. The invitation saved with the pending signup is checked internally for validity. It atomically creates the parent, links the job seeker, consumes the invitation and pending signup, and completes the child's parent-connection step. Returns HTTP 201 with public parent identity. Passwords and document keys are not returned.

Parent OTPs expire after five minutes and allow five failed attempts, including malformed codes. Resend using `POST /v1/parents/otp` with `{"code":"RETURNED_INVITATION_CODE"}` after 60 seconds. Pending parent signup lasts 30 minutes, without extending the seven-day invitation expiry. A valid saved invitation is required for every parent signup/OTP operation; parents cannot self-register by phone. Only invitation hashes and salted OTP hashes are stored. Parent passwords are bcrypt hashes. Failed submissions clean up newly uploaded documents, replacements clean up old pending documents after commit, and verification reuses the saved documents. Include the `parents/*` prefix in S3 PutObject/GetObject/DeleteObject permissions. Existing parents do not resubmit documents or repeat signup when another child links them.

### Parent account APIs

| Endpoint | Purpose |
| --- | --- |
| `POST /v1/parents/login` | Sign in with phone and password and obtain a parent JWT. A phone-verified parent can sign in while admin review is pending. |
| `GET /v1/parents/me` | Read the signed-in parent's own profile and current admin review status. |
| `GET /v1/parents/children` | List connected children with their signup status. |
| `GET /v1/parents/connection-requests` | List the signed-in parent's pending requests. |
| `POST /v1/parents/connection-requests/{job_seeker_id}/decision` | Submit `{"approved":true}` or `{"approved":false}`. Only an approval connects the child. |
| `POST /v1/parents/logout` | Revoke the current parent JWT. |
| `POST /v1/parents/forgot-password` | Request a password reset code. |
| `POST /v1/parents/verify-reset-otp` | Verify the code and receive a one-time reset token. |
| `POST /v1/parents/reset-password` | Submit the reset token and new password. A successful reset invalidates all previous parent JWTs. |

Parent JWTs use a distinct role. Login permits five failed attempts per normalized phone in a 15-minute window. Password reset OTPs are isolated from signup OTPs, expire after five minutes, and allow five failed attempts. Startup migrations preserve existing parent connections and add storage for new connection requests, login attempts and reset challenges. Deleting a job seeker clears that user's invitation and connection request while preserving the shared parent account and other children.

## Password reset

Use the same three public JSON endpoints under one of these prefixes: `/v1/job-seekers`, `/v1/parents`, `/v1/individual-employers`, or `/v1/legal-employers`.

1. `POST /forgot-password` with `{"phone":"09123456789"}` requests a six-digit OTP. The response is HTTP 202 for unknown numbers and for repeats inside the 60-second cooldown. No new SMS is sent in either case.
2. `POST /verify-reset-otp` with `{"phone":"09123456789","otp":"123456"}` consumes the OTP and returns HTTP 200 with `reset_token` and `expires_in: 300`. The code expires after five minutes and permits five failed attempts.
3. `POST /reset-password` with `{"reset_token":"RETURNED_TOKEN","new_password":"NewPassword123!"}` sets the password, consumes the token and invalidates existing sessions. The token expires after five minutes and works only for the account type that issued it. Job seeker and parent passwords must be 8–72 bytes. Employer passwords must be 12–72 bytes and include uppercase, lowercase, a digit and a symbol.

Reset OTPs are independent of signup and phone-change codes. Only hashes of reset codes and tokens are stored, and reset codes are kept out of SMS logs. A new OTP request after the cooldown replaces any prior reset challenge or token for that phone and account type. Codes issued before the two-step flow was deployed must be requested again. A job seeker phone change or account deletion invalidates that account's outstanding reset challenge.

## Job seeker phone changes and account deletion

All three endpoints require `Authorization: Bearer <access_token>` from a job seeker login and accept JSON.

| Method and endpoint | JSON body | Success |
| --- | --- | --- |
| `POST /v1/job-seekers/phone-change` | `{"new_phone":"09129876543"}` | 202: OTP sent to the new number |
| `POST /v1/job-seekers/phone-change/verify` | `{"new_phone":"09129876543","otp":"123456"}` | 200: updated account identity |
| `DELETE /v1/job-seekers/me` | `{"password":"your-current-password"}` | 200: `{"message":"Account deleted successfully"}` |

The current phone stays active until the new number is verified. Phone numbers are normalized just as during signup. The new number must differ from the current one and must not belong to another job seeker; availability is checked again during verification. Login uses the new number after success; existing JWT sessions stay valid because they identify the account by ID.

Repeat the phone-change request to resend after 60 seconds. Only the latest destination/code is valid. Each account has its own phone-change challenge, separate from signup: codes expire after five minutes and permit five failed verification attempts. Failed attempts persist, successful verification consumes the code, and changing the destination does not bypass the resend cooldown. Never use the signup OTP endpoints to confirm a phone change.

Deletion requires the current account password. Five failed password confirmations share the existing login throttle (five failures within 15 minutes). Deletion atomically removes the account, its phone-change challenge, and its login-attempt record. Existing access tokens immediately fail subsequent authentication. Its verification video is removed after the database commit; storage cleanup failures are logged for operational follow-up. Deletion is permanent. Startup/migration automatically adds `job_seeker_phone_change_otps` without changing existing profiles.

## Run everything with Docker

Docker Compose runs the Go API, PostgreSQL, and RustFS together. It waits for storage to be healthy, creates the private bucket automatically, and then starts the API. Database migrations run when the API starts. The API image includes `ffprobe` for video validation.

Keep your existing `.env`. For a new installation, copy `.env.docker.example` to `.env` and replace the database password, JWT secret, SMS token, and storage credentials. The JWT secret must be at least 32 bytes. Do not overwrite an existing `.env` containing your credentials.

If the standalone storage service is running, stop it first without deleting its files:

```sh
docker compose -f compose.storage.yaml down
```

Start the complete stack:

```sh
docker compose up -d --build --wait
docker compose ps
```

For local use, open Swagger at <http://127.0.0.1:8081/swagger/index.html> and the storage console at <http://127.0.0.1:9001>. On the VPN server, use <http://192.168.6.217:8081/swagger/index.html> and <http://192.168.6.217:9001>. Set `API_HOST_PORT=8082` in `.env` if port 8081 is already used. The API talks to `db:5432` and `s3:9000` on the Docker network.

PostgreSQL uses a separate persistent `zedlancer_postgres_data` volume; existing host database records are not imported automatically. RustFS reuses `zedlancer-storage_s3_data`, including files from the standalone setup. Use one storage setup at a time. PostgreSQL is accessible only inside the Docker network; the API and storage ports bind to localhost.

```sh
docker compose logs -f api
docker compose down
```

`down` stops the stack and retains data. To permanently delete this stack's database and uploaded files, use `docker compose down --volumes`. Rebuild with `docker compose up -d --build --wait` after code changes. This Compose setup uses local S3 storage; remote provider settings are used when running the backend directly instead.

## Admin login and account approval

Job seeker responses now expose three approval flags: `parent_connected`, `admin_approved` (the job seeker's own admin review), and `admin_approved_parent` (the linked parent's current admin review). Overall `signup_status` is `pending` or `approved`. For ages 15-17, all three flags must be true to be approved. For ages 18-22, only `admin_approved` must be true: a parent connection or parent review is not required. The old `is_approved` field is replaced by these explicit flags and status.

There is no stage tracking in the API, domain, or database. Clients use `parent_connected` for the parent flow, plus `signup_status` and the admin approval flags for acceptance. Admins can review a minor after the parent has connected, in either review order, but overall acceptance waits for both the parent and the child to be admin approved. Adults can be reviewed after signup OTP verification without using parent approval APIs. A shared parent's approval or rejection immediately changes overall acceptance for every linked minor on the next read; it does not overwrite the children's own admin review decisions. Adults are unaffected by their optional parent's review.

Pending/rejected job seekers may still log in and see their status. Detailed `admin_approval_status` (`pending`, `approved`, `rejected`), `admin_review_message`, and `admin_reviewed_at` remain available alongside the new acceptance fields. Parent signup verification returns its initial admin review status.

The `admins` table contains exactly `username` and `password`; `password` stores a bcrypt hash. There is no admin signup or OTP API. Startup creates the table. Run this SQL to add an admin:

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;
INSERT INTO admins (username, password)
VALUES ('admin', crypt('REPLACE_WITH_A_STRONG_PASSWORD', gen_salt('bf', 10)));
```

Admin login accepts any nonempty password up to 72 bytes that matches the stored bcrypt hash. Admin usernames are case-sensitive and limited to 100 bytes. `POST /v1/admin/login` accepts `{"username":"admin","password":"YOUR_PASSWORD"}` and returns an expiring admin JWT. Send it as `Authorization: Bearer <access_token>` on every other admin endpoint. Swagger authorizes automatically after login. User tokens cannot access admin APIs, and admin tokens cannot access user APIs. Five failed login attempts per username within 15 minutes trigger the same temporary lockout policy as user login; attempts are stored separately in `admin_login_attempts`. Removing an admin row invalidates their tokens on subsequent requests.

| Method and endpoint | Purpose |
| --- | --- |
| `GET /v1/admin/job-seekers` | List minors with connected parents and adults; defaults to pending overall acceptance |
| `GET /v1/admin/job-seekers/{id}` | View full job seeker profile and review status |
| `GET /v1/admin/job-seekers/{id}/documents/video` | Get a private five-minute verification-video download URL |
| `POST /v1/admin/job-seekers/{id}/review` | Review a minor with a connected parent, or a verified adult |
| `GET /v1/admin/parents` | List parents; defaults to pending |
| `GET /v1/admin/parents/{id}` | View parent identity and review status |
| `GET /v1/admin/parents/{id}/documents/{kind}` | Get a private five-minute download URL; kind is `identity` or `consent-video` |
| `POST /v1/admin/parents/{id}/review` | Review a parent; their decision also controls linked minors' acceptance |

Job seeker lists accept `status=pending|approved|rejected|all`; `pending` excludes job seekers whose own admin review is rejected; approved minors awaiting parent approval remain pending. `rejected` selects the job seeker's own admin rejection. Parent lists accept `status=pending|approved|rejected|all`; pending excludes rejected parents. Both lists default to `pending`. Use `rejected` or `all` to retrieve rejected accounts. Both accept `limit` (default 20, maximum 100), and nonnegative `offset`. Results include `data` and `total`, ordered oldest first. Review requests require `{"approved":true}` or `{"approved":false,"message":"Please submit a clearer video"}`. A message is optional, with a maximum of 1000 characters; a rejection without one gets a default message. Review responses return the updated account and a success message. Re-reviewing can change an earlier decision. Decisions store the reviewer username and UTC timestamp. Missing accounts return 404; reviewing a minor before a parent is connected returns 409. Passwords and private storage keys are never included in admin responses.

Swagger keeps the existing four sections in order and adds **Admin** at the end, with login followed by each account's list, profile, documents and review operation. Migrations add review fields to job seekers and parents; existing accounts default to `pending` without losing their profiles.

### Employer admin APIs

Use an admin JWT with these endpoints. Use the ID from the employer list for either profile type.

| Endpoint | Purpose |
| --- | --- |
| `GET /v1/admin/employers` | List verified employers with `type=individual\|legal\|all` and `status=pending_review\|approved\|rejected\|all`. Defaults to pending review across both types. Supports `limit` (1–100, default 20) and `offset` (default 0). |
| `GET /v1/admin/employers/{id}` | Get the applicable `individual` or `legal` profile and its `available_documents`. |
| `GET /v1/admin/employers/{id}/documents/{kind}` | Get a private five-minute link for a document listed on the profile. |
| `POST /v1/admin/employers/{id}/review` | Submit `{"approved":true}` to approve or `{"approved":false,"message":"Please submit a clearer document"}` to reject. The message is optional. |

A review records the decision, message, reviewer, and time. Employer `/me` returns the current `review_status`, decision message, and review time. Approved employers can manage job postings. Phone-verified employers can still sign in and check their status while pending or rejected.

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

Required profile fields: first/last/father name, valid national code, gender, Jalali `birth_year`/`birth_month`/`birth_day`, matching province/city, residence address, ten-digit postal code, phone, and password. Business name, business activity, website/social-page URL, email, and `has_legal_guardian` are optional. Passwords require 12â€“72 bytes including uppercase, lowercase, a number, and a symbol. The business license (`Ø¬ÙˆØ§Ø² Ú©Ø³Ø¨`) is optional and is saved as SQL NULL when absent.

`identity_document` is required: PDF, PNG, or JPEG, up to 5 MiB. Optional `business_license` uses the same limits. `verification_video` is required: MP4, MOV, or WebM, 1â€“20 seconds, up to 50 MiB. Uploaded content is inspected on the server; filenames and client Content-Type headers are not sufficient. Install `ffprobe` on PATH or set `FFPROBE_PATH` to its executable. ffmpeg is only needed to generate integration-test video fixtures.

All document bytes and the video are stored behind the `ports.Storage` interface using the S3 adapter. PostgreSQL stores the profile and object references/content types/sizes, not the binary files. Keys use random identifiers and do not contain names, phone numbers, or national codes. S3 uploads use no public-read ACL. Download URLs require authentication, are scoped to the requesting employer, expire after five minutes, and are returned with `Cache-Control: no-store`. Successful registration starts with `review_status=pending_review`; admins can review the employer with the API below. Phone-verified employers may login while review is pending.

### Local S3 storage

The local setup uses [RustFS](https://docs.rustfs.com/en/installation/container), with persistent Docker storage and ports bound to this computer. Set these values in the ignored `.env`, using your own random access key and secret:

```dotenv
S3_BUCKET=zedlancer-local
AWS_REGION=us-east-1
S3_ENDPOINT=http://localhost:9000
S3_DOCKER_ENDPOINT=http://host.docker.internal:9000
S3_PUBLIC_ENDPOINT=http://localhost:9000
S3_USE_PATH_STYLE=true
AWS_ACCESS_KEY_ID=your-local-access-key
AWS_SECRET_ACCESS_KEY=your-random-local-secret
```

Start Docker Desktop, then run from the repository root:

```sh
docker compose -f compose.storage.yaml up -d --wait
go run ./cmd/storage-init
```

The initializer creates the private bucket and can be rerun. The S3 endpoint is <http://localhost:9000>; open the console at <http://localhost:9001> and log in with `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` from `.env`. Run the backend with `go run ./cmd/app`. For the complete Docker stack, follow the Docker instructions above instead; it starts storage and initializes the bucket automatically. `S3_PUBLIC_ENDPOINT` keeps signed browser downloads accessible. Restart a running backend after changing `.env`.

Verify real uploads, private access, signed downloads, and deletion from PowerShell:

```powershell
$env:RUN_LOCAL_S3_INTEGRATION='1'
go test ./internal/infrastructure/storage -run TestLocalS3Integration -count=1 -v
Remove-Item Env:RUN_LOCAL_S3_INTEGRATION
```

Stop storage with `docker compose -f compose.storage.yaml down`. Its named volume retains uploaded files; adding `--volumes` deletes them.

For a remote S3 provider, configure its existing private bucket and credentials instead:

```dotenv
S3_BUCKET=your-private-bucket
AWS_REGION=your-bucket-region
S3_ENDPOINT=
S3_PUBLIC_ENDPOINT=
S3_USE_PATH_STYLE=false
FFPROBE_PATH=ffprobe
```

For a remote provider, unset `S3_DOCKER_ENDPOINT` (or set it to the remote endpoint) and leave `S3_PUBLIC_ENDPOINT` empty to sign downloads using the normal S3 endpoint.

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

Required files are `establishment_document` (establishment notice or official gazette), `identity_document` (representative identity), and `verification_video`. Documents accept PDF/PNG/JPEG up to 5 MiB each; video accepts MP4/MOV/WebM, 1-20 seconds, up to 50 MiB. Set `requires_activity_license=true` and upload `activity_license` when the activity requires licensing. Representative position `attorney` requires `authorization_document` (power of attorney). Optional documents persist as SQL NULL when absent. The signup request is limited to 76 MiB. Terms acceptance, password confirmation, and frontend step navigation remain frontend responsibilities.

In Swagger, fetch signup-options, submit the legal signup form and files once, verify the received code using verify-otp, then login. Codes last five minutes, permit five failed attempts, and pending submissions last 30 minutes. Login issues the configured one-hour token by default. Swagger automatically authorizes and preserves it across refresh; logout revokes it. Legal, individual employer, and job seeker roles cannot access one another's protected routes. Phone verification permits login while `review_status` is `pending_review`; document approval is separate.

Legal uploads use private S3 keys under `legal-employers/`; allow the same PutObject/GetObject/DeleteObject permissions on that prefix as for `individual-employers/`. Existing S3 credentials and ffprobe configuration are reused. Failed submissions clean up new objects, while failed replacements preserve the old submission and its files. Verification uses stored information and never re-uploads files. Unit tests and the opt-in PostgreSQL integration flow use fake SMS/storage, real bcrypt/JWT and ffprobe, and roll back database fixtures.

Admin parent list, detail, and review responses include `child_ids`, an array of connected job seeker IDs (for example, `"child_ids": [12, 34]`). Parents with no connected children return `"child_ids": []`.
